package emptybackendhandler

import (
	"net/http"

	"github.com/traefik/traefik/v2/pkg/healthcheck"
)

// EmptyBackend is a middleware that checks whether the current Backend
// has at least one active Server in respect to the healthchecks and if this
// is not the case, it will stop the middleware chain and respond with 503.
type emptyBackend struct {
	healthcheck.BalancerStatusHandler
}

// New creates a new EmptyBackend middleware.
func New(lb healthcheck.BalancerStatusHandler) http.Handler {
	return &emptyBackend{BalancerStatusHandler: lb}
}

// ServeHTTP responds with 503 when there is no active Server and otherwise
// invokes the next handler in the middleware chain.
func (e *emptyBackend) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if len(e.BalancerStatusHandler.Servers()) != 0 {
		e.BalancerStatusHandler.ServeHTTP(rw, req)
		return
	}

	rw.WriteHeader(http.StatusServiceUnavailable)
	if _, err := rw.Write([]byte(http.StatusText(http.StatusServiceUnavailable))); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}
