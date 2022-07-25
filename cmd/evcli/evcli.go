package main

import (
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-program"
)

var (
	p   *program.Program
	app *App

	buildId string

	skipConfirmations bool
	colorOutput       bool
)

func main() {
	// Command line
	p = program.NewProgram("evcli", "client for the eventline service")

	p.AddFlag("y", "yes", "skip all confirmations")
	p.AddFlag("", "no-color", "do not use colors")

	p.AddOption("", "project-id", "id", "",
		"the identifier of the current project")
	p.AddOption("p", "project-name", "name", "",
		"the name of the current project")

	addConfigCommands()
	addLoginCommand()
	addUpdateCommand()
	addProjectCommands()
	addEventCommands()
	addJobCommands()
	addJobExecutionCommands()

	p.AddCommand("version", "print the version of evcli and exit", cmdVersion)

	p.ParseCommandLine()

	// Config
	skipConfirmations = p.IsOptionSet("yes")

	config, err := LoadConfig()
	if err != nil {
		p.Fatal("cannot load configuration: %v", err)
	}

	colorOutput = config.Interface.Color && !p.IsOptionSet("no-color")

	// Application
	client, err := NewClient(config)
	if err != nil {
		p.Fatal("cannot create api client: %v", err)
	}

	optionValue := func(name string) *string {
		if !p.IsOptionSet(name) {
			return nil
		}

		value := p.OptionValue(name)
		return &value
	}

	app, err = NewApp(config, client)
	if err != nil {
		p.Fatal("%v", err)
	}

	if projectIdString := optionValue("project-id"); projectIdString != nil {
		var projectId eventline.Id
		if err := projectId.Parse(*projectIdString); err != nil {
			p.Fatal("invalid project id %q: %v", *projectIdString, err)
		}

		app.projectIdOption = &projectId
	}

	app.projectNameOption = optionValue("project-name")

	name := p.CommandName()

	loadAPIKey := true
	for _, cmdName := range noAPIKeyCommands() {
		if name == cmdName {
			loadAPIKey = false
			break
		}
	}

	if loadAPIKey {
		app.LoadAPIKey()
	}

	if !config.Misc.DisableUpdateCheck && name != "update" {
		app.LookForLastBuild()
	}

	p.Run()
}

func noAPIKeyCommands() []string {
	return []string{
		"get-config",
		"help",
		"login",
		"set-config",
		"show-config",
		"update",
		"version",
	}
}
