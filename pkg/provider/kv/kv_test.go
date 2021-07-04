package kv

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/abronan/valkeyrie/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/tls"
	"github.com/traefik/traefik/v2/pkg/types"
)

func Test_buildConfiguration(t *testing.T) {
	provider := newProviderMock(mapToPairs(map[string]string{
		"traefik/http/routers/Router0/entryPoints/0":                                                 "foobar",
		"traefik/http/routers/Router0/entryPoints/1":                                                 "foobar",
		"traefik/http/routers/Router0/middlewares/0":                                                 "foobar",
		"traefik/http/routers/Router0/middlewares/1":                                                 "foobar",
		"traefik/http/routers/Router0/service":                                                       "foobar",
		"traefik/http/routers/Router0/rule":                                                          "foobar",
		"traefik/http/routers/Router0/priority":                                                      "42",
		"traefik/http/routers/Router0/tls":                                                           "",
		"traefik/http/routers/Router1/rule":                                                          "foobar",
		"traefik/http/routers/Router1/priority":                                                      "42",
		"traefik/http/routers/Router1/tls/domains/0/main":                                            "foobar",
		"traefik/http/routers/Router1/tls/domains/0/sans/0":                                          "foobar",
		"traefik/http/routers/Router1/tls/domains/0/sans/1":                                          "foobar",
		"traefik/http/routers/Router1/tls/domains/1/main":                                            "foobar",
		"traefik/http/routers/Router1/tls/domains/1/sans/0":                                          "foobar",
		"traefik/http/routers/Router1/tls/domains/1/sans/1":                                          "foobar",
		"traefik/http/routers/Router1/tls/options":                                                   "foobar",
		"traefik/http/routers/Router1/tls/certResolver":                                              "foobar",
		"traefik/http/routers/Router1/entryPoints/0":                                                 "foobar",
		"traefik/http/routers/Router1/entryPoints/1":                                                 "foobar",
		"traefik/http/routers/Router1/middlewares/0":                                                 "foobar",
		"traefik/http/routers/Router1/middlewares/1":                                                 "foobar",
		"traefik/http/routers/Router1/service":                                                       "foobar",
		"traefik/http/services/Service01/loadBalancer/healthCheck/path":                              "foobar",
		"traefik/http/services/Service01/loadBalancer/healthCheck/port":                              "42",
		"traefik/http/services/Service01/loadBalancer/healthCheck/interval":                          "foobar",
		"traefik/http/services/Service01/loadBalancer/healthCheck/timeout":                           "foobar",
		"traefik/http/services/Service01/loadBalancer/healthCheck/hostname":                          "foobar",
		"traefik/http/services/Service01/loadBalancer/healthCheck/headers/name0":                     "foobar",
		"traefik/http/services/Service01/loadBalancer/healthCheck/headers/name1":                     "foobar",
		"traefik/http/services/Service01/loadBalancer/healthCheck/scheme":                            "foobar",
		"traefik/http/services/Service01/loadBalancer/healthCheck/followredirects":                   "true",
		"traefik/http/services/Service01/loadBalancer/responseForwarding/flushInterval":              "foobar",
		"traefik/http/services/Service01/loadBalancer/passHostHeader":                                "true",
		"traefik/http/services/Service01/loadBalancer/sticky/cookie/name":                            "foobar",
		"traefik/http/services/Service01/loadBalancer/sticky/cookie/secure":                          "true",
		"traefik/http/services/Service01/loadBalancer/sticky/cookie/httpOnly":                        "true",
		"traefik/http/services/Service01/loadBalancer/servers/0/url":                                 "foobar",
		"traefik/http/services/Service01/loadBalancer/servers/1/url":                                 "foobar",
		"traefik/http/services/Service02/mirroring/service":                                          "foobar",
		"traefik/http/services/Service02/mirroring/maxBodySize":                                      "42",
		"traefik/http/services/Service02/mirroring/mirrors/0/name":                                   "foobar",
		"traefik/http/services/Service02/mirroring/mirrors/0/percent":                                "42",
		"traefik/http/services/Service02/mirroring/mirrors/1/name":                                   "foobar",
		"traefik/http/services/Service02/mirroring/mirrors/1/percent":                                "42",
		"traefik/http/services/Service03/weighted/sticky/cookie/name":                                "foobar",
		"traefik/http/services/Service03/weighted/sticky/cookie/secure":                              "true",
		"traefik/http/services/Service03/weighted/sticky/cookie/httpOnly":                            "true",
		"traefik/http/services/Service03/weighted/services/0/name":                                   "foobar",
		"traefik/http/services/Service03/weighted/services/0/weight":                                 "42",
		"traefik/http/services/Service03/weighted/services/1/name":                                   "foobar",
		"traefik/http/services/Service03/weighted/services/1/weight":                                 "42",
		"traefik/http/middlewares/Middleware08/forwardAuth/authResponseHeaders/0":                    "foobar",
		"traefik/http/middlewares/Middleware08/forwardAuth/authResponseHeaders/1":                    "foobar",
		"traefik/http/middlewares/Middleware08/forwardAuth/authRequestHeaders/0":                     "foobar",
		"traefik/http/middlewares/Middleware08/forwardAuth/authRequestHeaders/1":                     "foobar",
		"traefik/http/middlewares/Middleware08/forwardAuth/tls/key":                                  "foobar",
		"traefik/http/middlewares/Middleware08/forwardAuth/tls/insecureSkipVerify":                   "true",
		"traefik/http/middlewares/Middleware08/forwardAuth/tls/ca":                                   "foobar",
		"traefik/http/middlewares/Middleware08/forwardAuth/tls/caOptional":                           "true",
		"traefik/http/middlewares/Middleware08/forwardAuth/tls/cert":                                 "foobar",
		"traefik/http/middlewares/Middleware08/forwardAuth/address":                                  "foobar",
		"traefik/http/middlewares/Middleware08/forwardAuth/trustForwardHeader":                       "true",
		"traefik/http/middlewares/Middleware15/redirectScheme/scheme":                                "foobar",
		"traefik/http/middlewares/Middleware15/redirectScheme/port":                                  "foobar",
		"traefik/http/middlewares/Middleware15/redirectScheme/permanent":                             "true",
		"traefik/http/middlewares/Middleware17/replacePathRegex/regex":                               "foobar",
		"traefik/http/middlewares/Middleware17/replacePathRegex/replacement":                         "foobar",
		"traefik/http/middlewares/Middleware14/redirectRegex/regex":                                  "foobar",
		"traefik/http/middlewares/Middleware14/redirectRegex/replacement":                            "foobar",
		"traefik/http/middlewares/Middleware14/redirectRegex/permanent":                              "true",
		"traefik/http/middlewares/Middleware16/replacePath/path":                                     "foobar",
		"traefik/http/middlewares/Middleware06/digestAuth/removeHeader":                              "true",
		"traefik/http/middlewares/Middleware06/digestAuth/realm":                                     "foobar",
		"traefik/http/middlewares/Middleware06/digestAuth/headerField":                               "foobar",
		"traefik/http/middlewares/Middleware06/digestAuth/users/0":                                   "foobar",
		"traefik/http/middlewares/Middleware06/digestAuth/users/1":                                   "foobar",
		"traefik/http/middlewares/Middleware06/digestAuth/usersFile":                                 "foobar",
		"traefik/http/middlewares/Middleware09/headers/accessControlAllowHeaders/0":                  "foobar",
		"traefik/http/middlewares/Middleware09/headers/accessControlAllowHeaders/1":                  "foobar",
		"traefik/http/middlewares/Middleware09/headers/accessControlAllowOriginList/0":               "foobar",
		"traefik/http/middlewares/Middleware09/headers/accessControlAllowOriginList/1":               "foobar",
		"traefik/http/middlewares/Middleware09/headers/accessControlAllowOriginListRegex/0":          "foobar",
		"traefik/http/middlewares/Middleware09/headers/accessControlAllowOriginListRegex/1":          "foobar",
		"traefik/http/middlewares/Middleware09/headers/contentTypeNosniff":                           "true",
		"traefik/http/middlewares/Middleware09/headers/accessControlAllowCredentials":                "true",
		"traefik/http/middlewares/Middleware09/headers/featurePolicy":                                "foobar",
		"traefik/http/middlewares/Middleware09/headers/permissionsPolicy":                            "foobar",
		"traefik/http/middlewares/Middleware09/headers/forceSTSHeader":                               "true",
		"traefik/http/middlewares/Middleware09/headers/sslRedirect":                                  "true",
		"traefik/http/middlewares/Middleware09/headers/sslHost":                                      "foobar",
		"traefik/http/middlewares/Middleware09/headers/sslForceHost":                                 "true",
		"traefik/http/middlewares/Middleware09/headers/sslProxyHeaders/name1":                        "foobar",
		"traefik/http/middlewares/Middleware09/headers/sslProxyHeaders/name0":                        "foobar",
		"traefik/http/middlewares/Middleware09/headers/allowedHosts/0":                               "foobar",
		"traefik/http/middlewares/Middleware09/headers/allowedHosts/1":                               "foobar",
		"traefik/http/middlewares/Middleware09/headers/stsPreload":                                   "true",
		"traefik/http/middlewares/Middleware09/headers/frameDeny":                                    "true",
		"traefik/http/middlewares/Middleware09/headers/isDevelopment":                                "true",
		"traefik/http/middlewares/Middleware09/headers/customResponseHeaders/name1":                  "foobar",
		"traefik/http/middlewares/Middleware09/headers/customResponseHeaders/name0":                  "foobar",
		"traefik/http/middlewares/Middleware09/headers/accessControlAllowMethods/0":                  "foobar",
		"traefik/http/middlewares/Middleware09/headers/accessControlAllowMethods/1":                  "foobar",
		"traefik/http/middlewares/Middleware09/headers/stsSeconds":                                   "42",
		"traefik/http/middlewares/Middleware09/headers/stsIncludeSubdomains":                         "true",
		"traefik/http/middlewares/Middleware09/headers/customFrameOptionsValue":                      "foobar",
		"traefik/http/middlewares/Middleware09/headers/accessControlMaxAge":                          "42",
		"traefik/http/middlewares/Middleware09/headers/addVaryHeader":                                "true",
		"traefik/http/middlewares/Middleware09/headers/hostsProxyHeaders/0":                          "foobar",
		"traefik/http/middlewares/Middleware09/headers/hostsProxyHeaders/1":                          "foobar",
		"traefik/http/middlewares/Middleware09/headers/sslTemporaryRedirect":                         "true",
		"traefik/http/middlewares/Middleware09/headers/customBrowserXSSValue":                        "foobar",
		"traefik/http/middlewares/Middleware09/headers/referrerPolicy":                               "foobar",
		"traefik/http/middlewares/Middleware09/headers/accessControlExposeHeaders/0":                 "foobar",
		"traefik/http/middlewares/Middleware09/headers/accessControlExposeHeaders/1":                 "foobar",
		"traefik/http/middlewares/Middleware09/headers/contentSecurityPolicy":                        "foobar",
		"traefik/http/middlewares/Middleware09/headers/publicKey":                                    "foobar",
		"traefik/http/middlewares/Middleware09/headers/customRequestHeaders/name0":                   "foobar",
		"traefik/http/middlewares/Middleware09/headers/customRequestHeaders/name1":                   "foobar",
		"traefik/http/middlewares/Middleware09/headers/browserXssFilter":                             "true",
		"traefik/http/middlewares/Middleware10/ipWhiteList/sourceRange/0":                            "foobar",
		"traefik/http/middlewares/Middleware10/ipWhiteList/sourceRange/1":                            "foobar",
		"traefik/http/middlewares/Middleware10/ipWhiteList/ipStrategy/excludedIPs/0":                 "foobar",
		"traefik/http/middlewares/Middleware10/ipWhiteList/ipStrategy/excludedIPs/1":                 "foobar",
		"traefik/http/middlewares/Middleware10/ipWhiteList/ipStrategy/depth":                         "42",
		"traefik/http/middlewares/Middleware11/inFlightReq/amount":                                   "42",
		"traefik/http/middlewares/Middleware11/inFlightReq/sourceCriterion/requestHost":              "true",
		"traefik/http/middlewares/Middleware11/inFlightReq/sourceCriterion/ipStrategy/depth":         "42",
		"traefik/http/middlewares/Middleware11/inFlightReq/sourceCriterion/ipStrategy/excludedIPs/0": "foobar",
		"traefik/http/middlewares/Middleware11/inFlightReq/sourceCriterion/ipStrategy/excludedIPs/1": "foobar",
		"traefik/http/middlewares/Middleware11/inFlightReq/sourceCriterion/requestHeaderName":        "foobar",
		"traefik/http/middlewares/Middleware12/passTLSClientCert/pem":                                "true",
		"traefik/http/middlewares/Middleware12/passTLSClientCert/info/notAfter":                      "true",
		"traefik/http/middlewares/Middleware12/passTLSClientCert/info/notBefore":                     "true",
		"traefik/http/middlewares/Middleware12/passTLSClientCert/info/sans":                          "true",
		"traefik/http/middlewares/Middleware12/passTLSClientCert/info/subject/country":               "true",
		"traefik/http/middlewares/Middleware12/passTLSClientCert/info/subject/province":              "true",
		"traefik/http/middlewares/Middleware12/passTLSClientCert/info/subject/locality":              "true",
		"traefik/http/middlewares/Middleware12/passTLSClientCert/info/subject/organization":          "true",
		"traefik/http/middlewares/Middleware12/passTLSClientCert/info/subject/organizationalunit":    "true",
		"traefik/http/middlewares/Middleware12/passTLSClientCert/info/subject/commonName":            "true",
		"traefik/http/middlewares/Middleware12/passTLSClientCert/info/subject/serialNumber":          "true",
		"traefik/http/middlewares/Middleware12/passTLSClientCert/info/subject/domainComponent":       "true",
		"traefik/http/middlewares/Middleware12/passTLSClientCert/info/issuer/country":                "true",
		"traefik/http/middlewares/Middleware12/passTLSClientCert/info/issuer/province":               "true",
		"traefik/http/middlewares/Middleware12/passTLSClientCert/info/issuer/locality":               "true",
		"traefik/http/middlewares/Middleware12/passTLSClientCert/info/issuer/organization":           "true",
		"traefik/http/middlewares/Middleware12/passTLSClientCert/info/issuer/commonName":             "true",
		"traefik/http/middlewares/Middleware12/passTLSClientCert/info/issuer/serialNumber":           "true",
		"traefik/http/middlewares/Middleware12/passTLSClientCert/info/issuer/domainComponent":        "true",
		"traefik/http/middlewares/Middleware00/addPrefix/prefix":                                     "foobar",
		"traefik/http/middlewares/Middleware03/chain/middlewares/0":                                  "foobar",
		"traefik/http/middlewares/Middleware03/chain/middlewares/1":                                  "foobar",
		"traefik/http/middlewares/Middleware04/circuitBreaker/expression":                            "foobar",
		"traefik/http/middlewares/Middleware07/errors/status/0":                                      "foobar",
		"traefik/http/middlewares/Middleware07/errors/status/1":                                      "foobar",
		"traefik/http/middlewares/Middleware07/errors/service":                                       "foobar",
		"traefik/http/middlewares/Middleware07/errors/query":                                         "foobar",
		"traefik/http/middlewares/Middleware13/rateLimit/average":                                    "42",
		"traefik/http/middlewares/Middleware13/rateLimit/period":                                     "1s",
		"traefik/http/middlewares/Middleware13/rateLimit/burst":                                      "42",
		"traefik/http/middlewares/Middleware13/rateLimit/sourceCriterion/requestHeaderName":          "foobar",
		"traefik/http/middlewares/Middleware13/rateLimit/sourceCriterion/requestHost":                "true",
		"traefik/http/middlewares/Middleware13/rateLimit/sourceCriterion/ipStrategy/depth":           "42",
		"traefik/http/middlewares/Middleware13/rateLimit/sourceCriterion/ipStrategy/excludedIPs/0":   "foobar",
		"traefik/http/middlewares/Middleware13/rateLimit/sourceCriterion/ipStrategy/excludedIPs/1":   "foobar",
		"traefik/http/middlewares/Middleware20/stripPrefixRegex/regex/0":                             "foobar",
		"traefik/http/middlewares/Middleware20/stripPrefixRegex/regex/1":                             "foobar",
		"traefik/http/middlewares/Middleware01/basicAuth/users/0":                                    "foobar",
		"traefik/http/middlewares/Middleware01/basicAuth/users/1":                                    "foobar",
		"traefik/http/middlewares/Middleware01/basicAuth/usersFile":                                  "foobar",
		"traefik/http/middlewares/Middleware01/basicAuth/realm":                                      "foobar",
		"traefik/http/middlewares/Middleware01/basicAuth/removeHeader":                               "true",
		"traefik/http/middlewares/Middleware01/basicAuth/headerField":                                "foobar",
		"traefik/http/middlewares/Middleware02/buffering/maxResponseBodyBytes":                       "42",
		"traefik/http/middlewares/Middleware02/buffering/memResponseBodyBytes":                       "42",
		"traefik/http/middlewares/Middleware02/buffering/retryExpression":                            "foobar",
		"traefik/http/middlewares/Middleware02/buffering/maxRequestBodyBytes":                        "42",
		"traefik/http/middlewares/Middleware02/buffering/memRequestBodyBytes":                        "42",
		"traefik/http/middlewares/Middleware05/compress":                                             "",
		"traefik/http/middlewares/Middleware18/retry/attempts":                                       "42",
		"traefik/http/middlewares/Middleware19/stripPrefix/prefixes/0":                               "foobar",
		"traefik/http/middlewares/Middleware19/stripPrefix/prefixes/1":                               "foobar",
		"traefik/http/middlewares/Middleware19/stripPrefix/forceSlash":                               "true",
		"traefik/tcp/routers/TCPRouter0/entryPoints/0":                                               "foobar",
		"traefik/tcp/routers/TCPRouter0/entryPoints/1":                                               "foobar",
		"traefik/tcp/routers/TCPRouter0/service":                                                     "foobar",
		"traefik/tcp/routers/TCPRouter0/rule":                                                        "foobar",
		"traefik/tcp/routers/TCPRouter0/tls/options":                                                 "foobar",
		"traefik/tcp/routers/TCPRouter0/tls/certResolver":                                            "foobar",
		"traefik/tcp/routers/TCPRouter0/tls/domains/0/main":                                          "foobar",
		"traefik/tcp/routers/TCPRouter0/tls/domains/0/sans/0":                                        "foobar",
		"traefik/tcp/routers/TCPRouter0/tls/domains/0/sans/1":                                        "foobar",
		"traefik/tcp/routers/TCPRouter0/tls/domains/1/main":                                          "foobar",
		"traefik/tcp/routers/TCPRouter0/tls/domains/1/sans/0":                                        "foobar",
		"traefik/tcp/routers/TCPRouter0/tls/domains/1/sans/1":                                        "foobar",
		"traefik/tcp/routers/TCPRouter0/tls/passthrough":                                             "true",
		"traefik/tcp/routers/TCPRouter1/entryPoints/0":                                               "foobar",
		"traefik/tcp/routers/TCPRouter1/entryPoints/1":                                               "foobar",
		"traefik/tcp/routers/TCPRouter1/service":                                                     "foobar",
		"traefik/tcp/routers/TCPRouter1/rule":                                                        "foobar",
		"traefik/tcp/routers/TCPRouter1/tls/domains/0/main":                                          "foobar",
		"traefik/tcp/routers/TCPRouter1/tls/domains/0/sans/0":                                        "foobar",
		"traefik/tcp/routers/TCPRouter1/tls/domains/0/sans/1":                                        "foobar",
		"traefik/tcp/routers/TCPRouter1/tls/domains/1/main":                                          "foobar",
		"traefik/tcp/routers/TCPRouter1/tls/domains/1/sans/0":                                        "foobar",
		"traefik/tcp/routers/TCPRouter1/tls/domains/1/sans/1":                                        "foobar",
		"traefik/tcp/routers/TCPRouter1/tls/passthrough":                                             "true",
		"traefik/tcp/routers/TCPRouter1/tls/options":                                                 "foobar",
		"traefik/tcp/routers/TCPRouter1/tls/certResolver":                                            "foobar",
		"traefik/tcp/services/TCPService01/loadBalancer/terminationDelay":                            "42",
		"traefik/tcp/services/TCPService01/loadBalancer/servers/0/address":                           "foobar",
		"traefik/tcp/services/TCPService01/loadBalancer/servers/1/address":                           "foobar",
		"traefik/tcp/services/TCPService02/weighted/services/0/name":                                 "foobar",
		"traefik/tcp/services/TCPService02/weighted/services/0/weight":                               "42",
		"traefik/tcp/services/TCPService02/weighted/services/1/name":                                 "foobar",
		"traefik/tcp/services/TCPService02/weighted/services/1/weight":                               "43",
		"traefik/udp/routers/UDPRouter0/entrypoints/0":                                               "foobar",
		"traefik/udp/routers/UDPRouter0/entrypoints/1":                                               "foobar",
		"traefik/udp/routers/UDPRouter0/service":                                                     "foobar",
		"traefik/udp/routers/UDPRouter1/entrypoints/0":                                               "foobar",
		"traefik/udp/routers/UDPRouter1/entrypoints/1":                                               "foobar",
		"traefik/udp/routers/UDPRouter1/service":                                                     "foobar",
		"traefik/udp/services/UDPService01/loadBalancer/servers/0/address":                           "foobar",
		"traefik/udp/services/UDPService01/loadBalancer/servers/1/address":                           "foobar",
		"traefik/udp/services/UDPService02/loadBalancer/servers/0/address":                           "foobar",
		"traefik/udp/services/UDPService02/loadBalancer/servers/1/address":                           "foobar",
		"traefik/tls/options/Options0/minVersion":                                                    "foobar",
		"traefik/tls/options/Options0/maxVersion":                                                    "foobar",
		"traefik/tls/options/Options0/cipherSuites/0":                                                "foobar",
		"traefik/tls/options/Options0/cipherSuites/1":                                                "foobar",
		"traefik/tls/options/Options0/sniStrict":                                                     "true",
		"traefik/tls/options/Options0/curvePreferences/0":                                            "foobar",
		"traefik/tls/options/Options0/curvePreferences/1":                                            "foobar",
		"traefik/tls/options/Options0/clientAuth/caFiles/0":                                          "foobar",
		"traefik/tls/options/Options0/clientAuth/caFiles/1":                                          "foobar",
		"traefik/tls/options/Options0/clientAuth/clientAuthType":                                     "foobar",
		"traefik/tls/options/Options1/sniStrict":                                                     "true",
		"traefik/tls/options/Options1/curvePreferences/0":                                            "foobar",
		"traefik/tls/options/Options1/curvePreferences/1":                                            "foobar",
		"traefik/tls/options/Options1/clientAuth/caFiles/0":                                          "foobar",
		"traefik/tls/options/Options1/clientAuth/caFiles/1":                                          "foobar",
		"traefik/tls/options/Options1/clientAuth/clientAuthType":                                     "foobar",
		"traefik/tls/options/Options1/minVersion":                                                    "foobar",
		"traefik/tls/options/Options1/maxVersion":                                                    "foobar",
		"traefik/tls/options/Options1/cipherSuites/0":                                                "foobar",
		"traefik/tls/options/Options1/cipherSuites/1":                                                "foobar",
		"traefik/tls/stores/Store0/defaultCertificate/certFile":                                      "foobar",
		"traefik/tls/stores/Store0/defaultCertificate/keyFile":                                       "foobar",
		"traefik/tls/stores/Store1/defaultCertificate/certFile":                                      "foobar",
		"traefik/tls/stores/Store1/defaultCertificate/keyFile":                                       "foobar",
		"traefik/tls/certificates/0/certFile":                                                        "foobar",
		"traefik/tls/certificates/0/keyFile":                                                         "foobar",
		"traefik/tls/certificates/0/stores/0":                                                        "foobar",
		"traefik/tls/certificates/0/stores/1":                                                        "foobar",
		"traefik/tls/certificates/1/certFile":                                                        "foobar",
		"traefik/tls/certificates/1/keyFile":                                                         "foobar",
		"traefik/tls/certificates/1/stores/0":                                                        "foobar",
		"traefik/tls/certificates/1/stores/1":                                                        "foobar",
	}))

	cfg, err := provider.buildConfiguration()
	require.NoError(t, err)

	expected := &dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Routers: map[string]*dynamic.Router{
				"Router1": {
					EntryPoints: []string{
						"foobar",
						"foobar",
					},
					Middlewares: []string{
						"foobar",
						"foobar",
					},
					Service:  "foobar",
					Rule:     "foobar",
					Priority: 42,
					TLS: &dynamic.RouterTLSConfig{
						Options:      "foobar",
						CertResolver: "foobar",
						Domains: []types.Domain{
							{
								Main: "foobar",
								SANs: []string{
									"foobar",
									"foobar",
								},
							},
							{
								Main: "foobar",
								SANs: []string{
									"foobar",
									"foobar",
								},
							},
						},
					},
				},
				"Router0": {
					EntryPoints: []string{
						"foobar",
						"foobar",
					},
					Middlewares: []string{
						"foobar",
						"foobar",
					},
					Service:  "foobar",
					Rule:     "foobar",
					Priority: 42,
					TLS:      &dynamic.RouterTLSConfig{},
				},
			},
			Middlewares: map[string]*dynamic.Middleware{
				"Middleware10": {
					IPWhiteList: &dynamic.IPWhiteList{
						SourceRange: []string{
							"foobar",
							"foobar",
						},
						IPStrategy: &dynamic.IPStrategy{
							Depth: 42,
							ExcludedIPs: []string{
								"foobar",
								"foobar",
							},
						},
					},
				},
				"Middleware13": {
					RateLimit: &dynamic.RateLimit{
						Average: 42,
						Burst:   42,
						Period:  ptypes.Duration(time.Second),
						SourceCriterion: &dynamic.SourceCriterion{
							IPStrategy: &dynamic.IPStrategy{
								Depth: 42,
								ExcludedIPs: []string{
									"foobar",
									"foobar",
								},
							},
							RequestHeaderName: "foobar",
							RequestHost:       true,
						},
					},
				},
				"Middleware19": {
					StripPrefix: &dynamic.StripPrefix{
						Prefixes: []string{
							"foobar",
							"foobar",
						},
						ForceSlash: true,
					},
				},
				"Middleware00": {
					AddPrefix: &dynamic.AddPrefix{
						Prefix: "foobar",
					},
				},
				"Middleware02": {
					Buffering: &dynamic.Buffering{
						MaxRequestBodyBytes:  42,
						MemRequestBodyBytes:  42,
						MaxResponseBodyBytes: 42,
						MemResponseBodyBytes: 42,
						RetryExpression:      "foobar",
					},
				},
				"Middleware04": {
					CircuitBreaker: &dynamic.CircuitBreaker{
						Expression: "foobar",
					},
				},
				"Middleware05": {
					Compress: &dynamic.Compress{},
				},
				"Middleware08": {
					ForwardAuth: &dynamic.ForwardAuth{
						Address: "foobar",
						TLS: &dynamic.ClientTLS{
							CA:                 "foobar",
							CAOptional:         true,
							Cert:               "foobar",
							Key:                "foobar",
							InsecureSkipVerify: true,
						},
						TrustForwardHeader: true,
						AuthResponseHeaders: []string{
							"foobar",
							"foobar",
						},
						AuthRequestHeaders: []string{
							"foobar",
							"foobar",
						},
					},
				},
				"Middleware06": {
					DigestAuth: &dynamic.DigestAuth{
						Users: dynamic.Users{
							"foobar",
							"foobar",
						},
						UsersFile:    "foobar",
						RemoveHeader: true,
						Realm:        "foobar",
						HeaderField:  "foobar",
					},
				},
				"Middleware18": {
					Retry: &dynamic.Retry{
						Attempts: 42,
					},
				},
				"Middleware16": {
					ReplacePath: &dynamic.ReplacePath{
						Path: "foobar",
					},
				},
				"Middleware20": {
					StripPrefixRegex: &dynamic.StripPrefixRegex{
						Regex: []string{
							"foobar",
							"foobar",
						},
					},
				},
				"Middleware03": {
					Chain: &dynamic.Chain{
						Middlewares: []string{
							"foobar",
							"foobar",
						},
					},
				},
				"Middleware11": {
					InFlightReq: &dynamic.InFlightReq{
						Amount: 42,
						SourceCriterion: &dynamic.SourceCriterion{
							IPStrategy: &dynamic.IPStrategy{
								Depth: 42,
								ExcludedIPs: []string{
									"foobar",
									"foobar",
								},
							},
							RequestHeaderName: "foobar",
							RequestHost:       true,
						},
					},
				},
				"Middleware12": {
					PassTLSClientCert: &dynamic.PassTLSClientCert{
						PEM: true,
						Info: &dynamic.TLSClientCertificateInfo{
							NotAfter:  true,
							NotBefore: true,
							Sans:      true,
							Subject: &dynamic.TLSCLientCertificateSubjectDNInfo{
								Country:            true,
								Province:           true,
								Locality:           true,
								Organization:       true,
								OrganizationalUnit: true,
								CommonName:         true,
								SerialNumber:       true,
								DomainComponent:    true,
							},
							Issuer: &dynamic.TLSCLientCertificateIssuerDNInfo{
								Country:         true,
								Province:        true,
								Locality:        true,
								Organization:    true,
								CommonName:      true,
								SerialNumber:    true,
								DomainComponent: true,
							},
						},
					},
				},
				"Middleware14": {
					RedirectRegex: &dynamic.RedirectRegex{
						Regex:       "foobar",
						Replacement: "foobar",
						Permanent:   true,
					},
				},
				"Middleware15": {
					RedirectScheme: &dynamic.RedirectScheme{
						Scheme:    "foobar",
						Port:      "foobar",
						Permanent: true,
					},
				},
				"Middleware01": {
					BasicAuth: &dynamic.BasicAuth{
						Users: dynamic.Users{
							"foobar",
							"foobar",
						},
						UsersFile:    "foobar",
						Realm:        "foobar",
						RemoveHeader: true,
						HeaderField:  "foobar",
					},
				},
				"Middleware07": {
					Errors: &dynamic.ErrorPage{
						Status: []string{
							"foobar",
							"foobar",
						},
						Service: "foobar",
						Query:   "foobar",
					},
				},
				"Middleware09": {
					Headers: &dynamic.Headers{
						CustomRequestHeaders: map[string]string{
							"name0": "foobar",
							"name1": "foobar",
						},
						CustomResponseHeaders: map[string]string{
							"name0": "foobar",
							"name1": "foobar",
						},
						AccessControlAllowCredentials: true,
						AccessControlAllowHeaders: []string{
							"foobar",
							"foobar",
						},
						AccessControlAllowMethods: []string{
							"foobar",
							"foobar",
						},
						AccessControlAllowOriginList: []string{
							"foobar",
							"foobar",
						},
						AccessControlAllowOriginListRegex: []string{
							"foobar",
							"foobar",
						},
						AccessControlExposeHeaders: []string{
							"foobar",
							"foobar",
						},
						AccessControlMaxAge: 42,
						AddVaryHeader:       true,
						AllowedHosts: []string{
							"foobar",
							"foobar",
						},
						HostsProxyHeaders: []string{
							"foobar",
							"foobar",
						},
						SSLRedirect:          true,
						SSLTemporaryRedirect: true,
						SSLHost:              "foobar",
						SSLProxyHeaders: map[string]string{
							"name1": "foobar",
							"name0": "foobar",
						},
						SSLForceHost:            true,
						STSSeconds:              42,
						STSIncludeSubdomains:    true,
						STSPreload:              true,
						ForceSTSHeader:          true,
						FrameDeny:               true,
						CustomFrameOptionsValue: "foobar",
						ContentTypeNosniff:      true,
						BrowserXSSFilter:        true,
						CustomBrowserXSSValue:   "foobar",
						ContentSecurityPolicy:   "foobar",
						PublicKey:               "foobar",
						ReferrerPolicy:          "foobar",
						FeaturePolicy:           "foobar",
						PermissionsPolicy:       "foobar",
						IsDevelopment:           true,
					},
				},
				"Middleware17": {
					ReplacePathRegex: &dynamic.ReplacePathRegex{
						Regex:       "foobar",
						Replacement: "foobar",
					},
				},
			},
			Services: map[string]*dynamic.Service{
				"Service01": {
					LoadBalancer: &dynamic.ServersLoadBalancer{
						Sticky: &dynamic.Sticky{
							Cookie: &dynamic.Cookie{
								Name:     "foobar",
								Secure:   true,
								HTTPOnly: true,
							},
						},
						Servers: []dynamic.Server{
							{
								URL:    "foobar",
								Scheme: "http",
							},
							{
								URL:    "foobar",
								Scheme: "http",
							},
						},
						HealthCheck: &dynamic.ServerHealthCheck{
							Scheme:          "foobar",
							Path:            "foobar",
							Port:            42,
							Interval:        "foobar",
							Timeout:         "foobar",
							Hostname:        "foobar",
							FollowRedirects: func(v bool) *bool { return &v }(true),
							Headers: map[string]string{
								"name0": "foobar",
								"name1": "foobar",
							},
						},
						PassHostHeader: func(v bool) *bool { return &v }(true),
						ResponseForwarding: &dynamic.ResponseForwarding{
							FlushInterval: "foobar",
						},
					},
				},
				"Service02": {
					Mirroring: &dynamic.Mirroring{
						Service:     "foobar",
						MaxBodySize: func(v int64) *int64 { return &v }(42),
						Mirrors: []dynamic.MirrorService{
							{
								Name:    "foobar",
								Percent: 42,
							},
							{
								Name:    "foobar",
								Percent: 42,
							},
						},
					},
				},
				"Service03": {
					Weighted: &dynamic.WeightedRoundRobin{
						Services: []dynamic.WRRService{
							{
								Name:   "foobar",
								Weight: func(v int) *int { return &v }(42),
							},
							{
								Name:   "foobar",
								Weight: func(v int) *int { return &v }(42),
							},
						},
						Sticky: &dynamic.Sticky{
							Cookie: &dynamic.Cookie{
								Name:     "foobar",
								Secure:   true,
								HTTPOnly: true,
							},
						},
					},
				},
			},
		},
		TCP: &dynamic.TCPConfiguration{
			Routers: map[string]*dynamic.TCPRouter{
				"TCPRouter0": {
					EntryPoints: []string{
						"foobar",
						"foobar",
					},
					Service: "foobar",
					Rule:    "foobar",
					TLS: &dynamic.RouterTCPTLSConfig{
						Passthrough:  true,
						Options:      "foobar",
						CertResolver: "foobar",
						Domains: []types.Domain{
							{
								Main: "foobar",
								SANs: []string{
									"foobar",
									"foobar",
								},
							},
							{
								Main: "foobar",
								SANs: []string{
									"foobar",
									"foobar",
								},
							},
						},
					},
				},
				"TCPRouter1": {
					EntryPoints: []string{
						"foobar",
						"foobar",
					},
					Service: "foobar",
					Rule:    "foobar",
					TLS: &dynamic.RouterTCPTLSConfig{
						Passthrough:  true,
						Options:      "foobar",
						CertResolver: "foobar",
						Domains: []types.Domain{
							{
								Main: "foobar",
								SANs: []string{
									"foobar",
									"foobar",
								},
							},
							{
								Main: "foobar",
								SANs: []string{
									"foobar",
									"foobar",
								},
							},
						},
					},
				},
			},
			Services: map[string]*dynamic.TCPService{
				"TCPService01": {
					LoadBalancer: &dynamic.TCPServersLoadBalancer{
						TerminationDelay: func(v int) *int { return &v }(42),
						Servers: []dynamic.TCPServer{
							{Address: "foobar"},
							{Address: "foobar"},
						},
					},
				},
				"TCPService02": {
					Weighted: &dynamic.TCPWeightedRoundRobin{
						Services: []dynamic.TCPWRRService{
							{
								Name:   "foobar",
								Weight: func(v int) *int { return &v }(42),
							},
							{
								Name:   "foobar",
								Weight: func(v int) *int { return &v }(43),
							},
						},
					},
				},
			},
		},
		UDP: &dynamic.UDPConfiguration{
			Routers: map[string]*dynamic.UDPRouter{
				"UDPRouter0": {
					EntryPoints: []string{"foobar", "foobar"},
					Service:     "foobar",
				},
				"UDPRouter1": {
					EntryPoints: []string{"foobar", "foobar"},
					Service:     "foobar",
				},
			},
			Services: map[string]*dynamic.UDPService{
				"UDPService01": {
					LoadBalancer: &dynamic.UDPServersLoadBalancer{
						Servers: []dynamic.UDPServer{
							{Address: "foobar"},
							{Address: "foobar"},
						},
					},
				},
				"UDPService02": {
					LoadBalancer: &dynamic.UDPServersLoadBalancer{
						Servers: []dynamic.UDPServer{
							{Address: "foobar"},
							{Address: "foobar"},
						},
					},
				},
			},
		},
		TLS: &dynamic.TLSConfiguration{
			Certificates: []*tls.CertAndStores{
				{
					Certificate: tls.Certificate{
						CertFile: tls.FileOrContent("foobar"),
						KeyFile:  tls.FileOrContent("foobar"),
					},
					Stores: []string{
						"foobar",
						"foobar",
					},
				},
				{
					Certificate: tls.Certificate{
						CertFile: tls.FileOrContent("foobar"),
						KeyFile:  tls.FileOrContent("foobar"),
					},
					Stores: []string{
						"foobar",
						"foobar",
					},
				},
			},
			Options: map[string]tls.Options{
				"Options0": {
					MinVersion: "foobar",
					MaxVersion: "foobar",
					CipherSuites: []string{
						"foobar",
						"foobar",
					},
					CurvePreferences: []string{
						"foobar",
						"foobar",
					},
					ClientAuth: tls.ClientAuth{
						CAFiles: []tls.FileOrContent{
							tls.FileOrContent("foobar"),
							tls.FileOrContent("foobar"),
						},
						ClientAuthType: "foobar",
					},
					SniStrict: true,
				},
				"Options1": {
					MinVersion: "foobar",
					MaxVersion: "foobar",
					CipherSuites: []string{
						"foobar",
						"foobar",
					},
					CurvePreferences: []string{
						"foobar",
						"foobar",
					},
					ClientAuth: tls.ClientAuth{
						CAFiles: []tls.FileOrContent{
							tls.FileOrContent("foobar"),
							tls.FileOrContent("foobar"),
						},
						ClientAuthType: "foobar",
					},
					SniStrict: true,
				},
			},
			Stores: map[string]tls.Store{
				"Store0": {
					DefaultCertificate: &tls.Certificate{
						CertFile: tls.FileOrContent("foobar"),
						KeyFile:  tls.FileOrContent("foobar"),
					},
				},
				"Store1": {
					DefaultCertificate: &tls.Certificate{
						CertFile: tls.FileOrContent("foobar"),
						KeyFile:  tls.FileOrContent("foobar"),
					},
				},
			},
		},
	}

	assert.Equal(t, expected, cfg)
}

func Test_buildConfiguration_KV_error(t *testing.T) {
	provider := &Provider{
		RootKey: "traefik",
		kvClient: &Mock{
			Error: KvError{
				List: errors.New("OOPS"),
			},
			KVPairs: mapToPairs(map[string]string{
				"traefik/foo": "bar",
			}),
		},
	}

	cfg, err := provider.buildConfiguration()
	require.Error(t, err)
	assert.Nil(t, cfg)
}

func TestKvWatchTree(t *testing.T) {
	returnedChans := make(chan chan []*store.KVPair)
	provider := Provider{
		kvClient: &Mock{
			WatchTreeMethod: func() <-chan []*store.KVPair {
				c := make(chan []*store.KVPair, 10)
				returnedChans <- c
				return c
			},
		},
	}

	configChan := make(chan dynamic.Message)
	go func() {
		err := provider.watchKv(context.Background(), configChan)
		require.NoError(t, err)
	}()

	select {
	case c1 := <-returnedChans:
		c1 <- []*store.KVPair{}
		<-configChan
		close(c1) // WatchTree chans can close due to error
	case <-time.After(1 * time.Second):
		t.Fatalf("Failed to create a new WatchTree chan")
	}

	select {
	case c2 := <-returnedChans:
		c2 <- []*store.KVPair{}
		<-configChan
	case <-time.After(1 * time.Second):
		t.Fatalf("Failed to create a new WatchTree chan")
	}

	select {
	case <-configChan:
		t.Fatalf("configChan should be empty")
	default:
	}
}

func mapToPairs(in map[string]string) []*store.KVPair {
	var out []*store.KVPair
	for k, v := range in {
		out = append(out, &store.KVPair{Key: k, Value: []byte(v)})
	}
	return out
}
