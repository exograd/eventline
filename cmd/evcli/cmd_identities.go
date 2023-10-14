package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/galdor/go-program"
)

func addIdentityCommands() {
	var c *program.Command

	// list-identities
	c = p.AddCommand("list-identities", "list all identities",
		cmdListIdentities)

	// create-identity
	c = p.AddCommand("create-identity", "create a new identity",
		cmdCreateIdentity)

	c.AddArgument("name", "the name of the identity")
	c.AddArgument("connector", "the name of the connector")
	c.AddArgument("type", "the type of the identity")
	c.AddTrailingArgument("field",
		"key/value identity data fields represented as \"<key>=<value>\" "+
			"arguments")

	// update-identity
	c = p.AddCommand("update-identity", "update an identity",
		cmdUpdateIdentity)

	c.AddArgument("name", "the name of the identity")
	c.AddArgument("connector", "the name of the connector")
	c.AddArgument("type", "the type of the identity")
	c.AddTrailingArgument("field",
		"key/value identity data fields represented as \"<key>=<value>\" "+
			"arguments")

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

func cmdCreateIdentity(p *program.Program) {
	app.IdentifyCurrentProject()

	name := p.ArgumentValue("name")
	connector := p.ArgumentValue("connector")
	itype := p.ArgumentValue("type")
	fieldStrings := p.TrailingArgumentValues("field")

	data, err := ParseIdentityFields(fieldStrings)
	if err != nil {
		p.Fatal("invalid fields: %v", err)
	}

	encodedData, err := json.Marshal(data)
	if err != nil {
		p.Fatal("cannot encode field data: %v", err)
	}

	newIdentity := eventline.RawNewIdentity{
		Name:      name,
		Connector: connector,
		Type:      itype,
		RawData:   encodedData,
	}

	identity, err := app.Client.CreateIdentity(&newIdentity)
	if err != nil {
		p.Fatal("cannot create identity: %v", err)
	}

	p.Info("identity %q created", identity.Name)
}

func cmdUpdateIdentity(p *program.Program) {
	app.IdentifyCurrentProject()

	name := p.ArgumentValue("name")
	connector := p.ArgumentValue("connector")
	itype := p.ArgumentValue("type")
	fieldStrings := p.TrailingArgumentValues("field")

	data, err := ParseIdentityFields(fieldStrings)
	if err != nil {
		p.Fatal("invalid fields: %v", err)
	}

	encodedData, err := json.Marshal(data)
	if err != nil {
		p.Fatal("cannot encode field data: %v", err)
	}

	identity, err := app.Client.FetchIdentityByName(name)
	if err != nil {
		p.Fatal("cannot fetch identity: %v", err)
	}

	newIdentity := eventline.RawNewIdentity{
		Name:      name,
		Connector: connector,
		Type:      itype,
		RawData:   encodedData,
	}

	identity2, err := app.Client.UpdateIdentity(identity.Id, &newIdentity)
	if err != nil {
		p.Fatal("cannot update identity: %v", err)
	}

	p.Info("identity %q updated", identity2.Name)
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

func ParseIdentityFields(ss []string) (map[string]interface{}, error) {
	// With current connectors, all non-oauth2 identities only use string
	// fields. If this changes, we will need a way to access identity
	// definitions in evcli through the API.

	data := make(map[string]interface{})

	for _, s := range ss {
		equal := strings.IndexByte(s, '=')
		if equal == -1 {
			return nil, fmt.Errorf("%q: invalid format", s)
		}

		key := s[:equal]
		if key == "" {
			return nil, fmt.Errorf("%q: empty key", s)
		}

		value := s[equal+1:]

		data[key] = value
	}

	return data, nil
}
