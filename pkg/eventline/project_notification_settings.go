package eventline

import (
	"errors"
	"net/mail"

	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/pg"
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

func (ps *ProjectNotificationSettings) Check(c *check.Checker) {
	c.WithChild("email_addresses", func() {
		for i, address := range ps.EmailAddresses {
			if _, err := mail.ParseAddress(address); err != nil {
				c.AddError(i, "invalid_email_address",
					"invalid email address: %v", err)
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
