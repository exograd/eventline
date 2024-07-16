package service

import (
	"encoding/json"

	"github.com/exograd/eventline/pkg/cryptoutils"
	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/ejson"
	"go.n16f.net/log"
	"go.n16f.net/service/pkg/influx"
	"go.n16f.net/service/pkg/pg"
	goservice "go.n16f.net/service/pkg/service"
	"go.n16f.net/service/pkg/shttp"
)

type ServiceCfg struct {
	Logger *log.LoggerCfg `json:"logger"`

	ServiceAPI *goservice.ServiceAPICfg `json:"service_api"`

	DataDirectory string `json:"data_directory"`

	APIHTTPServer *shttp.ServerCfg `json:"api_http_server"`
	WebHTTPServer *shttp.ServerCfg `json:"web_http_server"`

	Influx *influx.ClientCfg `json:"influx"`

	Pg *pg.ClientCfg `json:"pg"`

	EncryptionKey cryptoutils.AES256Key `json:"encryption_key"`

	WebHTTPServerURI    string `json:"web_http_server_uri"`
	InsecureHTTPCookies bool   `json:"insecure_http_cookies"`

	Connectors map[string]json.RawMessage `json:"connectors"`

	Workers map[string]eventline.WorkerCfg `json:"workers"`

	MaxParallelJobExecutions    int `json:"max_parallel_job_executions"`
	JobExecutionRetention       int `json:"job_execution_retention"`        // days
	JobExecutionRefreshInterval int `json:"job_execution_refresh_interval"` // seconds
	JobExecutionTimeout         int `json:"job_execution_timeout"`          // seconds

	SessionRetention int `json:"session_retention"` // days

	AllowedRunners []string                   `json:"allowed_runners"`
	Runners        map[string]json.RawMessage `json:"runners"`

	Notifications *NotificationsCfg `json:"notifications"`

	ProCfg ejson.Validatable `json:"pro"`
}

func DefaultServiceCfg() ServiceCfg {
	logger := &log.LoggerCfg{
		BackendType: "terminal",
		TerminalBackend: &log.TerminalBackendCfg{
			Color:       true,
			DomainWidth: 32,
		},
	}

	return ServiceCfg{
		Logger: logger,

		DataDirectory: "data",

		APIHTTPServer: &shttp.ServerCfg{
			Address: "localhost:8085",
		},
		WebHTTPServer: &shttp.ServerCfg{
			Address: "localhost:8087",
		},

		Pg: &pg.ClientCfg{
			URI:         "postgres://eventline:eventline@localhost:5432/eventline",
			SchemaNames: []string{"eventline"},
		},

		WebHTTPServerURI: "http://localhost:8087",

		JobExecutionRefreshInterval: 10,
		JobExecutionTimeout:         120,

		Notifications: DefaultNotificationsCfg(),
	}
}

func (cfg *ServiceCfg) Check(v *ejson.Validator, s *Service) {
	// Note that some fields are optional in the documentation but mandatory
	// here. These values are set in the default configuration: they must be
	// provided for Eventline to work, but we define reasonable default values
	// so that the user does not have to set them in most cases.
	//
	// Also, we do not check the value of the encryption key, since it is
	// validated by its UnmarshalJSON method. We still have to check that it
	// is present, since UnmarshalJSON will not be called if the field is not
	// set.
	//
	// Finally, connectors, workers and runners are currently handled later in
	// the initialization phase. This is clearly something we could improve in
	// the future.

	v.CheckObject("logger", cfg.Logger)
	v.CheckOptionalObject("service_api", cfg.ServiceAPI)

	v.CheckStringNotEmpty("data_directory", cfg.DataDirectory)

	v.CheckObject("api_http_server", cfg.APIHTTPServer)
	v.CheckObject("web_http_server", cfg.WebHTTPServer)

	v.CheckOptionalObject("influx", cfg.Influx)

	v.CheckObject("pg", cfg.Pg)

	v.Check("encryption_key", !cfg.EncryptionKey.IsZero(),
		"invalid_value", "missing encryption key")

	v.CheckStringURI("web_http_server_uri", cfg.WebHTTPServerURI)

	if cfg.MaxParallelJobExecutions != 0 {
		v.CheckIntMin("max_parallel_job_executions",
			cfg.MaxParallelJobExecutions, 1)
	}

	if cfg.JobExecutionRetention != 0 {
		v.CheckIntMin("job_execution_retention", cfg.JobExecutionRetention, 1)
	}

	if cfg.JobExecutionRefreshInterval != 0 {
		v.CheckIntMin("job_execution_refresh_interval",
			cfg.JobExecutionRefreshInterval, 1)
	}

	if cfg.JobExecutionTimeout != 0 {
		v.CheckIntMin("job_execution_timeout", cfg.JobExecutionTimeout, 1)
	}

	if cfg.SessionRetention != 0 {
		v.CheckIntMin("session_retention", cfg.SessionRetention, 1)
	}

	v.WithChild("allowed_runners", func() {
		for i, r := range cfg.AllowedRunners {
			v.CheckStringValue(i, r, s.runnerNames)
		}
	})

	v.CheckObject("notifications", cfg.Notifications)
}
