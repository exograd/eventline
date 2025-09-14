package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	rlocal "github.com/exograd/eventline/pkg/runners/local"
	"github.com/exograd/eventline/pkg/utils"
	"go.n16f.net/ejson"
	"go.n16f.net/service/pkg/pg"
	"go.n16f.net/uuid"
)

type JobSpecValidator struct {
	JobSpec *eventline.JobSpec

	Identities map[string]*eventline.Identity

	Service   *Service
	Validator *ejson.Validator
	Conn      pg.Conn
	Scope     eventline.Scope
}

func (s *Service) ValidateJobSpec(conn pg.Conn, spec *eventline.JobSpec, scope eventline.Scope) error {
	validator := ejson.NewValidator()
	spec.ValidateJSON(validator)

	var identities eventline.Identities
	err := identities.LoadByNamesForUpdate(conn, spec.IdentityNames(), scope)
	if err != nil {
		return fmt.Errorf("cannot load identities: %w", err)
	}

	identityTable := make(map[string]*eventline.Identity)
	for _, i := range identities {
		identityTable[i.Name] = i
	}

	v := JobSpecValidator{
		JobSpec: spec,

		Identities: identityTable,

		Service:   s,
		Validator: validator,
		Conn:      conn,
		Scope:     scope,
	}

	v.checkJobSpec()

	return v.Validator.Error()
}

func (v *JobSpecValidator) checkJobSpec() error {
	// Parameters
	hasMandatoryParams := false
	for _, p := range v.JobSpec.Parameters {
		if p.Default == nil {
			hasMandatoryParams = true
			break
		}
	}

	if hasMandatoryParams && v.JobSpec.Trigger != nil {
		v.Validator.AddError("trigger",
			"invalid_trigger_with_mandatory_parameters",
			"jobs with mandatory parameters cannot have a trigger")
	}

	// Trigger
	if trigger := v.JobSpec.Trigger; trigger != nil {
		v.Validator.WithChild("trigger", func() {
			v.checkTrigger(trigger)
		})
	}

	// Runner
	if runner := v.JobSpec.Runner; runner != nil {
		v.Validator.WithChild("runner", func() {
			if iname := runner.Identity; iname != "" {
				v.checkIdentityName("identity", iname)
			}
		})
	}

	// Identities
	v.Validator.WithChild("identities", func() {
		for idx, iname := range v.JobSpec.Identities {
			v.checkIdentityName(idx, iname)
		}
	})

	return nil
}

func (v *JobSpecValidator) checkTrigger(trigger *eventline.Trigger) {
	cname := trigger.Event.Connector

	// If the connector does not exist, a validation error was added but we
	// still run all other checks.
	if connector, found := eventline.FindConnector(cname); found {
		optConnector, ok := connector.(eventline.OptionalConnector)
		if ok && !optConnector.Enabled() {
			v.Validator.AddError("event", "disabled_connector",
				"connector %q is disabled and cannot be used in triggers",
				cname)
		}
	}

	if iname := trigger.Identity; iname != "" {
		v.checkIdentityName("identity", iname)
	}
}

func (v *JobSpecValidator) checkIdentityName(token interface{}, name string) {
	identity, found := v.Identities[name]
	if !found {
		v.Validator.AddError(token, "unknown_identity", "unknown identity %q",
			name)
		return
	}

	switch identity.Status {
	case eventline.IdentityStatusPending:
		v.Validator.AddError(token, "pending_identity",
			"identity %q is not ready and needs additional configuration",
			name)

	case eventline.IdentityStatusError:
		v.Validator.AddError(token, "malfunctioning_identity",
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

	runnerAllowed := len(s.Cfg.AllowedRunners) == 0 ||
		utils.StringsContain(s.Cfg.AllowedRunners, spec.Runner.Name)
	if !runnerAllowed {
		return nil, false,
			fmt.Errorf("runner %q is not allowed", spec.Runner.Name)
	}

	projectId := scope.(*eventline.ProjectScope).ProjectId

	now := time.Now().UTC()

	job := eventline.Job{
		Id:           uuid.MustGenerate(uuid.V7),
		ProjectId:    projectId,
		CreationTime: now,
		UpdateTime:   now,
		Spec:         spec,
	}

	id, err := job.Upsert(conn)
	if err != nil {
		return nil, false, err
	}

	job.Id = *id

	// Subscription handling
	subscription := new(eventline.Subscription)
	err = subscription.LoadByJobForUpdate(conn, job.Id, scope)
	if err != nil {
		var unknownJobSubscriptionErr *eventline.UnknownJobSubscriptionError

		if !errors.As(err, &unknownJobSubscriptionErr) {
			return nil, false, fmt.Errorf("cannot load subscription: %w", err)
		}

		subscription = nil
	}

	// If there is a trigger, check if it has changed
	triggerChanged := false

	if subscription == nil && spec.Trigger != nil {
		triggerChanged = true
	} else if subscription != nil && spec.Trigger != nil {
		oldSubParams := subscription.Parameters
		newSubParams := spec.Trigger.Parameters

		subParamsEqual :=
			eventline.SubscriptionParametersEqual(oldSubParams,
				newSubParams)

		var oldIdentityName string
		if subscription.IdentityId != nil {
			var oldIdentity eventline.Identity
			err := oldIdentity.Load(conn, *subscription.IdentityId, scope)
			if err != nil {
				return nil, false, fmt.Errorf("cannot load identity %q: %w",
					subscription.IdentityId, err)
			}

			oldIdentityName = oldIdentity.Name
		}

		newIdentityName := spec.Trigger.Identity

		triggerChanged = !subParamsEqual || oldIdentityName != newIdentityName
	}

	var subscriptionCreatedOrUpdated bool

	if subscription != nil && (spec.Trigger == nil || triggerChanged) {
		err := s.TerminateSubscription(conn, subscription, false, scope)
		if err != nil {
			return nil, false,
				fmt.Errorf("cannot terminate subscription: %w", err)
		}
	}

	if spec.Trigger != nil && triggerChanged {
		if _, err := s.CreateSubscription(conn, &job, scope); err != nil {
			return nil, false,
				fmt.Errorf("cannot create subscription: %w", err)
		}

		subscriptionCreatedOrUpdated = true
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

func (s *Service) RenameJob(conn pg.Conn, jobId uuid.UUID, data *eventline.JobRenamingData, scope eventline.Scope) (*eventline.Job, error) {
	var job eventline.Job

	if err := job.LoadForUpdate(conn, jobId, scope); err != nil {
		return nil, fmt.Errorf("cannot load job: %w", err)
	}

	job.Spec.Name = data.Name
	job.Spec.Description = data.Description

	if err := job.UpdateRename(conn, scope); err != nil {
		return nil, fmt.Errorf("cannot update job: %w", err)
	}

	return &job, nil
}

func (s *Service) InstantiateJob(conn pg.Conn, job *eventline.Job, event *eventline.Event, params map[string]interface{}, scope eventline.Scope) (*eventline.JobExecution, error) {
	now := time.Now().UTC()

	projectId := scope.(*eventline.ProjectScope).ProjectId

	// Job
	jobExecution := eventline.JobExecution{
		Id:           uuid.MustGenerate(uuid.V7),
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

	if err := jobExecution.Insert(conn); err != nil {
		return nil, fmt.Errorf("cannot insert job execution: %w", err)
	}

	// Steps
	stepExecutions := make(eventline.StepExecutions, len(job.Spec.Steps))
	for i := range job.Spec.Steps {
		stepExecution := eventline.StepExecution{
			Id:             uuid.MustGenerate(uuid.V7),
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

func (s *Service) EnableJob(conn pg.Conn, jobId uuid.UUID, scope eventline.Scope) (*eventline.Job, error) {
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

func (s *Service) DisableJob(conn pg.Conn, jobId uuid.UUID, scope eventline.Scope) (*eventline.Job, error) {
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

func (s *Service) ExecuteJob(conn pg.Conn, jobId uuid.UUID, input *eventline.JobExecutionInput, scope eventline.Scope) (*eventline.JobExecution, error) {
	var job eventline.Job
	if err := job.Load(conn, jobId, scope); err != nil {
		return nil, fmt.Errorf("cannot load job: %w", err)
	}

	v := ejson.NewValidator()
	job.Spec.Parameters.CheckValues(v, "parameters", input.Parameters)
	if err := v.Error(); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	return s.InstantiateJob(conn, &job, nil, input.Parameters, scope)
}
