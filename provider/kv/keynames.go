package kv

const (
	pathBackends                                = "/backends/"
	pathBackendCircuitBreakerExpression         = "/circuitbreaker/expression"
	pathBackendHealthCheckPath                  = "/healthcheck/path"
	pathBackendHealthCheckPort                  = "/healthcheck/port"
	pathBackendHealthCheckInterval              = "/healthcheck/interval"
	pathBackendLoadBalancerMethod               = "/loadbalancer/method"
	pathBackendLoadBalancerSticky               = "/loadbalancer/sticky"
	pathBackendLoadBalancerStickiness           = "/loadbalancer/stickiness"
	pathBackendLoadBalancerStickinessCookieName = "/loadbalancer/stickiness/cookiename"
	pathBackendMaxConnAmount                    = "/maxconn/amount"
	pathBackendMaxConnExtractorFunc             = "/maxconn/extractorfunc"
	pathBackendServers                          = "/servers/"
	pathBackendServerURL                        = "/url"
	pathBackendServerWeight                     = "/weight"
	pathBackendBufferingEnabled                 = "/buffering/enabled"
	pathBackendBufferingMaxResponseBodyBytes    = "/buffering/maxresponsebodybytes"
	pathBackendBufferingMemResponseBodyBytes    = "/buffering/memresponsebodybytes"
	pathBackendBufferingMaxRequestBodyBytes     = "/buffering/maxrequestbodybytes"
	pathBackendBufferingMemRequestBodyBytes     = "/buffering/memrequestbodybytes"
	pathBackendBufferingRetryExpression         = "/buffering/retryexpression"

	pathFrontends                      = "/frontends/"
	pathFrontendBackend                = "/backend"
	pathFrontendPriority               = "/priority"
	pathFrontendPassHostHeader         = "/passHostHeader"
	pathFrontendPassTLSCert            = "/passtlscert"
	pathFrontendWhiteListSourceRange   = "/whitelistsourcerange"
	pathFrontendBasicAuth              = "/basicauth"
	pathFrontendEntryPoints            = "/entrypoints"
	pathFrontendRedirectEntryPoint     = "/redirect/entrypoint"
	pathFrontendRedirectRegex          = "/redirect/regex"
	pathFrontendRedirectReplacement    = "/redirect/replacement"
	pathFrontendErrorPages             = "/errors/"
	pathFrontendErrorPagesBackend      = "/backend"
	pathFrontendErrorPagesQuery        = "/query"
	pathFrontendErrorPagesStatus       = "/status"
	pathFrontendRateLimit              = "/ratelimit/"
	pathFrontendRateLimitRateSet       = pathFrontendRateLimit + "rateset/"
	pathFrontendRateLimitExtractorFunc = pathFrontendRateLimit + "extractorfunc"
	pathFrontendRateLimitPeriod        = "/period"
	pathFrontendRateLimitAverage       = "/average"
	pathFrontendRateLimitBurst         = "/burst"

	pathFrontendCustomRequestHeaders    = "/headers/customrequestheaders/"
	pathFrontendCustomResponseHeaders   = "/headers/customresponseheaders/"
	pathFrontendAllowedHosts            = "/headers/allowedhosts"
	pathFrontendHostsProxyHeaders       = "/headers/hostsproxyheaders"
	pathFrontendSSLRedirect             = "/headers/sslredirect"
	pathFrontendSSLTemporaryRedirect    = "/headers/ssltemporaryredirect"
	pathFrontendSSLHost                 = "/headers/sslhost"
	pathFrontendSSLProxyHeaders         = "/headers/sslproxyheaders/"
	pathFrontendSTSSeconds              = "/headers/stsseconds"
	pathFrontendSTSIncludeSubdomains    = "/headers/stsincludesubdomains"
	pathFrontendSTSPreload              = "/headers/stspreload"
	pathFrontendForceSTSHeader          = "/headers/forcestsheader"
	pathFrontendFrameDeny               = "/headers/framedeny"
	pathFrontendCustomFrameOptionsValue = "/headers/customframeoptionsvalue"
	pathFrontendContentTypeNosniff      = "/headers/contenttypenosniff"
	pathFrontendBrowserXSSFilter        = "/headers/browserxssfilter"
	pathFrontendContentSecurityPolicy   = "/headers/contentsecuritypolicy"
	pathFrontendPublicKey               = "/headers/publickey"
	pathFrontendReferrerPolicy          = "/headers/referrerpolicy"
	pathFrontendIsDevelopment           = "/headers/isdevelopment"

	pathFrontendRoutes = "/routes/"
	pathFrontendRule   = "/rule"

	pathTLS            = "/tls/"
	pathTLSEntryPoints = "/entrypoints"
	pathTLSCertFile    = "/certificate/certfile"
	pathTLSKeyFile     = "/certificate/keyfile"

	pathTags      = "/tags"
	pathAlias     = "/alias"
	pathSeparator = "/"
)
