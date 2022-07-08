package service

import (
	"bytes"
	"fmt"
	"path"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/pg"
	"github.com/jhillyerd/enmime"
)

type NotificationsCfg struct {
	SMTPServer    *SMTPServerCfg `json:"smtp_server"`
	FromAddress   string         `json:"from_address"`
	SubjectPrefix string         `json:"subject_prefix"`
}

type SMTPServerCfg struct {
	Address  string `json:"address"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func (cfg *NotificationsCfg) Check(c *check.Checker) {
	c.CheckObject("smtp_server", cfg.SMTPServer)
	c.CheckStringNotEmpty("from_address", cfg.FromAddress)
}

func (cfg *SMTPServerCfg) Check(c *check.Checker) {
	c.CheckStringNotEmpty("address", cfg.Address)
}

func DefaultNotificationsCfg() *NotificationsCfg {
	return &NotificationsCfg{
		SMTPServer: &SMTPServerCfg{
			Address: "localhost:25",
		},

		FromAddress:   "no-reply@localhost",
		SubjectPrefix: "[eventline] ",
	}
}

func (s *Service) CreateNotification(conn pg.Conn, recipients []string, subject, templateName string, templateData interface{}, scope eventline.Scope) error {
	projectId := scope.(*eventline.ProjectScope).ProjectId

	now := time.Now().UTC()

	message, err := s.ComposeNotificationMessage(recipients, subject,
		templateName, templateData)
	if err != nil {
		return fmt.Errorf("cannot compose message: %w", err)
	}

	notification := eventline.Notification{
		Id:               eventline.GenerateId(),
		ProjectId:        projectId,
		Recipients:       recipients,
		Message:          message,
		NextDeliveryTime: now,
		DeliveryDelay:    0,
	}

	if err := notification.Insert(conn); err != nil {
		return fmt.Errorf("cannot insert notification: %w", err)
	}

	return nil
}

func (s *Service) ComposeNotificationMessage(recipients []string, subject, templateName string, templateData interface{}) ([]byte, error) {
	cfg := s.Cfg.Notifications

	// Careful here, the enmime builder functions all create a new copy of the
	// builder for *every single change*. Not much we can do about it
	// infortunately.

	builder := enmime.Builder()

	builder = builder.From("Eventline", cfg.FromAddress)

	for _, recipient := range recipients {
		builder = builder.To("", recipient)
	}

	builder = builder.Subject(cfg.SubjectPrefix + subject)

	body, err := s.RenderNotificationText(templateName, templateData)
	if err != nil {
		return nil, fmt.Errorf("cannot render message: %w", err)
	}
	builder = builder.Text(body)

	if err := builder.Error(); err != nil {
		return nil, err
	}

	part, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("cannot build mime part: %w", err)
	}

	var buf bytes.Buffer
	if err := part.Encode(&buf); err != nil {
		return nil, fmt.Errorf("cannot encode part: %w", err)
	}

	return buf.Bytes(), err
}

func (s *Service) RenderNotificationText(name string, data interface{}) ([]byte, error) {
	name = path.Join("notifications", name)

	obj := struct {
		Data interface{}
	}{
		Data: data,
	}

	var buf bytes.Buffer

	if err := s.TextTemplate.ExecuteTemplate(&buf, name, obj); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
