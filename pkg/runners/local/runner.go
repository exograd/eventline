package local

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"sync"
	"syscall"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-log"
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
			RootDirectory: "tmp/local-execution",
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

func (r *Runner) Init() error {
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

func (r *Runner) ExecuteStep(se *eventline.StepExecution, step *eventline.Step) error {
	// Interruption handling (i.e. when the server is being stopped while jobs
	// are running).
	ctx, cancel := context.WithCancel(context.Background())

	endChan := make(chan struct{})
	defer close(endChan)

	go func() {
		select {
		case <-r.runner.StopChan:
			r.log.Info("interrupting job")
			cancel()
			return

		case <-endChan:
			cancel()
			return
		}
	}()

	// Create the command
	cmdName, cmdArgs := r.runner.StepCommand(se, step, ".")

	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)

	cmd.Dir = r.rootPath

	cmd.Env = make([]string, 0, len(r.runner.Environment))
	for k, v := range r.runner.Environment {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("cannot create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("cannot create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cannot start commande: %w", err)
	}

	// Start readers for output pipes; note that we must not call Command.Wait
	// until both stdout and stderr have been closed (see the documentation of
	// the os/exec package).
	errChan := make(chan error, 2)
	defer close(errChan)

	var wg sync.WaitGroup
	wg.Add(2)
	go r.readOutput(se, stdout, "stdout", errChan, &wg)
	go r.readOutput(se, stderr, "stderr", errChan, &wg)

	wg.Wait()

	// Now that output readers are terminated, check the error channel for any
	// output error.
	select {
	case outputErr := <-errChan:
		if outputErr != nil {
			cmd.Wait()
			return outputErr
		}

	default:
	}

	// Wait for the command termination status
	err = cmd.Wait()

	// Handle the error if there is one. We translate it to get nice error
	// messages.
	if err != nil {
		var exitErr *exec.ExitError

		if errors.As(err, &exitErr) {
			return eventline.NewStepFailureError(r.translateExitError(exitErr))
		}

		return err
	}

	return nil
}

func (r *Runner) readOutput(se *eventline.StepExecution, output io.ReadCloser, name string, errChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	bufferedOutput := bufio.NewReader(output)

	var line []byte

	for {
		data, isPrefix, err := bufferedOutput.ReadLine()
		if err != nil && err != io.EOF {
			err = fmt.Errorf("cannot read command output %q: %v", name, err)
			errChan <- err
			return
		}

		if err == nil {
			line = append(line, data...)
			if isPrefix {
				continue
			}
		}

		// There is no point in updating se.Output because we are not going to
		// read it in the runner, so we may as well avoid allocating and
		// copying data. This only works because se.Update does not modify the
		// output column, so it will not be erased when updating the step
		// execution later.

		if len(line) > 0 {
			err = r.runner.UpdateStepExecutionOutput(se, append(line, '\n'))
			if err != nil {
				err = fmt.Errorf("cannot update step execution %q: %v",
					se.Id, err)
				errChan <- err
				return
			}

			line = nil
		}

		if err == io.EOF {
			break
		}
	}
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
