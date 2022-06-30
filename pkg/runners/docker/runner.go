package docker

import (
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-log"
)

type Runner struct {
	runner *eventline.Runner
	log    *log.Logger
}

func RunnerDef() *eventline.RunnerDef {
	return &eventline.RunnerDef{
		Name:                  "docker",
		Cfg:                   &RunnerCfg{},
		InstantiateParameters: NewRunnerParameters,
		InstantiateBehaviour:  NewRunner,
	}
}

func NewRunner(r *eventline.Runner) eventline.RunnerBehaviour {
	//cfg := r.Cfg.(*RunnerCfg)

	return &Runner{
		runner: r,
		log:    r.Log,
	}
}

func (r *Runner) DirPath() string {
	return "/eventline"
}

func (r *Runner) Init() error {
	// TODO

	return nil
}

func (r *Runner) Terminate() {
	// TODO
}

func (r *Runner) ExecuteStep(se *eventline.StepExecution, step *eventline.Step) error {
	// TODO

	return nil
}
