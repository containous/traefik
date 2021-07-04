package mirror

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/healthcheck"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/middlewares/accesslog"
	"github.com/traefik/traefik/v2/pkg/safe"
)

// Mirroring is an http.Handler that can mirror requests.
type Mirroring struct {
	handler        http.Handler
	mirrorHandlers []*mirrorHandler
	rw             http.ResponseWriter
	routinePool    *safe.Pool

	maxBodySize      int64
	wantsHealthCheck bool

	lock  sync.RWMutex
	total uint64
}

// New returns a new instance of *Mirroring.
func New(handler http.Handler, pool *safe.Pool, maxBodySize int64, hc *dynamic.HealthCheck) *Mirroring {
	return &Mirroring{
		routinePool:      pool,
		handler:          handler,
		rw:               blackHoleResponseWriter{},
		maxBodySize:      maxBodySize,
		wantsHealthCheck: hc != nil,
	}
}

func (m *Mirroring) inc() uint64 {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.total++
	return m.total
}

type mirrorHandler struct {
	http.Handler
	percent int

	lock  sync.RWMutex
	count uint64
}

func (m *Mirroring) getActiveMirrors() []http.Handler {
	total := m.inc()

	var mirrors []http.Handler
	for _, handler := range m.mirrorHandlers {
		handler.lock.Lock()
		if handler.count*100 < total*uint64(handler.percent) {
			handler.count++
			handler.lock.Unlock()
			mirrors = append(mirrors, handler)
		} else {
			handler.lock.Unlock()
		}
	}
	return mirrors
}

func (m *Mirroring) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	mirrors := m.getActiveMirrors()
	if len(mirrors) == 0 {
		m.handler.ServeHTTP(rw, req)
		return
	}

	logger := log.FromContext(req.Context())
	rr, bytesRead, err := newReusableRequest(req, m.maxBodySize)
	if err != nil && !errors.Is(err, errBodyTooLarge) {
		http.Error(rw, http.StatusText(http.StatusInternalServerError)+
			fmt.Sprintf("error creating reusable request: %v", err), http.StatusInternalServerError)
		return
	}

	if errors.Is(err, errBodyTooLarge) {
		req.Body = io.NopCloser(io.MultiReader(bytes.NewReader(bytesRead), req.Body))
		m.handler.ServeHTTP(rw, req)
		logger.Debugf("no mirroring, request body larger than allowed size")
		return
	}

	m.handler.ServeHTTP(rw, rr.clone(req.Context()))

	select {
	case <-req.Context().Done():
		// No mirroring if request has been canceled during main handler ServeHTTP
		logger.Warn("no mirroring, request has been canceled during main handler ServeHTTP")
		return
	default:
	}

	m.routinePool.GoCtx(func(_ context.Context) {
		for _, handler := range mirrors {
			// prepare request, update body from buffer
			r := rr.clone(req.Context())

			// In ServeHTTP, we rely on the presence of the accessLog datatable found in the request's context
			// to know whether we should mutate said datatable (and contribute some fields to the log).
			// In this instance, we do not want the mirrors mutating (i.e. changing the service name in)
			// the logs related to the mirrored server.
			// Especially since it would result in unguarded concurrent reads/writes on the datatable.
			// Therefore, we reset any potential datatable key in the new context that we pass around.
			ctx := context.WithValue(r.Context(), accesslog.DataTableKey, nil)

			// When a request served by m.handler is successful, req.Context will be canceled,
			// which would trigger a cancellation of the ongoing mirrored requests.
			// Therefore, we give a new, non-cancellable context  to each of the mirrored calls,
			// so they can terminate by themselves.
			handler.ServeHTTP(m.rw, r.WithContext(contextStopPropagation{ctx}))
		}
	})
}

// AddMirror adds an httpHandler to mirror to.
func (m *Mirroring) AddMirror(handler http.Handler, percent int) error {
	if percent < 0 || percent > 100 {
		return errors.New("percent must be between 0 and 100")
	}
	m.mirrorHandlers = append(m.mirrorHandlers, &mirrorHandler{Handler: handler, percent: percent})
	return nil
}

// RegisterStatusUpdater adds fn to the list of hooks that are run when the
// status of handler of the Mirroring changes.
// Not thread safe.
func (m *Mirroring) RegisterStatusUpdater(fn func(up bool)) error {
	// Since the status propagation is completely transparent through the
	// mirroring (because of the recursion on the underlying service), we could maybe
	// skip that below, and even not add HealthCheck as a field of
	// dynamic.Mirroring. But I think it's easier to understand for the user
	// if the HealthCheck is required absolutely everywhere in the config.
	if !m.wantsHealthCheck {
		return errors.New("healthCheck not enabled in config for this mirroring service")
	}

	updater, ok := m.handler.(healthcheck.StatusUpdater)
	if !ok {
		return fmt.Errorf("service of mirroring %T not a healthcheck.StatusUpdater", m.handler)
	}

	if err := updater.RegisterStatusUpdater(fn); err != nil {
		return fmt.Errorf("cannot register service of mirroring as updater: %w", err)
	}

	return nil
}

type blackHoleResponseWriter struct{}

func (b blackHoleResponseWriter) Flush() {}

func (b blackHoleResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, errors.New("connection on blackHoleResponseWriter cannot be hijacked")
}

func (b blackHoleResponseWriter) Header() http.Header {
	return http.Header{}
}

func (b blackHoleResponseWriter) Write(data []byte) (int, error) {
	return len(data), nil
}

func (b blackHoleResponseWriter) WriteHeader(statusCode int) {}

type contextStopPropagation struct {
	context.Context
}

func (c contextStopPropagation) Done() <-chan struct{} {
	return make(chan struct{})
}

// reusableRequest keeps in memory the body of the given request,
// so that the request can be fully cloned by each mirror.
type reusableRequest struct {
	req  *http.Request
	body []byte
}

var errBodyTooLarge = errors.New("request body too large")

// if the returned error is errBodyTooLarge, newReusableRequest also returns the
// bytes that were already consumed from the request's body.
func newReusableRequest(req *http.Request, maxBodySize int64) (*reusableRequest, []byte, error) {
	if req == nil {
		return nil, nil, errors.New("nil input request")
	}
	if req.Body == nil {
		return &reusableRequest{req: req}, nil, nil
	}

	// unbounded body size
	if maxBodySize < 0 {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, nil, err
		}
		return &reusableRequest{
			req:  req,
			body: body,
		}, nil, nil
	}

	// we purposefully try to read _more_ than maxBodySize to detect whether
	// the request body is larger than what we allow for the mirrors.
	body := make([]byte, maxBodySize+1)
	n, err := io.ReadFull(req.Body, body)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, nil, err
	}

	// we got an ErrUnexpectedEOF, which means there was less than maxBodySize data to read,
	// which permits us sending also to all the mirrors later.
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return &reusableRequest{
			req:  req,
			body: body[:n],
		}, nil, nil
	}

	// err == nil , which means data size > maxBodySize
	return nil, body[:n], errBodyTooLarge
}

func (rr reusableRequest) clone(ctx context.Context) *http.Request {
	req := rr.req.Clone(ctx)

	if rr.body != nil {
		req.Body = io.NopCloser(bytes.NewReader(rr.body))
	}

	return req
}
