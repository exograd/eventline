package generic

import (
	"github.com/exograd/evgo/pkg/eventline"
	"github.com/exograd/go-daemon/check"
)

type APIKeyIdentity struct {
	Key string `json:"key"`
}

func APIKeyIdentityDef() *eventline.IdentityDef {
	def := eventline.NewIdentityDef("api_key", &APIKeyIdentity{})
	return def
}

func (i *APIKeyIdentity) Check(c *check.Checker) {
	c.CheckStringNotEmpty("key", i.Key)
}

func (i *APIKeyIdentity) Def() *eventline.IdentityDataDef {
	view := eventline.NewIdentityDataDef()

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:      "key",
		Label:    "API key",
		Value:    i.Key,
		Type:     eventline.IdentityDataTypeString,
		Verbatim: true,
		Secret:   true,
	})

	return view
}

func (i *APIKeyIdentity) Environment() map[string]string {
	return map[string]string{}
}
