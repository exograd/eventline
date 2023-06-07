package main

import (
	"github.com/exograd/eventline/pkg/service"
	goservice "github.com/galdor/go-service/pkg/service"
)

var buildId string

func main() {
	sdata := service.ServiceData{
		Product:    "Eventline",
		BuildId:    buildId,
		Connectors: service.Connectors,
		Runners:    service.Runners,
	}

	s := service.NewService(sdata)

	goservice.Run("eventline", "job scheduling platform", s)
}
