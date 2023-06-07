package service

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"sync"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/galdor/go-ejson"
	"github.com/galdor/go-log"
	"github.com/galdor/go-service/pkg/pg"
	goservice "github.com/galdor/go-service/pkg/service"
	"github.com/galdor/go-service/pkg/shttp"
)

type Service struct {
	Data ServiceData
	Cfg  ServiceCfg

	Service *goservice.Service
	Log     *log.Logger

	Pg *pg.Client

	APIHTTPServer *APIHTTPServer
	WebHTTPServer *WebHTTPServer

	BuildIdHash      string
	WebHTTPServerURI *url.URL

	workers                map[string]*eventline.Worker
	workerStopChan         chan struct{}
	workerNotificationChan chan interface{}
	workerWg               sync.WaitGroup

	connectors map[string]eventline.Connector

	runnerNames    []string
	runnerDefs     map[string]*eventline.RunnerDef
	runnerStopChan chan struct{}
	runnerWg       sync.WaitGroup

	jobExecutionTerminationChan chan eventline.Id
}

func NewService(data ServiceData) *Service {
	hash := sha1.New()
	hash.Write([]byte(data.BuildId))
	buildIdHash := hex.EncodeToString(hash.Sum(nil))

	runnerNames := make([]string, len(data.Runners))
	for i, r := range data.Runners {
		runnerNames[i] = r.Name
	}

	s := &Service{
		Data: data,

		BuildIdHash: buildIdHash,

		workers:                make(map[string]*eventline.Worker),
		workerStopChan:         make(chan struct{}),
		workerNotificationChan: make(chan interface{}),

		connectors: make(map[string]eventline.Connector),

		runnerNames:    runnerNames,
		runnerDefs:     make(map[string]*eventline.RunnerDef),
		runnerStopChan: make(chan struct{}),

		jobExecutionTerminationChan: make(chan eventline.Id),
	}

	return s
}

func (s *Service) DefaultCfg() interface{} {
	cfg := DefaultServiceCfg()

	if ps := s.Data.ProService; ps != nil {
		cfg.ProCfg = ps.DefaultServiceCfg()
	}

	s.Cfg = cfg

	return &s.Cfg
}

func (s *Service) ValidateCfg() error {
	// Postprocessing
	if s.Cfg.Pg.SchemaDirectory == "" {
		s.Cfg.Pg.SchemaDirectory =
			path.Join(s.Cfg.DataDirectory, "pg", "schemas")
	}

	// Validation
	validator := ejson.NewValidator()

	s.Cfg.Check(validator, s)

	if scfg := s.Cfg.ProCfg; scfg != nil {
		scfg.ValidateJSON(validator)
	}

	return validator.Error()
}

func (s *Service) ServiceCfg() *goservice.ServiceCfg {
	// Service configuration is supposed to be a goservice.ServiceCfg
	// structure nested in the ServiceCfg structure of Eventline. But
	// Eventline was developed before go-service and we kept the configuration
	// backward compatible. Hence the messy go-service service configuration
	// initialization.

	s.Cfg.APIHTTPServer.ErrorHandler = shttp.JSONErrorHandler
	s.Cfg.WebHTTPServer.ErrorHandler = s.webErrorHandler

	cfg := goservice.ServiceCfg{
		Logger: s.Cfg.Logger,

		DataDirectory: s.Cfg.DataDirectory,

		Influx: s.Cfg.Influx,

		PgClients: map[string]*pg.ClientCfg{
			"main": s.Cfg.Pg,
		},

		HTTPServers: map[string]*shttp.ServerCfg{
			"api": s.Cfg.APIHTTPServer,
			"web": s.Cfg.WebHTTPServer,
		},

		ServiceAPI: s.Cfg.ServiceAPI,

		TemplateFuncMap: eventline.TemplateFuncMap,
	}

	return &cfg
}

func (s *Service) Init(ss *goservice.Service) error {
	s.Service = ss
	s.Log = ss.Log

	s.Pg = ss.PgClient("main")

	if err := s.initEncryptionKey(); err != nil {
		return err
	}

	if err := s.initWebHTTPServerURI(); err != nil {
		return err
	}

	if err := s.initConnectors(); err != nil {
		return err
	}

	if err := s.initRunners(); err != nil {
		return err
	}

	if err := s.initPg(); err != nil {
		return err
	}

	if ps := s.Data.ProService; ps != nil {
		if err := ps.Init(s); err != nil {
			return err
		}
	}

	apiHTTPServer, err := NewAPIHTTPServer(s)
	if err != nil {
		return err
	}
	s.APIHTTPServer = apiHTTPServer

	webHTTPServer, err := NewWebHTTPServer(s)
	if err != nil {
		return err
	}
	s.WebHTTPServer = webHTTPServer

	s.initJobExecutionTerminationWatcher()

	s.initWorkers()

	return nil
}

func (s *Service) initEncryptionKey() error {
	eventline.GlobalEncryptionKey = s.Cfg.EncryptionKey

	return nil
}

func (s *Service) initConnectors() error {
	for _, c := range s.Data.Connectors {
		if err := s.initConnector(c); err != nil {
			return fmt.Errorf("cannot initialize connector %q: %w",
				c.Name(), err)
		}
	}

	return nil
}

func (s *Service) initConnector(c eventline.Connector) error {
	def := c.Definition()
	name := def.Name

	initData := eventline.ConnectorInitData{
		Log:              s.Log.Child("connectors."+name, nil),
		Pg:               s.Pg,
		WebHTTPServerURI: s.WebHTTPServerURI,
	}

	cfg := c.DefaultCfg()

	if cfgData, found := s.Cfg.Connectors[name]; found {
		if err := ejson.Unmarshal(cfgData, cfg); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}
	}

	if err := c.Init(cfg, initData); err != nil {
		return err
	}

	s.connectors[name] = c

	eventline.Connectors[name] = c

	return nil
}

func (s *Service) initRunners() error {
	for _, def := range s.Data.Runners {
		if err := s.initRunner(def); err != nil {
			return fmt.Errorf("cannot initialize runner %q: %w",
				def.Name, err)
		}
	}

	return nil
}

func (s *Service) initRunner(def *eventline.RunnerDef) error {
	if cfgData, found := s.Cfg.Runners[def.Name]; found {
		if err := json.Unmarshal(cfgData, def.Cfg); err != nil {
			return fmt.Errorf("cannot decode configuration: %w", err)
		}

		validator := ejson.NewValidator()
		def.Cfg.ValidateJSON(validator)
		if err := validator.Error(); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}
	}

	s.runnerDefs[def.Name] = def

	eventline.RunnerDefs[def.Name] = def

	return nil
}

func (s *Service) initPg() error {
	return s.Pg.WithTx(func(conn pg.Conn) error {
		id1 := PgAdvisoryLockId1
		id2 := PgAdvisoryLockId2ServiceInit

		if err := pg.TakeAdvisoryTxLock(conn, id1, id2); err != nil {
			return fmt.Errorf("cannot take advisory lock: %w", err)
		}

		account, err := s.MaybeCreateDefaultAccount(conn)
		if err != nil {
			return err
		}

		if _, err := s.MaybeCreateDefaultProject(conn, account); err != nil {
			return err
		}

		return nil
	})
}

func (s *Service) initWebHTTPServerURI() error {
	if uriString := s.Cfg.WebHTTPServerURI; uriString == "" {
		s.WebHTTPServerURI = &url.URL{
			Scheme: "http",
			Host:   s.Cfg.WebHTTPServer.Address,
		}
	} else {
		uri, err := url.Parse(uriString)
		if err != nil {
			return fmt.Errorf("inavlid web http server uri %q: %w",
				uriString, err)
		}

		s.WebHTTPServerURI = uri
	}

	return nil
}

func (s *Service) initJobExecutionTerminationWatcher() {
	go func() {
		for jeId := range s.jobExecutionTerminationChan {
			if err := s.handleJobExecutionTermination(jeId); err != nil {
				s.Log.Error("cannot handle termination of job execution "+
					"%q: %v", jeId, err)
			}
		}
	}()
}

func (s *Service) initWorkers() {
	init := func(name string, behaviour eventline.WorkerBehaviour, notificationChan chan interface{}) {
		cfg, found := s.Cfg.Workers[name]
		if !found {
			cfg = eventline.WorkerCfg{}
		}

		if cfg.Disabled {
			s.Log.Info("skipping disabled worker %q", name)
			return
		}

		cfg.Log = s.Log.Child(name, nil)
		cfg.Pg = s.Pg

		cfg.Behaviour = behaviour

		cfg.NotificationChan = notificationChan
		cfg.StopChan = s.workerStopChan
		cfg.Wg = &s.workerWg

		s.workers[name] = eventline.NewWorker(name, cfg)
	}

	init("identity-refresher", NewIdentityRefresher(s), nil)
	init("subscription-worker", NewSubscriptionWorker(s), nil)
	init("event-worker", NewEventWorker(s), nil)
	init("job-scheduler", NewJobScheduler(s), nil)
	init("job-execution-gc", NewJobExecutionGC(s), nil)
	init("job-execution-watcher", NewJobExecutionWatcher(s), nil)
	init("notification-worker", NewNotificationWorker(s), nil)
	if s.Cfg.SessionRetention > 0 {
		init("session-gc", NewSessionGC(s), nil)
	}

	for name, c := range eventline.Connectors {
		cdef := c.Definition()

		if cdef.Worker != nil {
			init(name+"-connector", cdef.Worker, s.workerNotificationChan)
		}
	}
}

func (s *Service) FindWorker(name string) *eventline.Worker {
	// We do not need any lock because the worker map is never modified after
	// initialization.

	return s.workers[name]
}

func (s *Service) FindConnectorWorker(cname string) *eventline.Worker {
	return s.FindWorker(cname + "-connector")
}

func (s *Service) Start(ss *goservice.Service) error {
	go s.processWorkerNotifications()

	for _, w := range s.workers {
		if err := w.Start(); err != nil {
			return fmt.Errorf("cannot start worker %q: %w", w.Name, err)
		}
	}

	if ps := s.Data.ProService; ps != nil {
		if err := ps.Start(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) Stop(ss *goservice.Service) {
	// Note that we do *not* close the job execution termination chan until
	// all runners have terminated. If we did, they would crash when writing
	// the job execution id at the end.
	close(s.runnerStopChan)
	s.runnerWg.Wait()

	for _, c := range s.connectors {
		c.Terminate()
	}

	close(s.jobExecutionTerminationChan)

	close(s.workerStopChan)
	s.workerWg.Wait()
	close(s.workerNotificationChan)

	if ps := s.Data.ProService; ps != nil {
		ps.Stop()
	}
}

func (s *Service) Terminate(ss *goservice.Service) {
	if ps := s.Data.ProService; ps != nil {
		ps.Terminate()
	}
}

func (s *Service) processWorkerNotifications() {
	for value := range s.workerNotificationChan {
		if _, ok := value.(*eventline.Event); ok {
			if w := s.FindWorker("event-worker"); w != nil {
				w.WakeUp()
			}
		}
	}
}
