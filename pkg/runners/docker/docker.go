package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	dockertypes "github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	dockerjsonmessage "github.com/docker/docker/pkg/jsonmessage"
	"github.com/exograd/eventline/pkg/utils"
)

func newClient() (*dockerclient.Client, error) {
	opts := []dockerclient.Opt{
		dockerclient.WithAPIVersionNegotiation(),
	}

	return dockerclient.NewClientWithOpts(opts...)
}

func (r *Runner) pullImage() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-ctx.Done():

		case <-r.runner.StopChan:
			cancel()
		}
	}()

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

func (r *Runner) createContainer() error {
	ctx := context.Background()

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

	vcpu := 1      // TODO
	memory := 1000 // TODO
	maxNbPids := int64(4096)

	hostCfg := dockercontainer.HostConfig{
		Resources: dockercontainer.Resources{
			NanoCPUs:  int64(vcpu) * 1_000_000_000,
			Memory:    int64(memory) * 1_000_000,
			PidsLimit: utils.Ref(maxNbPids),
		},
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

	case <-r.runner.StopChan:
		return fmt.Errorf("container creation interrupted")
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
