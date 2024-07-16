package eventline

import (
	"fmt"

	"go.n16f.net/service/pkg/pg"
)

type FavouriteJob struct {
	AccountId Id
	ProjectId Id
	JobId     Id
}

func LoadFavouriteJobs(conn pg.Conn, scope Scope) (Jobs, error) {
	query := fmt.Sprintf(`
SELECT j.id, j.project_id, j.creation_time, j.update_time, j.disabled, j.spec
  FROM jobs AS j
  JOIN favourite_jobs AS fj ON fj.job_id = j.id
  WHERE %s
  ORDER BY j.spec->'name'
`, scope.SQLCondition2("fj"))

	var jobs Jobs
	if err := pg.QueryObjects(conn, &jobs, query); err != nil {
		return nil, err
	}

	return jobs, nil
}

func (fj *FavouriteJob) Upsert(conn pg.Conn) error {
	query := `
INSERT INTO favourite_jobs
    (account_id, project_id, job_id)
  VALUES
    ($1, $2, $3)
  ON CONFLICT (account_id, project_id, job_id) DO NOTHING;
`
	return pg.Exec(conn, query,
		fj.AccountId, fj.ProjectId, fj.JobId)
}

func DeleteFavouriteJob(conn pg.Conn, jobId Id, scope Scope) error {
	query := fmt.Sprintf(`
DELETE FROM favourite_jobs
  WHERE %s AND job_id = $1
`, scope.SQLCondition())

	return pg.Exec(conn, query, jobId)
}
