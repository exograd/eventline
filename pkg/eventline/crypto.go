package eventline

import (
	"github.com/exograd/eventline/pkg/cryptoutils"
)

var GlobalEncryptionKey cryptoutils.AES256Key

func EncryptAES256(data []byte) ([]byte, error) {
	return cryptoutils.EncryptAES256(data, GlobalEncryptionKey)
}

func DecryptAES256(data []byte) ([]byte, error) {
	return cryptoutils.DecryptAES256(data, GlobalEncryptionKey)
}
