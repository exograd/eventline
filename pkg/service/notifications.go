package service

import (
	"bytes"
	"fmt"
	"net"
	"net/smtp"
	"path"
	"time"

	"github.com/Shopify/gomail"
	"github.com/exograd/eventline/pkg/eventline"
	"go.n16f.net/ejson"
	"go.n16f.net/service/pkg/pg"
	"go.n16f.net/uuid"
)

type NotificationsCfg struct {
	SMTPServer     *SMTPServerCfg `json:"smtp_server"`
	FromAddress    string         `json:"from_address"`
	SubjectPrefix  string         `json:"subject_prefix"`
	Signature      string         `json:"signature"`
	AllowedDomains []string       `json:"allowed_domains"`
}

type SMTPServerCfg struct {
	Address  string `json:"address"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func (cfg *NotificationsCfg) ValidateJSON(v *ejson.Validator) {
	v.CheckObject("smtp_server", cfg.SMTPServer)

	v.CheckStringNotEmpty("from_address", cfg.FromAddress)

	v.WithChild("allowed_domains", func() {
		for i, domain := range cfg.AllowedDomains {
			v.CheckStringNotEmpty(i, domain)
		}
	})
}

func (cfg *SMTPServerCfg) ValidateJSON(v *ejson.Validator) {
	v.CheckStringNotEmpty("address", cfg.Address)
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

	// Render the message body
	body, err := s.RenderNotificationText(templateName, templateData)
	if err != nil {
		return fmt.Errorf("cannot render message: %w", err)
	}

	// It is really easy to get leading and trailing spaces in templates,
	// especially newline characters. Removing them here is a lot easier that
	// removing all newlines in complex if/else/end blocks.
	body = bytes.TrimSpace(body)

	if cfg.Signature != "" {
		body = append(body, []byte("\n\n--\n"+cfg.Signature+"\n")...)
	}

	// Compose and render the message
	m := gomail.NewMessage()

	m.SetAddressHeader("From", cfg.FromAddress, "Eventline")
	m.SetHeader("To", recipients...)
	m.SetHeader("Subject", cfg.SubjectPrefix+subject)
	m.SetBody("text/plain", string(body))

	var buf bytes.Buffer
	if _, err := m.WriteTo(&buf); err != nil {
		return fmt.Errorf("cannot render message: %w", err)
	}

	// Create the notification
	notification := eventline.Notification{
		Id:               uuid.MustGenerate(uuid.V7),
		ProjectId:        projectId,
		Recipients:       recipients,
		Message:          buf.Bytes(),
		NextDeliveryTime: now,
		DeliveryDelay:    0,
	}

	if err := notification.Insert(conn); err != nil {
		return fmt.Errorf("cannot insert notification: %w", err)
	}

	return nil
}

func (s *Service) RenderNotificationText(name string, data interface{}) ([]byte, error) {
	name = path.Join("notifications", name)

	obj := struct {
		Data interface{}
	}{
		Data: data,
	}

	return s.Service.RenderTextTemplate(name, obj)
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
