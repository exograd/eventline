package main

import (
	"github.com/galdor/go-program"
)

func addIdentityCommands() {
	var c *program.Command

	// delete-identity
	c = p.AddCommand("delete-identity", "delete an identity",
		cmdDeleteIdentity)

	c.AddArgument("name", "the name of the identity")
}

func cmdDeleteIdentity(p *program.Program) {
	app.IdentifyCurrentProject()

	name := p.ArgumentValue("name")

	identity, err := app.Client.FetchIdentityByName(name)
	if err != nil {
		p.Fatal("cannot fetch identity: %v", err)
	}

	if err := app.Client.DeleteIdentity(identity.Id.String()); err != nil {
		p.Fatal("cannot delete identity: %v", err)
	}

	p.Info("identity %q deleted", identity.Id)
}
