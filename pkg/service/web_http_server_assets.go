package service

import (
	"net/http"
	"os"
	"path"
	"regexp"
)

var assetCacheBustingRE = regexp.MustCompile(
	`^(.+)\.(?:[a-z0-9]+)\.(css|js|png)$`)

func (s *WebHTTPServer) setupAssetRoutes() {
	s.route("/favicon.ico", "GET",
		s.hFaviconGET,
		HTTPRouteOptions{Public: true})

	s.route("/assets/*", "GET",
		s.hAssetsGET,
		HTTPRouteOptions{Public: true})
}

func (s *WebHTTPServer) hFaviconGET(h *HTTPHandler) {
	h.ReplyEmpty(204)
}

func (s *WebHTTPServer) hAssetsGET(h *HTTPHandler) {
	// TODO etag support
	//
	// If the caller has set w's ETag header formatted per RFC 7232, section
	// 2.3, ServeContent uses it to handle requests using If-Match,
	// If-None-Match, or If-Rang

	subpath := h.RouteVariable("*")
	subpath = rewriteAssetPath(subpath)

	dirPath := path.Join(s.Service.Cfg.DataDirectory, "assets")
	filePath := path.Join(dirPath, subpath)

	info, err := os.Stat(filePath)
	if err != nil {
		h.ReplyInternalError(500, "cannot stat %q: %v", filePath, err)
		return
	}

	if !info.Mode().IsRegular() {
		h.ReplyInternalError(400, "%q is not a regular file", subpath)
		return
	}

	modTime := info.ModTime()

	body, err := os.Open(filePath)
	if err != nil {
		h.ReplyInternalError(500, "cannot open %q: %v", filePath, err)
	}
	defer body.Close()

	http.ServeContent(h.ResponseWriter, h.Request, subpath, modTime, body)
}

func rewriteAssetPath(p string) string {
	matches := assetCacheBustingRE.FindAllStringSubmatch(p, -1)
	if len(matches) < 1 {
		return p
	}

	groups := matches[0][1:]
	return groups[0] + "." + groups[1]
}
