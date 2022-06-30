package service

import (
	cgithub "github.com/exograd/eventline/pkg/connectors/github"
	"github.com/exograd/eventline/pkg/eventline"
	"github.com/google/go-github/v45/github"
)

func (s *WebHTTPServer) setupExternalRoutes() {
	s.route("/ext/connectors/generic/oauth2", "GET",
		s.hExtConnectorsGenericOAuth2GET,
		HTTPRouteOptions{Public: true})

	s.route("/ext/connectors/github/oauth2", "GET",
		s.hExtConnectorsGithubOAuth2GET,
		HTTPRouteOptions{Public: true})

	s.route("/ext/connectors/github/hooks/*", "POST",
		s.hExtConnectorsGithubHooksPOST,
		HTTPRouteOptions{Public: true})
}

func (s *WebHTTPServer) hExtConnectorsGenericOAuth2GET(h *HTTPHandler) {
	s.processOAuth2Request(h)
}

func (s *WebHTTPServer) hExtConnectorsGithubOAuth2GET(h *HTTPHandler) {
	s.processOAuth2Request(h)
}

func (s *WebHTTPServer) hExtConnectorsGithubHooksPOST(h *HTTPHandler) {
	if deliveryId := github.DeliveryID(h.Request); deliveryId != "" {
		h.Log.Data["github_delivery_id"] = deliveryId
	}

	target := h.RouteVariable("*")

	var params cgithub.Parameters
	params.ParseTarget(target)

	c := eventline.GetConnector("github")
	c2 := c.(*cgithub.Connector)

	if err := c2.ProcessWebhookRequest(h.Request, &params); err != nil {
		s.Log.Error("cannot process request: %v", err)
	}

	h.ReplyEmpty(204)
}
