package service

import (
	"fmt"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/log"
	"go.n16f.net/service/pkg/pg"
)

type NotificationWorker struct {
	Log     *log.Logger
	Service *Service
}

func NewNotificationWorker(s *Service) *NotificationWorker {
	return &NotificationWorker{
		Service: s,
	}
}

func (nw *NotificationWorker) Init(w *eventline.Worker) {
	nw.Log = w.Log
}

func (nw *NotificationWorker) Start() error {
	return nil
}

func (nw *NotificationWorker) Stop() {
}

func (nw *NotificationWorker) ProcessJob() (bool, error) {
	var processingErr error
	var processed bool

	err := nw.Service.Pg.WithTx(func(conn pg.Conn) error {
		notification, err := eventline.LoadNotificationForDelivery(conn)
		if err != nil {
			return fmt.Errorf("cannot load notification: %w", err)
		} else if notification == nil {
			return nil
		}

		nw.Log.Info("processing notification %q", notification.Id)

		deliveryErr := nw.Service.DeliverNotification(conn, notification)

		if deliveryErr == nil {
			if err := notification.Delete(conn); err != nil {
				return fmt.Errorf("cannot delete notification %q: %w",
					notification.Id, err)
			}
		} else {
			nw.Log.Error("cannot process notification: %v", deliveryErr)

			now := time.Now().UTC()

			var deliveryDelay int
			switch {
			case notification.DeliveryDelay == 0:
				deliveryDelay = 5
			case notification.DeliveryDelay < 40:
				deliveryDelay = notification.DeliveryDelay * 2
			case notification.DeliveryDelay >= 40:
				deliveryDelay = 60
			}

			deliveryDelayDuration := time.Duration(deliveryDelay) * time.Second

			notification.DeliveryDelay = deliveryDelay
			notification.NextDeliveryTime = now.Add(deliveryDelayDuration)

			if err := notification.Update(conn); err != nil {
				return fmt.Errorf("cannot update notification %q: %w",
					notification.Id, err)
			}
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
