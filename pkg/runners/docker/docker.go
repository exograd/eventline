package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	dockertypes "github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
	dockerjsonmessage "github.com/docker/docker/pkg/jsonmessage"
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
	// TODO

	return nil
}

func (r *Runner) deleteContainer() error {
	// TODO

	return nil
}

func registryAuth(authKey string) (string, error) {
	// For some reason, the Docker API expects registry authentication keys to
	// be base64-encoded JSON objects, even though all other systems seem to
	// be using "username:password" strings.

	// This function supports empty authentication keys; this makes it easier
	// to use without having to wonder if a key was set or not.
	if authKey == "" {
		return "", nil
	}

	parts := strings.SplitN(authKey, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid format")
	}

	username := parts[0]
	password := parts[1]

	auth := dockertypes.AuthConfig{
		Username: username,
		Password: password,
	}

	authData, err := json.Marshal(auth)
	if err != nil {
		return "", fmt.Errorf("cannot encode json value: %w", err)
	}

	return base64.StdEncoding.EncodeToString(authData), nil
}
