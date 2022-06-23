package service

func (s *APIHTTPServer) setupAccountRoutes() {
	s.route("/account", "GET", s.hAccountGET,
		HTTPRouteOptions{})
}

func (s *APIHTTPServer) hAccountGET(h *HTTPHandler) {
	account, err := s.LoadAccount(h)
	if err != nil {
		return
	}

	h.ReplyJSON(200, account)
}
