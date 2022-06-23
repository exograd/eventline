package generic

import (
	"github.com/exograd/evgo/pkg/eventline"
	"github.com/exograd/go-daemon/check"
)

type SSHKeyIdentity struct {
	PrivateKey  string `json:"private_key"`
	PublicKey   string `json:"public_key,omitempty"`
	Certificate string `json:"certificate,omitempty"`
}

func SSHKeyIdentityDef() *eventline.IdentityDef {
	def := eventline.NewIdentityDef("ssh_key", &SSHKeyIdentity{})
	return def
}

func (i *SSHKeyIdentity) Check(c *check.Checker) {
	c.CheckStringNotEmpty("private_key", i.PrivateKey)
}

func (i *SSHKeyIdentity) Def() *eventline.IdentityDataDef {
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
		Key:      "certificate",
		Label:    "Certificate",
		Value:    i.Certificate,
		Type:     eventline.IdentityDataTypeTextBlock,
		Optional: true,
		Verbatim: true,
	})

	return view
}

func (i *SSHKeyIdentity) Environment() map[string]string {
	return map[string]string{}
}
