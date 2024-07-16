package eventline

import (
	"fmt"
	"time"

	"go.n16f.net/service/pkg/pg"
	"github.com/jackc/pgx/v5"
)

type JobStats struct {
	JobId        Id             `json:"job_id"`
	NbExecutions int            `json:"nb_executions"`
	DurationP50  *time.Duration `json:"duration_p50,omitempty"`
	SuccessRatio float64        `json:"success_ratio"` // last 7 days
}

type JobStatsList []*JobStats

func (js *JobStats) SuccessPercentage() float64 {
	return js.SuccessRatio * 100.0
}

func (js *JobStats) SuccessPercentageString() string {
	return fmt.Sprintf("%.0f%%", js.SuccessPercentage())
}

func LoadJobStats(conn pg.Conn, jobIds Ids, scope Scope) (map[Id]*JobStats, error) {
	query := fmt.Sprintf(`
SELECT job_id,
       COUNT(id) AS nb_executions,
       (PERCENTILE_CONT(0.5) WITHIN GROUP
          (ORDER BY end_time - start_time)
          FILTER (WHERE status = 'successful')) AS duration_p50,
       (COUNT(id) FILTER (WHERE status = 'successful')
          / COUNT(id)::FLOAT) AS success_ratio
  FROM job_executions
  WHERE %s
    AND job_id = ANY ($1)
    AND end_time > $2
    GROUP BY job_id;
`, scope.SQLCondition())

	now := time.Now().UTC()
	minTime := now.AddDate(0, 0, -7)

	var jss JobStatsList
	err := pg.QueryObjects(conn, &jss, query, jobIds, minTime)
	if err != nil {
		return nil, err
	}

	table := make(map[Id]*JobStats)
	for _, js := range jss {
		table[js.JobId] = js
	}

	return table, nil
}

func (js *JobStats) FromRow(row pgx.Row) error {
	return row.Scan(&js.JobId, &js.NbExecutions, &js.DurationP50,
		&js.SuccessRatio)
}

func (jss *JobStatsList) AddFromRow(row pgx.Row) error {
	var js JobStats
	if err := js.FromRow(row); err != nil {
		return err
	}

	*jss = append(*jss, &js)
	return nil
}
