package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/exograd/go-program"
)

func addConfigCommands() {
	var c *program.Command

	// show-config
	c = p.AddCommand("show-config", "print the configuration",
		cmdShowConfig)

	c.AddFlag("e", "entries",
		"show a list of entries instead of the entire configuration")

	// get-config
	c = p.AddCommand("get-config",
		"extract a value from the configuration and print it",
		cmdGetConfig)

	c.AddArgument("name", "the name of the entry")

	// set-config
	c = p.AddCommand("set-config", "set a value in the configuration",
		cmdSetConfig)

	c.AddArgument("name", "the name of the entry")
	c.AddArgument("value", "the value of the entry")
}

func cmdShowConfig(p *program.Program) {
	if p.IsOptionSet("entries") {
		var names []string
		for _, e := range ConfigEntries {
			names = append(names, e.Name)
		}

		table := NewTable([]string{"name", "value"})
		for _, name := range names {
			value, err := app.Config.GetEntry(name)
			if err != nil {
				p.Error("cannot read entry %q: %v", name, err)
				continue
			}

			table.AddRow([]interface{}{name, value})
		}

		table.Write()
	} else {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		if err := encoder.Encode(app.Config); err != nil {
			p.Fatal("cannot encode configuration: %v", err)
		}
	}
}

func cmdGetConfig(p *program.Program) {
	name := p.ArgumentValue("name")

	value, err := app.Config.GetEntry(name)
	if err != nil {
		p.Fatal("%v", err)
	}

	fmt.Printf("%s\n", value)
}

func cmdSetConfig(p *program.Program) {
	name := p.ArgumentValue("name")
	value := p.ArgumentValue("value")

	if err := app.Config.SetEntry(name, value); err != nil {
		p.Fatal("%v", err)
	}

	if err := app.Config.Write(); err != nil {
		p.Fatal("%v", err)
	}
}
