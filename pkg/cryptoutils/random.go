package cryptoutils

import (
	"crypto/rand"
	"fmt"
)

func RandomBytes(n int) []byte {
	data := make([]byte, n)

	if _, err := rand.Read(data); err != nil {
		panic(fmt.Errorf("cannot generate random data: %w", err))
	}

	return data
}
