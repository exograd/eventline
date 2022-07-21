package service

import (
	"bytes"
	"fmt"
	"net"
	"net/smtp"
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
	Signature     string         `json:"signature"`
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
		Signature:     "This email is a notification sent by the Eventline job scheduling software.",
	}
}

func (s *Service) CreateNotification(conn pg.Conn, recipients []string, subject, templateName string, templateData interface{}, scope eventline.Scope) error {
	if len(recipients) == 0 {
		s.Log.Debug(1, "dropping notification: no recipients")
		return nil
	}

	cfg := s.Cfg.Notifications
	projectId := scope.(*eventline.ProjectScope).ProjectId
	now := time.Now().UTC()

	message, err := s.ComposeNotificationMessage(recipients, subject,
		templateName, templateData)
	if err != nil {
		return fmt.Errorf("cannot compose message: %w", err)
	}

	if cfg.Signature != "" {
		message = append(message, []byte("\n\n--\n"+cfg.Signature+"\n")...)
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

	// It is really easy to get leading and trailing spaces in templates,
	// especially newline characters. Removing them here is a lot easier that
	// removing all newlines in complex if/else/end blocks.
	body = bytes.TrimSpace(body)

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

func (s *Service) DeliverNotification(conn pg.Conn, n *eventline.Notification) error {
	cfg := s.Cfg.Notifications
	smtpCfg := cfg.SMTPServer

	host, _, err := net.SplitHostPort(smtpCfg.Address)
	if err != nil {
		return fmt.Errorf("invalid smtp server address %q: %w",
			smtpCfg.Address, err)
	}

	var auth smtp.Auth
	if smtpCfg.Username != "" && smtpCfg.Password != "" {
		auth = smtp.PlainAuth("", smtpCfg.Username, smtpCfg.Password, host)
	}

	return smtp.SendMail(smtpCfg.Address, auth, cfg.FromAddress, n.Recipients,
		n.Message)
}
