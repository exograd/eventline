package main

import (
	"fmt"

	"github.com/galdor/go-program"
)

func addEventCommands() {
	var c *program.Command

	// replay-event
	c = p.AddCommand("replay-event", "replay an existing event",
		cmdReplayEvent)

	c.AddArgument("event-id", "the identifier of the event")
}

func cmdReplayEvent(p *program.Program) {
	app.IdentifyCurrentProject()

	eventId := p.ArgumentValue("event-id")

	event, err := app.Client.ReplayEvent(eventId)
	if err != nil {
		p.Fatal("cannot replay event: %v", err)
	}

	p.Info("event %s created", event.Id)

	fmt.Printf("%s\n", event.Id)
}
