package docker

import (
	"fmt"

	cdockerhub "github.com/exograd/eventline/pkg/connectors/dockerhub"
	cgithub "github.com/exograd/eventline/pkg/connectors/github"
	"github.com/exograd/eventline/pkg/eventline"
)

func IdentityAuthenticationKey(identity *eventline.Identity) (key string, err error) {
	// Note that GitHub OAuth2 identities cannot be used for the ghcr.io
	// container registry. If this changes one day, feel free to contact me.

	switch i := identity.Data.(type) {
	case *cdockerhub.PasswordIdentity:
		key = i.Username + ":" + i.Password

	case *cdockerhub.TokenIdentity:
		key = i.Username + ":" + i.Token

	case *cgithub.TokenIdentity:
		key = i.Username + ":" + i.Token

	default:
		err = fmt.Errorf("identity %q cannot be used for docker registry "+
			"authentication", identity.Name)
	}

	return
}
