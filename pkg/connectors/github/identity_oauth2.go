package github

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-oauth2c"
)

type OAuth2Identity struct {
	Username string `json:"username"`

	ClientId     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	Scopes       []string `json:"scopes"`

	AccessToken    string     `json:"access_token,omitempty"`
	TTL            int        `json:"ttl"`
	ExpirationTime *time.Time `json:"expiration_time,omitempty"`
}

func OAuth2IdentityDef() *eventline.IdentityDef {
	def := eventline.NewIdentityDef("oauth2", &OAuth2Identity{})
	def.DeferredReadiness = true
	return def
}

func (i *OAuth2Identity) Check(c *check.Checker) {
	c.CheckStringNotEmpty("client_id", i.ClientId)
	c.CheckStringNotEmpty("client_secret", i.ClientSecret)

	c.CheckArrayNotEmpty("scopes", i.Scopes)
}

func (i *OAuth2Identity) Def() *eventline.IdentityDataDef {
	view := eventline.NewIdentityDataDef()

	view.AddEntry(&eventline.IdentityDataEntry{
		Key:   "username",
		Label: "Username",
		Value: i.Username,
		Type:  eventline.IdentityDataTypeString,
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
		Key:                   "scopes",
		Label:                 "Scopes",
		Value:                 i.Scopes,
		Type:                  eventline.IdentityDataTypeEnumList,
		EnumValues:            OAuth2Scopes(),
		PreselectedEnumValues: RequiredOAuth2Scopes(),
		MultiselectEnumSize:   8,
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

	i.AccessToken = res.AccessToken

	// GitHub OAuth2 tokens do not expire and cannot be refreshed. The
	// expire_in field is usually 0. We still handle a value greater than zero
	// if GitHub decides to support refresh later.
	if res.ExpiresIn == 0 {
		ttl := time.Duration(res.ExpiresIn) * time.Second
		expirationTime := time.Now().UTC().Add(ttl)

		i.TTL = int(res.ExpiresIn)
		i.ExpirationTime = &expirationTime
	} else {
		i.TTL = 0
		i.ExpirationTime = nil
	}

	return nil
}

func (i *OAuth2Identity) newOAuth2Client(httpClient *http.Client) (*oauth2c.Client, error) {
	issuer := "https://github.com/login/oauth"

	options := oauth2c.Options{
		HTTPClient: httpClient,

		AuthorizationEndpoint: "https://github.com/login/oauth/authorize",
		TokenEndpoint:         "https://github.com/login/oauth/access_token",
	}

	return oauth2c.NewClient(issuer, i.ClientId, i.ClientSecret, &options)
}

func (i *OAuth2Identity) Environment() map[string]string {
	return map[string]string{
		"GITHUB_USER":  i.Username,
		"GITHUB_TOKEN": i.AccessToken,
	}
}

func OAuth2Scopes() []string {
	// See
	// https://docs.github.com/en/developers/apps/building-oauth-apps/scopes-for-oauth-apps
	return []string{
		"admin:gpg_key",
		"admin:org",
		"admin:org_hook",
		"admin:public_key",
		"admin:repo_hook",
		"codespace",
		"delete:packages",
		"delete_repo",
		"gist",
		"notifications",
		"public_repo",
		"read:discussion",
		"read:gpg_key",
		"read:org",
		"read:packages",
		"read:public_key",
		"read:repo_hook",
		"read:user",
		"repo",
		"repo:invite",
		"repo:status",
		"repo_deployment",
		"security_events",
		"user",
		"user:email",
		"user:follow",
		"workflow",
		"write:discussion",
		"write:gpg_key",
		"write:org",
		"write:packages",
		"write:public_key",
		"write:repo_hook",
	}
}

func RequiredOAuth2Scopes() []string {
	return []string{
		"admin:repo_hook",
		"admin:org_hook",
		"write:repo_hook",
		"read:packages",
	}
}
