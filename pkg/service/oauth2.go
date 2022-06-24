package service

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/exograd/eventline/pkg/eventline"
)

func EncodeOAuth2State(identityId eventline.Id, sessionId eventline.Id) (string, error) {
	decryptedData := []byte(identityId.String() + ":" + sessionId.String())

	encryptedData, err := eventline.EncryptAES256(decryptedData)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(encryptedData), nil
}

func DecodeOAuth2State(state string) (eventline.Id, eventline.Id, error) {
	zeroId := eventline.ZeroId

	encryptedData, err := base64.StdEncoding.DecodeString(state)
	if err != nil {
		return zeroId, zeroId, fmt.Errorf("invalid base64 data: %w", err)
	}

	decryptedData, err := eventline.DecryptAES256(encryptedData)
	if err != nil {
		return zeroId, zeroId, err
	}

	parts := strings.SplitN(string(decryptedData), ":", 2)
	if len(parts) != 2 {
		return zeroId, zeroId, fmt.Errorf("invalid format")
	}

	var identityId eventline.Id
	if err := identityId.Parse(parts[0]); err != nil {
		return zeroId, zeroId,
			fmt.Errorf("invalid identity id %q: %w", parts[0], err)
	}

	var sessionId eventline.Id
	if err := sessionId.Parse(parts[1]); err != nil {
		return zeroId, zeroId,
			fmt.Errorf("invalid session id %q: %w", parts[1], err)
	}

	return identityId, sessionId, nil
}
