package dynamic

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/ip"
)

// +k8s:deepcopy-gen=true

// Middleware holds the Middleware configuration.
type Middleware struct {
	AddPrefix         *AddPrefix         `json:"addPrefix,omitempty" toml:"addPrefix,omitempty" yaml:"addPrefix,omitempty" export:"true"`
	StripPrefix       *StripPrefix       `json:"stripPrefix,omitempty" toml:"stripPrefix,omitempty" yaml:"stripPrefix,omitempty" export:"true"`
	StripPrefixRegex  *StripPrefixRegex  `json:"stripPrefixRegex,omitempty" toml:"stripPrefixRegex,omitempty" yaml:"stripPrefixRegex,omitempty" export:"true"`
	ReplacePath       *ReplacePath       `json:"replacePath,omitempty" toml:"replacePath,omitempty" yaml:"replacePath,omitempty" export:"true"`
	ReplacePathRegex  *ReplacePathRegex  `json:"replacePathRegex,omitempty" toml:"replacePathRegex,omitempty" yaml:"replacePathRegex,omitempty" export:"true"`
	Chain             *Chain             `json:"chain,omitempty" toml:"chain,omitempty" yaml:"chain,omitempty" export:"true"`
	IPWhiteList       *IPWhiteList       `json:"ipWhiteList,omitempty" toml:"ipWhiteList,omitempty" yaml:"ipWhiteList,omitempty" export:"true"`
	Headers           *Headers           `json:"headers,omitempty" toml:"headers,omitempty" yaml:"headers,omitempty" export:"true"`
	Errors            *ErrorPage         `json:"errors,omitempty" toml:"errors,omitempty" yaml:"errors,omitempty" export:"true"`
	RateLimit         *RateLimit         `json:"rateLimit,omitempty" toml:"rateLimit,omitempty" yaml:"rateLimit,omitempty" export:"true"`
	RedirectRegex     *RedirectRegex     `json:"redirectRegex,omitempty" toml:"redirectRegex,omitempty" yaml:"redirectRegex,omitempty" export:"true"`
	RedirectScheme    *RedirectScheme    `json:"redirectScheme,omitempty" toml:"redirectScheme,omitempty" yaml:"redirectScheme,omitempty" export:"true"`
	BasicAuth         *BasicAuth         `json:"basicAuth,omitempty" toml:"basicAuth,omitempty" yaml:"basicAuth,omitempty" export:"true"`
	DigestAuth        *DigestAuth        `json:"digestAuth,omitempty" toml:"digestAuth,omitempty" yaml:"digestAuth,omitempty" export:"true"`
	ForwardAuth       *ForwardAuth       `json:"forwardAuth,omitempty" toml:"forwardAuth,omitempty" yaml:"forwardAuth,omitempty" export:"true"`
	InFlightReq       *InFlightReq       `json:"inFlightReq,omitempty" toml:"inFlightReq,omitempty" yaml:"inFlightReq,omitempty" export:"true"`
	Buffering         *Buffering         `json:"buffering,omitempty" toml:"buffering,omitempty" yaml:"buffering,omitempty" export:"true"`
	CircuitBreaker    *CircuitBreaker    `json:"circuitBreaker,omitempty" toml:"circuitBreaker,omitempty" yaml:"circuitBreaker,omitempty" export:"true"`
	Compress          *Compress          `json:"compress,omitempty" toml:"compress,omitempty" yaml:"compress,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	PassTLSClientCert *PassTLSClientCert `json:"passTLSClientCert,omitempty" toml:"passTLSClientCert,omitempty" yaml:"passTLSClientCert,omitempty" export:"true"`
	Retry             *Retry             `json:"retry,omitempty" toml:"retry,omitempty" yaml:"retry,omitempty" export:"true"`
	ContentType       *ContentType       `json:"contentType,omitempty" toml:"contentType,omitempty" yaml:"contentType,omitempty" export:"true"`

	Plugin map[string]PluginConf `json:"plugin,omitempty" toml:"plugin,omitempty" yaml:"plugin,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// ContentType middleware - or rather its unique `autoDetect` option -
// specifies whether to let the `Content-Type` header,
// if it has not been set by the backend,
// be automatically set to a value derived from the contents of the response.
// As a proxy, the default behavior should be to leave the header alone,
// regardless of what the backend did with it.
// However, the historic default was to always auto-detect and set the header if it was nil,
// and it is going to be kept that way in order to support users currently relying on it.
// This middleware exists to enable the correct behavior until at least the default one can be changed in a future version.
type ContentType struct {
	AutoDetect bool `json:"autoDetect,omitempty" toml:"autoDetect,omitempty" yaml:"autoDetect,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// AddPrefix holds the AddPrefix configuration.
type AddPrefix struct {
	Prefix string `json:"prefix,omitempty" toml:"prefix,omitempty" yaml:"prefix,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// BasicAuth holds the HTTP basic authentication configuration.
type BasicAuth struct {
	Users        Users  `json:"users,omitempty" toml:"users,omitempty" yaml:"users,omitempty"`
	UsersFile    string `json:"usersFile,omitempty" toml:"usersFile,omitempty" yaml:"usersFile,omitempty"`
	Realm        string `json:"realm,omitempty" toml:"realm,omitempty" yaml:"realm,omitempty"`
	RemoveHeader bool   `json:"removeHeader,omitempty" toml:"removeHeader,omitempty" yaml:"removeHeader,omitempty" export:"true"`
	HeaderField  string `json:"headerField,omitempty" toml:"headerField,omitempty" yaml:"headerField,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// Buffering holds the request/response buffering configuration.
type Buffering struct {
	MaxRequestBodyBytes  int64  `json:"maxRequestBodyBytes,omitempty" toml:"maxRequestBodyBytes,omitempty" yaml:"maxRequestBodyBytes,omitempty" export:"true"`
	MemRequestBodyBytes  int64  `json:"memRequestBodyBytes,omitempty" toml:"memRequestBodyBytes,omitempty" yaml:"memRequestBodyBytes,omitempty" export:"true"`
	MaxResponseBodyBytes int64  `json:"maxResponseBodyBytes,omitempty" toml:"maxResponseBodyBytes,omitempty" yaml:"maxResponseBodyBytes,omitempty" export:"true"`
	MemResponseBodyBytes int64  `json:"memResponseBodyBytes,omitempty" toml:"memResponseBodyBytes,omitempty" yaml:"memResponseBodyBytes,omitempty" export:"true"`
	RetryExpression      string `json:"retryExpression,omitempty" toml:"retryExpression,omitempty" yaml:"retryExpression,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// Chain holds a chain of middlewares.
type Chain struct {
	Middlewares []string `json:"middlewares,omitempty" toml:"middlewares,omitempty" yaml:"middlewares,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// CircuitBreaker holds the circuit breaker configuration.
type CircuitBreaker struct {
	Expression string `json:"expression,omitempty" toml:"expression,omitempty" yaml:"expression,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// Compress holds the compress configuration.
type Compress struct {
	ExcludedContentTypes []string `json:"excludedContentTypes,omitempty" toml:"excludedContentTypes,omitempty" yaml:"excludedContentTypes,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// DigestAuth holds the Digest HTTP authentication configuration.
type DigestAuth struct {
	Users        Users  `json:"users,omitempty" toml:"users,omitempty" yaml:"users,omitempty"`
	UsersFile    string `json:"usersFile,omitempty" toml:"usersFile,omitempty" yaml:"usersFile,omitempty"`
	RemoveHeader bool   `json:"removeHeader,omitempty" toml:"removeHeader,omitempty" yaml:"removeHeader,omitempty" export:"true"`
	Realm        string `json:"realm,omitempty" toml:"realm,omitempty" yaml:"realm,omitempty"`
	HeaderField  string `json:"headerField,omitempty" toml:"headerField,omitempty" yaml:"headerField,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// ErrorPage holds the custom error page configuration.
type ErrorPage struct {
	Status  []string `json:"status,omitempty" toml:"status,omitempty" yaml:"status,omitempty" export:"true"`
	Service string   `json:"service,omitempty" toml:"service,omitempty" yaml:"service,omitempty" export:"true"`
	Query   string   `json:"query,omitempty" toml:"query,omitempty" yaml:"query,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// ForwardAuth holds the http forward authentication configuration.
type ForwardAuth struct {
	Address                  string     `json:"address,omitempty" toml:"address,omitempty" yaml:"address,omitempty"`
	TLS                      *ClientTLS `json:"tls,omitempty" toml:"tls,omitempty" yaml:"tls,omitempty" export:"true"`
	TrustForwardHeader       bool       `json:"trustForwardHeader,omitempty" toml:"trustForwardHeader,omitempty" yaml:"trustForwardHeader,omitempty" export:"true"`
	AuthResponseHeaders      []string   `json:"authResponseHeaders,omitempty" toml:"authResponseHeaders,omitempty" yaml:"authResponseHeaders,omitempty" export:"true"`
	AuthResponseHeadersRegex string     `json:"authResponseHeadersRegex,omitempty" toml:"authResponseHeadersRegex,omitempty" yaml:"authResponseHeadersRegex,omitempty" export:"true"`
	AuthRequestHeaders       []string   `json:"authRequestHeaders,omitempty" toml:"authRequestHeaders,omitempty" yaml:"authRequestHeaders,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// Headers holds the custom header configuration.
type Headers struct {
	CustomRequestHeaders  map[string]string `json:"customRequestHeaders,omitempty" toml:"customRequestHeaders,omitempty" yaml:"customRequestHeaders,omitempty" export:"true"`
	CustomResponseHeaders map[string]string `json:"customResponseHeaders,omitempty" toml:"customResponseHeaders,omitempty" yaml:"customResponseHeaders,omitempty" export:"true"`

	// AccessControlAllowCredentials is only valid if true. false is ignored.
	AccessControlAllowCredentials bool `json:"accessControlAllowCredentials,omitempty" toml:"accessControlAllowCredentials,omitempty" yaml:"accessControlAllowCredentials,omitempty" export:"true"`
	// AccessControlAllowHeaders must be used in response to a preflight request with Access-Control-Request-Headers set.
	AccessControlAllowHeaders []string `json:"accessControlAllowHeaders,omitempty" toml:"accessControlAllowHeaders,omitempty" yaml:"accessControlAllowHeaders,omitempty" export:"true"`
	// AccessControlAllowMethods must be used in response to a preflight request with Access-Control-Request-Method set.
	AccessControlAllowMethods []string `json:"accessControlAllowMethods,omitempty" toml:"accessControlAllowMethods,omitempty" yaml:"accessControlAllowMethods,omitempty" export:"true"`
	// AccessControlAllowOriginList is a list of allowable origins. Can also be a wildcard origin "*".
	AccessControlAllowOriginList []string `json:"accessControlAllowOriginList,omitempty" toml:"accessControlAllowOriginList,omitempty" yaml:"accessControlAllowOriginList,omitempty"`
	// AccessControlAllowOriginListRegex is a list of allowable origins written following the Regular Expression syntax (https://golang.org/pkg/regexp/).
	AccessControlAllowOriginListRegex []string `json:"accessControlAllowOriginListRegex,omitempty" toml:"accessControlAllowOriginListRegex,omitempty" yaml:"accessControlAllowOriginListRegex,omitempty"`
	// AccessControlExposeHeaders sets valid headers for the response.
	AccessControlExposeHeaders []string `json:"accessControlExposeHeaders,omitempty" toml:"accessControlExposeHeaders,omitempty" yaml:"accessControlExposeHeaders,omitempty" export:"true"`
	// AccessControlMaxAge sets the time that a preflight request may be cached.
	AccessControlMaxAge int64 `json:"accessControlMaxAge,omitempty" toml:"accessControlMaxAge,omitempty" yaml:"accessControlMaxAge,omitempty" export:"true"`
	// AddVaryHeader controls if the Vary header is automatically added/updated when the AccessControlAllowOriginList is set.
	AddVaryHeader bool `json:"addVaryHeader,omitempty" toml:"addVaryHeader,omitempty" yaml:"addVaryHeader,omitempty" export:"true"`

	AllowedHosts      []string `json:"allowedHosts,omitempty" toml:"allowedHosts,omitempty" yaml:"allowedHosts,omitempty"`
	HostsProxyHeaders []string `json:"hostsProxyHeaders,omitempty" toml:"hostsProxyHeaders,omitempty" yaml:"hostsProxyHeaders,omitempty" export:"true"`
	// Deprecated: use EntryPoint redirection or RedirectScheme instead.
	SSLRedirect bool `json:"sslRedirect,omitempty" toml:"sslRedirect,omitempty" yaml:"sslRedirect,omitempty" export:"true"`
	// Deprecated: use EntryPoint redirection or RedirectScheme instead.
	SSLTemporaryRedirect bool `json:"sslTemporaryRedirect,omitempty" toml:"sslTemporaryRedirect,omitempty" yaml:"sslTemporaryRedirect,omitempty" export:"true"`
	// Deprecated: use RedirectRegex instead.
	SSLHost         string            `json:"sslHost,omitempty" toml:"sslHost,omitempty" yaml:"sslHost,omitempty"`
	SSLProxyHeaders map[string]string `json:"sslProxyHeaders,omitempty" toml:"sslProxyHeaders,omitempty" yaml:"sslProxyHeaders,omitempty"`
	// Deprecated: use RedirectRegex instead.
	SSLForceHost            bool   `json:"sslForceHost,omitempty" toml:"sslForceHost,omitempty" yaml:"sslForceHost,omitempty" export:"true"`
	STSSeconds              int64  `json:"stsSeconds,omitempty" toml:"stsSeconds,omitempty" yaml:"stsSeconds,omitempty" export:"true"`
	STSIncludeSubdomains    bool   `json:"stsIncludeSubdomains,omitempty" toml:"stsIncludeSubdomains,omitempty" yaml:"stsIncludeSubdomains,omitempty" export:"true"`
	STSPreload              bool   `json:"stsPreload,omitempty" toml:"stsPreload,omitempty" yaml:"stsPreload,omitempty" export:"true"`
	ForceSTSHeader          bool   `json:"forceSTSHeader,omitempty" toml:"forceSTSHeader,omitempty" yaml:"forceSTSHeader,omitempty" export:"true"`
	FrameDeny               bool   `json:"frameDeny,omitempty" toml:"frameDeny,omitempty" yaml:"frameDeny,omitempty" export:"true"`
	CustomFrameOptionsValue string `json:"customFrameOptionsValue,omitempty" toml:"customFrameOptionsValue,omitempty" yaml:"customFrameOptionsValue,omitempty"`
	ContentTypeNosniff      bool   `json:"contentTypeNosniff,omitempty" toml:"contentTypeNosniff,omitempty" yaml:"contentTypeNosniff,omitempty" export:"true"`
	BrowserXSSFilter        bool   `json:"browserXssFilter,omitempty" toml:"browserXssFilter,omitempty" yaml:"browserXssFilter,omitempty" export:"true"`
	CustomBrowserXSSValue   string `json:"customBrowserXSSValue,omitempty" toml:"customBrowserXSSValue,omitempty" yaml:"customBrowserXSSValue,omitempty"`
	ContentSecurityPolicy   string `json:"contentSecurityPolicy,omitempty" toml:"contentSecurityPolicy,omitempty" yaml:"contentSecurityPolicy,omitempty"`
	PublicKey               string `json:"publicKey,omitempty" toml:"publicKey,omitempty" yaml:"publicKey,omitempty"`
	ReferrerPolicy          string `json:"referrerPolicy,omitempty" toml:"referrerPolicy,omitempty" yaml:"referrerPolicy,omitempty" export:"true"`
	// Deprecated: use PermissionsPolicy instead.
	FeaturePolicy     string `json:"featurePolicy,omitempty" toml:"featurePolicy,omitempty" yaml:"featurePolicy,omitempty" export:"true"`
	PermissionsPolicy string `json:"permissionsPolicy,omitempty" toml:"permissionsPolicy,omitempty" yaml:"permissionsPolicy,omitempty" export:"true"`
	IsDevelopment     bool   `json:"isDevelopment,omitempty" toml:"isDevelopment,omitempty" yaml:"isDevelopment,omitempty" export:"true"`
}

// HasCustomHeadersDefined checks to see if any of the custom header elements have been set.
func (h *Headers) HasCustomHeadersDefined() bool {
	return h != nil && (len(h.CustomResponseHeaders) != 0 ||
		len(h.CustomRequestHeaders) != 0)
}

// HasCorsHeadersDefined checks to see if any of the cors header elements have been set.
func (h *Headers) HasCorsHeadersDefined() bool {
	return h != nil && (h.AccessControlAllowCredentials ||
		len(h.AccessControlAllowHeaders) != 0 ||
		len(h.AccessControlAllowMethods) != 0 ||
		len(h.AccessControlAllowOriginList) != 0 ||
		len(h.AccessControlAllowOriginListRegex) != 0 ||
		len(h.AccessControlExposeHeaders) != 0 ||
		h.AccessControlMaxAge != 0 ||
		h.AddVaryHeader)
}

// HasSecureHeadersDefined checks to see if any of the secure header elements have been set.
func (h *Headers) HasSecureHeadersDefined() bool {
	return h != nil && (len(h.AllowedHosts) != 0 ||
		len(h.HostsProxyHeaders) != 0 ||
		h.SSLRedirect ||
		h.SSLTemporaryRedirect ||
		h.SSLForceHost ||
		h.SSLHost != "" ||
		len(h.SSLProxyHeaders) != 0 ||
		h.STSSeconds != 0 ||
		h.STSIncludeSubdomains ||
		h.STSPreload ||
		h.ForceSTSHeader ||
		h.FrameDeny ||
		h.CustomFrameOptionsValue != "" ||
		h.ContentTypeNosniff ||
		h.BrowserXSSFilter ||
		h.CustomBrowserXSSValue != "" ||
		h.ContentSecurityPolicy != "" ||
		h.PublicKey != "" ||
		h.ReferrerPolicy != "" ||
		h.FeaturePolicy != "" ||
		h.PermissionsPolicy != "" ||
		h.IsDevelopment)
}

// +k8s:deepcopy-gen=true

// IPStrategy holds the ip strategy configuration.
type IPStrategy struct {
	Depth       int      `json:"depth,omitempty" toml:"depth,omitempty" yaml:"depth,omitempty" export:"true"`
	ExcludedIPs []string `json:"excludedIPs,omitempty" toml:"excludedIPs,omitempty" yaml:"excludedIPs,omitempty"`
	// TODO(mpl): I think we should make RemoteAddr an explicit field. For one thing, it would yield better documentation.
}

// Get an IP selection strategy.
// If nil return the RemoteAddr strategy
// else return a strategy based on the configuration using the X-Forwarded-For Header.
// Depth override the ExcludedIPs.
func (s *IPStrategy) Get() (ip.Strategy, error) {
	if s == nil {
		return &ip.RemoteAddrStrategy{}, nil
	}

	if s.Depth > 0 {
		return &ip.DepthStrategy{
			Depth: s.Depth,
		}, nil
	}

	if len(s.ExcludedIPs) > 0 {
		checker, err := ip.NewChecker(s.ExcludedIPs)
		if err != nil {
			return nil, err
		}
		return &ip.PoolStrategy{
			Checker: checker,
		}, nil
	}

	return &ip.RemoteAddrStrategy{}, nil
}

// +k8s:deepcopy-gen=true

// IPWhiteList holds the ip white list configuration.
type IPWhiteList struct {
	SourceRange []string    `json:"sourceRange,omitempty" toml:"sourceRange,omitempty" yaml:"sourceRange,omitempty"`
	IPStrategy  *IPStrategy `json:"ipStrategy,omitempty" toml:"ipStrategy,omitempty" yaml:"ipStrategy,omitempty"  label:"allowEmpty" file:"allowEmpty" export:"true"`
}

// +k8s:deepcopy-gen=true

// InFlightReq limits the number of requests being processed and served concurrently.
type InFlightReq struct {
	Amount          int64            `json:"amount,omitempty" toml:"amount,omitempty" yaml:"amount,omitempty" export:"true"`
	SourceCriterion *SourceCriterion `json:"sourceCriterion,omitempty" toml:"sourceCriterion,omitempty" yaml:"sourceCriterion,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// PassTLSClientCert holds the TLS client cert headers configuration.
type PassTLSClientCert struct {
	PEM  bool                      `json:"pem,omitempty" toml:"pem,omitempty" yaml:"pem,omitempty" export:"true"`
	Info *TLSClientCertificateInfo `json:"info,omitempty" toml:"info,omitempty" yaml:"info,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// SourceCriterion defines what criterion is used to group requests as originating from a common source.
// If none are set, the default is to use the request's remote address field.
// All fields are mutually exclusive.
type SourceCriterion struct {
	IPStrategy        *IPStrategy `json:"ipStrategy,omitempty" toml:"ipStrategy,omitempty" yaml:"ipStrategy,omitempty" export:"true"`
	RequestHeaderName string      `json:"requestHeaderName,omitempty" toml:"requestHeaderName,omitempty" yaml:"requestHeaderName,omitempty" export:"true"`
	RequestHost       bool        `json:"requestHost,omitempty" toml:"requestHost,omitempty" yaml:"requestHost,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// RateLimit holds the rate limiting configuration for a given router.
type RateLimit struct {
	// Average is the maximum rate, by default in requests/s, allowed for the given source.
	// It defaults to 0, which means no rate limiting.
	// The rate is actually defined by dividing Average by Period. So for a rate below 1req/s,
	// one needs to define a Period larger than a second.
	Average int64 `json:"average,omitempty" toml:"average,omitempty" yaml:"average,omitempty" export:"true"`

	// Period, in combination with Average, defines the actual maximum rate, such as:
	// r = Average / Period. It defaults to a second.
	Period ptypes.Duration `json:"period,omitempty" toml:"period,omitempty" yaml:"period,omitempty" export:"true"`

	// Burst is the maximum number of requests allowed to arrive in the same arbitrarily small period of time.
	// It defaults to 1.
	Burst int64 `json:"burst,omitempty" toml:"burst,omitempty" yaml:"burst,omitempty" export:"true"`

	SourceCriterion *SourceCriterion `json:"sourceCriterion,omitempty" toml:"sourceCriterion,omitempty" yaml:"sourceCriterion,omitempty" export:"true"`
}

// SetDefaults sets the default values on a RateLimit.
func (r *RateLimit) SetDefaults() {
	r.Burst = 1
	r.Period = ptypes.Duration(time.Second)
}

// +k8s:deepcopy-gen=true

// RedirectRegex holds the redirection configuration.
type RedirectRegex struct {
	Regex       string `json:"regex,omitempty" toml:"regex,omitempty" yaml:"regex,omitempty"`
	Replacement string `json:"replacement,omitempty" toml:"replacement,omitempty" yaml:"replacement,omitempty"`
	Permanent   bool   `json:"permanent,omitempty" toml:"permanent,omitempty" yaml:"permanent,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// RedirectScheme holds the scheme redirection configuration.
type RedirectScheme struct {
	Scheme    string `json:"scheme,omitempty" toml:"scheme,omitempty" yaml:"scheme,omitempty" export:"true"`
	Port      string `json:"port,omitempty" toml:"port,omitempty" yaml:"port,omitempty" export:"true"`
	Permanent bool   `json:"permanent,omitempty" toml:"permanent,omitempty" yaml:"permanent,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// ReplacePath holds the ReplacePath configuration.
type ReplacePath struct {
	Path string `json:"path,omitempty" toml:"path,omitempty" yaml:"path,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// ReplacePathRegex holds the ReplacePathRegex configuration.
type ReplacePathRegex struct {
	Regex       string `json:"regex,omitempty" toml:"regex,omitempty" yaml:"regex,omitempty" export:"true"`
	Replacement string `json:"replacement,omitempty" toml:"replacement,omitempty" yaml:"replacement,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// Retry holds the retry configuration.
type Retry struct {
	Attempts        int             `json:"attempts,omitempty" toml:"attempts,omitempty" yaml:"attempts,omitempty" export:"true"`
	InitialInterval ptypes.Duration `json:"initialInterval,omitempty" toml:"initialInterval,omitempty" yaml:"initialInterval,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// StripPrefix holds the StripPrefix configuration.
type StripPrefix struct {
	Prefixes   []string `json:"prefixes,omitempty" toml:"prefixes,omitempty" yaml:"prefixes,omitempty" export:"true"`
	ForceSlash bool     `json:"forceSlash,omitempty" toml:"forceSlash,omitempty" yaml:"forceSlash,omitempty" export:"true"` // Deprecated
}

// SetDefaults Default values for a StripPrefix.
func (s *StripPrefix) SetDefaults() {
	s.ForceSlash = true
}

// +k8s:deepcopy-gen=true

// StripPrefixRegex holds the StripPrefixRegex configuration.
type StripPrefixRegex struct {
	Regex []string `json:"regex,omitempty" toml:"regex,omitempty" yaml:"regex,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// TLSClientCertificateInfo holds the client TLS certificate info configuration.
type TLSClientCertificateInfo struct {
	NotAfter     bool                               `json:"notAfter,omitempty" toml:"notAfter,omitempty" yaml:"notAfter,omitempty" export:"true"`
	NotBefore    bool                               `json:"notBefore,omitempty" toml:"notBefore,omitempty" yaml:"notBefore,omitempty" export:"true"`
	Sans         bool                               `json:"sans,omitempty" toml:"sans,omitempty" yaml:"sans,omitempty" export:"true"`
	Subject      *TLSCLientCertificateSubjectDNInfo `json:"subject,omitempty" toml:"subject,omitempty" yaml:"subject,omitempty" export:"true"`
	Issuer       *TLSCLientCertificateIssuerDNInfo  `json:"issuer,omitempty" toml:"issuer,omitempty" yaml:"issuer,omitempty" export:"true"`
	SerialNumber bool                               `json:"serialNumber,omitempty" toml:"serialNumber,omitempty" yaml:"serialNumber,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// TLSCLientCertificateIssuerDNInfo holds the client TLS certificate distinguished name info configuration.
// cf https://tools.ietf.org/html/rfc3739
type TLSCLientCertificateIssuerDNInfo struct {
	Country         bool `json:"country,omitempty" toml:"country,omitempty" yaml:"country,omitempty" export:"true"`
	Province        bool `json:"province,omitempty" toml:"province,omitempty" yaml:"province,omitempty" export:"true"`
	Locality        bool `json:"locality,omitempty" toml:"locality,omitempty" yaml:"locality,omitempty" export:"true"`
	Organization    bool `json:"organization,omitempty" toml:"organization,omitempty" yaml:"organization,omitempty" export:"true"`
	CommonName      bool `json:"commonName,omitempty" toml:"commonName,omitempty" yaml:"commonName,omitempty" export:"true"`
	SerialNumber    bool `json:"serialNumber,omitempty" toml:"serialNumber,omitempty" yaml:"serialNumber,omitempty" export:"true"`
	DomainComponent bool `json:"domainComponent,omitempty" toml:"domainComponent,omitempty" yaml:"domainComponent,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// TLSCLientCertificateSubjectDNInfo holds the client TLS certificate distinguished name info configuration.
// cf https://tools.ietf.org/html/rfc3739
type TLSCLientCertificateSubjectDNInfo struct {
	Country            bool `json:"country,omitempty" toml:"country,omitempty" yaml:"country,omitempty" export:"true"`
	Province           bool `json:"province,omitempty" toml:"province,omitempty" yaml:"province,omitempty" export:"true"`
	Locality           bool `json:"locality,omitempty" toml:"locality,omitempty" yaml:"locality,omitempty" export:"true"`
	Organization       bool `json:"organization,omitempty" toml:"organization,omitempty" yaml:"organization,omitempty" export:"true"`
	OrganizationalUnit bool `json:"organizationalUnit,omitempty" toml:"organizationalUnit,omitempty" yaml:"organizationalUnit,omitempty" export:"true"`
	CommonName         bool `json:"commonName,omitempty" toml:"commonName,omitempty" yaml:"commonName,omitempty" export:"true"`
	SerialNumber       bool `json:"serialNumber,omitempty" toml:"serialNumber,omitempty" yaml:"serialNumber,omitempty" export:"true"`
	DomainComponent    bool `json:"domainComponent,omitempty" toml:"domainComponent,omitempty" yaml:"domainComponent,omitempty" export:"true"`
}

// +k8s:deepcopy-gen=true

// Users holds a list of users.
type Users []string

// +k8s:deepcopy-gen=true

// ClientTLS holds the TLS specific configurations as client
// CA, Cert and Key can be either path or file contents.
type ClientTLS struct {
	CA                 string `json:"ca,omitempty" toml:"ca,omitempty" yaml:"ca,omitempty"`
	CAOptional         bool   `json:"caOptional,omitempty" toml:"caOptional,omitempty" yaml:"caOptional,omitempty" export:"true"`
	Cert               string `json:"cert,omitempty" toml:"cert,omitempty" yaml:"cert,omitempty"`
	Key                string `json:"key,omitempty" toml:"key,omitempty" yaml:"key,omitempty"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty" toml:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty" export:"true"`
}

// CreateTLSConfig creates a TLS config from ClientTLS structures.
func (c *ClientTLS) CreateTLSConfig() (*tls.Config, error) {
	if c == nil {
		return nil, nil
	}

	var err error
	caPool := x509.NewCertPool()
	clientAuth := tls.NoClientCert
	if c.CA != "" {
		var ca []byte
		if _, errCA := os.Stat(c.CA); errCA == nil {
			ca, err = os.ReadFile(c.CA)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA. %w", err)
			}
		} else {
			ca = []byte(c.CA)
		}

		if !caPool.AppendCertsFromPEM(ca) {
			return nil, fmt.Errorf("failed to parse CA")
		}

		if c.CAOptional {
			clientAuth = tls.VerifyClientCertIfGiven
		} else {
			clientAuth = tls.RequireAndVerifyClientCert
		}
	}

	cert := tls.Certificate{}
	_, errKeyIsFile := os.Stat(c.Key)

	if !c.InsecureSkipVerify && (len(c.Cert) == 0 || len(c.Key) == 0) {
		return nil, fmt.Errorf("TLS Certificate or Key file must be set when TLS configuration is created")
	}

	if len(c.Cert) > 0 && len(c.Key) > 0 {
		if _, errCertIsFile := os.Stat(c.Cert); errCertIsFile == nil {
			if errKeyIsFile == nil {
				cert, err = tls.LoadX509KeyPair(c.Cert, c.Key)
				if err != nil {
					return nil, fmt.Errorf("failed to load TLS keypair: %w", err)
				}
			} else {
				return nil, fmt.Errorf("tls cert is a file, but tls key is not")
			}
		} else {
			if errKeyIsFile != nil {
				cert, err = tls.X509KeyPair([]byte(c.Cert), []byte(c.Key))
				if err != nil {
					return nil, fmt.Errorf("failed to load TLS keypair: %w", err)
				}
			} else {
				return nil, fmt.Errorf("TLS key is a file, but tls cert is not")
			}
		}
	}

	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caPool,
		InsecureSkipVerify: c.InsecureSkipVerify,
		ClientAuth:         clientAuth,
	}, nil
}
