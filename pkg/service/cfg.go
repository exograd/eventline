package service

import (
	"encoding/json"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/dcrypto"
	"github.com/exograd/go-daemon/dhttp"
	"github.com/exograd/go-daemon/influx"
	"github.com/exograd/go-daemon/pg"
	"github.com/exograd/go-log"
)

type ServiceCfg struct {
	Logger *log.LoggerCfg `json:"logger"`

	DataDirectory string `json:"data_directory"`

	APIHTTPServer dhttp.ServerCfg `json:"api_http_server"`
	WebHTTPServer dhttp.ServerCfg `json:"web_http_server"`

	Influx *influx.ClientCfg `json:"influx"`

	Pg *pg.ClientCfg `json:"pg"`

	EncryptionKey dcrypto.AES256Key `json:"encryption_key"`

	WebHTTPServerURI string `json:"web_http_server_uri"`

	Connectors map[string]json.RawMessage `json:"connectors"`

	Workers map[string]eventline.WorkerCfg `json:"workers"`

	JobRetention int `json:"job_retention"` // days

	Runners map[string]json.RawMessage `json:"runners"`
}

func DefaultServiceCfg() ServiceCfg {
	logger := &log.LoggerCfg{
		BackendType: "terminal",
		Backend: &log.TerminalBackendCfg{
			Color:       true,
			DomainWidth: 32,
		},
		DebugLevel: 1,
	}

	return ServiceCfg{
		Logger: logger,

		DataDirectory: "data",

		APIHTTPServer: dhttp.ServerCfg{
			Address: "localhost:8085",
		},
		WebHTTPServer: dhttp.ServerCfg{
			Address: "localhost:8087",
		},

		Pg: &pg.ClientCfg{
			URI:             "postgres://eventline:eventline@localhost:5432/eventline",
			SchemaDirectory: "data/pg/schemas",
			SchemaNames:     []string{"eventline"},
		},

		WebHTTPServerURI: "http://localhost:8087",
	}
}
