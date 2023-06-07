package service

import (
	"github.com/exograd/eventline/pkg/eventline"
)

func (s *APIHTTPServer) setupProjectRoutes() {
	s.route("/projects", "GET", s.hProjectsGET,
		HTTPRouteOptions{})

	s.route("/projects", "POST", s.hProjectsPOST,
		HTTPRouteOptions{Admin: true})

	s.route("/projects/id/:id", "GET", s.hProjectsIdGET,
		HTTPRouteOptions{})

	s.route("/projects/name/:name", "GET", s.hProjectsNameGET,
		HTTPRouteOptions{})

	s.route("/projects/id/:id", "PUT", s.hProjectsIdPUT,
		HTTPRouteOptions{Admin: true})

	s.route("/projects/id/:id", "DELETE", s.hProjectsIdDELETE,
		HTTPRouteOptions{Admin: true})
}

func (s *APIHTTPServer) hProjectsGET(h *HTTPHandler) {
	page, err := s.LoadProjectPage(h)
	if err != nil {
		return
	}

	h.ReplyJSON(200, page)
}

func (s *APIHTTPServer) hProjectsPOST(h *HTTPHandler) {
	var newProject eventline.NewProject
	if err := h.JSONRequestObject(&newProject); err != nil {
		return
	}

	project, err := s.CreateProject(h, &newProject)
	if err != nil {
		return
	}

	h.ReplyJSON(201, project)
}

func (s *APIHTTPServer) hProjectsIdGET(h *HTTPHandler) {
	projectId, err := h.IdPathVariable("id")
	if err != nil {
		return
	}

	project, err := s.LoadProject(h, projectId)
	if err != nil {
		return
	}

	h.ReplyJSON(200, project)
}

func (s *APIHTTPServer) hProjectsNameGET(h *HTTPHandler) {
	projectName := h.PathVariable("name")

	project, err := s.LoadProjectByName(h, projectName)
	if err != nil {
		return
	}

	h.ReplyJSON(200, project)
}

func (s *APIHTTPServer) hProjectsIdPUT(h *HTTPHandler) {
	projectId, err := h.IdPathVariable("id")
	if err != nil {
		return
	}

	var newProject eventline.NewProject
	if err := h.JSONRequestObject(&newProject); err != nil {
		return
	}

	project, err := s.UpdateProject(h, projectId, &newProject)
	if err != nil {
		return
	}

	h.ReplyJSON(200, project)
}

func (s *APIHTTPServer) hProjectsIdDELETE(h *HTTPHandler) {
	projectId, err := h.IdPathVariable("id")
	if err != nil {
		return
	}

	if err := s.DeleteProject(h, projectId); err != nil {
		return
	}

	h.ReplyEmpty(204)
}
