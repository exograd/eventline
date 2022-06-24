package service

import (
	"errors"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/exograd/eventline/pkg/utils"
	"github.com/exograd/go-daemon/daemon"
	"github.com/exograd/go-daemon/dhttp"
	"github.com/exograd/go-daemon/pg"
	"github.com/exograd/go-log"
)

var (
	testService   *Service
	testAPIClient *dhttp.APIClient
)

func TestMain(m *testing.M) {
	setTestDirectory()

	resetTestDatabase()

	initTestService()
	initTestHTTPClient()

	os.Exit(m.Run())
}

func setTestDirectory() {
	// Go runs the tests of a package in the directory of this package. We
	// want tests to run at the root of the project.
	//
	// We obtain the filename of the caller which will be the path of the
	// current file. We can then get the path of the root directory of the
	// project by looking for the configuration file, and change the current
	// directory.
	_, filePath, _, _ := runtime.Caller(0)

	dirPath := path.Dir(filePath)
	for dirPath != "" {
		dirPath = path.Join(dirPath, "..")

		filePath := path.Join(dirPath, "eventline.yaml")

		_, err := os.Stat(filePath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			utils.Abort("cannot stat %q: %w", filePath, err)
		}

		break
	}

	if err := os.Chdir(dirPath); err != nil {
		utils.Abort("cannot change directory to %s: %v", dirPath, err)
	}
}

func resetTestDatabase() {
	clientCfg := pg.ClientCfg{
		URI: "postgres://eventline:eventline@localhost:5432/eventline_test",
	}

	client, err := pg.NewClient(clientCfg)
	if err != nil {
		utils.Abort("cannot connect to the test database: %v", err)
	}
	defer client.Close()

	err = client.WithConn(func(conn pg.Conn) (err error) {
		query := `
DROP SCHEMA public CASCADE;
CREATE SCHEMA public AUTHORIZATION eventline;
GRANT ALL ON SCHEMA public TO eventline;
`
		return pg.Exec(conn, query)
	})
	if err != nil {
		utils.Abort("cannot reset test database: %v", err)
	}
}

func initTestService() {
	testService = NewService()

	readyChan := make(chan struct{})

	go func() {
		daemon.RunTest("eventline", testService, "eventline-test.yaml",
			readyChan)
	}()

	select {
	case <-readyChan:
	}
}

func TestUnknownRoute(t *testing.T) {
	var req *TestRequest
	var err error

	req = NewTestWebRequest(t, "GET", "/foobar")
	_, err = req.Send()
	assertRequestError(t, err, 404, "route_not_found")
}

func initTestHTTPClient() {
	logger := log.DefaultLogger("http-client")
	logger.Data["client"] = "test"

	cfg := dhttp.ClientCfg{
		Log:         logger,
		LogRequests: true,
	}

	c, err := dhttp.NewClient(cfg)
	if err != nil {
		utils.Abort("cannot create http client: %v", err)
	}

	testAPIClient = dhttp.NewAPIClient(c)
}
