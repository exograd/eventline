package service

import (
	"encoding/json"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/daemon"
	"github.com/exograd/go-daemon/dcrypto"
	"github.com/exograd/go-daemon/dhttp"
	"github.com/exograd/go-daemon/dlog"
	"github.com/exograd/go-daemon/influx"
	"github.com/exograd/go-daemon/pg"
)

type ServiceCfg struct {
	Logger *dlog.LoggerCfg `json:"logger"`

	DaemonAPI *daemon.APICfg `json:"daemon_api"`

	DataDirectory string `json:"data_directory"`

	APIHTTPServer *dhttp.ServerCfg `json:"api_http_server"`
	WebHTTPServer *dhttp.ServerCfg `json:"web_http_server"`

	Influx *influx.ClientCfg `json:"influx"`

	Pg *pg.ClientCfg `json:"pg"`

	EncryptionKey dcrypto.AES256Key `json:"encryption_key"`

	WebHTTPServerURI string `json:"web_http_server_uri"`

	Connectors map[string]json.RawMessage `json:"connectors"`

	Workers map[string]eventline.WorkerCfg `json:"workers"`

	MaxParallelJobExecutions int `json:"max_parallel_job_executions"`
	JobExecutionRetention    int `json:"job_execution_retention"` // days

	SessionRetention int `json:"session_retention"` // days

	Runners map[string]json.RawMessage `json:"runners"`

	Notifications *NotificationsCfg `json:"notifications"`
}

func DefaultServiceCfg() ServiceCfg {
	logger := &dlog.LoggerCfg{
		BackendType: "terminal",
		Backend: &dlog.TerminalBackendCfg{
			Color:       true,
			DomainWidth: 32,
		},
	}

	return ServiceCfg{
		Logger: logger,

		DataDirectory: "data",

		APIHTTPServer: &dhttp.ServerCfg{
			Address: "localhost:8085",
		},
		WebHTTPServer: &dhttp.ServerCfg{
			Address: "localhost:8087",
		},

		Pg: &pg.ClientCfg{
			URI:         "postgres://eventline:eventline@localhost:5432/eventline",
			SchemaNames: []string{"eventline"},
		},

		WebHTTPServerURI: "http://localhost:8087",

		Notifications: DefaultNotificationsCfg(),
	}
}

func (cfg *ServiceCfg) Check(c *check.Checker) {
	// Note that some fields are optional in the documentation but mandatory
	// here. These values are set in the default configuration: they must be
	// provided for Eventline to work, but we define reasonable default values
	// so that the user does not have to set them in most cases.
	//
	// Also, we do not check the encryption key, since it is validated by its
	// UnmashalJSON method.
	//
	// Finally, connectors, workers and runners are currently handled later in
	// the initialization phase. This is clearly something we could improve in
	// the future.

	c.CheckObject("logger", cfg.Logger)
	c.CheckOptionalObject("daemon_api", cfg.DaemonAPI)

	c.CheckStringNotEmpty("data_directory", cfg.DataDirectory)

	c.CheckObject("api_http_server", cfg.APIHTTPServer)
	c.CheckObject("web_http_server", cfg.WebHTTPServer)

	c.CheckOptionalObject("influx", cfg.Influx)

	c.CheckObject("pg", cfg.Pg)

	c.CheckStringURI("web_http_server_uri", cfg.WebHTTPServerURI)

	if cfg.MaxParallelJobExecutions != 0 {
		c.CheckIntMin("max_parallel_job_executions",
			cfg.MaxParallelJobExecutions, 1)
	}

	if cfg.JobExecutionRetention != 0 {
		c.CheckIntMin("job_execution_retention", cfg.JobExecutionRetention, 1)
	}

	if cfg.SessionRetention != 0 {
		c.CheckIntMin("session_retention", cfg.SessionRetention, 1)
	}

	c.CheckObject("notifications", cfg.Notifications)
}
