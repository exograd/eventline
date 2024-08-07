package eventline

import (
	"bytes"
	"encoding/json"

	"github.com/exograd/eventline/pkg/utils"
	"go.n16f.net/ejson"
)

type SubscriptionParameters interface {
	ejson.Validatable
}

func SubscriptionParametersEqual(sp1, sp2 SubscriptionParameters) bool {
	// It is quite a hack, but it works

	sp1Data, err := json.Marshal(sp1)
	if err != nil {
		utils.Panicf("cannot encode subscription parameters %#v: %v", sp1, err)
	}

	sp2Data, err := json.Marshal(sp2)
	if err != nil {
		utils.Panicf("cannot encode subscription parameters %#v: %v", sp2, err)
	}

	return bytes.Equal(sp1Data, sp2Data)
}
