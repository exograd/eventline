package generic

import (
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/check"
)

type GPGKeyIdentity struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key,omitempty"`
	Password   string `json:"password,omitempty"`
}

func GPGKeyIdentityDef() *eventline.IdentityDef {
	def := eventline.NewIdentityDef("gpg_key", &GPGKeyIdentity{})
	return def
}

func (i *GPGKeyIdentity) Check(c *check.Checker) {
	c.CheckStringNotEmpty("private_key", i.PrivateKey)
}

func (i *GPGKeyIdentity) Def() *eventline.IdentityDataDef {
	view := eventline.NewIdentityDataDef()

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:      "private_key",
		Label:    "Private key",
		Value:    i.PrivateKey,
		Type:     eventline.IdentityDataTypeTextBlock,
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
