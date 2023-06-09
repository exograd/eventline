package service

import (
	"path"
)

func (s *WebHTTPServer) setupAssetRoutes() {
	s.route("/favicon.ico", "GET",
		s.hFaviconGET,
		HTTPRouteOptions{Public: true})

	s.route("/assets/*subpath", "GET",
		s.hAssetsGET,
		HTTPRouteOptions{Public: true})
}

func (s *WebHTTPServer) hFaviconGET(h *HTTPHandler) {
	h.ReplyEmpty(204)
}

func (s *WebHTTPServer) hAssetsGET(h *HTTPHandler) {
	subpath := h.PathVariable("subpath")

	dirPath := path.Join(s.Service.Cfg.DataDirectory, "assets")
	filePath := path.Join(dirPath, subpath)

	h.ReplyFile(filePath)
}
