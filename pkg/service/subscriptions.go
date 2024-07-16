package service

import (
	"fmt"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/service/pkg/pg"
)

func (s *Service) CreateSubscription(conn pg.Conn, job *eventline.Job, scope eventline.Scope) (*eventline.Subscription, error) {
	projectId := scope.(*eventline.ProjectScope).ProjectId

	triggerData := job.Spec.Trigger

	var identityId *eventline.Id

	if name := triggerData.Identity; name != "" {
		id, err := eventline.LoadIdentityIdByName(conn, name, scope)
		if err != nil {
			return nil, fmt.Errorf("cannot load identity %q: %w", name, err)
		}

		identityId = &id
	}

	now := time.Now().UTC()

	subscription := eventline.Subscription{
		Id:             eventline.GenerateId(),
		ProjectId:      &projectId,
		JobId:          &job.Id,
		IdentityId:     identityId,
		Connector:      triggerData.Event.Connector,
		Event:          triggerData.Event.Event,
		Parameters:     triggerData.Parameters, // should be a deep copy
		CreationTime:   now,
		Status:         eventline.SubscriptionStatusInactive,
		NextUpdateTime: &now,
	}

	if err := subscription.Insert(conn); err != nil {
		return nil, fmt.Errorf("cannot insert subscription: %w", err)
	}

	if err := subscription.UpdateOp(conn); err != nil {
		return nil, fmt.Errorf("cannot update subscription op: %w", err)
	}

	return &subscription, nil
}

func (s *Service) TerminateSubscription(conn pg.Conn, subscription *eventline.Subscription, projectDeletion bool, scope eventline.Scope) error {
	if subscription.Status == eventline.SubscriptionStatusInactive {
		if err := subscription.Delete(conn); err != nil {
			return fmt.Errorf("cannot delete subscription: %w", err)
		}

		return nil
	}

	if subscription.Status == eventline.SubscriptionStatusTerminating {
		return nil
	}

	now := time.Now().UTC()

	if projectDeletion {
		subscription.ProjectId = nil
	}
	subscription.JobId = nil
	subscription.Status = eventline.SubscriptionStatusTerminating
	subscription.UpdateDelay = 0
	subscription.NextUpdateTime = &now

	if err := subscription.Update(conn); err != nil {
		return fmt.Errorf("cannot update subscription: %w", err)
	}

	if err := subscription.UpdateOp(conn); err != nil {
		return fmt.Errorf("cannot update subscription op: %w", err)
	}

	return nil
}

func (s *Service) ProcessSubscription(conn pg.Conn, sctx *eventline.SubscriptionContext) error {
	switch sctx.Subscription.Status {
	case eventline.SubscriptionStatusInactive:
		return s.processInactiveSubscription(conn, sctx)

	case eventline.SubscriptionStatusTerminating:
		return s.processTerminatingSubscription(conn, sctx)
	}

	return fmt.Errorf("unexpected status %q", sctx.Subscription.Status)
}

func (s *Service) processInactiveSubscription(conn pg.Conn, sctx *eventline.SubscriptionContext) error {
	c := eventline.GetConnector(sctx.Subscription.Connector)

	if c2, ok := c.(eventline.SubscribableConnector); ok {
		if err := c2.Subscribe(conn, sctx); err != nil {
			return fmt.Errorf("cannot subscribe to event: %w", err)
		}
	}

	now := time.Now().UTC()

	sctx.Subscription.Status = eventline.SubscriptionStatusActive
	sctx.Subscription.UpdateDelay = 0
	sctx.Subscription.LastUpdateTime = &now
	sctx.Subscription.NextUpdateTime = nil

	return nil
}

func (s *Service) processTerminatingSubscription(conn pg.Conn, sctx *eventline.SubscriptionContext) error {
	c := eventline.GetConnector(sctx.Subscription.Connector)

	if c2, ok := c.(eventline.SubscribableConnector); ok {
		if err := c2.Unsubscribe(conn, sctx); err != nil {
			return fmt.Errorf("cannot unsubscribe from event: %w", err)
		}
	}

	if err := sctx.Subscription.Delete(conn); err != nil {
		return fmt.Errorf("cannot delete subscription: %w", err)
	}

	// If the project does not exist anymore and if it was the last
	// subscription referencing this identity, we can delete it.
	if sctx.Identity != nil && sctx.Subscription.ProjectId == nil {
		used, err := sctx.Identity.IsUsedBySubscription(conn)
		if err != nil {
			return fmt.Errorf("cannot check identity usage: %w", err)
		}

		if !used {
			if err := sctx.Identity.Delete(conn); err != nil {
				return fmt.Errorf("cannot delete identity: %w", err)
			}
		}
	}

	return nil
}
