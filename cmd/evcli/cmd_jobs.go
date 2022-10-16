package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/utils"
	"github.com/exograd/go-program"
)

func addJobCommands() {
	var c *program.Command

	// list-jobs
	c = p.AddCommand("list-jobs", "list all jobs",
		cmdListJobs)

	// export-job
	c = p.AddCommand("export-job", "export a job specification",
		cmdExportJob)

	c.AddOption("d", "directory", "path", ".", "the directory to write to")

	c.AddArgument("name", "the name of the job")

	// deploy-job
	c = p.AddCommand("deploy-job",
		"create or update a job from a specification file",
		cmdDeployJob)

	c.AddFlag("n", "dry-run", "validate the job but do not deploy it")

	c.AddArgument("path", "the path of the job specification file")

	// deploy-jobs
	c = p.AddCommand("deploy-jobs",
		"create or update jobs from specification files",
		cmdDeployJobs)

	c.AddFlag("n", "dry-run", "validate jobs but do not deploy them")
	c.AddFlag("r", "recursive", "find job files in nested directories")

	c.AddTrailingArgument("path",
		"the path of a job specification file or directory")

	// delete-job
	c = p.AddCommand("delete-job", "delete a job",
		cmdDeleteJob)

	c.AddArgument("name", "the name of the job")

	// rename-job
	c = p.AddCommand("rename-job", "rename a job",
		cmdRenameJob)

	c.AddArgument("name", "the current name of the job")
	c.AddArgument("new-name", "the new name of the job")

	c.AddOption("d", "description", "text", "",
		"the new description of the job")

	// describe-job
	c = p.AddCommand("describe-job", "print information about a job",
		cmdDescribeJob)

	c.AddArgument("name", "the name of the job")

	// execute-job
	c = p.AddCommand("execute-job", "execute a job",
		cmdExecuteJob)

	c.AddArgument("name", "the name of the job")
	c.AddTrailingArgument("parameter",
		"a parameter passed to the command as <name>=<value>")

	c.AddFlag("w", "wait", "wait for execution to finish")
	c.AddFlag("f", "fail",
		"exit with status 1 if execution does not complete successfully")
}

func cmdListJobs(p *program.Program) {
	app.IdentifyCurrentProject()

	jobs, err := app.Client.FetchJobs()
	if err != nil {
		p.Fatal("cannot fetch jobs: %v", err)
	}

	header := []string{"id", "name", "event", "runner"}
	table := NewTable(header)

	for _, j := range jobs {
		var triggerName string
		if t := j.Spec.Trigger; t != nil {
			triggerName = t.Event.String()
		}

		row := []interface{}{
			j.Id,
			j.Spec.Name,
			triggerName,
			j.Spec.Runner.Name,
		}

		table.AddRow(row)
	}

	table.Write()
}

func cmdExportJob(p *program.Program) {
	app.IdentifyCurrentProject()

	name := p.ArgumentValue("name")
	dirPath := p.OptionValue("directory")

	job, err := app.Client.FetchJobByName(name)
	if err != nil {
		p.Fatal("cannot fetch job: %v", err)
	}

	filePath, err := ExportJob(job.Spec, dirPath)
	if err != nil {
		p.Fatal("cannot export job %q: %v", name, err)
	}

	p.Info("job %q exported to %q", name, filePath)
}

func cmdDeployJob(p *program.Program) {
	app.IdentifyCurrentProject()

	filePath := p.ArgumentValue("path")
	dryRun := p.IsOptionSet("dry-run")

	spec, err := LoadJobFile(filePath)
	if err != nil {
		p.Fatal("cannot load job file: %v", err)
	}

	app.IdentifyCurrentProject()

	job, err := app.Client.DeployJob(spec, dryRun)
	if err != nil {
		isRequestBodyError, verrs := IsInvalidRequestBodyError(err)

		if isRequestBodyError {
			if dryRun {
				p.Error("invalid job")
			} else {
				p.Error("cannot deploy job")
			}

			for _, verr := range verrs {
				p.Error("%s: %v", filePath, verr)
			}

			os.Exit(1)
		} else {
			if dryRun {
				p.Fatal("invalid jobs: %v", err)
			} else {
				p.Fatal("cannot deploy jobs: %v", err)
			}
		}
	}

	if dryRun {
		p.Info("job validated successfully")
	} else {
		p.Info("job %q deployed", job.Id)
	}
}

func cmdDeployJobs(p *program.Program) {
	app.IdentifyCurrentProject()

	fileOrDirPaths := p.TrailingArgumentValues("path")
	dryRun := p.IsOptionSet("dry-run")
	recursive := p.IsOptionSet("recursive")

	app.IdentifyCurrentProject()

	filePaths, err := FindJobFiles(fileOrDirPaths, recursive)
	if err != nil {
		p.Fatal("%v", err)
	}

	specs := make(eventline.JobSpecs, len(filePaths))

	for i, filePath := range filePaths {
		spec, err := LoadJobFile(filePath)
		if err != nil {
			p.Fatal("cannot load %q: %v", filePath, err)
		}

		specs[i] = spec
	}

	p.Info("deploying %d jobs", len(specs))

	if _, err := app.Client.DeployJobs(specs, dryRun); err != nil {
		isRequestBodyError, verrs := IsInvalidRequestBodyError(err)

		if isRequestBodyError {
			if dryRun {
				p.Error("invalid jobs")
			} else {
				p.Error("cannot deploy jobs")
			}

			for _, verr := range verrs {
				if len(verr.Pointer) == 0 {
					p.Error("invalid empty error pointer")
					continue
				}

				i, err := strconv.Atoi(verr.Pointer[0])
				if err != nil || i < 0 || i >= len(filePaths) {
					p.Error("invalid error pointer %v", verr.Pointer)
					continue
				}

				filePath := filePaths[i]

				verr.Pointer = verr.Pointer[1:]

				p.Error("%s: %v", filePath, verr)
			}

			os.Exit(1)
		} else {
			if dryRun {
				p.Fatal("invalid jobs: %v", err)
			} else {
				p.Fatal("cannot deploy jobs: %v", err)
			}
		}
	}

	if dryRun {
		p.Info("jobs validated successfully")
	} else {
		plural := ""
		if len(filePaths) > 1 {
			plural = "s"
		}

		p.Info("%d job%s deployed", len(filePaths), plural)
	}
}

func cmdDeleteJob(p *program.Program) {
	app.IdentifyCurrentProject()

	name := p.ArgumentValue("name")

	job, err := app.Client.FetchJobByName(name)
	if err != nil {
		p.Fatal("cannot fetch job: %v", err)
	}

	if err := app.Client.DeleteJob(job.Id.String()); err != nil {
		p.Fatal("cannot delete job: %v", err)
	}

	p.Info("job %q deleted", job.Id)
}

func cmdRenameJob(p *program.Program) {
	app.IdentifyCurrentProject()

	name := p.ArgumentValue("name")
	newName := p.ArgumentValue("new-name")

	var description *string
	if p.IsOptionSet("description") {
		description = utils.Ref(p.OptionValue("description"))
	}

	job, err := app.Client.FetchJobByName(name)
	if err != nil {
		p.Fatal("cannot fetch job: %v", err)
	}

	data := eventline.JobRenamingData{
		Name:        newName,
		Description: job.Spec.Description,
	}

	if description != nil {
		data.Description = *description
	}

	if err := app.Client.RenameJob(job.Id.String(), &data); err != nil {
		p.Fatal("cannot rename job: %v", err)
	}

	p.Info("job %q renamed", job.Id)
}

func cmdDescribeJob(p *program.Program) {
	app.IdentifyCurrentProject()

	name := p.ArgumentValue("name")

	job, err := app.Client.FetchJobByName(name)
	if err != nil {
		p.Fatal("cannot fetch job: %v", err)
	}

	fmt.Printf("%s %s\n",
		Colorize(ColorYellow, "Name:"), job.Spec.Name)

	if job.Spec.Description != "" {
		fmt.Printf("%s %s\n",
			Colorize(ColorYellow, "Description:"), job.Spec.Description)
	}

	if job.Spec.Trigger != nil {
		fmt.Printf("%s %s\n",
			Colorize(ColorYellow, "Trigger event:"), job.Spec.Trigger.Event)
	}

	fmt.Printf("%s %s\n",
		Colorize(ColorYellow, "Runner name:"), job.Spec.Runner.Name)

	if len(job.Spec.Parameters) > 0 {
		fmt.Printf("%s\n", Colorize(ColorYellow, "Parameters:"))
		for _, p := range job.Spec.Parameters {
			fmt.Printf("  - %s: %s\n",
				Colorize(ColorYellow, p.Name),
				Colorize(ColorGreen, string(p.Type)))

			if p.Description != "" {
				fmt.Printf("    %s\n", p.Description)
			}

			if p.Default != nil {
				defaultString := fmt.Sprintf("%v", p.Default)
				fmt.Printf("    Default: %s\n",
					Colorize(ColorRed, defaultString))
			}
		}
	}
}

func cmdExecuteJob(p *program.Program) {
	app.IdentifyCurrentProject()

	name := p.ArgumentValue("name")
	paramStrings := p.TrailingArgumentValues("parameter")

	wait := p.IsOptionSet("wait")
	fail := p.IsOptionSet("fail")

	if fail && !wait {
		p.Fatal("the --fail option is only supported if the --wait option " +
			" is set")
	}

	job, err := app.Client.FetchJobByName(name)
	if err != nil {
		p.Fatal("cannot fetch job: %v", err)
	}

	params, err := parseCommandParameters(paramStrings, job.Spec.Parameters)
	if err != nil {
		p.Fatal("%v", err)
	}

	input := eventline.JobExecutionInput{
		Parameters: params,
	}

	jobExecution, err := app.Client.ExecuteJob(job.Id.String(), &input)
	if err != nil {
		var apiErr *APIError

		if errors.As(err, &apiErr) && apiErr.Code == "invalid_request_body" {
			const prefix = "/parameters/"

			data := apiErr.Data.(InvalidRequestBodyError)

			for _, verr := range data.ValidationErrors {
				pointer := verr.Pointer.String()
				if strings.HasPrefix(pointer, prefix) {
					name := pointer[len(prefix):]
					p.Error("invalid parameter %q: %s", name, verr.Message)
				} else {
					p.Error("%s", verr)
				}
			}

			p.Fatal("cannot execute job")
		} else {
			p.Fatal("cannot execute job: %v", err)
		}
	}

	jeId := jobExecution.Id

	p.Info("job execution %q created", jeId)

	if !wait {
		return
	}

	lastStatus := jobExecution.Status

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-sigChan:
			return
		case <-time.After(time.Second):
		}

		je, err := app.Client.FetchJobExecution(jeId)
		if err != nil {
			p.Fatal("cannot fetch job execution: %v", err)
		}

		if je.Status != lastStatus {
			switch je.Status {
			case eventline.JobExecutionStatusFailed:
				p.Info("job execution %s: %s", je.Status, je.FailureMessage)
			default:
				p.Info("job execution %s", je.Status)
			}

			lastStatus = je.Status
		}

		if je.Finished() {
			d := je.EndTime.Sub(*je.StartTime)
			p.Info("job execution finished in %s", utils.FormatDuration(d))

			if je.Status == eventline.JobExecutionStatusFailed && fail {
				os.Exit(1)
			}

			break
		}
	}
}

func parseCommandParameters(ss []string, params eventline.Parameters) (map[string]interface{}, error) {
	values := make(map[string]interface{})

	for _, s := range ss {
		name, value, err := parseCommandParameter(s, params)
		if err != nil {
			return nil, err
		}

		values[name] = value
	}

	for _, p := range params {
		if p.Default != nil {
			continue
		}

		if _, found := values[p.Name]; !found {
			return nil, fmt.Errorf("missing parameter %q", p.Name)
		}
	}

	return values, nil
}

func parseCommandParameter(s string, params eventline.Parameters) (string, interface{}, error) {
	parts := strings.SplitN(s, "=", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("invalid parameter format %q", s)
	}

	name := parts[0]
	valueString := parts[1]

	var p *eventline.Parameter
	for _, pp := range params {
		if pp.Name == name {
			p = pp
			break
		}
	}

	if p == nil {
		return "", nil, fmt.Errorf("unknown parameter %q", name)
	}

	var value interface{}

	switch p.Type {
	case "number":
		var i int64
		i, err := strconv.ParseInt(valueString, 10, 64)
		if err == nil {
			value = i
		} else {
			f, err := strconv.ParseFloat(valueString, 64)
			if err == nil {
				value = f
			} else {
				return "", nil,
					fmt.Errorf("invalid number value %q", valueString)
			}
		}

	case "string":
		value = valueString

	case "boolean":
		valueString = strings.ToLower(valueString)

		switch valueString {
		case "true":
			value = true
		case "false":
			value = false
		default:
			return "", nil, fmt.Errorf("invalid boolean value %q", valueString)
		}
	}

	return name, value, nil
}
