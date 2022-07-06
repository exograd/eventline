package main

import (
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

	c.AddTrailingArgument("path", "the paths of job specification files")

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

func cmdDeployJobs(p *program.Program) {
	app.IdentifyCurrentProject()

	filePaths := p.TrailingArgumentValues("path")
	dryRun := p.IsOptionSet("dry-run")

	app.IdentifyCurrentProject()

	for _, filePath := range filePaths {
		spec, err := LoadJobFile(filePath)
		if err != nil {
			p.Fatal("cannot load %q: %v", filePath, err)
		}

		p.Info("deploying job %q (%q)", spec.Name, filePath)

		if _, err := app.Client.DeployJob(spec, dryRun); err != nil {
			if dryRun {
				p.Fatal("invalid job %q (%q): %v",
					spec.Name, filePath, err)
			} else {
				p.Fatal("cannot deploy job %q (%q): %v",
					spec.Name, filePath, err)
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
