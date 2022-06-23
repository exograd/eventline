package github

import (
	"errors"

	"github.com/google/go-github/v45/github"
)

type HookId = int64

func IsNotFoundAPIError(err error) bool {
	var errRes *github.ErrorResponse

	if errors.As(err, &errRes) {
		return errRes.Response.StatusCode == 404
	}

	return false
}
