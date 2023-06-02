package eventline

import (
	"errors"

	"github.com/exograd/go-daemon/pg"
	"github.com/galdor/go-ejson"
	"github.com/jackc/pgx/v4"
)

type ProjectSettings struct {
	Id         Id     `json:"id"` // Ignored in input
	CodeHeader string `json:"code_header"`
}

func (ps *ProjectSettings) ValidateJSON(v *ejson.Validator) {
	var shebang Shebang
	err := shebang.Parse(ps.CodeHeader)
	v.Check("code_header", err == nil, "invalid_shebang",
		"invalid shebang: %v", err)
}

func (ps *ProjectSettings) Load(conn pg.Conn, id Id) error {
	query := `
SELECT id, code_header
  FROM project_settings
  WHERE id = $1
`
	err := pg.QueryObject(conn, ps, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return &UnknownProjectError{Id: id}
	}

	return nil
}

func (ps *ProjectSettings) Insert(conn pg.Conn) error {
	query := `
INSERT INTO project_settings
    (id, code_header)
  VALUES
    ($1, $2);
`
	return pg.Exec(conn, query,
		ps.Id, ps.CodeHeader)
}

func (ps *ProjectSettings) Update(conn pg.Conn) error {
	query := `
UPDATE project_settings SET
    code_header = $2
  WHERE id = $1
`
	return pg.Exec(conn, query,
		ps.Id, ps.CodeHeader)
}

func (ps *ProjectSettings) FromRow(row pgx.Row) error {
	return row.Scan(&ps.Id, &ps.CodeHeader)
}
