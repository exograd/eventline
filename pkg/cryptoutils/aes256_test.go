package cryptoutils

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAES256KeyHex(t *testing.T) {
	require := require.New(t)

	testKeyHex := "28278b7c0a25f01d3cab639633b9487f9ea1e9a2176dc9595a3f01323aa44284"
	testKey, _ := hex.DecodeString(testKeyHex)

	var key AES256Key
	require.NoError(key.FromHex(testKeyHex))
	require.Equal(testKey, key[:])

	require.Equal(testKeyHex, key.Hex())
}

func TestAES256(t *testing.T) {
	require := require.New(t)

	keyHex := "28278b7c0a25f01d3cab639633b9487f9ea1e9a2176dc9595a3f01323aa44284"
	var key AES256Key
	require.NoError(key.FromHex(keyHex))

	data := []byte("Hello world!")
	encryptedData, err := EncryptAES256(data, key)
	require.NoError(err)

	decryptedData, err := DecryptAES256(encryptedData, key)
	require.NoError(err)

	require.Equal(data, decryptedData)
}

func TestAES256InvalidData(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	keyHex := "28278b7c0a25f01d3cab639633b9487f9ea1e9a2176dc9595a3f01323aa44284"
	var key AES256Key
	require.NoError(key.FromHex(keyHex))

	var err error

	iv := make([]byte, AES256IVSize)

	_, err = DecryptAES256(append(iv, []byte("foo")...), key)
	assert.Error(err)
}
