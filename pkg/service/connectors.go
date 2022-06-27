package service

import (
	cgeneric "github.com/exograd/eventline/pkg/connectors/generic"
	cgithub "github.com/exograd/eventline/pkg/connectors/github"
	ctime "github.com/exograd/eventline/pkg/connectors/time"
	"github.com/exograd/eventline/pkg/eventline"
)

var Connectors = []eventline.Connector{
	cgeneric.NewConnector(),
	cgithub.NewConnector(),
	ctime.NewConnector(),
}

var Runners = []*eventline.RunnerDef{
	eventline.LocalRunnerDef(),
}
