package service

import (
	cdockerhub "github.com/exograd/eventline/pkg/connectors/dockerhub"
	ceventline "github.com/exograd/eventline/pkg/connectors/eventline"
	cgeneric "github.com/exograd/eventline/pkg/connectors/generic"
	cgithub "github.com/exograd/eventline/pkg/connectors/github"
	cpostgresql "github.com/exograd/eventline/pkg/connectors/postgresql"
	ctime "github.com/exograd/eventline/pkg/connectors/time"
	"github.com/exograd/eventline/pkg/eventline"
	rdocker "github.com/exograd/eventline/pkg/runners/docker"
	rlocal "github.com/exograd/eventline/pkg/runners/local"
)

type ServiceData struct {
	Product    string
	BuildId    string
	Connectors []eventline.Connector
	Runners    []*eventline.RunnerDef
}

var Connectors = []eventline.Connector{
	cdockerhub.NewConnector(),
	ceventline.NewConnector(),
	cgeneric.NewConnector(),
	cgithub.NewConnector(),
	cpostgresql.NewConnector(),
	ctime.NewConnector(),
}

var Runners = []*eventline.RunnerDef{
	rdocker.RunnerDef(),
	rlocal.RunnerDef(),
}
