package main

import (
	"fmt"
	"os"

	"go.n16f.net/program"
	"github.com/peterh/liner"
)

type loginInfo struct {
	Endpoint string
	Username string
	Password string
}

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

	line.SetCtrlCAborts(true)

	// Careful here: we introduce a separate function even though we do not
	// really need it to be sure to always call line.Close() on error. Using
	// defer here would not work because defer statements are not called on
	// exit.
	info, err := promptLoginInfo(line)
	if err != nil {
		line.Close()
		p.Fatal("%v", err)
	}

	line.Close()

	// Login
	if err := app.Client.SetEndpoint(info.Endpoint); err != nil {
		p.Fatal("invalid endpoint: %v", err)
	}

	p.Info("logging in to %s", info.Endpoint)

	res, err := app.Client.LogIn(info.Username, info.Password)
	if err != nil {
		p.Fatal("cannot log in: %v", err)
	}

	p.Info("login successful")

	p.Info("api key %q created", res.APIKey.Name)

	// Update the configuration file
	app.Config.API.Endpoint = info.Endpoint
	app.Config.API.Key = res.Key

	if err := app.Config.Write(); err != nil {
		p.Fatal("%v", err)
	}

	p.Info("configuration file updated")
}

func promptLoginInfo(line *liner.State) (*loginInfo, error) {
	cfg := app.Config

	defaultEndpoint := "http://localhost:8085"
	if cfg.API.Endpoint != "" {
		defaultEndpoint = cfg.API.Endpoint
	}

	endpoint, err := line.PromptWithSuggestion("API endpoint: ",
		defaultEndpoint, -1)
	if err != nil {
		return nil, fmt.Errorf("cannot read endpoint: %w", err)
	}

	username, err := line.PromptWithSuggestion("Username: ", "admin", -1)
	if err != nil {
		return nil, fmt.Errorf("cannot read username: %w", err)
	}

	password, err := line.PasswordPrompt("Password: ")
	if err != nil {
		return nil, fmt.Errorf("cannot read password: %w", err)
	}

	info := loginInfo{
		Endpoint: endpoint,
		Username: username,
		Password: password,
	}

	return &info, nil
}
