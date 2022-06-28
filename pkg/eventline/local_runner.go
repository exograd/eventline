package eventline

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strconv"
	"sync"
	"syscall"

	"github.com/exograd/eventline/pkg/utils"
	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/daemon"
	"github.com/exograd/go-daemon/pg"
	"github.com/exograd/go-log"
)

type StepFailureError struct {
	err error
}

func NewStepFailureError(err error) *StepFailureError {
	return &StepFailureError{err: err}
}

func (err *StepFailureError) Error() string {
	return err.err.Error()
}

func (err *StepFailureError) Unwrap() error {
	return err.err
}

type LocalRunnerCfg struct {
	RootDirectory string `json:"root_directory"`
}

func (cfg *LocalRunnerCfg) Check(c *check.Checker) {
	c.CheckStringNotEmpty("root_directory", cfg.RootDirectory)
}

type LocalRunner struct {
	runner *Runner
	log    *log.Logger
	daemon *daemon.Daemon

	jobExecution     *JobExecution
	stepExecutions   StepExecutions
	executionContext *ExecutionContext
	project          *Project
	projectSettings  *ProjectSettings
	scope            Scope

	rootPath    string
	environment map[string]string
}

func LocalRunnerDef() *RunnerDef {
	return &RunnerDef{
		Name: "local",
		Cfg: &LocalRunnerCfg{
			RootDirectory: "tmp/local-execution",
		},
		Instantiate: NewLocalRunner,
	}
}

func NewLocalRunner(r *Runner) RunnerBehaviour {
	cfg := r.cfg.(*LocalRunnerCfg)

	je := r.jobExecution

	rootDirPath := cfg.RootDirectory
	rootPath := path.Join(rootDirPath, je.Id.String())

	return &LocalRunner{
		runner: r,
		log:    r.log,
		daemon: r.daemon,

		jobExecution:     je,
		stepExecutions:   r.stepExecutions,
		executionContext: r.executionContext,
		project:          r.project,
		projectSettings:  r.projectSettings,
		scope:            r.scope,

		rootPath: rootPath,
	}
}

func (r *LocalRunner) Start() error {
	r.runner.wg.Add(1)
	go r.main()

	return nil
}

func (r *LocalRunner) Stopping() bool {
	select {
	case <-r.runner.stopChan:
		return true
	default:
		return false
	}
}

func (r *LocalRunner) main() {
	defer r.runner.wg.Done()
	defer r.clearEnvironment()

	var currentStepExecution *StepExecution

	jeId := r.jobExecution.Id

	defer func() {
		if value := recover(); value != nil {
			msg, trace := utils.RecoverValueData(value)
			r.log.Error("panic: %s\n%s", msg, trace)

			panicErr := fmt.Errorf("panic: %s", msg)

			if se := currentStepExecution; se != nil {
				_, _, err := r.runner.UpdateStepExecutionFailure(jeId, se.Id,
					panicErr, r.scope)
				if err != nil {
					r.handleError(fmt.Errorf("cannot update step %d: %w",
						se.Position, err))
					return
				}
			}

			r.handleError(panicErr)
		}
	}()

	r.log.Info("starting execution")

	if err := r.initEnvironment(); err != nil {
		r.handleError(fmt.Errorf("cannot initialize environment: %w", err))
		return
	}

	for i, se := range r.stepExecutions {
		if r.Stopping() {
			r.handleInterruption()
			return
		}

		step := r.jobExecution.JobSpec.Steps[i]

		r.log.Info("executing step %d", se.Position)

		_, _, err := r.runner.UpdateStepExecutionStart(jeId, se.Id, r.scope)
		if err != nil {
			r.handleError(fmt.Errorf("cannot update step %d: %w",
				se.Position, err))
			return
		}

		currentStepExecution = se

		err = r.executeStep(i, se)

		var stepFailureErr *StepFailureError
		if errors.As(err, &stepFailureErr) {
			_, _, updateErr := r.runner.UpdateStepExecutionFailure(jeId,
				se.Id, err, r.scope)
			if updateErr != nil {
				r.handleError(fmt.Errorf("cannot update step %d: %w",
					se.Position, err))
				return
			}
		}

		if err != nil {
			if stepFailureErr == nil || step.AbortOnFailure() {
				r.handleError(fmt.Errorf("cannot execute step %d: %w",
					se.Position, err))
				return
			}
		}

		if stepFailureErr == nil {
			_, _, err := r.runner.UpdateStepExecutionSuccess(jeId, se.Id,
				r.scope)
			if err != nil {
				r.handleError(fmt.Errorf("cannot update step %d: %w",
					se.Position, err))
				return
			}
		}
	}

	r.log.Info("execution finished")

	_, err := r.runner.UpdateJobExecutionSuccess(jeId, r.scope)
	if err != nil {
		r.log.Error("cannot update job: %v", err)
		return
	}
}

func (r *LocalRunner) handleInterruption() {
	r.log.Info("execution interrupted")

	je, ses, err := r.runner.UpdateJobExecutionAbortion(r.jobExecution.Id,
		r.scope)
	if err != nil {
		r.log.Error("%v", err)
	}

	r.jobExecution = je
	r.stepExecutions = ses
}

func (r *LocalRunner) handleError(err error) {
	r.log.Error("%v", err)

	je, ses, err := r.runner.UpdateJobExecutionFailure(r.jobExecution.Id,
		err, r.scope)
	if err != nil {
		r.log.Error("%v", err)
	}

	r.jobExecution = je
	r.stepExecutions = ses
}

func (r *LocalRunner) initEnvironment() error {
	// Root directory
	if err := os.RemoveAll(r.rootPath); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("cannot delete directory %q: %w",
				r.rootPath, err)
		}
	}

	if err := os.MkdirAll(r.rootPath, 0700); err != nil {
		return fmt.Errorf("cannot create directory %q: %w", r.rootPath, err)
	}

	// Execution context file
	ectxPath := path.Join(r.rootPath, "context.json")

	if err := r.executionContext.WriteFile(ectxPath); err != nil {
		return fmt.Errorf("cannot write execution context to %q: %w",
			ectxPath, err)
	}

	// Step scripts
	stepDirPath := path.Join(r.rootPath, "steps")

	if err := os.MkdirAll(stepDirPath, 0700); err != nil {
		return fmt.Errorf("cannot create directory %q: %w", stepDirPath, err)
	}

	for i, step := range r.jobExecution.JobSpec.Steps {
		if err := r.writeStepData(i, step, stepDirPath); err != nil {
			return fmt.Errorf("cannot write data for step %d: %w",
				i+1, err)
		}
	}

	return nil
}

func (r *LocalRunner) writeStepData(i int, step *Step, stepDirPath string) error {
	if step.Code != "" || step.Script != nil {
		var code string

		if step.Code != "" {
			code = step.Code
		} else if step.Script != nil {
			code = step.Script.Content
		}

		var buf bytes.Buffer
		if !StartsWithShebang(code) {
			buf.WriteString(r.projectSettings.CodeHeader)
		}
		buf.WriteString(code)

		filePath := path.Join(stepDirPath, strconv.Itoa(i+1))
		if err := os.WriteFile(filePath, buf.Bytes(), 0700); err != nil {
			return fmt.Errorf("cannot write %q: %w", filePath, err)
		}
	} else if step.Bundle != nil {
		bundlePath := path.Join(stepDirPath, strconv.Itoa(i+1))
		if err := os.MkdirAll(bundlePath, 0700); err != nil {
			return fmt.Errorf("cannot create directory %q: %w",
				bundlePath, err)
		}

		for _, bundleFile := range step.Bundle.Files {
			filePath := path.Join(bundlePath, bundleFile.Name)
			fileDirPath := path.Dir(filePath)

			if err := os.MkdirAll(fileDirPath, 0700); err != nil {
				return fmt.Errorf("cannot create directory %q: %w",
					fileDirPath, err)
			}

			content := []byte(bundleFile.Content)

			err := os.WriteFile(filePath, content, bundleFile.Mode)
			if err != nil {
				return fmt.Errorf("cannot write %q: %w", filePath, err)
			}
		}
	}

	return nil
}

func (r *LocalRunner) clearEnvironment() {
	if err := os.RemoveAll(r.rootPath); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			r.log.Error("cannot delete directory %q: %v", r.rootPath, err)
		}
	}
}

func (r *LocalRunner) executeStep(i int, se *StepExecution) error {
	// Interruption handling (i.e. when the server is being stopped while jobs
	// are running).
	ctx, cancel := context.WithCancel(context.Background())

	endChan := make(chan struct{})
	defer close(endChan)

	go func() {
		select {
		case <-r.runner.stopChan:
			r.log.Info("interrupting job")
			cancel()
			return

		case <-endChan:
			cancel()
			return
		}
	}()

	// Create the command
	s := r.jobExecution.JobSpec.Steps[i]
	cmdName, cmdArgs := r.stepCommand(se, s)

	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)

	cmd.Dir = r.rootPath

	cmd.Env = make([]string, 0, len(r.environment))
	for k, v := range r.runner.environment {
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

	var wg sync.WaitGroup
	wg.Add(2)
	go r.readOutput(se, stdout, "stdout", errChan, &wg)
	go r.readOutput(se, stderr, "stderr", errChan, &wg)

	wg.Wait()

	// Now that output readers are terminated, check the error channel for any
	// output error.
	var outputErr error

	select {
	case outputErr = <-errChan:
		if outputErr != nil {
			cmd.Wait()
			close(errChan)
			return outputErr
		}

	default:
		close(errChan)
	}

	// Wait for the command termination status
	err = cmd.Wait()

	// Handle the error if there is one. We translate it to get nice error
	// messages.
	if err != nil {
		var exitErr *exec.ExitError

		if errors.As(err, &exitErr) {
			return NewStepFailureError(r.translateExitError(exitErr))
		}

		return err
	}

	return nil
}

func (r *LocalRunner) readOutput(se *StepExecution, output io.ReadCloser, name string, errChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	bufferedOutput := bufio.NewReader(output)

	var line []byte

	// TODO Only flush every X seconds

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
			err = r.daemon.Pg.WithConn(func(conn pg.Conn) (err error) {
				err = se.UpdateOutput(conn, append(line, '\n'))
				return
			})
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

func (r *LocalRunner) stepCommand(se *StepExecution, s *Step) (name string, args []string) {
	switch {
	case s.Code != "":
		name = path.Join("steps", strconv.Itoa(se.Position))

	case s.Command != nil:
		name = s.Command.Name
		args = s.Command.Arguments

	case s.Script != nil:
		name = path.Join("steps", strconv.Itoa(se.Position))
		args = s.Script.Arguments

	case s.Bundle != nil:
		name = path.Join("steps", strconv.Itoa(se.Position), s.Bundle.Command)
		args = s.Bundle.Arguments

	default:
		utils.Panicf("unhandled step %#v", s)
	}

	return
}

func (r *LocalRunner) translateExitError(err *exec.ExitError) error {
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
