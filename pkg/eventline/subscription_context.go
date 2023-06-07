package eventline

import (
	"fmt"
	"time"

	"github.com/galdor/go-service/pkg/pg"
)

type SubscriptionContext struct {
	Subscription *Subscription
	Identity     *Identity
	Job          *Job

	Scope Scope
}

func (sctx *SubscriptionContext) Load(conn pg.Conn, subscription *Subscription) error {
	if subscription.ProjectId == nil {
		// If the project has been deleted, the identity does not have a
		// project id anymore, so we have to look for it in the null project
		// scope.
		sctx.Scope = NewNullProjectScope()
	} else {
		sctx.Scope = NewProjectScope(*subscription.ProjectId)
	}

	sctx.Subscription = subscription

	if id := subscription.IdentityId; id != nil {
		var identity Identity
		if err := identity.LoadForUpdate(conn, *id, sctx.Scope); err != nil {
			return fmt.Errorf("cannot load identity: %w", err)
		}

		now := time.Now().UTC()
		identity.LastUseTime = &now

		if err := identity.UpdateLastUseTime(conn); err != nil {
			return fmt.Errorf("cannot update identity: %w", err)
		}

		sctx.Identity = &identity
	}

	if id := subscription.JobId; id != nil {
		var job Job
		if err := job.Load(conn, *id, sctx.Scope); err != nil {
			return fmt.Errorf("cannot load job: %w", err)
		}

		sctx.Job = &job
	}

	return nil
}
