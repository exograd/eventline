package cryptoutils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type AES256Key [32]byte

const (
	AES256IVSize int = aes.BlockSize
)

var Zero = AES256Key{}

func (key AES256Key) Bytes() []byte {
	return key[:]
}

func (key AES256Key) IsZero() bool {
	return bytes.Equal(key.Bytes(), Zero.Bytes())
}

func (key AES256Key) Hex() string {
	return hex.EncodeToString(key[:])
}

func (key *AES256Key) FromHex(s string) error {
	data, err := hex.DecodeString(s)
	if err != nil {
		return err
	}

	if len(data) != 32 {
		return fmt.Errorf("invalid key size")
	}

	copy((*key)[:], data[:32])

	return nil
}

func (key *AES256Key) FromBase64(s string) error {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return err
	}

	if len(data) != 32 {
		return fmt.Errorf("invalid key size")
	}

	copy((*key)[:], data[:32])

	return nil
}

func (key AES256Key) MarshalJSON() ([]byte, error) {
	s := base64.StdEncoding.EncodeToString(key[:])
	return json.Marshal(s)
}

func (pkey *AES256Key) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	key, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return fmt.Errorf("cannot decode base64 value: %w", err)
	}

	if len(key) != 32 {
		return fmt.Errorf("invalid key size (must be 32 bytes long)")
	}

	copy(pkey[:], key)

	return nil
}

func EncryptAES256(inputData []byte, key AES256Key) ([]byte, error) {
	blockCipher, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("cannot create cipher: %w", err)
	}

	paddedData := PadPKCS5(inputData, aes.BlockSize)

	outputData := make([]byte, AES256IVSize+len(paddedData))

	iv := outputData[:AES256IVSize]
	encryptedData := outputData[AES256IVSize:]

	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("cannot generate iv: %w", err)
	}

	encrypter := cipher.NewCBCEncrypter(blockCipher, iv)
	encrypter.CryptBlocks(encryptedData, paddedData)

	return outputData, nil
}

func DecryptAES256(inputData []byte, key AES256Key) ([]byte, error) {
	blockCipher, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("cannot create cipher: %w", err)
	}

	if len(inputData) < AES256IVSize {
		return nil, fmt.Errorf("truncated data")
	}

	iv := inputData[:AES256IVSize]
	paddedData := inputData[AES256IVSize:]

	if len(paddedData)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("invalid padded data length")
	}

	decrypter := cipher.NewCBCDecrypter(blockCipher, iv)
	decrypter.CryptBlocks(paddedData, paddedData)

	outputData, err := UnpadPKCS5(paddedData, aes.BlockSize)
	if err != nil {
		return nil, fmt.Errorf("invalid padded data: %w", err)
	}

	return outputData, nil
}
