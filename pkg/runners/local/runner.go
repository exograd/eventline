package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"syscall"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/galdor/go-log"
)

type Runner struct {
	runner *eventline.Runner
	log    *log.Logger

	rootPath string
}

func RunnerDef() *eventline.RunnerDef {
	return &eventline.RunnerDef{
		Name: "local",
		Cfg: &RunnerCfg{
			RootDirectory: "/tmp/eventline/execution",
		},
		InstantiateParameters: NewRunnerParameters,
		InstantiateBehaviour:  NewRunner,
	}
}

func NewRunner(r *eventline.Runner) eventline.RunnerBehaviour {
	cfg := r.Cfg.(*RunnerCfg)

	je := r.JobExecution

	rootDirPath := cfg.RootDirectory
	rootPath := path.Join(rootDirPath, je.Id.String())

	return &Runner{
		runner: r,
		log:    r.Log,

		rootPath: rootPath,
	}
}

func (r *Runner) DirPath() string {
	return r.rootPath
}

func (r *Runner) Init(ctx context.Context) error {
	r.runner.Environment["HOME"] = r.rootPath

	if err := r.runner.FileSet.Write(r.rootPath); err != nil {
		return err
	}

	return nil
}

func (r *Runner) Terminate() {
	if err := os.RemoveAll(r.rootPath); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			r.log.Error("cannot delete directory %q: %v", r.rootPath, err)
		}
	}
}

func (r *Runner) ExecuteStep(ctx context.Context, se *eventline.StepExecution, step *eventline.Step, stdout, stderr io.WriteCloser) error {
	// Create the command
	cmdName, cmdArgs := r.runner.StepCommand(se, step, ".")

	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)

	cmd.Dir = r.rootPath

	cmd.Env = make([]string, 0, len(r.runner.Environment))
	for k, v := range r.runner.Environment {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// Run the command
	err := cmd.Run()

	// Handle the error if there is one. We translate it to get nice error
	// messages.
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return eventline.NewStepFailureError(r.translateExitError(exitErr))
	}

	return err
}

func (r *Runner) translateExitError(err *exec.ExitError) error {
	state := err.ProcessState
	status := state.Sys().(syscall.WaitStatus)

	switch {
	case status.Exited():
		if code := status.ExitStatus(); code < 128 {
			return fmt.Errorf("program exited with status %d", code)
		} else {
			return fmt.Errorf("program killed by signal %d", code-128)
		}

	case status.Signaled():
		return fmt.Errorf("program killed by signal %d", status.Signal())

	default:
		return err
	}
}
