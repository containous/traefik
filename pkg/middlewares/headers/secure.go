package headers

import (
	"net/http"

	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/unrolled/secure"
)

type secureHeader struct {
	next   http.Handler
	secure *secure.Secure
	cfg    dynamic.Headers
}

// newSecure constructs a new secure instance with supplied options.
func newSecure(next http.Handler, cfg dynamic.Headers, contextKey string) *secureHeader {
	opt := secure.Options{
		BrowserXssFilter:        cfg.BrowserXSSFilter,
		ContentTypeNosniff:      cfg.ContentTypeNosniff,
		ForceSTSHeader:          cfg.ForceSTSHeader,
		FrameDeny:               cfg.FrameDeny,
		IsDevelopment:           cfg.IsDevelopment,
		SSLRedirect:             cfg.SSLRedirect,
		SSLForceHost:            cfg.SSLForceHost,
		SSLTemporaryRedirect:    cfg.SSLTemporaryRedirect,
		STSIncludeSubdomains:    cfg.STSIncludeSubdomains,
		STSPreload:              cfg.STSPreload,
		ContentSecurityPolicy:   cfg.ContentSecurityPolicy,
		CustomBrowserXssValue:   cfg.CustomBrowserXSSValue,
		CustomFrameOptionsValue: cfg.CustomFrameOptionsValue,
		PublicKey:               cfg.PublicKey,
		ReferrerPolicy:          cfg.ReferrerPolicy,
		SSLHost:                 cfg.SSLHost,
		AllowedHosts:            cfg.AllowedHosts,
		HostsProxyHeaders:       cfg.HostsProxyHeaders,
		SSLProxyHeaders:         cfg.SSLProxyHeaders,
		STSSeconds:              cfg.STSSeconds,
		FeaturePolicy:           cfg.FeaturePolicy,
		PermissionsPolicy:       cfg.PermissionsPolicy,
		SecureContextKey:        contextKey,
	}

	return &secureHeader{
		next:   next,
		secure: secure.New(opt),
		cfg:    cfg,
	}
}

func (s secureHeader) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	s.secure.HandlerFuncWithNextForRequestOnly(rw, req, func(writer http.ResponseWriter, request *http.Request) {
		s.next.ServeHTTP(newResponseModifier(writer, request, s.secure.ModifyResponseHeaders), request)
	})
}
