package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/exograd/evgo/pkg/eventline"
	"github.com/exograd/go-daemon/pg"
	"github.com/exograd/go-log"
)

type SubscriptionWorker struct {
	Log     *log.Logger
	Service *Service
}

func NewSubscriptionWorker(s *Service) *SubscriptionWorker {
	return &SubscriptionWorker{
		Service: s,
	}
}

func (sw *SubscriptionWorker) Init(w *eventline.Worker) {
	sw.Log = w.Log
}

func (sw *SubscriptionWorker) Start() error {
	return nil
}

func (sw *SubscriptionWorker) Stop() {
}

func (sw *SubscriptionWorker) ProcessJob() (bool, error) {
	var processingErr error
	var processed bool

	err := sw.Service.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		subscription, err := eventline.LoadSubscriptionForProcessing(conn)
		if err != nil {
			return fmt.Errorf("cannot load subscription: %w", err)
		} else if subscription == nil {
			return nil
		}

		sw.Log.Info("processing subscription %q", subscription.Id)

		var sctx eventline.SubscriptionContext
		if err := sctx.Load(conn, subscription); err != nil {
			return err
		}

		err = sw.Service.ProcessSubscription(conn, &sctx)

		var externalErr *eventline.ExternalSubscriptionError
		isExternalErr := errors.As(err, &externalErr)

		if err != nil && !isExternalErr {
			return fmt.Errorf("cannot process subscription: %w", err)
		}

		if err != nil && isExternalErr {
			processingErr = externalErr.Err

			sw.Log.Error("cannot process subscription: %v", err)

			now := time.Now().UTC()

			var updateDelay int
			switch {
			case subscription.UpdateDelay == 0:
				updateDelay = 5
			case subscription.UpdateDelay < 40:
				updateDelay = subscription.UpdateDelay * 2
			case subscription.UpdateDelay >= 40:
				updateDelay = 60
			}

			updateDelayDuration := time.Duration(updateDelay) * time.Second
			nextUpdate := now.Add(updateDelayDuration)

			subscription.UpdateDelay = updateDelay
			subscription.LastUpdate = &now
			subscription.NextUpdate = &nextUpdate
		}

		if err := subscription.Update(conn); err != nil {
			return fmt.Errorf("cannot update subscription %q: %w",
				subscription.Id, err)
		}

		processed = true
		return nil
	})
	if err != nil {
		return false, err
	}

	if processingErr != nil {
		return false, processingErr
	}

	return processed, nil
}
