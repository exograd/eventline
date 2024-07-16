package github

import (
	"context"
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/service/pkg/pg"
	"github.com/google/go-github/v45/github"
)

// GitHub limits the number of hooks which can be created on an organization
// or repository. We want to be able to create an unlimited number of jobs
// subscribing to the same organization or repository. So we create wildcard
// hooks (i.e. hooks listening for all events) and share them between
// subscriptions.

func (c *Connector) MaybeCreateHook(conn pg.Conn, params *Parameters, identity *eventline.Identity) (*HookId, error) {
	if err := LockHooks(conn); err != nil {
		return nil, err
	}

	hookId, err := LoadHookIdByParameters(conn, params)
	if err != nil {
		return nil, fmt.Errorf("cannot load hook id: %w", err)
	}

	if hookId != nil {
		return hookId, nil
	}

	return c.CreateHook(conn, params, identity)
}

func (c *Connector) CreateHook(conn pg.Conn, params *Parameters, identity *eventline.Identity) (*HookId, error) {
	client, err := c.NewClient(identity)
	if err != nil {
		return nil, fmt.Errorf("cannot create client: %w", err)
	}

	ctx := context.Background()

	active := true

	if params.Repository == "" {
		svc := client.Organizations

		hook := github.Hook{
			Name:   github.String("web"),
			Active: &active,
			Events: []string{"*"},
			Config: map[string]interface{}{
				"url":          c.WebhookURI(params),
				"content_type": "json",
				"secret":       c.Cfg.WebhookSecret,
			},
		}

		hook2, _, err := svc.CreateHook(ctx, params.Organization, &hook)
		if err != nil {
			return nil, err
		}

		return hook2.ID, nil
	} else {
		svc := client.Repositories

		hook := github.Hook{
			Active: &active,
			Events: []string{"*"},
			Config: map[string]interface{}{
				"url":          c.WebhookURI(params),
				"content_type": "json",
				"secret":       c.Cfg.WebhookSecret,
			},
		}

		hook2, _, err := svc.CreateHook(ctx, params.Organization,
			params.Repository, &hook)
		if err != nil {
			return nil, err
		}

		return hook2.ID, nil
	}
}

func (c *Connector) MaybeDeleteHook(conn pg.Conn, params *Parameters, identity *eventline.Identity, hookId HookId) error {
	if err := LockHooks(conn); err != nil {
		return err
	}

	n, err := CountSubscriptionsByHookId(conn, hookId)
	if err != nil {
		return fmt.Errorf("cannot count subscriptions: %w", err)
	}

	if n != 1 {
		return nil
	}

	return c.DeleteHook(conn, params, identity, hookId)
}

func (c *Connector) DeleteHook(conn pg.Conn, params *Parameters, identity *eventline.Identity, hookId HookId) error {
	client, err := c.NewClient(identity)
	if err != nil {
		return fmt.Errorf("cannot create client: %w", err)
	}

	ctx := context.Background()

	if params.Repository == "" {
		svc := client.Organizations

		_, err = svc.DeleteHook(ctx, params.Organization, hookId)
	} else {
		svc := client.Repositories

		_, err = svc.DeleteHook(ctx, params.Organization, params.Repository,
			hookId)
	}

	if err != nil {
		// If the hook does not exist, it may have been deleted manually by
		// the user. It sounds obvious, but another possible explaination is
		// that the identity used does not have permission to access the hook
		// anymore (GitHub uses 404 errors in that case, nothing we can
		// do about it). In both cases, we do not want the subscription to be
		// stuck in a terminating loop.

		if IsNotFoundAPIError(err) {
			return nil
		}

		return nil
	}

	return nil
}

func LockHooks(conn pg.Conn) error {
	id1 := PgAdvisoryLockId1
	id2 := PgAdvisoryLockId2GitHubHooks

	if err := pg.TakeAdvisoryTxLock(conn, id1, id2); err != nil {
		return fmt.Errorf("cannot take advisory lock: %w", err)
	}

	return nil
}
