package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/exograd/go-program"
)

func addEventCommands() {
	var c *program.Command

	// create-event
	c = p.AddCommand("create-event", "create a new custom event",
		cmdCreateEvent)

	c.AddOption("t", "event-time", "timestamp", "",
		"the date and time the event occurred (RFC 3339 format)")

	c.AddArgument("connector", "the name of the connector")
	c.AddArgument("event", "the name of the event")
	c.AddArgument("data",
		"the JSON object representing event data (\"-\" to read stdin)")

	// replay-event
	c = p.AddCommand("replay-event", "replay an existing event",
		cmdReplayEvent)

	c.AddArgument("event-id", "the identifier of the event")
}

func cmdCreateEvent(p *program.Program) {
	app.IdentifyCurrentProject()

	var eventTime time.Time
	if p.IsOptionSet("event-time") {
		eventTimeString := p.OptionValue("event-time")

		var err error
		eventTime, err = time.Parse(time.RFC3339, eventTimeString)
		if err != nil {
			p.Fatal("invalid event time: %v", err)
		}
	} else {
		eventTime = time.Now().UTC()
	}

	connector := p.ArgumentValue("connector")
	name := p.ArgumentValue("event")

	data := []byte(p.ArgumentValue("data"))
	if len(data) == 1 && data[0] == '-' {
		var err error
		data, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			p.Fatal("cannot read stdin: %v", err)
		}
	}

	newEvent := NewEvent{
		EventTime: &eventTime,
		Connector: connector,
		Name:      name,
		Data:      data,
	}

	_, err := app.Client.CreateEvent(&newEvent)
	if err != nil {
		p.Fatal("cannot create event: %v", err)
	}

	p.Info("events created")
}

func cmdReplayEvent(p *program.Program) {
	app.IdentifyCurrentProject()

	EventId := p.ArgumentValue("event-id")

	event, err := app.Client.ReplayEvent(EventId)
	if err != nil {
		p.Fatal("cannot replay event: %v", err)
	}

	p.Info("event %s created", event.Id)

	fmt.Printf("%s\n", event.Id)
}
