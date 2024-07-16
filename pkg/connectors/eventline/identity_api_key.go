package eventline

import (
	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/ejson"
)

type APIKeyIdentity struct {
	Key string `json:"key"`
}

func APIKeyIdentityDef() *eventline.IdentityDef {
	def := eventline.NewIdentityDef("api_key", &APIKeyIdentity{})
	return def
}

func (i *APIKeyIdentity) ValidateJSON(v *ejson.Validator) {
	v.CheckStringNotEmpty("key", i.Key)
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
	return map[string]string{
		"EVENTLINE_API_KEY": i.Key,
	}
}
