package service

import (
	"sort"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/web"
)

func NewLoginMenu(selectedEntry string) *web.Menu {
	return &web.Menu{
		Entries: []*web.MenuEntry{
			{
				Id:           "login",
				Icon:         "login",
				Label:        "Login",
				URI:          "/login",
				WhenLoggedIn: false,
			},
		},

		SelectedEntry: selectedEntry,
	}
}

func NewMainMenu(selectedEntry string) *web.Menu {
	return &web.Menu{
		Entries: []*web.MenuEntry{
			{
				Id:    "jobs",
				Icon:  "play-box-multiple-outline",
				Label: "Jobs",
				URI:   "/jobs",
			},
			{
				Id:    "identities",
				Icon:  "lock-outline",
				Label: "Identities",
				URI:   "/identities",
			},
			{
				Id:    "events",
				Icon:  "file-table-box-multiple-outline",
				Label: "Events",
				URI:   "/events",
			},
			// ---------------------------------------------------------------
			{
				Id:    "account",
				Icon:  "account-outline",
				Label: "Account",
				URI:   "/account",
				Apart: true,
			},
			{
				Id:    "admin",
				Icon:  "shield-outline",
				Label: "Administration",
				URI:   "/admin",
			},
		},

		SelectedEntry: selectedEntry,
	}
}

func IdentityConnectorSelect(name string) *web.Select {
	options := make([]web.SelectOption, 0, len(eventline.Connectors))

	for cname, c := range eventline.Connectors {
		cdef := c.Definition()

		if len(cdef.Identities) == 0 {
			continue
		}

		options = append(options, web.SelectOption{
			Name:  cname,
			Label: cname,
		})
	}

	sort.Slice(options, func(i, j int) bool {
		return options[i].Label < options[j].Label
	})

	return &web.Select{
		Name:    name,
		Options: options,
	}
}

func IdentityTypeSelect(cdef *eventline.ConnectorDef, name string) *web.Select {
	options := make([]web.SelectOption, len(cdef.Identities))

	i := 0

	for itype := range cdef.Identities {
		options[i] = web.SelectOption{
			Name:  itype,
			Label: itype,
		}

		i++
	}

	sort.Slice(options, func(i, j int) bool {
		return options[i].Label < options[j].Label
	})

	return &web.Select{
		Name:    name,
		Options: options,
	}
}
