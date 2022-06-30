package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/pg"
	"github.com/google/go-github/v45/github"
)

func (c *Connector) WebhookURI(params *Parameters) string {
	targetPart := url.PathEscape(params.Target())
	path := "/ext/connectors/github/hooks/" + targetPart
	uri := c.webHTTPServerURI.ResolveReference(&url.URL{Path: path})
	return uri.String()
}

func (c *Connector) WebhookSecret(params *Parameters) string {
	key := c.webhookKey
	value := []byte(params.Target())

	mac := hmac.New(sha256.New, key)
	mac.Write(value)
	code := mac.Sum(nil)

	return hex.EncodeToString(code)
}

func (c *Connector) ProcessWebhookRequest(req *http.Request, params *Parameters) error {
	secret := c.WebhookSecret(params)
	payload, err := github.ValidatePayload(req, []byte(secret))
	if err != nil {
		return fmt.Errorf("invalid signature: %w", err)
	}

	var rawMsg interface{}
	if err := json.Unmarshal(payload, &rawMsg); err != nil {
		return fmt.Errorf("cannot decode payload: %w", err)
	}

	rawEventData := RawEvent{
		DeliveryId: github.DeliveryID(req),
		EventType:  github.WebHookType(req),
		Event:      rawMsg,
	}

	if err := c.CreateEvents("raw", nil, &rawEventData, params); err != nil {
		return fmt.Errorf("cannot create event: %w", err)
	}

	return nil
}

func (c *Connector) CreateEvents(ename string, eventTime *time.Time, eventData eventline.EventData, params *Parameters) error {
	return c.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		var subs eventline.Subscriptions

		subs, err := LoadSubscriptionsByParams(conn, ename, params)
		if err != nil {
			return fmt.Errorf("cannot load subscriptions: %w", err)
		}

		for _, sub := range subs {
			event := sub.NewEvent(c.Def.Name, ename, eventTime, eventData)

			if err := event.Insert(conn); err != nil {
				return fmt.Errorf("cannot insert event: %w", err)
			}
		}

		return nil
	})
}

func LoadSubscriptionsByParams(conn pg.Conn, ename string, params *Parameters) (eventline.Subscriptions, error) {
	repoCond := "TRUE"
	if params.Repository != "" {
		repoCond = "gs.repository = $3"
	}

	query := fmt.Sprintf(`
SELECT es.id, es.project_id, es.job_id, es.identity_id, es.connector, es.event,
       es.parameters, es.creation_time, es.status, es.update_delay,
       es.last_update, es.next_update
  FROM subscriptions AS es
  JOIN c_github_subscriptions AS gs ON gs.id = es.id
  WHERE es.connector = 'github'
    AND es.event = $1
    AND gs.organization = $2
    AND %s
`, repoCond)

	args := []interface{}{ename, params.Organization}
	if params.Repository != "" {
		args = append(args, params.Repository)
	}

	var subs eventline.Subscriptions
	if err := pg.QueryObjects(conn, &subs, query, args...); err != nil {
		return nil, err
	}

	return subs, nil
}
