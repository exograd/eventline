package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/go-daemon/pg"
)

type DuplicateUsernameError struct {
	Username string
}

func (err DuplicateUsernameError) Error() string {
	return fmt.Sprintf("duplicate account username %q", err.Username)
}

func (s *Service) CreateAccount(newAccount *eventline.NewAccount) (*eventline.Account, error) {
	var account *eventline.Account

	err := s.Daemon.Pg.WithTx(func(conn pg.Conn) (err error) {
		account, err = s.createAccount(conn, newAccount)
		return
	})
	if err != nil {
		return nil, err
	}

	return account, nil
}

func (s *Service) createAccount(conn pg.Conn, newAccount *eventline.NewAccount) (*eventline.Account, error) {
	var account *eventline.Account

	now := time.Now().UTC()

	salt := eventline.GenerateSalt()
	passwordHash := eventline.HashPassword(newAccount.Password, salt)

	exists, err := eventline.UsernameExists(conn, newAccount.Username)
	if err != nil {
		return nil, fmt.Errorf("cannot check username existence: %w", err)
	} else if exists {
		return nil, &DuplicateUsernameError{Username: newAccount.Username}
	}

	account = &eventline.Account{
		Id:           eventline.GenerateId(),
		CreationTime: now,
		Username:     newAccount.Username,
		Salt:         salt,
		PasswordHash: passwordHash,
		Role:         newAccount.Role,
		Settings:     eventline.DefaultAccountSettings(),
	}

	if err := account.Insert(conn); err != nil {
		return nil, fmt.Errorf("cannot insert account: %w", err)
	}

	return account, nil
}

func (s *Service) MaybeCreateDefaultAccount(conn pg.Conn) (*eventline.Account, error) {
	var existingAccount eventline.Account
	err := existingAccount.LoadByUsernameForUpdate(conn, "admin")
	if err == nil {
		return &existingAccount, nil
	}

	if err != nil {
		var unknownUsernameErr *eventline.UnknownUsernameError
		if !errors.As(err, &unknownUsernameErr) {
			return nil, fmt.Errorf("cannot load account: %w", err)
		}
	}

	newAccount := eventline.NewAccount{
		Username:             "admin",
		Password:             "admin",
		PasswordConfirmation: "admin",
		Role:                 eventline.AccountRoleAdmin,
	}

	s.Log.Info("creating default %q account", newAccount.Username)

	account, err := s.createAccount(conn, &newAccount)
	if err != nil {
		return nil, fmt.Errorf("cannot create account: %w", err)
	}

	return account, nil
}

func (s *Service) SelectAccountProject(projectId eventline.Id, hctx *HTTPContext) error {
	return s.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		var project eventline.Project
		if err := project.Load(conn, projectId); err != nil {
			return fmt.Errorf("cannot load project: %w", err)
		}

		err := eventline.UpdateAccountLastProjectId(conn, *hctx.AccountId,
			&projectId)
		if err != nil {
			return fmt.Errorf("cannot update account last project id: %w", err)
		}

		session := hctx.Session

		session.Data.ProjectId = &projectId
		if err := session.UpdateData(conn); err != nil {
			return fmt.Errorf("cannot update session: %w", err)
		}

		hctx.ProjectId = &projectId
		hctx.ProjectName = project.Name

		return nil
	})
}

func (s *Service) SelfUpdateAccount(accountId eventline.Id, update *eventline.AccountSelfUpdate, hctx *HTTPContext) error {
	return s.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		var account eventline.Account
		if err := account.LoadForUpdate(conn, accountId); err != nil {
			return fmt.Errorf("cannot load account: %w", err)
		}

		account.Settings = update.Settings

		if err := account.SelfUpdate(conn); err != nil {
			return fmt.Errorf("cannot update account: %w", err)
		}

		hctx.AccountSettings = account.Settings

		return nil
	})
}

func (s *Service) SelfUpdateAccountPassword(accountId eventline.Id, update *eventline.AccountPasswordUpdate) error {
	return s.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		var account eventline.Account
		if err := account.LoadForUpdate(conn, accountId); err != nil {
			return fmt.Errorf("cannot load account: %w", err)
		}

		salt := eventline.GenerateSalt()
		passwordHash := eventline.HashPassword(update.Password, salt)

		account.Salt = salt
		account.PasswordHash = passwordHash

		if err := account.SelfUpdate(conn); err != nil {
			return fmt.Errorf("cannot update account: %w", err)
		}

		return nil
	})
}

func (s *Service) UpdateAccount(accountId eventline.Id, update *eventline.AccountUpdate) (*eventline.Account, error) {
	var account eventline.Account

	err := s.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		if err := account.LoadForUpdate(conn, accountId); err != nil {
			return fmt.Errorf("cannot load account: %w", err)
		}

		if update.Username != account.Username {
			exists, err := eventline.UsernameExists(conn, update.Username)
			if err != nil {
				return fmt.Errorf("cannot check username existence: %w", err)
			} else if exists {
				return &DuplicateUsernameError{Username: update.Username}
			}
		}

		account.Username = update.Username
		account.Role = update.Role

		if err := account.Update(conn); err != nil {
			return fmt.Errorf("cannot update account: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &account, nil
}

func (s *Service) UpdateAccountPassword(accountId eventline.Id, update *eventline.AccountPasswordUpdate) (*eventline.Account, error) {
	var account eventline.Account

	err := s.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		if err := account.LoadForUpdate(conn, accountId); err != nil {
			return fmt.Errorf("cannot load account: %w", err)
		}

		salt := eventline.GenerateSalt()
		passwordHash := eventline.HashPassword(update.Password, salt)

		account.Salt = salt
		account.PasswordHash = passwordHash

		if err := account.Update(conn); err != nil {
			return fmt.Errorf("cannot update account: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &account, nil
}

func (s *Service) DeleteAccount(accountId eventline.Id) error {
	return s.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		if err := eventline.DeleteAccount(conn, accountId); err != nil {
			return err
		}

		return nil
	})
}
