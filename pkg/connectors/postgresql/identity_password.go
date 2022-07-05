package postgresql

import (
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/check"
)

type PasswordIdentity struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

func PasswordIdentityDef() *eventline.IdentityDef {
	def := eventline.NewIdentityDef("password", &PasswordIdentity{})
	return def
}

func (i *PasswordIdentity) Check(c *check.Checker) {
	c.CheckStringNotEmpty("user", i.User)
	c.CheckStringNotEmpty("password", i.Password)
}

func (i *PasswordIdentity) Def() *eventline.IdentityDataDef {
	view := eventline.NewIdentityDataDef()

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:   "user",
		Label: "User",
		Value: i.User,
		Type:  eventline.IdentityDataTypeString,
	})

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:    "password",
		Label:  "Password",
		Value:  i.Password,
		Type:   eventline.IdentityDataTypeString,
		Secret: true,
	})

	return view
}

func (i *PasswordIdentity) Environment() map[string]string {
	return map[string]string{
		"PGUSER":     i.User,
		"PGPASSWORD": i.Password,
	}
}
