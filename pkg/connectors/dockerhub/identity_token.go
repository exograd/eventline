package dockerhub

import (
	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/ejson"
)

type TokenIdentity struct {
	Username string `json:"username"`
	Token    string `json:"token"`
}

func TokenIdentityDef() *eventline.IdentityDef {
	def := eventline.NewIdentityDef("token", &TokenIdentity{})
	return def
}

func (i *TokenIdentity) ValidateJSON(v *ejson.Validator) {
	v.CheckStringNotEmpty("username", i.Username)
	v.CheckStringNotEmpty("token", i.Token)
}

func (i *TokenIdentity) Def() *eventline.IdentityDataDef {
	view := eventline.NewIdentityDataDef()

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:   "username",
		Label: "Username",
		Value: i.Username,
		Type:  eventline.IdentityDataTypeString,
	})

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:    "token",
		Label:  "Token",
		Value:  i.Token,
		Type:   eventline.IdentityDataTypeString,
		Secret: true,
	})

	return view
}

func (i *TokenIdentity) Environment() map[string]string {
	return map[string]string{}
}
