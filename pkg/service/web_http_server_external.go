package service

import (
	"path"

	cgithub "github.com/exograd/eventline/pkg/connectors/github"
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/google/go-github/v45/github"
)

func (s *WebHTTPServer) setupExternalRoutes() {
	for _, c := range s.Service.Data.Connectors {
		for iname, idef := range c.Definition().Identities {
			if !idef.IsOAuth2() {
				continue
			}

			s.route(path.Join("/ext", "connectors", c.Name(), iname), "GET",
				func(h *HTTPHandler) {
					s.processOAuth2Request(h)
				},
				HTTPRouteOptions{Public: true})
		}
	}

	s.route("/ext/connectors/github/hooks/*subpath", "POST",
		s.hExtConnectorsGithubHooksPOST,
		HTTPRouteOptions{Public: true})
}

func (s *WebHTTPServer) hExtConnectorsGithubHooksPOST(h *HTTPHandler) {
	if deliveryId := github.DeliveryID(h.Request); deliveryId != "" {
		h.Log.Data["github_delivery_id"] = deliveryId
	}

	target := h.PathVariable("subpath")

	var params cgithub.Parameters
	params.ParseTarget(target)

	c := eventline.GetConnector("github")
	c2 := c.(*cgithub.Connector)

	if err := c2.ProcessWebhookRequest(h.Request, &params); err != nil {
		h.Log.Error("cannot process request: %v", err)
	}

	h.ReplyEmpty(204)
}
