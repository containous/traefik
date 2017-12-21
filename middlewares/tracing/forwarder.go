package tracing

import (
	"fmt"
	"net/http"

	"github.com/containous/traefik/log"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/urfave/negroni"
)

type fwdMiddleware struct {
	frontend string
	backend  string
	opName   string
	*Tracing
}

// NewForwarder creates a new forwarder middleware that the final outgoing request
func (t *Tracing) NewForwarder(frontend, backend string) negroni.Handler {
	log.Debugf("Added outgoing tracing middleware %s", frontend)
	return &fwdMiddleware{
		Tracing:  t,
		frontend: frontend,
		backend:  backend,
		opName:   fmt.Sprintf("forward %s/%s", frontend, backend),
	}
}

func (f *fwdMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	span, r, finish := StartSpan(r, f.opName, true)
	defer finish()
	span.SetTag("frontend.name", f.frontend)
	span.SetTag("backend.name", f.backend)
	ext.HTTPMethod.Set(span, r.Method)
	ext.HTTPUrl.Set(span, r.URL.String())
	span.SetTag("http.host", r.Host)

	InjectHeadersInRequest(r)

	w = &statusCodeTracker{w, 200}

	next(w, r)

	LogResponseCode(span, w.(*statusCodeTracker).status)
}
