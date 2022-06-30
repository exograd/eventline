package service

import (
	cgeneric "github.com/exograd/eventline/pkg/connectors/generic"
	cgithub "github.com/exograd/eventline/pkg/connectors/github"
	ctime "github.com/exograd/eventline/pkg/connectors/time"
	"github.com/exograd/eventline/pkg/eventline"
	rlocal "github.com/exograd/eventline/pkg/runners/local"
)

type ServiceData struct {
	Connectors []eventline.Connector
	Runners    []*eventline.RunnerDef
}

var Connectors = []eventline.Connector{
	cgeneric.NewConnector(),
	cgithub.NewConnector(),
	ctime.NewConnector(),
}

var Runners = []*eventline.RunnerDef{
	rlocal.RunnerDef(),
}