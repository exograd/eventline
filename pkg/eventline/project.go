package eventline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"go.n16f.net/ejson"
	"go.n16f.net/program"
	"go.n16f.net/service/pkg/pg"
)

var ProjectSorts Sorts = Sorts{
	Sorts: map[string]string{
		"id":   "id",
		"name": "name",
	},

	Default: "name",
}

type UnknownProjectError struct {
	Id Id
}

func (err UnknownProjectError) Error() string {
	return fmt.Sprintf("unknown project %q", err.Id)
}

type UnknownProjectNameError struct {
	Name string
}

func (err UnknownProjectNameError) Error() string {
	return fmt.Sprintf("unknown project %q", err.Name)
}

type NewProject struct {
	Name string `json:"name"`
}

type Project struct {
	Id           Id        `json:"id"`
	Name         string    `json:"name"`
	CreationTime time.Time `json:"creation_time"`
	UpdateTime   time.Time `json:"update_time"`
}

type Projects []*Project

func (np *NewProject) ValidateJSON(v *ejson.Validator) {
	CheckName(v, "name", np.Name)
}

func (p *Project) SortKey(sort string) (key string) {
	switch sort {
	case "id":
		key = p.Id.String()
	case "name":
		key = p.Name
	default:
		program.Panicf("unknown project sort %q", sort)
	}

	return
}

func ProjectNameExists(conn pg.Conn, name string) (bool, error) {
	ctx := context.Background()

	query := `
SELECT COUNT(*)
  FROM projects
  WHERE name = $1
`
	var count int64
	err := conn.QueryRow(ctx, query, name).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (p *Project) Load(conn pg.Conn, id Id) error {
	query := `
SELECT id, name, creation_time, update_time
  FROM projects
  WHERE id = $1
`
	err := pg.QueryObject(conn, p, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownProjectError{Id: id}
	}

	return err
}

func (p *Project) LoadByName(conn pg.Conn, name string) error {
	query := `
SELECT id, name, creation_time, update_time
  FROM projects
  WHERE name = $1
`
	err := pg.QueryObject(conn, p, query, name)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownProjectNameError{Name: name}
	}

	return err
}

func LoadMostRecentProject(conn pg.Conn) (*Project, error) {
	query := `
SELECT id, name, creation_time, update_time
  FROM projects
  ORDER BY creation_time DESC
  LIMIT 1;
`
	var p Project
	err := pg.QueryObject(conn, &p, query)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	return &p, nil
}

func (p *Project) LoadForUpdate(conn pg.Conn, id Id) error {
	query := `
SELECT id, name, creation_time, update_time
  FROM projects
  WHERE id = $1
  FOR UPDATE
`
	err := pg.QueryObject(conn, p, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownProjectError{Id: id}
	}

	return err
}

func (ps *Projects) LoadAll(conn pg.Conn) error {
	query := `
SELECT id, name, creation_time, update_time
  FROM projects
  ORDER BY name
`
	return pg.QueryObjects(conn, ps, query)
}

func LoadProjectPage(conn pg.Conn, cursor *Cursor) (*Page, error) {
	query := fmt.Sprintf(`
SELECT id, name, creation_time, update_time
  FROM projects
  WHERE %s
`, cursor.SQLConditionOrderLimit(ProjectSorts))

	var projects Projects
	if err := pg.QueryObjects(conn, &projects, query); err != nil {
		return nil, err
	}

	return projects.Page(cursor), nil
}

func (p *Project) Insert(conn pg.Conn) error {
	query := `
INSERT INTO projects
    (id, name, creation_time, update_time)
  VALUES
    ($1, $2, $3, $4);
`
	return pg.Exec(conn, query,
		p.Id, p.Name, p.CreationTime, p.UpdateTime)
}

func (p *Project) Update(conn pg.Conn) error {
	query := `
UPDATE projects SET
    name = $2
  WHERE id = $1
`
	return pg.Exec(conn, query,
		p.Id, p.Name)
}

func (p *Project) Delete(conn pg.Conn) error {
	query := `
DELETE FROM projects
  WHERE id = $1;
`
	return pg.Exec(conn, query, p.Id)
}

func (ps Projects) Page(cursor *Cursor) *Page {
	elements := make([]PageElement, len(ps))
	for i, p := range ps {
		elements[i] = p
	}

	return NewPage(cursor, elements, ProjectSorts)
}

func (p *Project) FromRow(row pgx.Row) error {
	return row.Scan(&p.Id, &p.Name, &p.CreationTime, &p.UpdateTime)
}

func (ps *Projects) AddFromRow(row pgx.Row) error {
	var p Project
	if err := p.FromRow(row); err != nil {
		return err
	}

	*ps = append(*ps, &p)
	return nil
}
