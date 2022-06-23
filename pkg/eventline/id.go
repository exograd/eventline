package eventline

import (
	"github.com/exograd/go-daemon/ksuid"
)

type Id = ksuid.KSUID

type Ids = ksuid.KSUIDs

var ZeroId = ksuid.Zero

func GenerateId() Id {
	return ksuid.Generate()
}
