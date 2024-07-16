package generic

import (
	"strings"

	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/ejson"
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

func (i *SSHKeyIdentity) ValidateJSON(v *ejson.Validator) {
	v.CheckStringNotEmpty("private_key", i.PrivateKey)

	// OpenSSH will fail with strange errors if key files do not end with a
	// newline character; we add it ourselves if it is not here.

	if !strings.HasSuffix(i.PrivateKey, "\n") {
		i.PrivateKey += "\n"
	}

	if i.PublicKey != "" && !strings.HasSuffix(i.PublicKey, "\n") {
		i.PublicKey += "\n"
	}

	if i.Certificate != "" && !strings.HasSuffix(i.Certificate, "\n") {
		i.Certificate += "\n"
	}
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
