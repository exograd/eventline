package docker

import (
	"fmt"
	"io"

	dockerclient "github.com/docker/docker/client"
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-log"
)

type Runner struct {
	runner *eventline.Runner
	log    *log.Logger

	client *dockerclient.Client

	imageRef    string
	containerId string
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
	// Create the client
	client, err := newClient()
	if err != nil {
		return fmt.Errorf("cannot create client: %w", err)
	}

	r.client = client

	// Pull the image
	if err := r.pullImage(); err != nil {
		return fmt.Errorf("cannot pull image: %w", err)
	}

	// Create the container
	if err := r.createContainer(); err != nil {
		return fmt.Errorf("cannot create container: %w", err)
	}

	// Copy files to the container
	if err := r.copyFiles(); err != nil {
		return fmt.Errorf("cannot copy files: %w", err)
	}

	// Start the container
	if err := r.startContainer(); err != nil {
		return fmt.Errorf("cannot start container: %w", err)
	}

	return nil
}

func (r *Runner) Terminate() {
	if r.client == nil {
		return
	}

	if r.containerId != "" {
		if err := r.deleteContainer(); err != nil {
			r.log.Error("cannot delete container %q: %v", r.containerId, err)
		}
	}
}

func (r *Runner) ExecuteStep(se *eventline.StepExecution, step *eventline.Step, stdout, stderr io.WriteCloser) error {
	return r.exec(se, step, stdout, stderr)
}
