package eventline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/exograd/eventline/pkg/utils"
	"github.com/galdor/go-ejson"
	"github.com/galdor/go-service/pkg/pg"
	"github.com/jackc/pgx/v5"
)

var IdentitySorts Sorts = Sorts{
	Sorts: map[string]string{
		"id":   "id",
		"name": "name",
	},

	Default: "name",
}

type UnknownIdentityError struct {
	Id Id
}

func (err UnknownIdentityError) Error() string {
	return fmt.Sprintf("unknown identity %q", err.Id)
}

type UnknownIdentityNameError struct {
	Name string
}

func (err UnknownIdentityNameError) Error() string {
	return fmt.Sprintf("unknown identity %q", err.Name)
}

type IdentityStatus string

const (
	IdentityStatusPending IdentityStatus = "pending"
	IdentityStatusReady   IdentityStatus = "ready"
	IdentityStatusError   IdentityStatus = "error"
)

type NewIdentity struct {
	Name      string          `json:"name"`
	Connector string          `json:"connector"`
	Type      string          `json:"type"`
	Data      IdentityData    `json:"-"`
	RawData   json.RawMessage `json:"data"`
}

type Identity struct {
	Id           Id              `json:"id"`
	ProjectId    *Id             `json:"project_id"`
	Name         string          `json:"name"`
	Status       IdentityStatus  `json:"status"`
	ErrorMessage string          `json:"error_message,omitempty"`
	CreationTime time.Time       `json:"creation_time"`
	UpdateTime   time.Time       `json:"update_time"`
	LastUseTime  *time.Time      `json:"last_use_time,omitempty"`
	RefreshTime  *time.Time      `json:"refresh_time,omitempty"`
	Connector    string          `json:"connector"`
	Type         string          `json:"type"`
	Data         IdentityData    `json:"-"`
	RawData      json.RawMessage `json:"data"`
}

type Identities []*Identity

type RawIdentity Identity

type RawIdentities []*RawIdentity

func (ni *NewIdentity) ValidateJSON(v *ejson.Validator) {
	// Note that connector and type have already been validated by
	// UnmarshalJSON.

	CheckName(v, "name", ni.Name)

	v.WithChild("data", func() {
		ni.Data.ValidateJSON(v)
	})
}

func (i *Identity) SortKey(sort string) (key string) {
	switch sort {
	case "id":
		key = i.Id.String()
	case "name":
		key = i.Name
	default:
		utils.Panicf("unknown identity sort %q", sort)
	}

	return
}

func (i *Identity) Refreshable() bool {
	cdef := GetConnectorDef(i.Connector)
	idef := cdef.Identity(i.Type)
	return idef.Refreshable
}

func (pni *NewIdentity) MarshalJSON() ([]byte, error) {
	type NewIdentity2 NewIdentity

	ni := NewIdentity2(*pni)
	data, err := json.Marshal(ni.Data)
	if err != nil {
		return nil, fmt.Errorf("cannot encode data: %w", err)
	}

	ni.RawData = data

	return json.Marshal(ni)
}

func (pni *NewIdentity) UnmarshalJSON(data []byte) error {
	type NewIdentity2 NewIdentity

	ni := NewIdentity2(*pni)
	if err := json.Unmarshal(data, &ni); err != nil {
		return err
	}

	if err := ValidateConnectorName(ni.Connector); err != nil {
		return err
	}
	cdef := GetConnectorDef(ni.Connector)

	if err := cdef.ValidateIdentityType(ni.Type); err != nil {
		return err
	}
	idef := cdef.Identity(ni.Type)

	idata, err := idef.DecodeData(ni.RawData)
	if err != nil {
		return fmt.Errorf("cannot decode data: %w", err)
	}

	ni.Data = idata

	*pni = NewIdentity(ni)
	return nil
}

func (pi *Identity) MarshalJSON() ([]byte, error) {
	type Identity2 Identity

	i := Identity2(*pi)
	data, err := json.Marshal(i.Data)
	if err != nil {
		return nil, fmt.Errorf("cannot encode data: %w", err)
	}

	i.RawData = data

	return json.Marshal(i)
}

func (pi *Identity) UnmarshalJSON(data []byte) error {
	type Identity2 Identity

	i := Identity2(*pi)
	if err := json.Unmarshal(data, &i); err != nil {
		return err
	}

	if err := ValidateConnectorName(i.Connector); err != nil {
		return err
	}
	cdef := GetConnectorDef(i.Connector)

	if err := cdef.ValidateIdentityType(i.Type); err != nil {
		return err
	}
	idef := cdef.Identity(i.Type)

	idata, err := idef.DecodeData(i.RawData)
	if err != nil {
		return fmt.Errorf("cannot decode data: %w", err)
	}

	i.Data = idata

	*pi = Identity(i)
	return nil
}

func IdentityNameExists(conn pg.Conn, name string, scope Scope) (bool, error) {
	ctx := context.Background()

	query := fmt.Sprintf(`
SELECT COUNT(*)
  FROM identities
  WHERE %s AND name = $1
`, scope.SQLCondition())

	var count int64
	err := conn.QueryRow(ctx, query, name).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (i *Identity) IsUsed(conn pg.Conn, scope Scope) (bool, error) {
	if used, err := i.IsUsedBySubscription(conn); err != nil {
		return false, fmt.Errorf("cannot check subscriptions")
	} else if used {
		return true, nil
	}

	if used, err := i.IsUsedByJob(conn, scope); err != nil {
		return false, fmt.Errorf("cannot check jobs")
	} else if used {
		return true, nil
	}

	return false, nil
}

func (i *Identity) IsUsedBySubscription(conn pg.Conn) (bool, error) {
	// We do *not* use a scope, we need to be able to find out if an identity
	// is used by a subscription *after* the project has been deleted, i.e.
	// when the project id is null.

	ctx := context.Background()

	query := `
SELECT COUNT(*)
  FROM subscriptions
  WHERE identity_id = $1
`
	var count int64
	err := conn.QueryRow(ctx, query, i.Id).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (i *Identity) IsUsedByJob(conn pg.Conn, scope Scope) (bool, error) {
	ctx := context.Background()

	query := fmt.Sprintf(`
SELECT 1
  WHERE EXISTS
          (SELECT id
             FROM jobs
             WHERE %s
               AND (spec->'trigger'->>'identity' = $1
                    OR spec->'runner'->>'identity' = $1
                    OR spec->'identities' ? $1));
`, scope.SQLCondition())

	var n int64
	err := conn.QueryRow(ctx, query, i.Name).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func (i *Identity) Load(conn pg.Conn, id Id, scope Scope) error {
	query := fmt.Sprintf(`
SELECT id, project_id, name, status, error_message,
       creation_time, update_time, last_use_time, refresh_time,
       connector, type, data
  FROM identities
  WHERE %s AND id = $1
`, scope.SQLCondition())

	err := pg.QueryObject(conn, i, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownIdentityError{Id: id}
	}

	return err
}

func (i *Identity) LoadForUpdate(conn pg.Conn, id Id, scope Scope) error {
	query := fmt.Sprintf(`
SELECT id, project_id, name, status, error_message,
       creation_time, update_time, last_use_time, refresh_time,
       connector, type, data
  FROM identities
  WHERE %s AND id = $1
  FOR UPDATE
`, scope.SQLCondition())

	err := pg.QueryObject(conn, i, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownIdentityError{Id: id}
	}

	return err
}

func (i *Identity) LoadByName(conn pg.Conn, name string, scope Scope) error {
	query := fmt.Sprintf(`
SELECT id, project_id, name, status, error_message,
       creation_time, update_time, last_use_time, refresh_time,
       connector, type, data
  FROM identities
  WHERE %s AND name = $1
`, scope.SQLCondition())

	err := pg.QueryObject(conn, i, query, name)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownIdentityNameError{Name: name}
	}

	return err
}

func (is *Identities) LoadByNames(conn pg.Conn, names []string, scope Scope) error {
	query := fmt.Sprintf(`
SELECT id, project_id, name, status, error_message,
       creation_time, update_time, last_use_time, refresh_time,
       connector, type, data
  FROM identities
  WHERE %s AND name = ANY ($1);
`, scope.SQLCondition())

	return pg.QueryObjects(conn, is, query, names)
}

func (is *Identities) LoadByNamesForUpdate(conn pg.Conn, names []string, scope Scope) error {
	query := fmt.Sprintf(`
SELECT id, project_id, name, status, error_message,
       creation_time, update_time, last_use_time, refresh_time,
       connector, type, data
  FROM identities
  WHERE %s AND name = ANY ($1)
  FOR UPDATE;
`, scope.SQLCondition())

	return pg.QueryObjects(conn, is, query, names)
}

func (is *Identities) LoadAllForUpdate(conn pg.Conn, scope Scope) error {
	query := fmt.Sprintf(`
SELECT id, project_id, name, status, error_message,
       creation_time, update_time, last_use_time, refresh_time,
       connector, type, data
  FROM identities
  WHERE %s
  FOR UPDATE
`, scope.SQLCondition())

	return pg.QueryObjects(conn, is, query)
}

func LoadIdentityIdByName(conn pg.Conn, name string, scope Scope) (Id, error) {
	ctx := context.Background()

	query := fmt.Sprintf(`
SELECT id
  FROM identities
  WHERE %s AND name = $1
`, scope.SQLCondition())

	var id Id
	if err := conn.QueryRow(ctx, query, name).Scan(&id); err != nil {
		return ZeroId, err
	}

	return id, nil
}

func LoadIdentityForRefresh(conn pg.Conn) (*Identity, error) {
	now := time.Now().UTC()

	query := `
SELECT id, project_id, name, status, error_message,
       creation_time, update_time, last_use_time, refresh_time,
       connector, type, data
  FROM identities
  WHERE refresh_time < $1
  ORDER BY refresh_time
  LIMIT 1
  FOR UPDATE SKIP LOCKED
`
	var i Identity

	err := pg.QueryObject(conn, &i, query, now)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &i, nil
}

func LoadIdentityPage(conn pg.Conn, cursor *Cursor, scope Scope) (*Page, error) {
	query := fmt.Sprintf(`
SELECT id, project_id, name, status, error_message,
       creation_time, update_time, last_use_time, refresh_time,
       connector, type, data
  FROM identities
  WHERE %s AND %s
`, scope.SQLCondition(), cursor.SQLConditionOrderLimit(IdentitySorts))

	var identities Identities
	if err := pg.QueryObjects(conn, &identities, query); err != nil {
		return nil, err
	}

	return identities.Page(cursor), nil
}

func (i *Identity) Insert(conn pg.Conn) error {
	query := `
INSERT INTO identities
    (id, project_id, name, status, error_message,
     creation_time, update_time, last_use_time, refresh_time,
     connector, type, data)
  VALUES
    ($1, $2, $3, $4, $5,
     $6, $7, $8, $9,
     $10, $11, $12);
`
	encryptedData, err := i.encodeAndEncryptData()
	if err != nil {
		return err
	}

	return pg.Exec(conn, query,
		i.Id, i.ProjectId, i.Name, i.Status, i.ErrorMessage,
		i.CreationTime, i.UpdateTime, i.LastUseTime, i.RefreshTime,
		i.Connector, i.Type, encryptedData)
}

func (i *Identity) Update(conn pg.Conn) error {
	query := `
UPDATE identities SET
    name = $2,
    status = $3,
    error_message = $4,
    update_time = $5,
    last_use_time = $6,
    refresh_time = $7,
    connector = $8,
    type = $9,
    data = $10
  WHERE id = $1
`

	encryptedData, err := i.encodeAndEncryptData()
	if err != nil {
		return err
	}

	return pg.Exec(conn, query,
		i.Id, i.Name, i.Status, i.ErrorMessage, i.UpdateTime, i.LastUseTime,
		i.RefreshTime, i.Connector, i.Type, encryptedData)
}

func (i *Identity) UpdateLastUseTime(conn pg.Conn) error {
	query := `
UPDATE identities SET
    last_use_time = $2
  WHERE id = $1
`
	return pg.Exec(conn, query,
		i.Id, i.LastUseTime)
}

func (i *Identity) UpdateForProjectDeletion(conn pg.Conn) error {
	query := `
UPDATE identities SET
    project_id = $2,
    refresh_time = $3
  WHERE id = $1
`
	return pg.Exec(conn, query,
		i.Id, i.ProjectId, i.RefreshTime)
}

func (i *Identity) Delete(conn pg.Conn) error {
	query := `
DELETE FROM identities
  WHERE id = $1
`
	return pg.Exec(conn, query, i.Id)
}

func (i *Identity) encodeAndEncryptData() ([]byte, error) {
	decryptedData, err := json.Marshal(i.Data)
	if err != nil {
		return nil, fmt.Errorf("cannot encode data: %w", err)
	}

	encryptedData, err := EncryptAES256(decryptedData)
	if err != nil {
		return nil, fmt.Errorf("cannot encrypt data: %w", err)
	}

	return encryptedData, nil
}

func (is Identities) Page(cursor *Cursor) *Page {
	elements := make([]PageElement, len(is))
	for idx, i := range is {
		elements[idx] = i
	}

	return NewPage(cursor, elements, IdentitySorts)
}

func (i *Identity) FromRow(row pgx.Row) error {
	var projectId Id
	var encryptedData []byte

	err := row.Scan(&i.Id, &projectId, &i.Name, &i.Status, &i.ErrorMessage,
		&i.CreationTime, &i.UpdateTime, &i.LastUseTime, &i.RefreshTime,
		&i.Connector, &i.Type, &encryptedData)
	if err != nil {
		return err
	}

	if !projectId.IsZero() {
		i.ProjectId = &projectId
	}

	i.RawData, err = DecryptAES256(encryptedData)
	if err != nil {
		return fmt.Errorf("cannot decrypt data of identity %q: %w", i.Id, err)
	}

	cdef := GetConnectorDef(i.Connector)
	idef := cdef.Identity(i.Type)

	idata, err := idef.DecodeData(i.RawData)
	if err != nil {
		return fmt.Errorf("cannot decode data of identity %q: %w", i.Id, err)
	}

	i.Data = idata

	return nil
}

func (is *Identities) AddFromRow(row pgx.Row) error {
	var i Identity
	if err := i.FromRow(row); err != nil {
		return err
	}

	*is = append(*is, &i)
	return nil
}
