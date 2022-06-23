package service

import (
	"testing"

	cgeneric "github.com/exograd/evgo/pkg/connectors/generic"
	"github.com/exograd/evgo/pkg/eventline"
	"github.com/exograd/evgo/pkg/test"
	"github.com/exograd/go-daemon/pg"
	"github.com/stretchr/testify/require"
)

func createTestProject(t *testing.T, nameSuffix string) *eventline.Project {
	var project *eventline.Project

	err := testService.Daemon.Pg.WithConn(func(conn pg.Conn) (err error) {
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

	err := testService.Daemon.Pg.WithConn(func(conn pg.Conn) (err error) {
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
