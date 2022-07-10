package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	rlocal "github.com/exograd/eventline/pkg/runners/local"
	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/pg"
)

type JobSpecChecker struct {
	JobSpec *eventline.JobSpec

	Identities map[string]*eventline.Identity

	Service *Service
	Checker *check.Checker
	Conn    pg.Conn
	Scope   eventline.Scope
}

func (s *Service) ValidateJobSpec(conn pg.Conn, spec *eventline.JobSpec, scope eventline.Scope) error {
	checker := check.NewChecker()
	spec.Check(checker)

	var identities eventline.Identities
	err := identities.LoadByNamesForUpdate(conn, spec.IdentityNames(), scope)
	if err != nil {
		return fmt.Errorf("cannot load identities: %w", err)
	}

	identityTable := make(map[string]*eventline.Identity)
	for _, i := range identities {
		identityTable[i.Name] = i
	}

	c := JobSpecChecker{
		JobSpec: spec,

		Identities: identityTable,

		Service: s,
		Checker: checker,
		Conn:    conn,
		Scope:   scope,
	}

	c.checkJobSpec()

	return c.Checker.Error()
}

func (c *JobSpecChecker) checkJobSpec() error {
	// Parameters
	hasMandatoryParams := false
	for _, p := range c.JobSpec.Parameters {
		if p.Default == nil {
			hasMandatoryParams = true
			break
		}
	}

	if hasMandatoryParams && c.JobSpec.Trigger != nil {
		c.Checker.AddError("trigger",
			"invalid_trigger_with_mandatory_parameters",
			"jobs with mandatory parameters cannot have a trigger")
	}

	// Trigger
	if trigger := c.JobSpec.Trigger; trigger != nil {
		c.Checker.WithChild("trigger", func() {
			c.checkTrigger(trigger)
		})
	}

	// Runner
	if runner := c.JobSpec.Runner; runner != nil {
		c.Checker.WithChild("runner", func() {
			if iname := runner.Identity; iname != "" {
				c.checkIdentityName("identity", iname)
			}
		})
	}

	// Identities
	c.Checker.WithChild("identities", func() {
		for idx, iname := range c.JobSpec.Identities {
			c.checkIdentityName(idx, iname)
		}
	})

	return nil
}

func (c *JobSpecChecker) checkTrigger(trigger *eventline.Trigger) {
	cname := trigger.Event.Connector

	// If the connector does not exist, a validation error was added but we
	// still run all other checks.
	if connector, found := eventline.FindConnector(cname); found {
		optConnector, ok := connector.(eventline.OptionalConnector)
		if ok && !optConnector.Enabled() {
			c.Checker.AddError("event", "disabled_connector",
				"connector %q is disabled and cannot be used in triggers",
				cname)
		}
	}

	if iname := trigger.Identity; iname != "" {
		c.checkIdentityName("identity", iname)
	}
}

func (c *JobSpecChecker) checkIdentityName(token interface{}, name string) {
	identity, found := c.Identities[name]
	if !found {
		c.Checker.AddError(token, "unknown_identity", "unknown identity %q",
			name)
		return
	}

	switch identity.Status {
	case eventline.IdentityStatusPending:
		c.Checker.AddError(token, "pending_identity",
			"identity %q is not ready and needs additional configuration",
			name)

	case eventline.IdentityStatusError:
		c.Checker.AddError(token, "malfunctioning_identity",
			"identity %q is currently malfunctioning", name)
	}
}

func (s *Service) CreateOrUpdateJob(conn pg.Conn, spec *eventline.JobSpec, scope eventline.Scope) (*eventline.Job, bool, error) {
	if spec.Runner == nil {
		spec.Runner = &eventline.JobRunner{
			Name:       "local",
			Parameters: &rlocal.RunnerParameters{},
		}
	}

	projectId := scope.(*eventline.ProjectScope).ProjectId

	now := time.Now().UTC()

	job := eventline.Job{
		Id:           eventline.GenerateId(),
		ProjectId:    projectId,
		CreationTime: now,
		UpdateTime:   now,
		Spec:         spec,
	}

	id, err := job.Upsert(conn)
	if err != nil {
		return nil, false, err
	}
	job.Id = id

	var subscriptionCreatedOrUpdated bool

	if spec.Trigger != nil {
		var unknownJobSubscriptionErr *eventline.UnknownJobSubscriptionError

		var subscription eventline.Subscription
		err := subscription.LoadByJobForUpdate(conn, job.Id, scope)
		subscriptionExists := (err == nil)
		if err != nil && !errors.As(err, &unknownJobSubscriptionErr) {
			return nil, false, fmt.Errorf("cannot load subscription: %w", err)
		}

		var subParamsEqual bool

		if subscriptionExists {
			oldSubParams := subscription.Parameters
			newSubParams := spec.Trigger.Parameters

			subParamsEqual =
				eventline.SubscriptionParametersEqual(oldSubParams,
					newSubParams)
		}

		if !subParamsEqual {
			if subscriptionExists {
				err = s.TerminateSubscription(conn, &subscription, false,
					scope)
				if err != nil {
					return nil, false,
						fmt.Errorf("cannot terminate subscription: %w", err)
				}
			}

			if _, err := s.CreateSubscription(conn, &job, scope); err != nil {
				return nil, false,
					fmt.Errorf("cannot create subscription: %w", err)
			}

			subscriptionCreatedOrUpdated = true
		}
	}

	return &job, subscriptionCreatedOrUpdated, nil
}

func (s *Service) DeleteJob(conn pg.Conn, job *eventline.Job, scope eventline.Scope) error {
	if job.Spec.Trigger != nil {
		var subscription eventline.Subscription
		err := subscription.LoadByJobForUpdate(conn, job.Id, scope)
		if err != nil {
			return fmt.Errorf("cannot load subscription: %w", err)
		}

		err = s.TerminateSubscription(conn, &subscription, false, scope)
		if err != nil {
			return fmt.Errorf("cannot terminate subscription: %w", err)
		}
	}

	if err := job.Delete(conn, scope); err != nil {
		return err
	}

	return nil
}

func (s *Service) InstantiateJob(conn pg.Conn, job *eventline.Job, event *eventline.Event, params map[string]interface{}, scope eventline.Scope) (*eventline.JobExecution, error) {
	now := time.Now().UTC()

	projectId := scope.(*eventline.ProjectScope).ProjectId

	// Job
	jobExecution := eventline.JobExecution{
		Id:           eventline.GenerateId(),
		ProjectId:    projectId,
		JobId:        job.Id,
		JobSpec:      job.Spec,
		Parameters:   params,
		CreationTime: now,
		UpdateTime:   now,
		Status:       eventline.JobExecutionStatusCreated,
	}

	if event == nil {
		jobExecution.ScheduledTime = now
	} else {
		jobExecution.EventId = &event.Id
		jobExecution.ScheduledTime = event.EventTime
	}

	retention := s.Cfg.JobRetention
	if job.Spec.Retention != 0 {
		retention = job.Spec.Retention
	}
	if retention > 0 {
		expirationTime := now.AddDate(0, 0, retention)
		jobExecution.ExpirationTime = &expirationTime
	}

	if err := jobExecution.Insert(conn); err != nil {
		return nil, fmt.Errorf("cannot insert job execution: %w", err)
	}

	// Steps
	stepExecutions := make(eventline.StepExecutions, len(job.Spec.Steps))
	for i := range job.Spec.Steps {
		stepExecution := eventline.StepExecution{
			Id:             eventline.GenerateId(),
			ProjectId:      projectId,
			JobExecutionId: jobExecution.Id,
			Position:       i + 1,
			Status:         eventline.StepExecutionStatusCreated,
		}

		stepExecutions[i] = &stepExecution
	}

	for _, stepExecution := range stepExecutions {
		if err := stepExecution.Insert(conn); err != nil {
			return nil, fmt.Errorf("cannot insert step execution: %w", err)
		}
	}

	return &jobExecution, nil
}

func (s *Service) EnableJob(conn pg.Conn, jobId eventline.Id, scope eventline.Scope) (*eventline.Job, error) {
	var job eventline.Job

	if err := job.LoadForUpdate(conn, jobId, scope); err != nil {
		return nil, fmt.Errorf("cannot load job: %w", err)
	}

	if !job.Disabled {
		return &job, nil
	}

	now := time.Now().UTC()

	job.Disabled = false
	job.UpdateTime = now

	if err := job.Update(conn, scope); err != nil {
		return nil, fmt.Errorf("cannot update job: %w", err)
	}

	return &job, nil
}

func (s *Service) DisableJob(conn pg.Conn, jobId eventline.Id, scope eventline.Scope) (*eventline.Job, error) {
	var job eventline.Job

	if err := job.LoadForUpdate(conn, jobId, scope); err != nil {
		return nil, fmt.Errorf("cannot load job: %w", err)
	}

	if job.Disabled {
		return &job, nil
	}

	now := time.Now().UTC()

	job.Disabled = true
	job.UpdateTime = now

	if err := job.Update(conn, scope); err != nil {
		return nil, fmt.Errorf("cannot update job: %w", err)
	}

	return &job, nil
}

func (s *Service) ExecuteJob(conn pg.Conn, jobId eventline.Id, input *eventline.JobExecutionInput, scope eventline.Scope) (*eventline.JobExecution, error) {
	var job eventline.Job
	if err := job.Load(conn, jobId, scope); err != nil {
		return nil, fmt.Errorf("cannot load job: %w", err)
	}

	c := check.NewChecker()
	job.Spec.Parameters.CheckValues(c, "parameters", input.Parameters)
	if err := c.Error(); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	return s.InstantiateJob(conn, &job, nil, input.Parameters, scope)
}
