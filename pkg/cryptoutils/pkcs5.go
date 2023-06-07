package cryptoutils

import (
	"bytes"
	"fmt"
)

func PadPKCS5(data []byte, blockSize int) []byte {
	paddingSize := blockSize - len(data)%blockSize
	padding := bytes.Repeat([]byte{byte(paddingSize)}, paddingSize)

	return append(data, padding...)
}

func UnpadPKCS5(data []byte, blockSize int) ([]byte, error) {
	dataSize := len(data)

	if dataSize%blockSize != 0 {
		return nil, fmt.Errorf("truncated data")
	}

	if len(data) == 0 {
		return data, nil
	} else {
		paddingSize := int(data[dataSize-1])
		if paddingSize > dataSize || paddingSize > blockSize {
			return nil, fmt.Errorf("invalid padding size %d", paddingSize)
		}

		return data[:dataSize-paddingSize], nil
	}
}
