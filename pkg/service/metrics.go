package service

import (
	"fmt"
	"time"

	"github.com/exograd/eventline/pkg/eventline"
)

func (h *HTTPHandler) ParseMetricParameters() (*eventline.MetricParameters, error) {
	var params eventline.MetricParameters

	now := time.Now().UTC()

	params.Start = now.AddDate(0, 0, -30)
	params.End = now

	if t, err := h.TimestampQueryParameter("start"); err != nil {
		return nil, err
	} else if t != nil {
		params.Start = *t
	}

	if t, err := h.TimestampQueryParameter("end"); err != nil {
		return nil, err
	} else if t != nil {
		params.End = *t
	}

	granularity := h.QueryParameter("granularity")
	switch granularity {
	case "":
		params.Granularity = eventline.MetricGranularityDay
	case "day":
		params.Granularity = eventline.MetricGranularityDay
	case "hour":
		params.Granularity = eventline.MetricGranularityHour

	default:
		err := fmt.Errorf("invalid granularity %q", granularity)
		h.ReplyError(400, "invalid_query_parameter", "%v", err)
		return nil, err
	}

	return &params, nil
}
