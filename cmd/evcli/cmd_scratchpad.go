package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/exograd/go-program"
)

func addScratchpadCommands() {
	var c *program.Command

	addCommonOptions := func(c *program.Command) {
		c.AddOption("", "pipeline-id", "id", "",
			"the identifier of the pipeline")
	}

	scratchpadCmd := func(f func(*program.Program, string)) func(*program.Program) {
		return func(p *program.Program) {
			app.IdentifyCurrentProject()

			id := os.Getenv("EVENTLINE_PIPELINE_ID")
			if id == "" {
				id = p.OptionValue("pipeline-id")
			}

			if id == "" {
				p.Error("missing pipeline id")
				p.Info("\nTo identify the scratchpad, you must provide a " +
					"pipeline id. You can either use the --pipeline-id " +
					"option or set the EVENTLINE_PIPELINE_ID environment " +
					"variable.")
				os.Exit(1)
			}

			f(p, id)
		}
	}

	// show-scratchpad
	c = p.AddCommand("show-scratchpad", "list scratchpad entries",
		scratchpadCmd(cmdShowScratchpad))

	addCommonOptions(c)

	// clear-scratchpad
	c = p.AddCommand("clear-scratchpad",
		"delete all entries in the scratchpad",
		scratchpadCmd(cmdClearScratchpad))

	addCommonOptions(c)

	// get-scratchpad-entry
	c = p.AddCommand("get-scratchpad-entry",
		"get the value of a scratchpad entry",
		scratchpadCmd(cmdGetScratchpadEntry))

	addCommonOptions(c)

	c.AddArgument("key", "the key of the entry")

	// set-scratchpad-entry
	c = p.AddCommand("set-scratchpad-entry",
		"set the value of a scratchpad entry",
		scratchpadCmd(cmdSetScratchpadEntry))

	addCommonOptions(c)

	c.AddArgument("key", "the key of the entry")
	c.AddArgument("value", "the value of the entry")

	// delete-scratchpad-entry
	c = p.AddCommand("delete-scratchpad-entry", "delete a scratchpad entry",
		scratchpadCmd(cmdDeleteScratchpadEntry))

	addCommonOptions(c)

	c.AddArgument("key", "the key of the entry")
}

func cmdShowScratchpad(p *program.Program, id string) {
	entries, err := app.Client.GetScratchpad(id)
	if err != nil {
		p.Fatal("cannot fetch scratchpad: %v", err)
	}

	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i] <= keys[j]
	})

	table := NewTable([]string{"key", "value"})
	for _, key := range keys {
		table.AddRow([]interface{}{key, entries[key]})
	}
	table.Write()
}

func cmdClearScratchpad(p *program.Program, id string) {
	if err := app.Client.ClearScratchpad(id); err != nil {
		p.Fatal("cannot clear scratchpad: %v", err)
	}

	p.Info("scratchpad cleared")
}

func cmdGetScratchpadEntry(p *program.Program, id string) {
	key := p.ArgumentValue("key")

	value, err := app.Client.GetScratchpadEntry(id, key)
	if err != nil {
		p.Fatal("cannot fetch scratchpad entry: %v", err)
	}

	fmt.Print(value)
}

func cmdSetScratchpadEntry(p *program.Program, id string) {
	key := p.ArgumentValue("key")
	value := p.ArgumentValue("value")

	if err := app.Client.SetScratchpadEntry(id, key, value); err != nil {
		p.Fatal("cannot set scratchpad entry: %v", err)
	}

	p.Info("scratchpad entry %q set", key)
}

func cmdDeleteScratchpadEntry(p *program.Program, id string) {
	key := p.ArgumentValue("key")

	if err := app.Client.DeleteScratchpadEntry(id, key); err != nil {
		p.Fatal("cannot delete scratchpad entry: %v", err)
	}

	p.Info("scratchpad entry %q deleted", key)
}
