package eventline

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type UnknownEventDefError struct {
	Connector string
	Name      string
}

func (err UnknownEventDefError) Error() string {
	return fmt.Sprintf("unknown event %q in connector %q",
		err.Name, err.Connector)
}

type EventDef struct {
	Name string

	Data                   EventData
	SubscriptionParameters SubscriptionParameters
}

type EventData interface {
}

func NewEventDef(name string, data EventData, subscriptionParameters SubscriptionParameters) *EventDef {
	return &EventDef{
		Name: name,

		Data:                   data,
		SubscriptionParameters: subscriptionParameters,
	}
}

func (edef *EventDef) DecodeSubscriptionParameters(data []byte) (SubscriptionParameters, error) {
	eparams := reflect.New(reflect.TypeOf(edef.SubscriptionParameters).Elem()).Interface()
	if err := json.Unmarshal(data, eparams); err != nil {
		return nil, err
	}

	return eparams.(SubscriptionParameters), nil
}

func (edef *EventDef) DecodeData(data []byte) (EventData, error) {
	edata := reflect.New(reflect.TypeOf(edef.Data).Elem()).Interface()
	if err := json.Unmarshal(data, edata); err != nil {
		return nil, err
	}

	return edata.(EventData), nil
}
