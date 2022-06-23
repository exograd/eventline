package eventline

import (
	"fmt"

	"github.com/exograd/go-daemon/pg"
)

func (j *Job) LoadStatusCounts(conn pg.Conn, params *MetricParameters) (MetricPoints, error) {
	query := fmt.Sprintf(`
SELECT EXTRACT(EPOCH FROM date_trunc('%s', start_time))::INT8,
       COUNT(status),
       COUNT(status) FILTER (WHERE status = 'successful'),
       COUNT(status) FILTER (WHERE status = 'aborted'),
       COUNT(status) FILTER (WHERE status = 'failed')
  FROM job_executions
  WHERE job_id = $1
    AND start_time BETWEEN $2 AND $3
    GROUP BY date_trunc('%s', start_time);
`, string(params.Granularity), string(params.Granularity))

	var points MetricPoints
	err := pg.QueryObjects(conn, &points, query,
		j.Id, params.Start, params.End)
	if err != nil {
		return nil, err
	}

	return points, nil
}

func (j *Job) LoadRunningTimes(conn pg.Conn, params *MetricParameters) (MetricPoints, error) {
	query := fmt.Sprintf(`
WITH data AS
       (SELECT EXTRACT(EPOCH FROM date_trunc('%s', start_time))::INT8 AS date,
               percentile_cont(ARRAY[0.99, 0.8, 0.5])
                 WITHIN GROUP
                 (ORDER BY EXTRACT(EPOCH FROM end_time - start_time))
                 AS percentiles
          FROM job_executions
          WHERE job_id = $1
            AND status = 'successful'
            AND start_time BETWEEN $2 AND $3
            AND end_time IS NOT NULL
            GROUP BY date_trunc('%s', start_time))
  SELECT date, percentiles[1], percentiles[2], percentiles[3] FROM data
`, string(params.Granularity), string(params.Granularity))

	var points MetricPoints
	err := pg.QueryObjects(conn, &points, query,
		j.Id, params.Start, params.End)
	if err != nil {
		return nil, err
	}

	return points, nil
}
