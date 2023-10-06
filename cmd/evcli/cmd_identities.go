package main

import (
	"github.com/galdor/go-program"
)

func addIdentityCommands() {
	var c *program.Command

	// // list-identities
	c = p.AddCommand("list-identities", "list all identities",
		cmdListIdentities)

	// delete-identity
	c = p.AddCommand("delete-identity", "delete an identity",
		cmdDeleteIdentity)

	c.AddArgument("name", "the name of the identity")
}

func cmdListIdentities(p *program.Program) {
	app.IdentifyCurrentProject()

	identities, err := app.Client.FetchIdentities()
	if err != nil {
		p.Fatal("cannot fetch identities: %v", err)
	}

	header := []string{"id", "name", "type", "status"}
	table := NewTable(header)

	for _, i := range identities {
		typeString := i.Connector + "/" + i.Type

		row := []interface{}{
			i.Id,
			i.Name,
			typeString,
			i.Status,
		}

		table.AddRow(row)
	}

	table.Write()
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
