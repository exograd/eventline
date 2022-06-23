package eventline

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccountCheckPassword(t *testing.T) {
	assert := assert.New(t)

	salt := GenerateSalt()

	account := Account{
		Salt:         salt,
		PasswordHash: HashPassword("foo", salt),
	}

	assert.False(account.CheckPassword(""))
	assert.False(account.CheckPassword("foobar"))
	assert.True(account.CheckPassword("foo"))
}

func TestHashPassword(t *testing.T) {
	assert := assert.New(t)

	salt := GenerateSalt()
	assert.Equal(SaltSize, len(salt))

	password := "foo"
	hash := HashPassword(password, salt)
	assert.Equal(32, len(hash))
}
