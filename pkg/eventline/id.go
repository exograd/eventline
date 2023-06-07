package eventline

import (
	"github.com/exograd/eventline/pkg/ksuid"
)

type Id = ksuid.KSUID

type Ids = ksuid.KSUIDs

var ZeroId = ksuid.Zero

func GenerateId() Id {
	return ksuid.Generate()
}
