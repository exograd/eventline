package generic

import (
	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/ejson"
)

type GPGKeyIdentity struct {
	PrivateKey string `json:"private_key,omitempty"`
	PublicKey  string `json:"public_key,omitempty"`
	Password   string `json:"password,omitempty"`
}

func GPGKeyIdentityDef() *eventline.IdentityDef {
	def := eventline.NewIdentityDef("gpg_key", &GPGKeyIdentity{})
	return def
}

func (i *GPGKeyIdentity) ValidateJSON(v *ejson.Validator) {
	if i.PrivateKey == "" && i.PublicKey == "" {
		msg := "gpg key must contain either a private key or a public key"
		v.AddError("private_key", "missing_value", msg)
		v.AddError("public_key", "missing_value", msg)
	}
}

func (i *GPGKeyIdentity) Def() *eventline.IdentityDataDef {
	view := eventline.NewIdentityDataDef()

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:      "private_key",
		Label:    "Private key",
		Value:    i.PrivateKey,
		Type:     eventline.IdentityDataTypeTextBlock,
		Optional: true,
		Secret:   true,
		Verbatim: true,
	})

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:      "public_key",
		Label:    "Public key",
		Value:    i.PublicKey,
		Type:     eventline.IdentityDataTypeTextBlock,
		Optional: true,
		Verbatim: true,
	})

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:      "password",
		Label:    "Password",
		Value:    i.Password,
		Type:     eventline.IdentityDataTypeString,
		Optional: true,
		Secret:   true,
	})

	return view
}

func (i *GPGKeyIdentity) Environment() map[string]string {
	return map[string]string{}
}
