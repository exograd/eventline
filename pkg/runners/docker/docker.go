package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"

	dockercontainer "github.com/docker/docker/api/types/container"
	dockerimage "github.com/docker/docker/api/types/image"
	dockermount "github.com/docker/docker/api/types/mount"
	dockerclient "github.com/docker/docker/client"
	dockerjsonmessage "github.com/docker/docker/pkg/jsonmessage"
	dockerstdcopy "github.com/docker/docker/pkg/stdcopy"
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/utils"
)

func (r Runner) newClient() (*dockerclient.Client, error) {
	opts := []dockerclient.Opt{
		dockerclient.WithAPIVersionNegotiation(),
	}

	if r.cfg.URI != "" {
		uri, err := url.Parse(r.cfg.URI)
		if err != nil {
			return nil, fmt.Errorf("cannot parse uri: %w", err)
		}

		switch uri.Scheme {
		case "unix":
			opts = append(opts, dockerclient.WithHost(r.cfg.URI))

		case "tcp":
			opts = append(opts, dockerclient.WithHost(r.cfg.URI))

			opts = append(opts,
				dockerclient.WithTLSClientConfig(r.cfg.CACertificatePath,
					r.cfg.CertificatePath, r.cfg.PrivateKeyPath))

		default:
			return nil, fmt.Errorf("unhandled uri scheme %q", uri.Scheme)
		}
	}

	return dockerclient.NewClientWithOpts(opts...)
}

func (r *Runner) pullImage(ctx context.Context) error {
	je := r.runner.JobExecution
	params := je.JobSpec.Runner.Parameters.(*RunnerParameters)

	var authKey string
	if r.runner.RunnerIdentity != nil {
		key, err := identityAuthenticationKey(r.runner.RunnerIdentity)
		if err != nil {
			return err
		}

		authKey = key
	}

	var options dockerimage.PullOptions

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

	mounts := make([]dockermount.Mount, len(r.cfg.MountPoints))
	for i, p := range r.cfg.MountPoints {
		mounts[i] = dockermount.Mount{
			Type:     dockermount.TypeBind,
			Source:   p.Source,
			Target:   p.Target,
			ReadOnly: p.ReadOnly,
		}
	}

	hostCfg := dockercontainer.HostConfig{
		Resources: resources,
		Mounts:    mounts,
	}

	res, err := r.client.ContainerCreate(ctx, &containerCfg, &hostCfg, nil,
		nil, containerName)
	if err != nil {
		return fmt.Errorf("cannot create container: %w", err)
	}

	r.containerId = res.ID

	statusChan, errChan := r.client.ContainerWait(ctx, r.containerId,
		"not-running")

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

	options := dockercontainer.RemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	return r.client.ContainerRemove(ctx, r.containerId, options)
}

func (r *Runner) copyFiles(ctx context.Context) error {
	options := dockercontainer.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	}

	r.runner.FileSet.AddPrefix(r.DirPath())

	// We do not control which user is going to execute the code (it depends
	// on the image). Therefore we have to make files readable (and
	// executable) by any user.
	for _, file := range r.runner.FileSet.Files {
		if (file.Mode & 0700) != 0 {
			file.Mode = file.Mode | 0755
		} else {
			file.Mode = file.Mode | 0644
		}
	}

	var buf bytes.Buffer
	if err := r.runner.FileSet.TarArchive(&buf); err != nil {
		return fmt.Errorf("cannot generate tar archive: %w", err)
	}

	return r.client.CopyToContainer(ctx, r.containerId, "/", &buf, options)
}

func (r *Runner) startContainer(ctx context.Context) error {
	options := dockercontainer.StartOptions{}

	return r.client.ContainerStart(ctx, r.containerId, options)
}

func (r *Runner) exec(ctx context.Context, se *eventline.StepExecution, step *eventline.Step, stdout, stderr io.WriteCloser) error {
	// Create an execution process
	cmdName, cmdArgs := r.runner.StepCommand(se, step, r.DirPath())
	cmd := append([]string{cmdName}, cmdArgs...)

	execCfg := dockercontainer.ExecOptions{
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
	execStartCheck := dockercontainer.ExecAttachOptions{}

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
