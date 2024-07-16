package main

import (
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/program"
)

func addProjectCommands() {
	var c *program.Command

	// list-projects
	c = p.AddCommand("list-projects", "list all projects",
		cmdListProjects)

	// create-project
	c = p.AddCommand("create-project", "create a new project",
		cmdCreateProject)

	c.AddArgument("name", "the name of the project")

	// delete-project
	c = p.AddCommand("delete-project", "delete a project",
		cmdDeleteProject)

	c.AddArgument("name", "the name of the project")
}

func cmdListProjects(p *program.Program) {
	projects, err := app.Client.FetchProjects()
	if err != nil {
		p.Fatal("cannot fetch projects: %v", err)
	}

	header := []string{"id", "name"}
	table := NewTable(header)
	for _, p := range projects {
		row := []interface{}{p.Id, p.Name}
		table.AddRow(row)
	}

	table.Write()
}

func cmdCreateProject(p *program.Program) {
	name := p.ArgumentValue("name")

	newProject := eventline.NewProject{
		Name: name,
	}

	project, err := app.Client.CreateProject(&newProject)
	if err != nil {
		p.Fatal("cannot create project: %v", err)
	}

	p.Info("project %q created", project.Name)
}

func cmdDeleteProject(p *program.Program) {
	name := p.ArgumentValue("name")

	prompt := fmt.Sprintf("Do you want to delete project %q ? All resources "+
		"associated with it will be deleted as well.", name)
	if Confirm(prompt) == false {
		p.Info("deletion aborted")
		return
	}

	project, err := app.Client.FetchProjectByName(name)
	if err != nil {
		p.Fatal("cannot fetch project: %v", err)
	}

	if err := app.Client.DeleteProject(project.Id); err != nil {
		p.Fatal("cannot delete project: %v", err)
	}
}
