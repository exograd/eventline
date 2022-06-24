package eventline

import (
	"sync"
	"time"

	"github.com/exograd/eventline/pkg/utils"
	"github.com/exograd/go-daemon/daemon"
	"github.com/exograd/go-log"
)

type WorkerCfg struct {
	Log    *log.Logger    `json:"-"`
	Daemon *daemon.Daemon `json:"-"`

	Behaviour WorkerBehaviour `json:"-"`

	Disabled bool `json:"disabled"`

	InitialDelay  int `json:"initial_delay"`  // millisecond
	ErrorDelay    int `json:"error_delay"`    // millisecond
	SleepDuration int `json:"sleep_duration"` // millisecond

	NotificationChan chan<- interface{} `json:"-"`
	StopChan         <-chan struct{}    `json:"-"`
	Wg               *sync.WaitGroup    `json:"-"`
}

type WorkerBehaviour interface {
	Init(*Worker)
	Start() error
	Stop()
	ProcessJob() (bool, error)
}

type Worker struct {
	Name   string
	Cfg    WorkerCfg
	Log    *log.Logger
	Daemon *daemon.Daemon

	timer            *time.Timer
	wakeUpChan       chan bool
	notificationChan chan<- interface{}

	initialDelay  time.Duration
	errorDelay    time.Duration
	sleepDuration time.Duration

	stopChan <-chan struct{}
	wg       *sync.WaitGroup
}

func NewWorker(name string, cfg WorkerCfg) *Worker {
	w := &Worker{
		Name:   name,
		Cfg:    cfg,
		Log:    cfg.Log,
		Daemon: cfg.Daemon,

		wakeUpChan:       make(chan bool, 1),
		notificationChan: cfg.NotificationChan,

		stopChan: cfg.StopChan,
		wg:       cfg.Wg,
	}

	initDuration := func(ms, defaultMs int) time.Duration {
		if ms == 0 {
			ms = defaultMs
		}

		return time.Duration(ms) * time.Millisecond
	}

	w.initialDelay = initDuration(cfg.InitialDelay, 1000)
	w.errorDelay = initDuration(cfg.ErrorDelay, 5000)
	w.sleepDuration = initDuration(cfg.SleepDuration, 5000)

	w.Cfg.Behaviour.Init(w)

	return w
}

func (w *Worker) Start() error {
	w.Log.Info("starting")

	if err := w.Cfg.Behaviour.Start(); err != nil {
		return err
	}

	w.timer = time.NewTimer(w.initialDelay)

	w.wg.Add(1)
	go w.main()

	return nil
}

func (w *Worker) WakeUp() {
	// We do not want to block when writing on the wake-up chan. This can
	// happen if we are trying to wake up the worker while it is processing a
	// job.

	select {
	case w.wakeUpChan <- true:
	default:
	}
}

func (w *Worker) main() {
	defer func() {
		close(w.wakeUpChan)
		w.wg.Done()
	}()

	running := true

	for running {
		func() {
			defer func() {
				if value := recover(); value != nil {
					msg, trace := utils.RecoverValueData(value)
					w.Log.Error("panic: %s\n%s", msg, trace)
					time.Sleep(w.sleepDuration)
				}
			}()

			select {
			case <-w.stopChan:
				running = false
				return

			case wakeUp := <-w.wakeUpChan:
				if wakeUp {
					w.processJobs()
				}

			case <-w.timer.C:
				w.processJobs()
			}
		}()
	}
}

func (w *Worker) processJobs() {
	for !w.Stopping() {
		processed, err := w.Cfg.Behaviour.ProcessJob()
		if err != nil {
			w.Log.Error("%v", err)
			w.timer.Reset(w.errorDelay)
			return
		}

		if !processed {
			w.timer.Reset(w.sleepDuration)
			return
		}
	}
}

func (w *Worker) Stopping() bool {
	select {
	case <-w.stopChan:
		return true
	default:
		return false
	}
}
