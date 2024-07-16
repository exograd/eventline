package eventline

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"go.n16f.net/ejson"
)

type UnknownIdentityDefError struct {
	Connector string
	Type      string
}

func (err UnknownIdentityDefError) Error() string {
	return fmt.Sprintf("unknown identity %q in connector %q",
		err.Type, err.Connector)
}

type IdentityDef struct {
	Type string

	DeferredReadiness bool
	Refreshable       bool

	Data    IdentityData
	DataDef *IdentityDataDef // used when there is no actual identity
}

type IdentityData interface {
	ejson.Validatable

	Def() *IdentityDataDef
	Environment() map[string]string
}

type OAuth2IdentityData interface {
	IdentityData

	RedirectionURI(*http.Client, string, string) (string, error)
	FetchTokenData(*http.Client, string, string) error
}

type RefreshableOAuth2IdentityData interface {
	OAuth2IdentityData

	Refresh(*http.Client) error
	RefreshTime() time.Time
}

func NewIdentityDef(typeName string, dataValue IdentityData) *IdentityDef {
	return &IdentityDef{
		Type: typeName,
		Data: dataValue,
	}
}

func (idef *IdentityDef) IsOAuth2() bool {
	_, ok := idef.Data.(OAuth2IdentityData)
	return ok
}

func (idef *IdentityDef) DecodeData(data []byte) (IdentityData, error) {
	idata := reflect.New(reflect.TypeOf(idef.Data).Elem()).Interface()
	if err := json.Unmarshal(data, idata); err != nil {
		return nil, err
	}

	return idata.(IdentityData), nil
}
