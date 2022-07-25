package service

func (s *APIHTTPServer) setupJobExecutionRoutes() {
	s.route("/job_executions/id/{id}", "GET", s.hJobExecutionsIdGET,
		HTTPRouteOptions{Project: true})

	s.route("/job_executions/id/{id}/abort", "POST",
		s.hJobExecutionsIdAbortPOST,
		HTTPRouteOptions{Project: true})

	s.route("/job_executions/id/{id}/restart", "POST",
		s.hJobExecutionsIdRestartPOST,
		HTTPRouteOptions{Project: true})
}

func (s *APIHTTPServer) hJobExecutionsIdGET(h *HTTPHandler) {
	jeId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	je, err := s.LoadJobExecution(h, jeId)
	if err != nil {
		return
	}

	h.ReplyJSON(200, je)
}

func (s *APIHTTPServer) hJobExecutionsIdAbortPOST(h *HTTPHandler) {
	jeId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	if err := s.AbortJobExecution(h, jeId); err != nil {
		return
	}

	h.ReplyEmpty(204)
}

func (s *APIHTTPServer) hJobExecutionsIdRestartPOST(h *HTTPHandler) {
	jeId, err := h.IdRouteVariable("id")
	if err != nil {
		return
	}

	if err := s.RestartJobExecution(h, jeId); err != nil {
		return
	}

	h.ReplyEmpty(204)
}
