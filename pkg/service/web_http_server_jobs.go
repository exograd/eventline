package service

import (
	"errors"
	"fmt"

	"github.com/exograd/eventline/pkg/eventline"
	"github.com/exograd/eventline/pkg/utils"
	"github.com/exograd/eventline/pkg/web"
	"github.com/exograd/go-daemon/pg"
)

func (s *WebHTTPServer) setupJobRoutes() {
	s.route("/jobs", "GET",
		s.hJobsGET,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/{id}", "GET",
		s.hJobsIdGET,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/{id}/timeline", "GET",
		s.hJobsIdTimelineGET,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/{id}/delete", "POST",
		s.hJobsIdDeletePOST,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/{id}/enable", "POST",
		s.hJobsIdEnablePOST,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/{id}/disable", "POST",
		s.hJobsIdDisablePOST,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/{id}/add_favourite", "POST",
		s.hJobsIdAddFavouritePOST,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/{id}/remove_favourite", "POST",
		s.hJobsIdRemoveFavouritePOST,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/{id}/execute", "GET",
		s.hJobsIdExecuteGET,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/{id}/execute", "POST",
		s.hJobsIdExecutePOST,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/{id}/definition", "GET",
		s.hJobsIdDefinitionGET,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/{id}/metrics", "GET",
		s.hJobsIdMetricsGET,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/{id}/metrics/status_counts", "GET",
		s.hJobsIdMetricsStatusCountsGET,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/{id}/metrics/running_times", "GET",
		s.hJobsIdMetricsRunningTimesGET,
		HTTPRouteOptions{Project: true})

	s.route("/jobs/id/{id}/metrics", "GET",
		s.hJobsIdMetricsGET,
		HTTPRouteOptions{Project: true})
}

func (s *WebHTTPServer) hJobsGET(h *HTTPHandler) {
	scope := h.Context.ProjectScope()
	accountProjectScope := h.Context.AccountProjectScope()

	cursor, err := h.ParseCursor(eventline.JobSorts)
	if err != nil {
		return
	}

	var page *eventline.Page
	var lastJobExecutions map[eventline.Id]*eventline.JobExecution
	var jobStats map[eventline.Id]*eventline.JobStats
	var favouriteJobTable = map[eventline.Id]bool{}

	err = s.Pg.WithConn(func(conn pg.Conn) error {
		var err error

		pageOptions := eventline.JobPageOptions{
			ExcludeFavouriteJobAccountId: h.Context.AccountId,
		}

		page, err = eventline.LoadJobPage(conn, pageOptions, cursor, scope)
		if err != nil {
			return fmt.Errorf("cannot load jobs: %w", err)
		}

		favouriteJobs, err := eventline.LoadFavouriteJobs(conn,
			accountProjectScope)
		if err != nil {
			return fmt.Errorf("cannot load favourite jobs: %w", err)
		}

		var pageElements []eventline.PageElement
		for _, j := range favouriteJobs {
			pageElements = append(pageElements, j)
			favouriteJobTable[j.Id] = true
		}
		for _, e := range page.Elements {
			pageElements = append(pageElements, e)
		}
		page.Elements = pageElements

		jobIds := make(eventline.Ids, len(page.Elements))
		for i, e := range page.Elements {
			jobIds[i] = e.(*eventline.Job).Id
		}

		lastJobExecutions, err = eventline.LoadLastJobExecutions(conn, jobIds,
			scope)
		if err != nil {
			return fmt.Errorf("cannot load job executions: %w", err)
		}

		jobStats, err = eventline.LoadJobStats(conn, jobIds, scope)
		if err != nil {
			return fmt.Errorf("cannot load job stats: %w", err)
		}

		return nil
	})
	if err != nil {
		h.ReplyInternalError(500, "%v", err)
		return
	}

	bodyData := struct {
		Page              *eventline.Page
		LastJobExecutions map[eventline.Id]*eventline.JobExecution
		JobStats          map[eventline.Id]*eventline.JobStats
		FavouriteJobTable map[eventline.Id]bool
	}{
		Page:              page,
		LastJobExecutions: lastJobExecutions,
		JobStats:          jobStats,
		FavouriteJobTable: favouriteJobTable,
	}

	h.ReplyView(200, &web.View{
		Title:      "Jobs",
		Menu:       NewMainMenu("jobs"),
		Breadcrumb: jobsBreadcrumb(),
		Body:       s.NewTemplate("jobs.html", bodyData),
	})
}

func (s *WebHTTPServer) hJobsIdGET(h *HTTPHandler) {
	jobId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	h.ReplyRedirect(302, "/jobs/id/"+jobId.String()+"/timeline")
}

func (s *WebHTTPServer) hJobsIdTimelineGET(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	jobId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	cursor, err := h.ParseCursor(eventline.JobExecutionSorts)
	if err != nil {
		return
	}
	if cursor.Order == "" {
		cursor.Order = eventline.OrderDesc
	}

	var page *eventline.Page
	var job eventline.Job

	err = s.Pg.WithConn(func(conn pg.Conn) error {
		var err error

		if err := job.Load(conn, jobId, scope); err != nil {
			return fmt.Errorf("cannot load job: %w", err)
		}

		pageOptions := eventline.JobExecutionPageOptions{
			JobId: &jobId,
		}

		page, err = eventline.LoadJobExecutionPage(conn, pageOptions, cursor,
			scope)
		if err != nil {
			return fmt.Errorf("cannot load job executions: %w", err)
		}

		return nil
	})
	if err != nil {
		var unknownJobErr *eventline.UnknownJobError

		if errors.As(err, &unknownJobErr) {
			h.ReplyError(404, "unknown_job", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return
	}

	bodyData := struct {
		Page *eventline.Page
	}{
		Page: page,
	}

	h.ReplyView(200, &web.View{
		Title:      "Job executions",
		Menu:       NewMainMenu("jobs"),
		Breadcrumb: jobBreadcrumb(&job),
		Tabs:       jobTabs(&job, "timeline"),
		Body:       s.NewTemplate("job_timeline.html", bodyData),
	})
}

func (s *WebHTTPServer) hJobsIdDeletePOST(h *HTTPHandler) {
	jobId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	if err := s.DeleteJob(h, jobId); err != nil {
		return
	}

	h.ReplyJSONLocation(200, "/jobs", nil)
}

func (s *WebHTTPServer) hJobsIdEnablePOST(h *HTTPHandler) {
	jobId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	if err := s.EnableJob(h, jobId); err != nil {
		return
	}

	h.ReplyEmpty(204)
}

func (s *WebHTTPServer) hJobsIdDisablePOST(h *HTTPHandler) {
	jobId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	if err := s.DisableJob(h, jobId); err != nil {
		return
	}

	h.ReplyEmpty(204)
}

func (s *WebHTTPServer) hJobsIdAddFavouritePOST(h *HTTPHandler) {
	scope := h.Context.AccountProjectScope()

	jobId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	err = s.Service.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		if err := s.Service.AddFavouriteJob(conn, jobId, scope); err != nil {
			return fmt.Errorf("cannot add favourite job: %w", err)
		}

		return nil
	})
	if err != nil {
		var unknownJobErr *eventline.UnknownJobError

		if errors.As(err, &unknownJobErr) {
			h.ReplyError(404, "unknown_job", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return
	}

	h.ReplyEmpty(204)
}

func (s *WebHTTPServer) hJobsIdRemoveFavouritePOST(h *HTTPHandler) {
	scope := h.Context.AccountProjectScope()

	jobId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	err = s.Service.Daemon.Pg.WithTx(func(conn pg.Conn) error {
		if err := s.Service.RemoveFavouriteJob(conn, jobId, scope); err != nil {
			return fmt.Errorf("cannot remove favourite job: %w", err)
		}

		return nil
	})
	if err != nil {
		var unknownJobErr *eventline.UnknownJobError

		if errors.As(err, &unknownJobErr) {
			h.ReplyError(404, "unknown_job", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return
	}

	h.ReplyEmpty(204)
}

func (s *WebHTTPServer) hJobsIdExecuteGET(h *HTTPHandler) {
	jobId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	job, err := s.LoadJob(h, jobId)
	if err != nil {
		return
	}

	bodyData := struct {
		Job *eventline.Job
	}{
		Job: job,
	}

	breadcrumb := jobBreadcrumb(job)
	breadcrumb.AddEntry(&web.BreadcrumbEntry{Label: "Execute"})

	h.ReplyView(200, &web.View{
		Title:      "Job execution",
		Menu:       NewMainMenu("jobs"),
		Breadcrumb: breadcrumb,
		Tabs:       jobTabs(job, "execution"),
		Body:       s.NewTemplate("job_execution.html", bodyData),
	})
}

func (s *WebHTTPServer) hJobsIdExecutePOST(h *HTTPHandler) {
	jobId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	var input eventline.JobExecutionInput
	if err := h.JSONRequestData(&input); err != nil {
		return
	}

	jobExecution, err := s.ExecuteJob(h, jobId, &input)
	if err != nil {
		return
	}

	location := "/job_executions/id/" + jobExecution.Id.String()

	extra := map[string]interface{}{
		"job_execution_id": jobExecution.Id.String(),
	}

	h.ReplyJSONLocation(201, location, extra)
}

func (s *WebHTTPServer) hJobsIdDefinitionGET(h *HTTPHandler) {
	jobId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	job, err := s.LoadJob(h, jobId)
	if err != nil {
		return
	}

	jobSpecData, err := utils.YAMLEncode(job.Spec)
	if err != nil {
		h.ReplyInternalError(500, "cannot encode job specification data: %v",
			err)
		return
	}

	bodyData := struct {
		JobSpecData string
	}{
		JobSpecData: string(jobSpecData),
	}

	breadcrumb := jobBreadcrumb(job)
	breadcrumb.AddEntry(&web.BreadcrumbEntry{Label: "Definition"})

	h.ReplyView(200, &web.View{
		Title:      "Job definition",
		Menu:       NewMainMenu("jobs"),
		Breadcrumb: breadcrumb,
		Tabs:       jobTabs(job, "definition"),
		Body:       s.NewTemplate("job_definition.html", bodyData),
	})
}

func (s *WebHTTPServer) hJobsIdMetricsGET(h *HTTPHandler) {
	jobId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	job, err := s.LoadJob(h, jobId)
	if err != nil {
		return
	}

	bodyData := struct {
		Job *eventline.Job
	}{
		Job: job,
	}

	breadcrumb := jobBreadcrumb(job)
	breadcrumb.AddEntry(&web.BreadcrumbEntry{Label: "Metrics"})

	h.ReplyView(200, &web.View{
		Title:      "Job metrics",
		Menu:       NewMainMenu("jobs"),
		Breadcrumb: breadcrumb,
		Tabs:       jobTabs(job, "metrics"),
		Body:       s.NewTemplate("job_metrics.html", bodyData),
	})
}

func (s *WebHTTPServer) hJobsIdMetricsStatusCountsGET(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	jobId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	params, err := h.ParseMetricParameters()
	if err != nil {
		return
	}

	var job eventline.Job
	var points eventline.MetricPoints

	err = s.Pg.WithConn(func(conn pg.Conn) error {
		var err error

		if err = job.Load(conn, jobId, scope); err != nil {
			return fmt.Errorf("cannot load job: %w", err)
		}

		points, err = job.LoadStatusCounts(conn, params)
		if err != nil {
			return fmt.Errorf("cannot load metrics: %w", err)
		}

		return nil
	})
	if err != nil {
		var unknownJobErr *eventline.UnknownJobError

		if errors.As(err, &unknownJobErr) {
			h.ReplyError(404, "unknown_job", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return
	}

	h.ReplyCompactJSON(200, points)
}

func (s *WebHTTPServer) hJobsIdMetricsRunningTimesGET(h *HTTPHandler) {
	scope := h.Context.ProjectScope()

	jobId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	params, err := h.ParseMetricParameters()
	if err != nil {
		return
	}

	var job eventline.Job
	var points eventline.MetricPoints

	err = s.Pg.WithConn(func(conn pg.Conn) error {
		var err error

		if err = job.Load(conn, jobId, scope); err != nil {
			return fmt.Errorf("cannot load job: %w", err)
		}

		points, err = job.LoadRunningTimes(conn, params)
		if err != nil {
			return fmt.Errorf("cannot load metrics: %w", err)
		}

		return nil
	})
	if err != nil {
		var unknownJobErr *eventline.UnknownJobError

		if errors.As(err, &unknownJobErr) {
			h.ReplyError(404, "unknown_job", "%v", err)
		} else {
			h.ReplyInternalError(500, "%v", err)
		}

		return
	}

	h.ReplyCompactJSON(200, points)
}

func jobsBreadcrumb() *web.Breadcrumb {
	breadcrumb := web.NewBreadcrumb()

	breadcrumb.AddEntry(&web.BreadcrumbEntry{
		Label: "Jobs",
		URI:   "/jobs",
	})

	return breadcrumb
}

func jobBreadcrumb(job *eventline.Job) *web.Breadcrumb {
	breadcrumb := jobsBreadcrumb()

	breadcrumb.AddEntry(&web.BreadcrumbEntry{
		Label: job.Spec.Name,
		URI:   "/jobs/id/" + job.Id.String(),
	})

	return breadcrumb
}

func jobTabs(job *eventline.Job, selectedTab string) *web.Tabs {
	tabs := web.NewTabs()
	tabs.SelectedTab = selectedTab

	baseURI := "/jobs/id/" + job.Id.String()

	tabs.AddTab(&web.Tab{
		Id:    "timeline",
		Icon:  "play-box-multiple-outline",
		Label: "Timeline",
		URI:   baseURI + "/timeline",
	})

	tabs.AddTab(&web.Tab{
		Id:    "execution",
		Icon:  "play-box-outline",
		Label: "Execution",
		URI:   baseURI + "/execute",
	})

	tabs.AddTab(&web.Tab{
		Id:    "definition",
		Icon:  "file-code-outline",
		Label: "Definition",
		URI:   baseURI + "/definition",
	})

	tabs.AddTab(&web.Tab{
		Id:    "metrics",
		Icon:  "chart-timeline-variant",
		Label: "Metrics",
		URI:   baseURI + "/metrics",
	})

	return tabs
}
