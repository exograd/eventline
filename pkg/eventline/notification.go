package eventline

import "time"

type Notification struct {
	Id               Id
	ProjectId        Id
	Recipients       []string
	Message          []byte
	NextDeliveryTime time.Time
	DeliveryDelay    int // seconds
}
