package reroute

import (
	"context"
	"net/http"
	"strings"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/log"
)

const (
	typeName = "Reroute"
)

// Reroute is a component to make outgoing SPNEGO calls
type Reroute struct {
	next   http.Handler
	config *dynamic.RerouteService
}

// New cretes a Reroute service
func New(ctx context.Context, next http.Handler, service *dynamic.RerouteService, name string) (http.Handler, error) {
	logger := log.FromContext(ctx)
	logger.Debug("Creating Reroute service")

	var err error

	reroute := &Reroute{
		next:   next,
		config: service,
	}

	return reroute, err
}

func (r *Reroute) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	if len(path) > 0 && path[0] != '/' {
		path = "/" + path
	}
	comps := strings.Split(req.URL.Path, "/")

	if len(r.config.Scheme) > 0 {
		req.URL.Scheme = r.config.Scheme
	} else {
		req.URL.Scheme = "HTTP"
	}

	// TargetHostSegment is an index of segments when you split URL by '/'.
	// 1st segment is 1.
	// if router's rule is "PathPrefix(`/spnegohttp/`)" and
	// you have a target domain name in the following segment,
	// TargetHostSegment should be set to 2.
	// when you make a request to http://traefikhost:port/spnegohttp/foo.com:12345/a/b/c,
	// it'll redirect the traffic to foo.com:12345 with URL Path = /a/b/c
	// If spnegoOut is defined in loadBalancer service, taergetHostSegment should be 0
	// to skip target host name override.
	if r.config.TargetHostSegment > 0 {
		req.URL.Host = comps[r.config.TargetHostSegment]
		req.URL.Path = "/" + strings.Join(comps[r.config.TargetHostSegment+1:], "/")
	}
	req.Host = req.URL.Host
	req.RequestURI = req.URL.Path

	r.next.ServeHTTP(rw, req)
}
