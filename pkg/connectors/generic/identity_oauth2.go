package generic

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-oauth2c"
	"go.n16f.net/ejson"
)

type OAuth2Identity struct {
	Issuer                string   `json:"issuer"`
	Discovery             bool     `json:"discovery,omitempty"`
	DiscoveryEndpoint     string   `json:"discovery_endpoint,omitempty"`
	AuthorizationEndpoint string   `json:"authorization_endpoint,omitempty"`
	TokenEndpoint         string   `json:"token_endpoint,omitempty"`
	ClientId              string   `json:"client_id"`
	ClientSecret          string   `json:"client_secret"`
	Scopes                []string `json:"scopes"`

	AccessToken    string     `json:"access_token,omitempty"`
	RefreshToken   string     `json:"refresh_token,omitempty"`
	TTL            int        `json:"ttl"`
	ExpirationTime *time.Time `json:"expiration_time,omitempty"`
}

func OAuth2IdentityDef() *eventline.IdentityDef {
	def := eventline.NewIdentityDef("oauth2", &OAuth2Identity{})
	def.DeferredReadiness = true
	def.Refreshable = true
	return def
}

func (i *OAuth2Identity) ValidateJSON(v *ejson.Validator) {
	v.CheckStringURI("issuer", i.Issuer)

	if i.DiscoveryEndpoint != "" {
		v.CheckStringURI("discovery_endpoint", i.DiscoveryEndpoint)
	}

	if i.AuthorizationEndpoint != "" {
		v.CheckStringURI("authorization_endpoint", i.AuthorizationEndpoint)
	}

	if i.TokenEndpoint != "" {
		v.CheckStringURI("token_endpoint", i.TokenEndpoint)
	}

	v.CheckStringNotEmpty("client_id", i.ClientId)
	v.CheckStringNotEmpty("client_secret", i.ClientSecret)

	v.CheckArrayNotEmpty("scopes", i.Scopes)
}

func (i *OAuth2Identity) Def() *eventline.IdentityDataDef {
	view := eventline.NewIdentityDataDef()

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:   "issuer",
		Label: "Issuer",
		Value: i.Issuer,
		Type:  eventline.IdentityDataTypeURI,
	})

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:      "discovery",
		Label:    "Discovery",
		Value:    i.Discovery,
		Type:     eventline.IdentityDataTypeBoolean,
		Optional: true,
	})

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:      "discovery_endpoint",
		Label:    "Discovery endpoint",
		Value:    i.DiscoveryEndpoint,
		Type:     eventline.IdentityDataTypeURI,
		Optional: true,
	})

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:      "authorization_endpoint",
		Label:    "Authorization endpoint",
		Value:    i.AuthorizationEndpoint,
		Type:     eventline.IdentityDataTypeURI,
		Optional: true,
	})

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:      "token_endpoint",
		Label:    "Token endpoint",
		Value:    i.TokenEndpoint,
		Type:     eventline.IdentityDataTypeURI,
		Optional: true,
	})

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:      "client_id",
		Label:    "Client id",
		Value:    i.ClientId,
		Type:     eventline.IdentityDataTypeString,
		Verbatim: true,
	})

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:      "client_secret",
		Label:    "Client secret",
		Value:    i.ClientSecret,
		Type:     eventline.IdentityDataTypeString,
		Secret:   true,
		Verbatim: true,
	})

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:   "scopes",
		Label: "Scopes",
		Value: i.Scopes,
		Type:  eventline.IdentityDataTypeStringList,
	})

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:      "access_token",
		Label:    "Access token",
		Value:    i.AccessToken,
		Type:     eventline.IdentityDataTypeString,
		Optional: true,
		Secret:   true,
		Verbatim: true,
		Internal: true,
	})

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:      "refresh_token",
		Label:    "Refresh token",
		Value:    i.RefreshToken,
		Type:     eventline.IdentityDataTypeString,
		Optional: true,
		Secret:   true,
		Verbatim: true,
		Internal: true,
	})

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:      "expiration_date",
		Label:    "Expiration date",
		Value:    i.ExpirationTime,
		Type:     eventline.IdentityDataTypeDate,
		Optional: true,
		Internal: true,
	})

	return view
}

func (i *OAuth2Identity) RedirectionURI(httpClient *http.Client, state, redirectionURI string) (string, error) {
	client, err := i.newOAuth2Client(httpClient)
	if err != nil {
		return "", fmt.Errorf("cannot create oauth2 client: %w", err)
	}

	req := oauth2c.AuthorizeRequest{
		RedirectURI: redirectionURI,
		State:       state,
		Scope:       i.Scopes,
	}

	uri := client.AuthorizeURL("code", &req)

	return uri.String(), nil
}

func (i *OAuth2Identity) FetchTokenData(httpClient *http.Client, code, redirectionURI string) error {
	client, err := i.newOAuth2Client(httpClient)
	if err != nil {
		return fmt.Errorf("cannot create oauth2 client: %w", err)
	}

	req := oauth2c.TokenCodeRequest{
		Code:        code,
		RedirectURI: redirectionURI,
	}

	res, err := client.Token(context.Background(), "authorization_code", &req)
	if err != nil {
		return err
	}

	ttl := time.Duration(res.ExpiresIn) * time.Second
	expirationTime := time.Now().UTC().Add(ttl)

	i.AccessToken = res.AccessToken
	i.RefreshToken = res.RefreshToken
	i.TTL = int(res.ExpiresIn)
	i.ExpirationTime = &expirationTime

	return nil
}

func (i *OAuth2Identity) Refresh(httpClient *http.Client) error {
	if i.RefreshToken == "" {
		return fmt.Errorf("missing refresh token")
	}

	client, err := i.newOAuth2Client(httpClient)
	if err != nil {
		return fmt.Errorf("cannot create oauth2 client: %w", err)
	}

	req := oauth2c.TokenRefreshRequest{
		RefreshToken: i.RefreshToken,
	}

	res, err := client.Token(context.Background(), "refresh_token", &req)
	if err != nil {
		return err
	}

	ttl := time.Duration(res.ExpiresIn) * time.Second
	expirationTime := time.Now().UTC().Add(ttl)

	i.AccessToken = res.AccessToken
	i.RefreshToken = res.RefreshToken
	i.TTL = int(res.ExpiresIn)
	i.ExpirationTime = &expirationTime

	return nil
}

func (i *OAuth2Identity) RefreshTime() time.Time {
	now := time.Now().UTC()

	halfTTL := time.Duration(math.Ceil(float64(i.TTL)/2.0)) * time.Second

	return now.Add(halfTTL)
}

func (i *OAuth2Identity) newOAuth2Client(httpClient *http.Client) (*oauth2c.Client, error) {
	options := oauth2c.Options{
		HTTPClient: httpClient,

		Discover:              i.Discovery,
		DiscoveryEndpoint:     i.DiscoveryEndpoint,
		AuthorizationEndpoint: i.AuthorizationEndpoint,
		TokenEndpoint:         i.TokenEndpoint,
	}

	return oauth2c.NewClient(i.Issuer, i.ClientId, i.ClientSecret, &options)
}

func (i *OAuth2Identity) Environment() map[string]string {
	return map[string]string{}
}
