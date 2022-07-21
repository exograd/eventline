package service

func (s *APIHTTPServer) setupJobExecutionRoutes() {
	s.route("/job_executions/id/{id}", "GET", s.hJobExecutionsIdGET,
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
