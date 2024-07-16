package service

import (
	"testing"

	cgeneric "github.com/exograd/eventline/pkg/connectors/generic"
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/test"
	"go.n16f.net/service/pkg/pg"
	"github.com/stretchr/testify/require"
)

func createTestProject(t *testing.T, nameSuffix string) *eventline.Project {
	var project *eventline.Project

	err := testService.Pg.WithConn(func(conn pg.Conn) (err error) {
		newProject := &eventline.NewProject{
			Name: test.RandomName("project", nameSuffix),
		}

		project, err = testService.CreateProject(newProject, nil)
		return
	})

	require.NoError(t, err)

	return project
}

func createTestIdentity(t *testing.T, nameSuffix string, scope eventline.Scope) *eventline.Identity {
	var identity *eventline.Identity

	err := testService.Pg.WithConn(func(conn pg.Conn) (err error) {
		newIdentity := &eventline.NewIdentity{
			Name:      test.RandomName("identity", nameSuffix),
			Connector: "generic",
			Type:      "password",
			Data: &cgeneric.PasswordIdentity{
				Password: "password",
			},
		}

		identity, err = testService.CreateIdentity(newIdentity, scope)
		return
	})

	require.NoError(t, err)

	return identity
}
