package eventline

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"go.n16f.net/ejson"
	"go.n16f.net/program"
	"go.n16f.net/service/pkg/pg"
)

var APIKeySorts Sorts = Sorts{
	Sorts: map[string]string{
		"id":   "id",
		"name": "name",
	},

	Default: "name",
}

type UnknownAPIKeyError struct {
	Id *Id
}

func (err UnknownAPIKeyError) Error() string {
	if err.Id == nil {
		return "unknown api key"
	} else {
		return fmt.Sprintf("unknown api key %q", err.Id)
	}
}

type NewAPIKey struct {
	Name string `json:"name"`
}

type APIKey struct {
	Id           Id         `json:"id"`
	AccountId    Id         `json:"account_id"`
	Name         string     `json:"name"`
	CreationTime time.Time  `json:"creation_time"`
	LastUseTime  *time.Time `json:"last_use_time,omitempty"`
	KeyHash      []byte     `json:"-"`
}

type APIKeys []*APIKey

func (nk *NewAPIKey) ValidateJSON(v *ejson.Validator) {
	CheckName(v, "name", nk.Name)
}

func (k *APIKey) SortKey(sort string) (key string) {
	switch sort {
	case "id":
		key = k.Id.String()
	case "name":
		key = k.Name
	default:
		program.Panic("unknown api key sort %q", sort)
	}

	return
}

func APIKeyNameExists(conn pg.Conn, name string, scope Scope) (bool, error) {
	ctx := context.Background()

	query := fmt.Sprintf(`
SELECT COUNT(*)
  FROM api_keys
  WHERE %s AND name = $1
`, scope.SQLCondition())

	var count int64
	err := conn.QueryRow(ctx, query, name).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (k *APIKey) LoadForUpdate(conn pg.Conn, id Id, scope Scope) error {
	query := `
SELECT id, account_id, name, creation_time,
       last_use_time, key_hash
  FROM api_keys
  WHERE id = $1
  FOR UPDATE
`
	err := pg.QueryObject(conn, k, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownAPIKeyError{Id: &id}
	}

	return err
}

func (k *APIKey) LoadUpdateByKeyHash(conn pg.Conn, keyHash []byte) error {
	now := time.Now().UTC()

	query := `
UPDATE api_keys SET
    last_use_time = $2
  WHERE key_hash = $1
  RETURNING id, account_id, name, creation_time,
            last_use_time, key_hash
`
	err := pg.QueryObject(conn, k, query, keyHash, now)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownAPIKeyError{}
	}

	return err
}

func LoadAPIKeyPage(conn pg.Conn, cursor *Cursor, scope Scope) (*Page, error) {
	query := fmt.Sprintf(`
SELECT id, account_id, name, creation_time,
       last_use_time, key_hash
  FROM api_keys
  WHERE %s AND %s
`, scope.SQLCondition(), cursor.SQLConditionOrderLimit(APIKeySorts))

	var apiKeys APIKeys
	if err := pg.QueryObjects(conn, &apiKeys, query); err != nil {
		return nil, err
	}

	return apiKeys.Page(cursor), nil
}

func (k *APIKey) Insert(conn pg.Conn) error {
	query := `
INSERT INTO api_keys
    (id, account_id, name, creation_time,
     last_use_time, key_hash)
  VALUES
    ($1, $2, $3, $4,
     $5, $6);
`
	return pg.Exec(conn, query,
		k.Id, k.AccountId, k.Name, k.CreationTime,
		k.LastUseTime, k.KeyHash)
}

func (k *APIKey) Delete(conn pg.Conn, scope Scope) error {
	query := fmt.Sprintf(`
DELETE FROM api_keys
  WHERE %s AND id = $1;
`, scope.SQLCondition())

	return pg.Exec(conn, query, k.Id)
}

func (ks APIKeys) Page(cursor *Cursor) *Page {
	elements := make([]PageElement, len(ks))
	for i, k := range ks {
		elements[i] = k
	}

	return NewPage(cursor, elements, APIKeySorts)
}

func (k *APIKey) FromRow(row pgx.Row) error {
	return row.Scan(&k.Id, &k.AccountId, &k.Name, &k.CreationTime,
		&k.LastUseTime, &k.KeyHash)
}

func (ks *APIKeys) AddFromRow(row pgx.Row) error {
	var k APIKey
	if err := k.FromRow(row); err != nil {
		return err
	}

	*ks = append(*ks, &k)
	return nil
}

func HashAPIKey(key string) []byte {
	hash := sha256.Sum256([]byte(key))
	return hash[:]
}
