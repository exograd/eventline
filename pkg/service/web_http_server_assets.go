package service

import (
	"path"
	"strings"
)

func (s *WebHTTPServer) setupAssetRoutes() {
	s.route("/favicon.ico", "GET",
		s.hFaviconGET,
		HTTPRouteOptions{Public: true})

	s.route("/assets/", "GET",
		s.hAssetsGET,
		HTTPRouteOptions{Public: true})
}

func (s *WebHTTPServer) hFaviconGET(h *HTTPHandler) {
	h.ReplyEmpty(204)
}

func (s *WebHTTPServer) hAssetsGET(h *HTTPHandler) {
	subpath := strings.TrimPrefix(h.Request.URL.Path, "/assets/")

	dirPath := path.Join(s.Service.Cfg.DataDirectory, "assets")
	filePath := path.Join(dirPath, subpath)

	h.ReplyFile(filePath)
}
