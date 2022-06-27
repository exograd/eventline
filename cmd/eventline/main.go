package main

import (
	"github.com/exograd/eventline/pkg/service"
	"github.com/exograd/go-daemon/daemon"
)

func main() {
	sdata := service.ServiceData{
		Connectors: service.Connectors,
		Runners:    service.Runners,
	}

	s := service.NewService(sdata)

	daemon.Run("eventline", "job scheduling platform", s)
}
