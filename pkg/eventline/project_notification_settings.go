package eventline

import (
	"errors"
	"net/mail"
	"strings"

	"github.com/exograd/go-daemon/pg"
	"github.com/galdor/go-ejson"
	"github.com/jackc/pgx/v4"
)

type ProjectNotificationSettings struct {
	Id                     Id       `json:"id"` // ignored in input
	OnSuccessfulJob        bool     `json:"on_successful_job,omitempty"`
	OnFirstSuccessfulJob   bool     `json:"on_first_successful_job,omitempty"`
	OnFailedJob            bool     `json:"on_failed_job,omitempty"`
	OnAbortedJob           bool     `json:"on_aborted_job,omitempty"`
	OnIdentityRefreshError bool     `json:"on_identity_refresh_error,omitempty"`
	EmailAddresses         []string `json:"email_addresses"`
}

func (ps *ProjectNotificationSettings) Check(v *ejson.Validator) {
	// Email addresses are validated in CheckEmailAddresses because we need
	// access to the list of allowed domains.
}

func (ps *ProjectNotificationSettings) CheckEmailAddresses(v *ejson.Validator, allowedDomains []string) {
	v.WithChild("email_addresses", func() {
		for i, as := range ps.EmailAddresses {
			a, err := mail.ParseAddress(as)
			if err != nil {
				v.AddError(i, "invalid_email_address",
					"invalid email address: %v", err)
				continue
			}

			address := a.Address

			idx := strings.LastIndex(address, "@")
			if idx == -1 {
				v.AddError(i, "invalid_email_address",
					"invalid email address: missing '@'")
				continue
			}

			if idx == len(address)-1 {
				v.AddError(i, "invalid_email_address",
					"invalid email address: empty domain")
				continue
			}

			domain := address[idx+1:]

			if len(allowedDomains) == 0 {
				// All domains are allowed by default
				continue
			}

			allowed := false
			for _, d := range allowedDomains {
				if domain == d {
					allowed = true
				}
			}

			if !allowed {
				v.AddError(i, "email_address_domain_not_allowed",
					"email address domain %q is not allowed", domain)
				continue
			}
		}
	})
}

func (ps *ProjectNotificationSettings) Load(conn pg.Conn, id Id) error {
	query := `
SELECT id, on_successful_job, on_first_successful_job,
       on_failed_job, on_aborted_job, on_identity_refresh_error,
       email_addresses
  FROM project_notification_settings
  WHERE id = $1
`
	err := pg.QueryObject(conn, ps, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownProjectError{Id: id}
	}

	return nil
}

func (ps *ProjectNotificationSettings) Insert(conn pg.Conn) error {
	query := `
INSERT INTO project_notification_settings
    (id, on_successful_job, on_first_successful_job,
     on_failed_job, on_aborted_job, on_identity_refresh_error,
     email_addresses)
  VALUES
    ($1, $2, $3,
     $4, $5, $6,
     $7);
`
	return pg.Exec(conn, query,
		ps.Id, ps.OnSuccessfulJob, ps.OnFirstSuccessfulJob,
		ps.OnFailedJob, ps.OnAbortedJob, ps.OnIdentityRefreshError,
		ps.EmailAddresses)
}

func (ps *ProjectNotificationSettings) Update(conn pg.Conn) error {
	query := `
UPDATE project_notification_settings SET
    on_successful_job = $2,
    on_first_successful_job = $3,
    on_failed_job = $4,
    on_aborted_job = $5,
    on_identity_refresh_error = $6,
    email_addresses = $7
  WHERE id = $1
`
	return pg.Exec(conn, query,
		ps.Id, ps.OnSuccessfulJob, ps.OnFirstSuccessfulJob,
		ps.OnFailedJob, ps.OnAbortedJob, ps.OnIdentityRefreshError,
		ps.EmailAddresses)
}

func (ps *ProjectNotificationSettings) FromRow(row pgx.Row) error {
	return row.Scan(&ps.Id, &ps.OnSuccessfulJob, &ps.OnFirstSuccessfulJob,
		&ps.OnFailedJob, &ps.OnAbortedJob, &ps.OnIdentityRefreshError,
		&ps.EmailAddresses)
}
