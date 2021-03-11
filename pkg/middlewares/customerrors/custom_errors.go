package customerrors

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/opentracing/opentracing-go/ext"
	"github.com/sirupsen/logrus"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/middlewares"
	"github.com/traefik/traefik/v2/pkg/tracing"
	"github.com/traefik/traefik/v2/pkg/types"
	"github.com/vulcand/oxy/utils"
)

// Compile time validation that the response recorder implements http interfaces correctly.
var (
	_ middlewares.Stateful = &responseRecorderWithCloseNotify{}
	_ middlewares.Stateful = &codeCatcherWithCloseNotify{}
)

const (
	typeName   = "customError"
	backendURL = "http://0.0.0.0"
)

type serviceBuilder interface {
	BuildHTTP(ctx context.Context, serviceName string) (http.Handler, error)
}

// customErrors is a middleware that provides the custom error pages..
type customErrors struct {
	name           string
	next           http.Handler
	backendHandler http.Handler
	httpCodeRanges types.HTTPCodeRanges
	backendQuery   string
}

// New creates a new custom error pages middleware.
func New(ctx context.Context, next http.Handler, config dynamic.ErrorPage, serviceBuilder serviceBuilder, name string) (http.Handler, error) {
	log.FromContext(middlewares.GetLoggerCtx(ctx, name, typeName)).Debug("Creating middleware")

	httpCodeRanges, err := types.NewHTTPCodeRanges(config.Status)
	if err != nil {
		return nil, err
	}

	backend, err := serviceBuilder.BuildHTTP(ctx, config.Service)
	if err != nil {
		return nil, err
	}

	return &customErrors{
		name:           name,
		next:           next,
		backendHandler: backend,
		httpCodeRanges: httpCodeRanges,
		backendQuery:   config.Query,
	}, nil
}

func (c *customErrors) GetTracingInformation() (string, ext.SpanKindEnum) {
	return c.name, tracing.SpanKindNoneEnum
}

func (c *customErrors) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	ctx := middlewares.GetLoggerCtx(req.Context(), c.name, typeName)
	logger := log.FromContext(ctx)

	if c.backendHandler == nil {
		logger.Error("Error pages: no backend handler.")
		tracing.SetErrorWithEvent(req, "Error pages: no backend handler.")
		c.next.ServeHTTP(rw, req)
		return
	}

	catcher := newCodeCatcher(rw, c.httpCodeRanges)
	c.next.ServeHTTP(catcher, req)
	if !catcher.isFilteredCode() {
		return
	}

	// check the recorder code against the configured http status code ranges
	code := catcher.getCode()
	for _, block := range c.httpCodeRanges {
		if code < block[0] || code > block[1] {
			continue
		}

		logger.Debugf("Caught HTTP Status Code %d, returning error page", code)

		var query string
		if len(c.backendQuery) > 0 {
			query = "/" + strings.TrimPrefix(c.backendQuery, "/")
			query = strings.ReplaceAll(query, "{status}", strconv.Itoa(code))
		}

		pageReq, err := newRequest(backendURL + query)
		if err != nil {
			logger.Error(err)
			rw.WriteHeader(code)
			_, err = fmt.Fprint(rw, http.StatusText(code))
			if err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		recorderErrorPage := newResponseRecorder(ctx, rw)
		utils.CopyHeaders(pageReq.Header, req.Header)

		c.backendHandler.ServeHTTP(recorderErrorPage, pageReq.WithContext(req.Context()))

		utils.CopyHeaders(rw.Header(), recorderErrorPage.Header())
		rw.WriteHeader(code)

		if _, err = rw.Write(recorderErrorPage.GetBody().Bytes()); err != nil {
			logger.Error(err)
		}

		return
	}
}

func newRequest(baseURL string) (*http.Request, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("error pages: error when parse URL: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error pages: error when create query: %w", err)
	}

	req.RequestURI = u.RequestURI()
	return req, nil
}

type responseInterceptor interface {
	http.ResponseWriter
	http.Flusher
	getCode() int
	isFilteredCode() bool
}

// codeCatcher is a response writer that detects as soon as possible whether the
// response is a code within the ranges of codes it watches for. If it is, it
// simply drops the data from the response. Otherwise, it forwards it directly to
// the original client (its responseWriter) without any buffering.
type codeCatcher struct {
	headerMap          http.Header
	code               int
	httpCodeRanges     types.HTTPCodeRanges
	firstWrite         bool
	caughtFilteredCode bool
	responseWriter     http.ResponseWriter
	headersSent        bool
}

type codeCatcherWithCloseNotify struct {
	*codeCatcher
}

// CloseNotify returns a channel that receives at most a
// single value (true) when the client connection has gone away.
func (cc *codeCatcherWithCloseNotify) CloseNotify() <-chan bool {
	return cc.responseWriter.(http.CloseNotifier).CloseNotify()
}

func newCodeCatcher(rw http.ResponseWriter, httpCodeRanges types.HTTPCodeRanges) responseInterceptor {
	catcher := &codeCatcher{
		headerMap:      make(http.Header),
		code:           http.StatusOK, // If backend does not call WriteHeader on us, we consider it's a 200.
		responseWriter: rw,
		httpCodeRanges: httpCodeRanges,
		firstWrite:     true,
	}
	if _, ok := rw.(http.CloseNotifier); ok {
		return &codeCatcherWithCloseNotify{catcher}
	}
	return catcher
}

func (cc *codeCatcher) Header() http.Header {
	if cc.headerMap == nil {
		cc.headerMap = make(http.Header)
	}

	return cc.headerMap
}

func (cc *codeCatcher) getCode() int {
	return cc.code
}

// isFilteredCode returns whether the codeCatcher received a response code among the ones it is watching,
// and for which the response should be deferred to the error handler.
func (cc *codeCatcher) isFilteredCode() bool {
	return cc.caughtFilteredCode
}

func (cc *codeCatcher) Write(buf []byte) (int, error) {
	if !cc.firstWrite {
		if cc.caughtFilteredCode {
			// We don't care about the contents of the response,
			// since we want to serve the ones from the error page,
			// so we just drop them.
			return len(buf), nil
		}
		return cc.responseWriter.Write(buf)
	}
	cc.firstWrite = false

	// If WriteHeader was already called from the caller, this is a NOOP.
	// Otherwise, cc.code is actually a 200 here.
	cc.WriteHeader(cc.code)

	if cc.caughtFilteredCode {
		return len(buf), nil
	}
	return cc.responseWriter.Write(buf)
}

func (cc *codeCatcher) WriteHeader(code int) {
	if cc.headersSent || cc.caughtFilteredCode {
		return
	}

	cc.code = code
	for _, block := range cc.httpCodeRanges {
		if cc.code >= block[0] && cc.code <= block[1] {
			cc.caughtFilteredCode = true
			break
		}
	}
	// it will be up to the other response recorder to send the headers,
	// so it is out of our hands now.
	if cc.caughtFilteredCode {
		return
	}
	utils.CopyHeaders(cc.responseWriter.Header(), cc.Header())
	cc.responseWriter.WriteHeader(cc.code)
	cc.headersSent = true
}

// Hijack hijacks the connection.
func (cc *codeCatcher) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := cc.responseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("%T is not a http.Hijacker", cc.responseWriter)
}

// Flush sends any buffered data to the client.
func (cc *codeCatcher) Flush() {
	// If WriteHeader was already called from the caller, this is a NOOP.
	// Otherwise, cc.code is actually a 200 here.
	cc.WriteHeader(cc.code)

	if flusher, ok := cc.responseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

type responseRecorder interface {
	http.ResponseWriter
	http.Flusher
	GetCode() int
	GetBody() *bytes.Buffer
	IsStreamingResponseStarted() bool
}

// newResponseRecorder returns an initialized responseRecorder.
func newResponseRecorder(ctx context.Context, rw http.ResponseWriter) responseRecorder {
	recorder := &responseRecorderWithoutCloseNotify{
		HeaderMap:      make(http.Header),
		Body:           new(bytes.Buffer),
		Code:           http.StatusOK,
		responseWriter: rw,
		logger:         log.FromContext(ctx),
	}
	if _, ok := rw.(http.CloseNotifier); ok {
		return &responseRecorderWithCloseNotify{recorder}
	}
	return recorder
}

// responseRecorderWithoutCloseNotify is an implementation of http.ResponseWriter that
// records its mutations for later inspection.
type responseRecorderWithoutCloseNotify struct {
	Code      int           // the HTTP response code from WriteHeader
	HeaderMap http.Header   // the HTTP response headers
	Body      *bytes.Buffer // if non-nil, the bytes.Buffer to append written data to

	responseWriter           http.ResponseWriter
	err                      error
	streamingResponseStarted bool
	logger                   logrus.FieldLogger
}

type responseRecorderWithCloseNotify struct {
	*responseRecorderWithoutCloseNotify
}

// CloseNotify returns a channel that receives at most a
// single value (true) when the client connection has gone away.
func (r *responseRecorderWithCloseNotify) CloseNotify() <-chan bool {
	return r.responseWriter.(http.CloseNotifier).CloseNotify()
}

// Header returns the response headers.
func (r *responseRecorderWithoutCloseNotify) Header() http.Header {
	if r.HeaderMap == nil {
		r.HeaderMap = make(http.Header)
	}

	return r.HeaderMap
}

func (r *responseRecorderWithoutCloseNotify) GetCode() int {
	return r.Code
}

func (r *responseRecorderWithoutCloseNotify) GetBody() *bytes.Buffer {
	return r.Body
}

func (r *responseRecorderWithoutCloseNotify) IsStreamingResponseStarted() bool {
	return r.streamingResponseStarted
}

// Write always succeeds and writes to rw.Body, if not nil.
func (r *responseRecorderWithoutCloseNotify) Write(buf []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	return r.Body.Write(buf)
}

// WriteHeader sets rw.Code.
func (r *responseRecorderWithoutCloseNotify) WriteHeader(code int) {
	r.Code = code
}

// Hijack hijacks the connection.
func (r *responseRecorderWithoutCloseNotify) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return r.responseWriter.(http.Hijacker).Hijack()
}

// Flush sends any buffered data to the client.
func (r *responseRecorderWithoutCloseNotify) Flush() {
	if !r.streamingResponseStarted {
		utils.CopyHeaders(r.responseWriter.Header(), r.Header())
		r.responseWriter.WriteHeader(r.Code)
		r.streamingResponseStarted = true
	}

	_, err := r.responseWriter.Write(r.Body.Bytes())
	if err != nil {
		r.logger.Errorf("Error writing response in responseRecorder: %v", err)
		r.err = err
	}
	r.Body.Reset()

	if flusher, ok := r.responseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
