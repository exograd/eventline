package eventline

import "github.com/exograd/go-daemon/dcrypto"

var GlobalEncryptionKey dcrypto.AES256Key

func EncryptAES256(data []byte) ([]byte, error) {
	return dcrypto.EncryptAES256(data, GlobalEncryptionKey)
}

func DecryptAES256(data []byte) ([]byte, error) {
	return dcrypto.DecryptAES256(data, GlobalEncryptionKey)
}
