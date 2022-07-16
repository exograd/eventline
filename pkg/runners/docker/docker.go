package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	dockertypes "github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	dockerjsonmessage "github.com/docker/docker/pkg/jsonmessage"
	dockerstdcopy "github.com/docker/docker/pkg/stdcopy"
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/utils"
)

func newLocalClient() (*dockerclient.Client, error) {
	opts := []dockerclient.Opt{
		dockerclient.WithAPIVersionNegotiation(),
	}

	return dockerclient.NewClientWithOpts(opts...)
}

func newTCPClient(uri, caCertPath, certPath, keyPath string) (*dockerclient.Client, error) {
	opts := []dockerclient.Opt{
		dockerclient.WithAPIVersionNegotiation(),
		dockerclient.WithHost(uri),
		dockerclient.WithTLSClientConfig(caCertPath, certPath, keyPath),
	}

	return dockerclient.NewClientWithOpts(opts...)
}

func (r *Runner) pullImage(ctx context.Context) error {
	je := r.runner.JobExecution
	params := je.JobSpec.Runner.Parameters.(*RunnerParameters)

	var authKey string
	if r.runner.RunnerIdentity != nil {
		key, err := IdentityAuthenticationKey(r.runner.RunnerIdentity)
		if err != nil {
			return err
		}

		authKey = key
	}

	var options dockertypes.ImagePullOptions

	if authKey != "" {
		registryAuth, err := registryAuth(authKey)
		if err != nil {
			return fmt.Errorf("cannot build registry authentication key: %w",
				err)
		}

		options.RegistryAuth = registryAuth
	}

	r.log.Info("pulling image %q", params.Image)

	statusReader, err := r.client.ImagePull(ctx, params.Image, options)
	if err != nil {
		return err
	}
	defer statusReader.Close()

	decodeChan := make(chan error)

	go func() {
		defer close(decodeChan)

		decoder := json.NewDecoder(statusReader)
		for {
			var msg dockerjsonmessage.JSONMessage

			err := decoder.Decode(&msg)
			if errors.Is(err, io.EOF) {
				break
			} else if errors.Is(err, context.Canceled) {
				decodeChan <- fmt.Errorf("execution interrupted")
			} else if err != nil {
				decodeChan <- fmt.Errorf("cannot decode message: %w", err)
				return
			}

			r.log.Debug(2, "image pull status: %s", msg.Status)
		}
	}()

	err = <-decodeChan

	return err
}

func (r *Runner) createContainer(ctx context.Context) error {
	je := r.runner.JobExecution
	params := je.JobSpec.Runner.Parameters.(*RunnerParameters)

	env := make([]string, 0, len(r.runner.Environment))
	for k, v := range r.runner.Environment {
		env = append(env, k+"="+v)
	}

	labels := map[string]string{
		"net.eventline.project-id":       r.runner.Project.Id.String(),
		"net.eventline.job-name":         je.JobSpec.Name,
		"net.eventline.job-execution-id": je.Id.String(),
	}

	containerName := "eventline-job-" + je.Id.String()

	containerCfg := dockercontainer.Config{
		Hostname: containerName,
		Image:    params.Image,
		Env:      env,
		Cmd:      []string{"sleep", "86400"},

		Labels:      labels,
		StopTimeout: utils.Ref(1), // seconds
	}

	var resources dockercontainer.Resources

	if limit := params.CPULimit; limit != 0 {
		resources.NanoCPUs = int64(limit * 1e9)
	}

	if limit := params.MemoryLimit; limit != 0 {
		resources.Memory = int64(limit * 1_000_000)
	}

	hostCfg := dockercontainer.HostConfig{
		Resources: resources,
	}

	res, err := r.client.ContainerCreate(ctx, &containerCfg, &hostCfg, nil,
		nil, containerName)
	if err != nil {
		return fmt.Errorf("cannot create container: %w", err)
	}

	r.containerId = res.ID

	statusChan, errChan := r.client.ContainerWait(ctx, r.containerId,
		"created")

	select {
	case <-statusChan:

	case err := <-errChan:
		return fmt.Errorf("container error: %w", err)

	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

func (r *Runner) deleteContainer() error {
	ctx := context.Background()

	options := dockertypes.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	return r.client.ContainerRemove(ctx, r.containerId, options)
}

func (r *Runner) copyFiles(ctx context.Context) error {
	options := dockertypes.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	}

	r.runner.FileSet.AddPrefix("/eventline")

	var buf bytes.Buffer
	if err := r.runner.FileSet.TarArchive(&buf); err != nil {
		return fmt.Errorf("cannot generate tar archive: %w", err)
	}

	return r.client.CopyToContainer(ctx, r.containerId, "/", &buf, options)
}

func (r *Runner) startContainer(ctx context.Context) error {
	options := dockertypes.ContainerStartOptions{}

	return r.client.ContainerStart(ctx, r.containerId, options)
}

func (r *Runner) exec(ctx context.Context, se *eventline.StepExecution, step *eventline.Step, stdout, stderr io.WriteCloser) error {
	// Create an execution process
	cmdName, cmdArgs := r.runner.StepCommand(se, step, "/eventline")
	cmd := append([]string{cmdName}, cmdArgs...)

	execCfg := dockertypes.ExecConfig{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	createRes, err := r.client.ContainerExecCreate(ctx, r.containerId, execCfg)
	if err != nil {
		return fmt.Errorf("cannot create execution process: %w", err)
	}

	execId := createRes.ID

	// Start the execution process
	execStartCheck := dockertypes.ExecStartCheck{}

	startRes, err := r.client.ContainerExecAttach(ctx, execId, execStartCheck)
	if err != nil {
		return fmt.Errorf("cannot start execution process: %w", err)
	}
	defer startRes.Close()

	// Read the output until the process terminates
	copyChan := make(chan error)

	go func() {
		defer close(copyChan)

		_, err := dockerstdcopy.StdCopy(stdout, stderr, startRes.Reader)
		if err != nil {
			copyChan <- fmt.Errorf("cannot read process output: %w", err)
			return
		}
	}()

	select {
	case err := <-copyChan:
		if err != nil {
			return err
		}

	case <-ctx.Done():
		return ctx.Err()
	}

	// Check execution status
	inspectRes, err := r.client.ContainerExecInspect(ctx, execId)
	if err != nil {
		return fmt.Errorf("cannot inspect execution process: %w", err)
	}

	if code := inspectRes.ExitCode; code != 0 {
		var err error

		if code < 128 {
			err = fmt.Errorf("program exited with status %d", code)
		} else {
			err = fmt.Errorf("program killed by signal %d", code-128)
		}

		return eventline.NewStepFailureError(err)
	}

	return nil
}
