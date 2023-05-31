package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/galdor/go-program"
	"github.com/google/go-github/v40/github"
)

type App struct {
	Config *Config
	Client *Client

	HTTPClient *http.Client
	UserAgent  string

	HomePath string

	projectIdOption   *eventline.Id
	projectNameOption *string
}

func NewApp(config *Config, client *Client) (*App, error) {
	a := &App{
		Config: config,
		Client: client,

		HTTPClient: NewHTTPClient(),
	}

	a.UserAgent = fmt.Sprintf("evcli/%s (%s; %s)",
		buildId, runtime.GOOS, runtime.GOARCH)

	homePath, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot locate user home directory: %w", err)
	}
	a.HomePath = homePath

	client.httpClient = a.HTTPClient

	return a, nil
}

func (a *App) LoadAPIKey() {
	if key := os.Getenv("EVENTLINE_API_KEY"); key != "" {
		p.Debug(1, "using api key from EVENTLINE_API_KEY environment variable")
		a.Client.APIKey = key
		return
	}

	if key := a.Config.API.Key; key != "" {
		p.Debug(1, "using api key from configuration")
		a.Client.APIKey = key
		return
	}

	p.Error("missing or empty API key")
	p.Info("\nYou need to provide an API key to interact with Eventline. " +
		"You can either edit the evcli configuration file or use the " +
		"following command:")
	p.Info("\n\tevcli set-config api.key <key>")
	p.Info("\nAlternatively, you can set the EVENTLINE_API_KEY environment " +
		"variable.")
	os.Exit(1)
}

func (a *App) IdentifyCurrentProject() {
	id, err := a.identifyCurrentProject()
	if err != nil {
		p.Fatal("%v", err)
	}

	p.Debug(1, "using project %s as current project", id)

	a.Client.ProjectId = &id
}

func (a *App) identifyCurrentProject() (eventline.Id, error) {
	if a.projectIdOption != nil {
		return *a.projectIdOption, nil
	}

	if a.projectNameOption != nil {
		return a.identifyCurrentProjectByName(*a.projectNameOption)
	}

	if idString := os.Getenv("EVENTLINE_PROJECT_ID"); idString != "" {
		var id eventline.Id
		if err := id.Parse(idString); err != nil {
			return eventline.ZeroId,
				fmt.Errorf("EVENTLINE_PROJECT_ID: invalid project id %q: %w",
					idString, err)
		}

		return id, nil
	}

	if name := os.Getenv("EVENTLINE_PROJECT_NAME"); name != "" {
		return a.identifyCurrentProjectByName(name)
	}

	return a.identifyCurrentProjectByName("main")
}

func (a *App) identifyCurrentProjectByName(name string) (eventline.Id, error) {
	project, err := a.Client.FetchProjectByName(name)
	if err != nil {
		return eventline.ZeroId,
			fmt.Errorf("cannot fetch project %q: %w", name, err)
	}

	return project.Id, nil
}

func (a *App) LookForLastBuild() {
	lastCheck, err := a.loadLastBuildIdCheckDate()
	if err != nil {
		p.Error("cannot look for last build: %v", err)
	}

	now := time.Now()

	if lastCheck != nil && now.Sub(*lastCheck) < 24*time.Hour {
		return
	}

	lastBuildId, err := a.lookForLastBuild()
	if err != nil {
		p.Error("cannot find last build: %v", err)
		return
	}

	if err := a.updateLastBuildIdCheckDate(now); err != nil {
		p.Error("cannot update last build check date: %v", err)
	}

	if lastBuildId == nil {
		p.Debug(1, "evcli is up-to-date")
		return
	}

	p.Info("evcli %v is now available: run \"evcli update\" to install it", lastBuildId)
}

func (a *App) lookForLastBuild() (*program.BuildId, error) {
	p.Debug(1, "looking for the last build")

	currentBuildId := a.currentBuildId()

	p.Debug(1, "current build: %v", currentBuildId)

	lastBuildId, err := a.lastBuildId()
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve last build id: %w", err)
	} else if lastBuildId == nil {
		return nil, nil
	}

	p.Debug(1, "last build: %v", lastBuildId)

	if lastBuildId.LowerThanOrEqualTo(currentBuildId) {
		return nil, nil
	}

	return lastBuildId, nil
}

func (a *App) currentBuildId() program.BuildId {
	var id program.BuildId
	id.Parse(buildId)
	return id
}

func (a *App) lastBuildId() (*program.BuildId, error) {
	httpClient := a.HTTPClient
	client := github.NewClient(httpClient)

	ctx := context.Background()

	release, _, err := client.Repositories.GetLatestRelease(ctx,
		"exograd", "eventline")
	if err != nil {
		return nil, fmt.Errorf("cannot fetch latest release: %w", err)
	}

	tagName := release.GetTagName()

	var buildId program.BuildId
	if err := buildId.Parse(tagName); err != nil {
		return nil, fmt.Errorf("invalid build id %q: %w", tagName, err)
	}

	return &buildId, nil
}

func (a *App) loadLastBuildIdCheckDate() (*time.Time, error) {
	filePath := a.lastBuildIdCheckPath()

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("cannot read %s: %w", filePath, err)
	}

	i, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid data", filePath)
	}

	t := time.Unix(i, 0)

	return &t, nil
}

func (a *App) updateLastBuildIdCheckDate(t time.Time) error {
	filePath := a.lastBuildIdCheckPath()

	data := []byte(strconv.FormatInt(t.Unix(), 10))

	dirPath := filepath.Dir(filePath)
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return fmt.Errorf("cannot create directory %s: %w", dirPath, err)
	}

	if err := ioutil.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("cannot write %s: %w", filePath, err)
	}

	return nil
}

func (a *App) lastBuildIdCheckPath() string {
	return path.Join(a.HomePath, ".evcli", "last-build-id-check")
}
