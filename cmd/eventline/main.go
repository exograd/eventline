package main

import (
	"github.com/exograd/eventline/pkg/service"
	"github.com/exograd/go-daemon/daemon"
)

func main() {
	daemon.Run("eventline", "job scheduling platform", service.NewService())
}
