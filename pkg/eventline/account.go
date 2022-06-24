package eventline

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"github.com/exograd/eventline/pkg/utils"
	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/dcrypto"
	"github.com/exograd/go-daemon/pg"
	"github.com/jackc/pgx/v4"
	"golang.org/x/crypto/pbkdf2"
)

var AccountSorts Sorts = Sorts{
	Sorts: map[string]string{
		"id":       "id",
		"username": "username",
	},

	Default: "username",
}

type AccountRole string

const (
	AccountRoleUser  AccountRole = "user"
	AccountRoleAdmin AccountRole = "admin"
)

var AccountRoleValues = []AccountRole{
	AccountRoleUser,
	AccountRoleAdmin,
}

const (
	MinUsernameLength = 3
	MaxUsernameLength = 100

	MinPasswordLength = 8
	MaxPasswordLength = 100

	SaltSize = 32 // bytes

	// Current OWASP recommendations are 310'000+
	// (https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html#pbkdf2).
	//
	// Note that changing it requires re-hashing all password hashes in the
	// database. Do not change it.
	NbPBKDF2Iterations = 350_000
)

type UnknownAccountError struct {
	Id Id
}

func (err UnknownAccountError) Error() string {
	return fmt.Sprintf("unknown account %q", err.Id)
}

type UnknownUsernameError struct {
	Username string
}

func (err UnknownUsernameError) Error() string {
	return fmt.Sprintf("unknown username %q", err.Username)
}

type NewAccount struct {
	Username             string      `json:"username"`
	Password             string      `json:"password"`
	PasswordConfirmation string      `json:"password_confirmation"`
	Role                 AccountRole `json:"role"`
}

type AccountUpdate struct {
	Username string      `json:"username"`
	Role     AccountRole `json:"role"`
}

type AccountPasswordUpdate struct {
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
}

type AccountSelfUpdate struct {
	Settings *AccountSettings `json:"settings"`
}

type Account struct {
	Id            Id               `json:"id"`
	CreationTime  time.Time        `json:"creation_time"`
	Username      string           `json:"username"`
	Salt          []byte           `json:"-"`
	PasswordHash  []byte           `json:"-"`
	Role          AccountRole      `json:"role"`
	LastLoginTime *time.Time       `json:"last_login_time,omitempty"`
	LastProjectId *Id              `json:"last_project_id,omitempty"`
	Settings      *AccountSettings `json:"settings"`
}

type Accounts []*Account

type AccountSettings struct {
	DateFormat DateFormat `json:"date_format,omitempty"`
	PageSize   int        `json:"page_size,omitempty"`
}

func DefaultAccountSettings() *AccountSettings {
	return &AccountSettings{
		DateFormat: "relative",
		PageSize:   20,
	}
}

func (as *AccountSettings) Check(c *check.Checker) {
	c.CheckStringValue("date_format", as.DateFormat, DateFormatValues)
	c.CheckIntMinMax("page_size", as.PageSize, MinPageSize, MaxPageSize)
}

func (na *NewAccount) Check(c *check.Checker) {
	c.CheckStringLengthMinMax("username", na.Username,
		MinUsernameLength, MaxUsernameLength)

	c.CheckStringLengthMinMax("password", na.Password,
		MinPasswordLength, MaxPasswordLength)

	c.Check("password_confirmation", na.PasswordConfirmation == na.Password,
		"password_mismatch", "password confirmation and password do not match")

	c.CheckStringValue("role", na.Role, AccountRoleValues)
}

func (au *AccountUpdate) Check(c *check.Checker) {
	c.CheckStringLengthMinMax("username", au.Username,
		MinUsernameLength, MaxUsernameLength)

	c.CheckStringValue("role", au.Role, AccountRoleValues)
}

func (au *AccountPasswordUpdate) Check(c *check.Checker) {
	c.CheckStringLengthMinMax("password", au.Password,
		MinPasswordLength, MaxPasswordLength)

	c.Check("password_confirmation", au.PasswordConfirmation == au.Password,
		"password_mismatch", "password confirmation and password do not match")
}

func (au *AccountSelfUpdate) Check(c *check.Checker) {
	c.CheckOptionalObject("settings", au.Settings)
}

func (a *Account) CheckPassword(password string) bool {
	attempt := HashPassword(password, a.Salt)
	return subtle.ConstantTimeCompare(attempt, a.PasswordHash) == 1
}

func GenerateSalt() []byte {
	return dcrypto.RandomBytes(SaltSize)
}

func HashPassword(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, NbPBKDF2Iterations, 32,
		sha256.New)
}

func (a *Account) SortKey(sort string) (key string) {
	switch sort {
	case "id":
		key = a.Id.String()
	case "username":
		key = a.Username
	default:
		utils.Panicf("unknown account sort %q", sort)
	}

	return
}

func UsernameExists(conn pg.Conn, username string) (bool, error) {
	ctx := context.Background()

	query := `
SELECT COUNT(*)
  FROM accounts
  WHERE username = $1
`
	var count int64
	err := conn.QueryRow(ctx, query, username).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (as *Accounts) LoadAll(conn pg.Conn) error {
	query := `
SELECT id, creation_time, username, salt,
       password_hash, role, last_login_time, last_project_id,
       settings
  FROM accounts
  ORDER BY username
`
	return pg.QueryObjects(conn, as, query)
}

func (a *Account) Load(conn pg.Conn, id Id) error {
	query := `
SELECT id, creation_time, username, salt,
       password_hash, role, last_login_time, last_project_id,
       settings
  FROM accounts
  WHERE id = $1
`
	err := pg.QueryObject(conn, a, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownAccountError{Id: id}
	}

	return err
}

func (a *Account) LoadForUpdate(conn pg.Conn, id Id) error {
	query := `
SELECT id, creation_time, username, salt,
       password_hash, role, last_login_time, last_project_id,
       settings
  FROM accounts
  WHERE id = $1
  FOR UPDATE
`
	err := pg.QueryObject(conn, a, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownAccountError{Id: id}
	}

	return err
}

func (a *Account) LoadByUsernameForUpdate(conn pg.Conn, username string) error {
	query := `
SELECT id, creation_time, username, salt,
       password_hash, role, last_login_time, last_project_id,
       settings
  FROM accounts
  WHERE username = $1
  FOR UPDATE;
`
	err := pg.QueryObject(conn, a, query, username)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownUsernameError{Username: username}
	}

	return err
}

func LoadAccountPage(conn pg.Conn, cursor *Cursor) (*Page, error) {
	query := fmt.Sprintf(`
SELECT id, creation_time, username, salt,
       password_hash, role, last_login_time, last_project_id,
       settings
  FROM accounts
  WHERE %s
`, cursor.SQLConditionOrderLimit(AccountSorts))

	var accounts Accounts
	if err := pg.QueryObjects(conn, &accounts, query); err != nil {
		return nil, err
	}

	return accounts.Page(cursor), nil
}

func (a *Account) Insert(conn pg.Conn) error {
	query := `
INSERT INTO accounts
    (id, creation_time, username, salt, password_hash,
     role, last_login_time, last_project_id, settings)
  VALUES
    ($1, $2, $3, $4, $5,
     $6, $7, $8, $9);
`
	return pg.Exec(conn, query,
		a.Id, a.CreationTime, a.Username, a.Salt, a.PasswordHash,
		a.Role, a.LastLoginTime, a.LastProjectId, a.Settings)
}

func (a *Account) UpdateForLogin(conn pg.Conn) error {
	query := `
UPDATE accounts SET
    last_login_time = $2,
    last_project_id = $3
  WHERE id = $1;
`
	return pg.Exec(conn, query, a.Id, a.LastLoginTime, a.LastProjectId)
}

func (a *Account) SelfUpdate(conn pg.Conn) error {
	query := `
UPDATE accounts SET
    settings = $2,
    salt = $3,
    password_hash = $4
  WHERE id = $1;
`
	return pg.Exec(conn, query, a.Id, a.Settings, a.Salt, a.PasswordHash)
}

func UpdateAccountLastProjectId(conn pg.Conn, accountId Id, projectId *Id) error {
	query := `
UPDATE accounts SET
    last_project_id = $2
  WHERE id = $1;
`
	return pg.Exec(conn, query, accountId, projectId)
}

func UpdateAccountsForProjectDeletion(conn pg.Conn, projectId Id) error {
	query := `
UPDATE accounts SET
    last_project_id = NULL
  WHERE last_project_id = $1
`
	return pg.Exec(conn, query, projectId)
}

func (a *Account) Update(conn pg.Conn) error {
	query := `
UPDATE accounts SET
    username = $2,
    salt = $3,
    password_hash = $4,
    role = $5
  WHERE id = $1
`
	return pg.Exec(conn, query,
		a.Id, a.Username, a.Salt, a.PasswordHash, a.Role)
}

func DeleteAccount(conn pg.Conn, accountId Id) error {
	query := `
DELETE FROM accounts
  WHERE id = $1;
`
	return pg.Exec(conn, query, accountId)
}

func (as Accounts) Page(cursor *Cursor) *Page {
	elements := make([]PageElement, len(as))
	for i, a := range as {
		elements[i] = a
	}

	return NewPage(cursor, elements, AccountSorts)
}

func (a *Account) FromRow(row pgx.Row) error {
	var lastProjectId Id
	var settings AccountSettings

	err := row.Scan(&a.Id, &a.CreationTime, &a.Username, &a.Salt,
		&a.PasswordHash, &a.Role, &a.LastLoginTime, &lastProjectId,
		&settings)
	if err != nil {
		return err
	}

	if !lastProjectId.IsZero() {
		a.LastProjectId = &lastProjectId
	}

	a.Settings = &settings

	return nil
}

func (as *Accounts) AddFromRow(row pgx.Row) error {
	var a Account
	if err := a.FromRow(row); err != nil {
		return err
	}

	*as = append(*as, &a)
	return nil
}
