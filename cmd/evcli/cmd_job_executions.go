package main

import (
	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/program"
)

func addJobExecutionCommands() {
	var c *program.Command

	// abort-job-execution
	c = p.AddCommand("abort-job-execution",
		"abort a created or started job execution.",
		cmdAbortJobExecution)

	c.AddArgument("job-execution-id", "the identifier of the job execution")

	// restart-job-execution
	c = p.AddCommand("restart-job-execution",
		"restart a finished job execution.",
		cmdRestartJobExecution)

	c.AddArgument("job-execution-id", "the identifier of the job execution")
}

func cmdAbortJobExecution(p *program.Program) {
	app.IdentifyCurrentProject()

	jeIdString := p.ArgumentValue("job-execution-id")

	var jeId eventline.Id
	if err := jeId.Parse(jeIdString); err != nil {
		p.Fatal("invalid id %q: %w", jeIdString, err)
	}

	if err := app.Client.AbortJobExecution(jeId); err != nil {
		p.Fatal("cannot abort job execution: %v", err)
	}

	p.Info("job execution %q aborted", jeId)
}

func cmdRestartJobExecution(p *program.Program) {
	app.IdentifyCurrentProject()

	jeIdString := p.ArgumentValue("job-execution-id")

	var jeId eventline.Id
	if err := jeId.Parse(jeIdString); err != nil {
		p.Fatal("invalid id %q: %w", jeIdString, err)
	}

	if err := app.Client.RestartJobExecution(jeId); err != nil {
		p.Fatal("cannot restart job execution: %v", err)
	}

	p.Info("job execution %q restarted", jeId)
}
