package docker

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	dockertypes "github.com/docker/docker/api/types"
	cdockerhub "github.com/exograd/eventline/pkg/connectors/dockerhub"
	cgithub "github.com/exograd/eventline/pkg/connectors/github"
	"github.com/exograd/eventline/pkg/eventline"
)

func identityAuthenticationKey(identity *eventline.Identity) (key string, err error) {
	switch i := identity.Data.(type) {
	case *cdockerhub.PasswordIdentity:
		key = i.Username + ":" + i.Password

	case *cdockerhub.TokenIdentity:
		key = i.Username + ":" + i.Token

	case *cgithub.OAuth2Identity:
		key = i.Username + ":" + i.AccessToken

	case *cgithub.TokenIdentity:
		key = i.Username + ":" + i.Token

	default:
		err = fmt.Errorf("identity %q cannot be used for docker registry "+
			"authentication", identity.Name)
	}

	return
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
