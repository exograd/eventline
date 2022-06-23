package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/exograd/evgo/pkg/utils"
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

	c.AddArgument("name", "the name of the job")

	c.AddFlag("", "json",
		"encode the specification using json instead of yaml")

	// deploy-job
	c = p.AddCommand("deploy-job",
		"create or update a job from a specification file",
		cmdDeployJob)

	c.AddFlag("n", "dry-run", "validate the job but do not deploy it")

	c.AddArgument("path", "the path of the job specification file")

	// delete-job
	c = p.AddCommand("delete-job", "delete a job",
		cmdDeleteJob)

	c.AddArgument("name", "the name of the job")
}

func cmdListJobs(p *program.Program) {
	app.IdentifyCurrentProject()

	jobs, err := app.Client.FetchJobs()
	if err != nil {
		p.Fatal("cannot fetch jobs: %v", err)
	}

	header := []string{"id", "name", "event", "runtime"}
	table := NewTable(header)

	for _, j := range jobs {
		var triggerName string
		if t := j.Spec.Trigger; t != nil {
			triggerName = fmt.Sprintf("%s/%s", t.Connector, t.Event)
		}

		row := []interface{}{
			j.Id,
			j.Spec.Name,
			triggerName,
			j.Spec.Runtime.Name,
		}

		table.AddRow(row)
	}

	table.Write()
}

func cmdExportJob(p *program.Program) {
	app.IdentifyCurrentProject()

	name := p.ArgumentValue("name")

	useJSON := p.IsOptionSet("json")

	job, err := app.Client.FetchJobByName(name)
	if err != nil {
		p.Fatal("cannot fetch job: %v", err)
	}

	var data []byte

	if useJSON {
		var buf bytes.Buffer

		encoder := json.NewEncoder(&buf)
		encoder.SetIndent("", "  ")

		err = encoder.Encode(job.Spec)
		data = buf.Bytes()
	} else {
		data, err = utils.YAMLEncode(job.Spec)
	}

	if err != nil {
		p.Fatal("cannot encode job specification: %w", err)
	}

	io.Copy(os.Stdout, bytes.NewReader(data))
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
		if dryRun {
			p.Fatal("invalid job: %v", err)
		} else {
			p.Fatal("cannot deploy job: %v", err)
		}
	}

	if dryRun {
		p.Info("job validated successfully")
	} else {
		p.Info("job %q deployed", job.Id)
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
