package github

import (
	"regexp"

	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/ejson"
)

var (
	tokenRE = regexp.MustCompile(`^ghp_.{36}$`)
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

	if v.CheckStringNotEmpty("token", i.Token) {
		v.CheckStringMatch("token", i.Token, tokenRE)
	}
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
		Key:      "token",
		Label:    "Token",
		Value:    i.Token,
		Type:     eventline.IdentityDataTypeString,
		Verbatim: true,
		Secret:   true,
	})

	return view
}

func (i *TokenIdentity) Environment() map[string]string {
	return map[string]string{
		"GITHUB_USER":  i.Username,
		"GITHUB_TOKEN": i.Token,
	}
}
