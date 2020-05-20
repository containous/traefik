package auth

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/containous/traefik/v2/pkg/config/dynamic"
	"github.com/containous/traefik/v2/pkg/log"
	"github.com/containous/traefik/v2/pkg/middlewares"
	"github.com/containous/traefik/v2/pkg/tracing"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/vulcand/oxy/forward"
	"github.com/vulcand/oxy/utils"
)

const (
	xForwardedURI     = "X-Forwarded-Uri"
	xForwardedMethod  = "X-Forwarded-Method"
	forwardedTypeName = "ForwardedAuthType"
)

type forwardAuth struct {
	address             string
	authResponseHeaders []string
	next                http.Handler
	name                string
	client              http.Client
	trustForwardHeader  bool
}

// NewForward creates a forward auth middleware.
func NewForward(ctx context.Context, next http.Handler, config dynamic.ForwardAuth, name string) (http.Handler, error) {
	log.FromContext(middlewares.GetLoggerCtx(ctx, name, forwardedTypeName)).Debug("Creating middleware")

	fa := &forwardAuth{
		address:             config.Address,
		authResponseHeaders: config.AuthResponseHeaders,
		next:                next,
		name:                name,
		trustForwardHeader:  config.TrustForwardHeader,
	}

	// Ensure our request client does not follow redirects
	fa.client = http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 30 * time.Second,
	}

	if config.TLS != nil {
		tlsConfig, err := config.TLS.CreateTLSConfig()
		if err != nil {
			return nil, err
		}

		tr := http.DefaultTransport.(*http.Transport).Clone()
		tr.TLSClientConfig = tlsConfig
		fa.client.Transport = tr
	}

	return fa, nil
}

func (fa *forwardAuth) GetTracingInformation() (string, ext.SpanKindEnum) {
	return fa.name, ext.SpanKindRPCClientEnum
}

func (fa *forwardAuth) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	logger := log.FromContext(middlewares.GetLoggerCtx(req.Context(), fa.name, forwardedTypeName))

	forwardReq, err := http.NewRequest(http.MethodGet, fa.address, nil)
	tracing.LogRequest(tracing.GetSpan(req), forwardReq)
	if err != nil {
		logMessage := fmt.Sprintf("Error calling %s. Cause %s", fa.address, err)
		logger.Debug(logMessage)
		tracing.SetErrorWithEvent(req, logMessage)

		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Ensure tracing headers are in the request before we copy the headers to the
	// forwardReq.
	tracing.InjectRequestHeaders(req)

	writeHeader(req, forwardReq, fa.trustForwardHeader)

	forwardResponse, forwardErr := fa.client.Do(forwardReq)
	if forwardErr != nil {
		logMessage := fmt.Sprintf("Error calling %s. Cause: %s", fa.address, forwardErr)
		logger.Debug(logMessage)
		tracing.SetErrorWithEvent(req, logMessage)

		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	body, readError := ioutil.ReadAll(forwardResponse.Body)
	if readError != nil {
		logMessage := fmt.Sprintf("Error reading body %s. Cause: %s", fa.address, readError)
		logger.Debug(logMessage)
		tracing.SetErrorWithEvent(req, logMessage)

		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer forwardResponse.Body.Close()

	// Pass the forward response's body and selected headers if it
	// didn't return a response within the range of [200, 300).
	if forwardResponse.StatusCode < http.StatusOK || forwardResponse.StatusCode >= http.StatusMultipleChoices {
		logger.Debugf("Remote error %s. StatusCode: %d", fa.address, forwardResponse.StatusCode)

		utils.CopyHeaders(rw.Header(), forwardResponse.Header)
		utils.RemoveHeaders(rw.Header(), forward.HopHeaders...)

		// Grab the location header, if any.
		redirectURL, err := forwardResponse.Location()

		if err != nil {
			if err != http.ErrNoLocation {
				logMessage := fmt.Sprintf("Error reading response location header %s. Cause: %s", fa.address, err)
				logger.Debug(logMessage)
				tracing.SetErrorWithEvent(req, logMessage)

				rw.WriteHeader(http.StatusInternalServerError)
				return
			}
		} else if redirectURL.String() != "" {
			// Set the location in our response if one was sent back.
			rw.Header().Set("Location", redirectURL.String())
		}

		tracing.LogResponseCode(tracing.GetSpan(req), forwardResponse.StatusCode)
		rw.WriteHeader(forwardResponse.StatusCode)

		if _, err = rw.Write(body); err != nil {
			logger.Error(err)
		}
		return
	}

	// Map of headers that already set by wildcard pattern to prevent double check
	setHeaders := make(map[string]bool)

	for _, headerName := range fa.authResponseHeaders {
		configHeader := http.CanonicalHeaderKey(headerName)

		// Check if config header is a wildcard pattern
		if strings.ContainsAny(configHeader, "*?[]") {
			// Remove original request's headers that match pattern even if it's not set in forward response
			for originalHeader := range req.Header {
				if match, _ := filepath.Match(configHeader, originalHeader); match {
					req.Header.Del(originalHeader)
				}
			}

			for forwardResponseHeader, forwardResponseHeaderValue := range forwardResponse.Header {
				if setHeaders[forwardResponseHeader] || len(forwardResponseHeaderValue) == 0 {
					continue
				}
				if match, _ := filepath.Match(configHeader, forwardResponseHeader); match {
					setHeaders[forwardResponseHeader] = true
					req.Header[forwardResponseHeader] = append([]string(nil), forwardResponseHeaderValue...)
				}
			}
		} else if len(forwardResponse.Header[configHeader]) > 0 {
			setHeaders[configHeader] = true
			req.Header.Del(configHeader)
			req.Header[configHeader] = append([]string(nil), forwardResponse.Header[configHeader]...)
		}
	}

	req.RequestURI = req.URL.RequestURI()
	fa.next.ServeHTTP(rw, req)
}

func writeHeader(req *http.Request, forwardReq *http.Request, trustForwardHeader bool) {
	utils.CopyHeaders(forwardReq.Header, req.Header)
	utils.RemoveHeaders(forwardReq.Header, forward.HopHeaders...)

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		if trustForwardHeader {
			if prior, ok := req.Header[forward.XForwardedFor]; ok {
				clientIP = strings.Join(prior, ", ") + ", " + clientIP
			}
		}
		forwardReq.Header.Set(forward.XForwardedFor, clientIP)
	}

	xMethod := req.Header.Get(xForwardedMethod)
	switch {
	case xMethod != "" && trustForwardHeader:
		forwardReq.Header.Set(xForwardedMethod, xMethod)
	case req.Method != "":
		forwardReq.Header.Set(xForwardedMethod, req.Method)
	default:
		forwardReq.Header.Del(xForwardedMethod)
	}

	xfp := req.Header.Get(forward.XForwardedProto)
	switch {
	case xfp != "" && trustForwardHeader:
		forwardReq.Header.Set(forward.XForwardedProto, xfp)
	case req.TLS != nil:
		forwardReq.Header.Set(forward.XForwardedProto, "https")
	default:
		forwardReq.Header.Set(forward.XForwardedProto, "http")
	}

	if xfp := req.Header.Get(forward.XForwardedPort); xfp != "" && trustForwardHeader {
		forwardReq.Header.Set(forward.XForwardedPort, xfp)
	}

	xfh := req.Header.Get(forward.XForwardedHost)
	switch {
	case xfh != "" && trustForwardHeader:
		forwardReq.Header.Set(forward.XForwardedHost, xfh)
	case req.Host != "":
		forwardReq.Header.Set(forward.XForwardedHost, req.Host)
	default:
		forwardReq.Header.Del(forward.XForwardedHost)
	}

	xfURI := req.Header.Get(xForwardedURI)
	switch {
	case xfURI != "" && trustForwardHeader:
		forwardReq.Header.Set(xForwardedURI, xfURI)
	case req.URL.RequestURI() != "":
		forwardReq.Header.Set(xForwardedURI, req.URL.RequestURI())
	default:
		forwardReq.Header.Del(xForwardedURI)
	}
}
