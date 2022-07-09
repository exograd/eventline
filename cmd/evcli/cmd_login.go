package main

import (
	"os"

	"github.com/exograd/go-program"
	"github.com/peterh/liner"
)

func addLoginCommand() {
	var c *program.Command

	// login
	c = p.AddCommand("login", "log in and create an API key",
		cmdLogin)

	c.AddFlag("f", "force",
		"replace the current api key if there is one")
}

func cmdLogin(p *program.Program) {
	force := p.IsOptionSet("force")

	cfg := app.Config

	if cfg.API.Key != "" {
		if !force {
			p.Error("API key already set")
			p.Info("\nThe Evcli configuration file already contains an " +
				"API key.\nYou can either delete it manually or use the " +
				"--force option to replace it.")
			os.Exit(1)
		}
	}

	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	// Endpoint
	defaultEndpoint := "http://localhost:8085"
	if cfg.API.Endpoint != "" {
		defaultEndpoint = cfg.API.Endpoint
	}

	endpoint, err := line.PromptWithSuggestion("API endpoint: ",
		defaultEndpoint, -1)
	if err != nil {
		p.Fatal("cannot read endpoint: %v", err)
	}

	// Username
	username, err := line.PromptWithSuggestion("Username: ", "admin", -1)
	if err != nil {
		p.Fatal("cannot read username: %v", err)
	}

	// Password
	password, err := line.PasswordPrompt("Password: ")
	if err != nil {
		p.Fatal("cannot read password: %v", err)
	}

	// Login
	if err := app.Client.SetEndpoint(endpoint); err != nil {
		p.Fatal("invalid endpoint: %v", err)
	}

	p.Info("logging in to %s", endpoint)

	res, err := app.Client.LogIn(username, password)
	if err != nil {
		p.Fatal("cannot log in: %v", err)
	}

	p.Info("login successful")

	p.Info("api key %q created", res.APIKey.Name)

	// Update the configuration file
	app.Config.API.Endpoint = endpoint
	app.Config.API.Key = res.Key

	if err := app.Config.Write(); err != nil {
		p.Fatal("%v", err)
	}

	p.Info("configuration file updated")
}
