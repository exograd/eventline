package eventline

import (
	"time"

	"github.com/jackc/pgx/v5"
)

type MetricGranularity string

const (
	MetricGranularityDay  MetricGranularity = "day"
	MetricGranularityHour MetricGranularity = "hour"
)

type MetricParameters struct {
	Start       time.Time
	End         time.Time
	Granularity MetricGranularity
}

type MetricPoint []interface{}
type MetricPoints []MetricPoint

func (p *MetricPoint) FromRow(row pgx.Row) error {
	rows := row.(pgx.Rows)

	values, err := rows.Values()
	if err != nil {
		return err
	}

	points := make(MetricPoint, len(values))

	for i, value := range values {
		points[i] = value
	}

	*p = points

	return nil
}

func (ps *MetricPoints) AddFromRow(row pgx.Row) error {
	var p MetricPoint
	if err := p.FromRow(row); err != nil {
		return err
	}

	*ps = append(*ps, p)
	return nil
}
