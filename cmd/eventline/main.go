package main

import (
	"github.com/exograd/evgo/pkg/service"
	"github.com/exograd/go-daemon/daemon"
)

func main() {
	daemon.Run("eventline", "pipeline scheduler", service.NewService())
}
