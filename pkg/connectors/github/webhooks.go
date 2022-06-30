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
	"github.com/exograd/eventline/pkg/utils"
	"github.com/exograd/go-daemon/pg"
	"github.com/google/go-github/v45/github"
)

type InvalidWebhookEventError struct {
	Msg string
}

func NewInvalidWebhookEventError(format string, args ...interface{}) *InvalidWebhookEventError {
	return &InvalidWebhookEventError{Msg: fmt.Sprintf(format, args...)}
}

func (err *InvalidWebhookEventError) Error() string {
	return fmt.Sprintf("invalid webhook event: %s", err.Msg)
}

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

	// Raw events are generated for all types of payloads
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

	// Decode the payload to determine which high level events to create
	event, err := github.ParseWebHook(github.WebHookType(req), payload)
	if err != nil {
		return fmt.Errorf("cannot parse webhook event: %w", err)
	}

	switch e := event.(type) {
	case *github.RepositoryEvent:
		if e.Action == nil {
			return NewInvalidWebhookEventError("missing action")
		}

		switch *e.Action {
		case "created":
			return c.processWebhookEventRepositoryCreated(e, params)
		case "deleted":
			return c.processWebhookEventRepositoryDeleted(e, params)
		}

	case *github.PushEvent:
		return c.processWebhookEventPush(e, params)
	}

	return nil
}

func (c *Connector) processWebhookEventRepositoryCreated(e *github.RepositoryEvent, params *Parameters) error {
	if e.Org == nil {
		return NewInvalidWebhookEventError("missing organization")
	}

	if e.Org.Login == nil {
		return NewInvalidWebhookEventError("missing organization login")
	}

	if e.Repo == nil {
		return NewInvalidWebhookEventError("missing repository")
	}

	if e.Repo.Name == nil {
		return NewInvalidWebhookEventError("missing repository name")
	}

	var eventTime *time.Time
	if e.Repo.CreatedAt != nil {
		eventTime = utils.Ref(e.Repo.CreatedAt.UTC())
	}

	eventData := RepositoryCreationEvent{
		Organization: *e.Org.Login,
		Repository:   *e.Repo.Name,
	}

	err := c.CreateEvents("repository_creation", eventTime, &eventData, params)
	if err != nil {
		return fmt.Errorf("cannot create event: %w", err)
	}

	return nil
}

func (c *Connector) processWebhookEventRepositoryDeleted(e *github.RepositoryEvent, params *Parameters) error {
	if e.Org == nil {
		return NewInvalidWebhookEventError("missing organization")
	}

	if e.Org.Login == nil {
		return NewInvalidWebhookEventError("missing organization login")
	}

	if e.Repo == nil {
		return NewInvalidWebhookEventError("missing repository")
	}

	if e.Repo.Name == nil {
		return NewInvalidWebhookEventError("missing repository name")
	}

	var eventTime *time.Time
	if e.Repo.UpdatedAt != nil {
		eventTime = utils.Ref(e.Repo.UpdatedAt.UTC())
	}

	eventData := RepositoryDeletionEvent{
		Organization: *e.Org.Login,
		Repository:   *e.Repo.Name,
	}

	err := c.CreateEvents("repository_deletion", eventTime, &eventData, params)
	if err != nil {
		return fmt.Errorf("cannot create event: %w", err)
	}

	return nil
}

func (c *Connector) processWebhookEventPush(e *github.PushEvent, params *Parameters) error {
	// TODO

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
