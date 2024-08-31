package eventline

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/exograd/eventline/pkg/utils"
	"go.n16f.net/ejson"
	"go.n16f.net/log"
	"go.n16f.net/program"
	"go.n16f.net/service/pkg/pg"
)

var RunnerDefs = map[string]*RunnerDef{}

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

type RunnerCfg interface {
	ejson.Validatable
}

type RunnerDef struct {
	Name                  string
	Cfg                   RunnerCfg
	InstantiateParameters func() RunnerParameters
	InstantiateBehaviour  func(*Runner) RunnerBehaviour
}

type RunnerInitData struct {
	Log *log.Logger
	Pg  *pg.Client

	Def  *RunnerDef
	Cfg  RunnerCfg
	Data *RunnerData

	TerminationChan chan<- Id

	RefreshInterval time.Duration

	StopChan <-chan struct{}
	Wg       *sync.WaitGroup
}

type RunnerData struct {
	JobExecution     *JobExecution
	StepExecutions   StepExecutions
	ExecutionContext *ExecutionContext
	Project          *Project
	ProjectSettings  *ProjectSettings
}

type RunnerBehaviour interface {
	DirPath() string

	Init(ctx context.Context) error
	Terminate()

	ExecuteStep(context.Context, *StepExecution, *Step, io.WriteCloser, io.WriteCloser) error
}

type Runner struct {
	Log       *log.Logger
	Pg        *pg.Client
	Cfg       RunnerCfg
	Behaviour RunnerBehaviour

	JobExecution     *JobExecution
	StepExecutions   StepExecutions
	ExecutionContext *ExecutionContext
	Project          *Project
	ProjectSettings  *ProjectSettings

	RunnerIdentity *Identity

	Environment map[string]string
	FileSet     *FileSet
	Scope       Scope

	// Keep a private copy to avoid potential concurrency issues with
	// JobExecution since we need the id in both the main goroutine and the
	// refresh goroutine.
	jeId Id

	refreshInterval time.Duration

	terminationChan chan<- Id

	StopChan <-chan struct{}
	Wg       *sync.WaitGroup
}

func NewRunner(data RunnerInitData) (*Runner, error) {
	fileSet, err := data.Data.FileSet()
	if err != nil {
		return nil, fmt.Errorf("cannot create file set: %w", err)
	}

	r := &Runner{
		Log: data.Log,
		Pg:  data.Pg,
		Cfg: data.Cfg,

		JobExecution:     data.Data.JobExecution,
		StepExecutions:   data.Data.StepExecutions,
		ExecutionContext: data.Data.ExecutionContext,
		Project:          data.Data.Project,
		ProjectSettings:  data.Data.ProjectSettings,

		Environment: data.Data.Environment(),
		FileSet:     fileSet,
		Scope:       NewProjectScope(data.Data.Project.Id),

		jeId: data.Data.JobExecution.Id,

		refreshInterval: data.RefreshInterval,

		terminationChan: data.TerminationChan,

		StopChan: data.StopChan,
		Wg:       data.Wg,
	}

	if runner := data.Data.JobExecution.JobSpec.Runner; runner != nil {
		if iname := runner.Identity; iname != "" {
			identities := data.Data.ExecutionContext.Identities

			identity, found := identities[iname]
			if !found {
				// That should never happen since ExecutionContext.Load is
				// supposed to load all identities referenced by the job
				// specification.
				return nil, fmt.Errorf("missing runner identity %q", iname)
			}

			r.RunnerIdentity = identity
		}
	}

	r.Behaviour = data.Def.InstantiateBehaviour(r)

	r.Environment["EVENTLINE_DIR"] = r.Behaviour.DirPath()

	return r, nil
}

func (r *Runner) Start() error {
	r.Wg.Add(1)
	go r.main()

	return nil
}

func (r *Runner) Stopping() bool {
	select {
	case <-r.StopChan:
		return true
	default:
		return false
	}
}

func (r *Runner) main() {
	defer r.Wg.Done()

	defer r.Behaviour.Terminate()

	var cse *StepExecution

	defer func() { r.terminationChan <- r.jeId }()

	defer func() {
		if value := recover(); value != nil {
			msg := program.RecoverValueString(value)
			trace := program.StackTrace(0, 20, true)

			r.Log.Error("panic: %s\n%s", msg, trace)

			panicErr := fmt.Errorf("panic: %s", msg)

			if cse != nil {
				_, _, err := r.updateStepExecutionFailure(r.jeId, cse.Id,
					panicErr, r.Scope)
				if err != nil {
					r.HandleError(fmt.Errorf("cannot update step %d: %w",
						cse.Position, err))
					return
				}
			}

			r.HandleError(panicErr)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-r.StopChan:
			cancel()

		case <-ctx.Done():
		}
	}()

	if err := r.initExecution(ctx); err != nil {
		r.HandleError(err)
		return
	}

	go r.mainRefresh(ctx, cancel)

	r.Log.Info("starting execution")

	for i, se := range r.StepExecutions {
		if r.Stopping() {
			r.HandleInterruption()
			return
		}

		step := r.JobExecution.JobSpec.Steps[i]

		r.Log.Info("executing step %d", se.Position)

		// We update the step execution here and not in Runner.executeStep
		// because we want to set the current step execution (cse) after the
		// start but before calling executeStep, to make sure the recovery
		// function works as intended.
		_, _, err := r.updateStepExecutionStart(r.jeId, se.Id, r.Scope)
		if err != nil {
			r.HandleError(fmt.Errorf("cannot update step %d: %w",
				se.Position, err))
			return
		}

		cse = se

		if err := r.executeStep(ctx, se, step); err != nil {
			r.HandleError(err)
			return
		}
	}

	r.Log.Info("execution finished")

	if _, err := r.updateJobExecutionSuccess(r.jeId, r.Scope); err != nil {
		r.Log.Error("cannot update job execution: %v", err)
		return
	}
}

func (r *Runner) mainRefresh(ctx context.Context, cancel context.CancelFunc) {
	ticker := time.NewTicker(r.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:

		case <-ctx.Done():
			return
		}

		if _, err := r.refreshJobExecution(); err != nil {
			r.Log.Error("cannot refresh job execution: %v", err)

			var jobExecutionFinishedErr *JobExecutionFinishedError
			if errors.As(err, &jobExecutionFinishedErr) {
				cancel()
			}

			return
		}
	}
}

func (r *Runner) initExecution(ctx context.Context) error {
	if err := r.Behaviour.Init(ctx); err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			return fmt.Errorf("initialization interrupted")

		case errors.Is(err, context.DeadlineExceeded):
			return fmt.Errorf("initialization timeout")

		default:
			return err
		}
	}

	return nil
}

func (r *Runner) executeStep(ctx context.Context, se *StepExecution, step *Step) error {
	jeId := r.JobExecution.Id

	// Create pipes used to read the output of the executed program
	stdoutRead, stdoutWrite := io.Pipe()
	stderrRead, stderrWrite := io.Pipe()

	// Create output readers
	errChan := make(chan error, 2)
	defer close(errChan)

	var wg sync.WaitGroup
	wg.Add(2)
	go r.readOutput(se, stdoutRead, "stdout", errChan, &wg)
	go r.readOutput(se, stderrRead, "stderr", errChan, &wg)

	// Execute the step
	err := r.Behaviour.ExecuteStep(ctx, se, step, stdoutWrite, stderrWrite)

	// Close pipes and wait for output readers to terminate
	stdoutRead.Close()
	stderrRead.Close()

	wg.Wait()

	// Check for reader errors; in practice, the only possible error is an
	// unability to update the step execution.
	select {
	case outputErr := <-errChan:
		if outputErr != nil {
			return outputErr
		}

	default:
	}

	// Handle the execution result
	//
	// We separate errors indicating that the command failed and errors
	// associated with execution itself.
	//
	// We only handle the step execution; if this function returns an error,
	// the caller will update the job execution.
	if err != nil {
		var stepFailureErr *StepFailureError

		switch {
		case errors.Is(err, context.Canceled):
			return fmt.Errorf("execution of step %d interrupted", se.Position)

		case errors.Is(err, context.DeadlineExceeded):
			return fmt.Errorf("execution of step %d timed out", se.Position)

		case errors.As(err, &stepFailureErr):
			_, _, updateErr := r.updateStepExecutionFailure(jeId, se.Id, err,
				r.Scope)
			if updateErr != nil {
				return fmt.Errorf("cannot update step execution %q: %w",
					se.Id, err)
			}

			if step.AbortOnFailure() {
				return fmt.Errorf("cannot execute step %d: %w",
					se.Position, err)
			}

			return nil

		default:
			// Even if the job is supposed to continue (i.e. if the step has
			// on_failure equal to 'continue'), an execution error always
			// causes the job to fail: if we cannot execute a job, we probably
			// will not be able to execute the next one.
			return fmt.Errorf("cannot execute step %d: %w", se.Position, err)
		}
	}

	// Mark the step as successful
	_, _, err = r.updateStepExecutionSuccess(jeId, se.Id, r.Scope)
	if err != nil {
		return fmt.Errorf("cannot update step %d: %w", se.Position, err)
	}

	return nil
}

func (r *Runner) readOutput(se *StepExecution, output io.ReadCloser, name string, errChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	bufferedOutput := bufio.NewReader(output)
	var outputSize int
	var line []byte

	lastUpdate := time.Now()
	updatePeriod := time.Duration(1 * time.Second)

	for {
		data, isPrefix, err := bufferedOutput.ReadLine()
		if err != nil && !errors.Is(err, io.ErrClosedPipe) {
			errChan <- fmt.Errorf("cannot read command output %q: %v",
				name, err)
			return
		}

		if err == nil {
			line = append(line, data...)
			if isPrefix {
				continue
			}
		}

		if len(line) > 0 && line[len(line)-1] != '\n' {
			line = append(line, '\n')
		}

		// There is no point in updating se.Output because we are not going to
		// read it in the runner, so we may as well avoid allocating and
		// copying data. This only works because se.Update does not modify the
		// output column, so it will not be erased when updating the step
		// execution later.

		isEOF := errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe)

		if len(line) > 0 && (time.Since(lastUpdate) >= updatePeriod || isEOF) {
			err = r.UpdateStepExecutionOutput(se, line)
			if err != nil {
				errChan <- fmt.Errorf("cannot update step execution %q: %v",
					se.Id, err)
				return
			}

			outputSize += len(line)
			line = nil

			lastUpdate = time.Now()
		}

		if isEOF {
			break
		}
	}
}

func (r *Runner) HandleInterruption() {
	r.Log.Info("execution interrupted")

	je, ses, err := r.updateJobExecutionAbortion(r.JobExecution.Id, r.Scope)
	if err != nil {
		r.Log.Error("%v", err)
	}

	r.JobExecution = je
	r.StepExecutions = ses
}

func (r *Runner) HandleError(err error) {
	r.Log.Error("%v", err)

	je, ses, err := r.updateJobExecutionFailure(r.JobExecution.Id, err,
		r.Scope)
	if err != nil {
		r.Log.Error("%v", err)
	}

	r.JobExecution = je
	r.StepExecutions = ses
}

func (rd *RunnerData) Environment() map[string]string {
	env := map[string]string{
		"EVENTLINE":                  "true",
		"EVENTLINE_PROJECT_ID":       rd.Project.Id.String(),
		"EVENTLINE_PROJECT_NAME":     rd.Project.Name,
		"EVENTLINE_JOB_ID":           rd.JobExecution.JobId.String(),
		"EVENTLINE_JOB_NAME":         rd.JobExecution.JobSpec.Name,
		"EVENTLINE_JOB_EXECUTION_ID": rd.JobExecution.Id.String(),
	}

	for _, i := range rd.ExecutionContext.Identities {
		for name, value := range i.Data.Environment() {
			env[name] = value
		}
	}

	for name, value := range rd.JobExecution.JobSpec.Environment {
		env[name] = value
	}

	for _, param := range rd.JobExecution.JobSpec.Parameters {
		if param.Environment == "" {
			continue
		}

		if value, found := rd.JobExecution.Parameters[param.Name]; found {
			env[param.Environment] = param.ValueString(value)
		}
	}

	return env
}

func (rd *RunnerData) FileSet() (*FileSet, error) {
	fs := NewFileSet()

	// Execution context
	ectxData, err := rd.ExecutionContext.Encode()
	if err != nil {
		return nil, fmt.Errorf("cannot encode execution context: %w", err)
	}
	fs.AddFile("context.json", ectxData, 0600)

	// Step files
	for i, step := range rd.JobExecution.JobSpec.Steps {
		if step.Code != "" || step.Script != nil {
			var code string

			if step.Code != "" {
				code = step.Code
			} else if step.Script != nil {
				code = step.Script.Content
			}

			var buf bytes.Buffer
			if !StartsWithShebang(code) {
				buf.WriteString(rd.ProjectSettings.CodeHeader)
			}
			buf.WriteString(code)

			filePath := path.Join("steps", strconv.Itoa(i+1))

			fs.AddFile(filePath, buf.Bytes(), 0700)
		}
	}

	// Parameters
	parameterFields, err := JSONFields(rd.ExecutionContext.Parameters)
	if err != nil {
		return nil, fmt.Errorf("cannot extract parameter fields: %w", err)
	}

	for name, value := range parameterFields {
		filePath := path.Join("parameters", name)
		fs.AddFile(filePath, []byte(value), 0600)
	}

	// Event fields
	if rd.ExecutionContext.Event != nil {
		eventFields, err := JSONFields(rd.ExecutionContext.Event.Data)
		if err != nil {
			return nil, fmt.Errorf("cannot extract event fields: %w", err)
		}

		for name, value := range eventFields {
			filePath := path.Join("event", name)
			fs.AddFile(filePath, []byte(value), 0600)
		}
	}

	// Identity fields
	for iname, identity := range rd.ExecutionContext.Identities {
		identityFields, err := JSONFields(identity.Data)
		if err != nil {
			return nil, fmt.Errorf("cannot extract identity fields: %w", err)
		}

		for name, value := range identityFields {
			filePath := path.Join("identities", iname, name)
			fs.AddFile(filePath, []byte(value), 0600)
		}
	}

	return fs, nil
}

func (r *Runner) StepCommand(se *StepExecution, s *Step, rootPath string) (name string, args []string) {
	switch {
	case s.Code != "":
		name = path.Join(rootPath, "steps", strconv.Itoa(se.Position))

	case s.Command != nil:
		name = s.Command.Name
		args = s.Command.Arguments

	case s.Script != nil:
		name = path.Join(rootPath, "steps", strconv.Itoa(se.Position))
		args = s.Script.Arguments

	default:
		program.Panicf("unhandled step %#v", s)
	}

	return
}

func (r *Runner) StepCommandString(se *StepExecution, s *Step, rootPath string) string {
	name, args := r.StepCommand(se, s, rootPath)

	var buf bytes.Buffer

	buf.WriteString(name)

	for _, arg := range args {
		buf.WriteByte(' ')
		buf.WriteString(utils.ShellEscape(arg))
	}

	return buf.String()
}

func (r *Runner) updateJobExecutionSuccess(jeId Id, scope Scope) (*JobExecution, error) {
	var je JobExecution

	err := r.Pg.WithTx(func(conn pg.Conn) error {
		if err := je.LoadForUpdate(conn, jeId, scope); err != nil {
			return fmt.Errorf("cannot load job execution: %w", err)
		}

		if je.Status == JobExecutionStatusAborted {
			return &JobExecutionAbortedError{Id: jeId}
		}

		now := time.Now().UTC()

		je.Status = JobExecutionStatusSuccessful
		je.EndTime = &now
		je.RefreshTime = nil

		if err := je.Update(conn); err != nil {
			return fmt.Errorf("cannot update job execution: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &je, nil
}

func (r *Runner) updateJobExecutionAbortion(jeId Id, scope Scope) (*JobExecution, StepExecutions, error) {
	var je JobExecution
	var ses StepExecutions

	err := r.Pg.WithTx(func(conn pg.Conn) error {
		if err := je.LoadForUpdate(conn, jeId, scope); err != nil {
			return fmt.Errorf("cannot load job execution: %w", err)
		}

		if je.Status == JobExecutionStatusAborted {
			return &JobExecutionAbortedError{Id: jeId}
		}

		now := time.Now().UTC()

		je.Status = JobExecutionStatusAborted
		je.EndTime = &now
		je.RefreshTime = nil

		if err := je.Update(conn); err != nil {
			return fmt.Errorf("cannot update job execution: %w", err)
		}

		if err := ses.LoadByJobExecutionId(conn, jeId); err != nil {
			return fmt.Errorf("cannot load step executions: %w", err)
		}

		for _, se := range ses {
			if !se.Finished() {
				se.Status = StepExecutionStatusAborted
				if se.StartTime != nil {
					se.EndTime = &now
				}

				if err := se.Update(conn); err != nil {
					return fmt.Errorf("cannot update step %d: %w",
						se.Position, err)
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return &je, ses, nil
}

func (r *Runner) updateJobExecutionFailure(jeId Id, jeErr error, scope Scope) (*JobExecution, StepExecutions, error) {
	var je JobExecution
	var ses StepExecutions

	err := r.Pg.WithTx(func(conn pg.Conn) error {
		if err := je.LoadForUpdate(conn, jeId, scope); err != nil {
			return fmt.Errorf("cannot load job execution: %w", err)
		}

		if je.Status == JobExecutionStatusAborted {
			return &JobExecutionAbortedError{Id: jeId}
		}

		now := time.Now().UTC()

		je.Status = JobExecutionStatusFailed
		je.EndTime = &now
		je.FailureMessage = jeErr.Error()
		je.RefreshTime = nil

		if err := je.Update(conn); err != nil {
			return fmt.Errorf("cannot update job execution: %w", err)
		}

		if err := ses.LoadByJobExecutionId(conn, jeId); err != nil {
			return fmt.Errorf("cannot load step executions: %w", err)
		}

		for _, se := range ses {
			if !se.Finished() {
				se.Status = StepExecutionStatusAborted
				if se.StartTime != nil {
					se.EndTime = &now
				}

				if err := se.Update(conn); err != nil {
					return fmt.Errorf("cannot update step %d: %w",
						se.Position, err)
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return &je, ses, nil
}

func (r *Runner) updateStepExecutionStart(jeId, seId Id, scope Scope) (*JobExecution, *StepExecution, error) {
	return r.updateStepExecution(jeId, seId, func(se *StepExecution) {
		now := time.Now().UTC()

		se.Status = StepExecutionStatusStarted
		se.StartTime = &now
		se.FailureMessage = ""
		se.Output = ""
	}, scope)
}

func (r *Runner) updateStepExecutionAborted(jeId, seId Id, scope Scope) (*JobExecution, *StepExecution, error) {
	return r.updateStepExecution(jeId, seId, func(se *StepExecution) {
		now := time.Now().UTC()

		se.Status = StepExecutionStatusAborted
		se.EndTime = &now
	}, scope)
}

func (r *Runner) updateStepExecutionSuccess(jeId, seId Id, scope Scope) (*JobExecution, *StepExecution, error) {
	return r.updateStepExecution(jeId, seId, func(se *StepExecution) {
		now := time.Now().UTC()

		se.Status = StepExecutionStatusSuccessful
		se.EndTime = &now
	}, scope)
}

func (r *Runner) updateStepExecutionFailure(jeId, seId Id, err error, scope Scope) (*JobExecution, *StepExecution, error) {
	return r.updateStepExecution(jeId, seId, func(se *StepExecution) {
		now := time.Now().UTC()

		se.Status = StepExecutionStatusFailed
		se.EndTime = &now
		se.FailureMessage = err.Error()
	}, scope)
}

func (r *Runner) updateStepExecution(jeId, seId Id, fn func(*StepExecution), scope Scope) (*JobExecution, *StepExecution, error) {
	var je JobExecution
	var se StepExecution

	err := r.Pg.WithTx(func(conn pg.Conn) error {
		if err := je.LoadForUpdate(conn, jeId, scope); err != nil {
			return fmt.Errorf("cannot load job execution: %w", err)
		}

		if je.Status == JobExecutionStatusAborted {
			return &JobExecutionAbortedError{Id: jeId}
		}

		if err := se.Load(conn, seId, scope); err != nil {
			return fmt.Errorf("cannot load step execution: %w", err)
		}

		fn(&se)

		if err := se.Update(conn); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return &je, &se, nil
}

func (r *Runner) UpdateStepExecutionOutput(se *StepExecution, data []byte) error {
	return r.Pg.WithConn(func(conn pg.Conn) (err error) {
		err = se.UpdateOutput(conn, data)
		return
	})
}

func (r *Runner) refreshJobExecution() (*JobExecution, error) {
	var je JobExecution

	err := r.Pg.WithTx(func(conn pg.Conn) error {
		if err := je.LoadForUpdate(conn, r.jeId, r.Scope); err != nil {
			return fmt.Errorf("cannot load job execution: %w", err)
		}

		if je.Finished() {
			return &JobExecutionFinishedError{Id: r.jeId}
		}

		now := time.Now().UTC()

		je.RefreshTime = &now

		if err := je.UpdateRefreshTime(conn); err != nil {
			return fmt.Errorf("cannot update job execution: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &je, nil
}
