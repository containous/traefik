package integration

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-check/check"
	"github.com/miekg/dns"
	"github.com/traefik/traefik/v2/integration/try"
	"github.com/traefik/traefik/v2/pkg/config/static"
	"github.com/traefik/traefik/v2/pkg/provider/acme"
	"github.com/traefik/traefik/v2/pkg/testhelpers"
	"github.com/traefik/traefik/v2/pkg/types"
	checker "github.com/vdemeester/shakers"
)

// ACME test suites (using libcompose).
type AcmeSuite struct {
	BaseSuite
	pebbleIP      string
	fakeDNSServer *dns.Server
}

type subCases struct {
	host               string
	expectedCommonName string
	expectedAlgorithm  x509.PublicKeyAlgorithm
}

type acmeTestCase struct {
	template            templateModel
	traefikConfFilePath string
	subCases            []subCases
}

type templateModel struct {
	Domains   []types.Domain
	PortHTTP  string
	PortHTTPS string
	Acme      map[string]static.CertificateResolver
}

const (
	// Domain to check
	acmeDomain = "traefik.acme.wtf"

	// Wildcard domain to check
	wildcardDomain = "*.acme.wtf"
)

func (s *AcmeSuite) getAcmeURL() string {
	return fmt.Sprintf("https://%s:14000/dir", s.pebbleIP)
}

func setupPebbleRootCA() (*http.Transport, error) {
	path, err := filepath.Abs("fixtures/acme/ssl/pebble.minica.pem")
	if err != nil {
		return nil, err
	}

	os.Setenv("LEGO_CA_CERTIFICATES", path)
	os.Setenv("LEGO_CA_SERVER_NAME", "pebble")

	customCAs, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(customCAs); !ok {
		return nil, fmt.Errorf("error creating x509 cert pool from %q: %w", path, err)
	}

	return &http.Transport{
		TLSClientConfig: &tls.Config{
			ServerName: "pebble",
			RootCAs:    certPool,
		},
	}, nil
}

func (s *AcmeSuite) SetUpSuite(c *check.C) {
	s.createComposeProject(c, "pebble")
	s.composeProject.Start(c)

	s.fakeDNSServer = startFakeDNSServer()

	s.pebbleIP = s.composeProject.Container(c, "pebble").NetworkSettings.IPAddress

	pebbleTransport, err := setupPebbleRootCA()
	if err != nil {
		c.Fatal(err)
	}

	// wait for pebble
	req := testhelpers.MustNewRequest(http.MethodGet, s.getAcmeURL(), nil)

	client := &http.Client{
		Transport: pebbleTransport,
	}

	err = try.Do(5*time.Second, func() error {
		resp, errGet := client.Do(req)
		if errGet != nil {
			return errGet
		}
		return try.StatusCodeIs(http.StatusOK)(resp)
	})
	c.Assert(err, checker.IsNil)
}

func (s *AcmeSuite) TearDownSuite(c *check.C) {
	err := s.fakeDNSServer.Shutdown()
	if err != nil {
		c.Log(err)
	}

	// shutdown and delete compose project
	if s.composeProject != nil {
		s.composeProject.Stop(c)
	}
}

func (s *AcmeSuite) TestHTTP01Domains(c *check.C) {
	testCase := acmeTestCase{
		traefikConfFilePath: "fixtures/acme/acme_domains.toml",
		subCases: []subCases{{
			host:               acmeDomain,
			expectedCommonName: acmeDomain,
			expectedAlgorithm:  x509.RSA,
		}},
		template: templateModel{
			Domains: []types.Domain{{
				Main: "traefik.acme.wtf",
			}},
			Acme: map[string]static.CertificateResolver{
				"default": {ACME: &acme.Configuration{
					HTTPChallenge: &acme.HTTPChallenge{EntryPoint: "web"},
				}},
			},
		},
	}

	s.retrieveAcmeCertificate(c, testCase)
}

func (s *AcmeSuite) TestHTTP01DomainsInSAN(c *check.C) {
	testCase := acmeTestCase{
		traefikConfFilePath: "fixtures/acme/acme_domains.toml",
		subCases: []subCases{{
			host:               acmeDomain,
			expectedCommonName: "acme.wtf",
			expectedAlgorithm:  x509.RSA,
		}},
		template: templateModel{
			Domains: []types.Domain{{
				Main: "acme.wtf",
				SANs: []string{"traefik.acme.wtf"},
			}},
			Acme: map[string]static.CertificateResolver{
				"default": {ACME: &acme.Configuration{
					HTTPChallenge: &acme.HTTPChallenge{EntryPoint: "web"},
				}},
			},
		},
	}

	s.retrieveAcmeCertificate(c, testCase)
}

func (s *AcmeSuite) TestHTTP01OnHostRule(c *check.C) {
	testCase := acmeTestCase{
		traefikConfFilePath: "fixtures/acme/acme_base.toml",
		subCases: []subCases{{
			host:               acmeDomain,
			expectedCommonName: acmeDomain,
			expectedAlgorithm:  x509.RSA,
		}},
		template: templateModel{
			Acme: map[string]static.CertificateResolver{
				"default": {ACME: &acme.Configuration{
					HTTPChallenge: &acme.HTTPChallenge{EntryPoint: "web"},
				}},
			},
		},
	}

	s.retrieveAcmeCertificate(c, testCase)
}

func (s *AcmeSuite) TestMultipleResolver(c *check.C) {
	testCase := acmeTestCase{
		traefikConfFilePath: "fixtures/acme/acme_multiple_resolvers.toml",
		subCases: []subCases{
			{
				host:               acmeDomain,
				expectedCommonName: acmeDomain,
				expectedAlgorithm:  x509.RSA,
			},
			{
				host:               "tchouk.acme.wtf",
				expectedCommonName: "tchouk.acme.wtf",
				expectedAlgorithm:  x509.ECDSA,
			},
		},
		template: templateModel{
			Acme: map[string]static.CertificateResolver{
				"default": {ACME: &acme.Configuration{
					HTTPChallenge: &acme.HTTPChallenge{EntryPoint: "web"},
				}},
				"tchouk": {ACME: &acme.Configuration{
					TLSChallenge: &acme.TLSChallenge{},
					KeyType:      "EC256",
				}},
			},
		},
	}

	s.retrieveAcmeCertificate(c, testCase)
}

func (s *AcmeSuite) TestHTTP01OnHostRuleECDSA(c *check.C) {
	testCase := acmeTestCase{
		traefikConfFilePath: "fixtures/acme/acme_base.toml",
		subCases: []subCases{{
			host:               acmeDomain,
			expectedCommonName: acmeDomain,
			expectedAlgorithm:  x509.ECDSA,
		}},
		template: templateModel{
			Acme: map[string]static.CertificateResolver{
				"default": {ACME: &acme.Configuration{
					HTTPChallenge: &acme.HTTPChallenge{EntryPoint: "web"},
					KeyType:       "EC384",
				}},
			},
		},
	}

	s.retrieveAcmeCertificate(c, testCase)
}

func (s *AcmeSuite) TestHTTP01OnHostRuleInvalidAlgo(c *check.C) {
	testCase := acmeTestCase{
		traefikConfFilePath: "fixtures/acme/acme_base.toml",
		subCases: []subCases{{
			host:               acmeDomain,
			expectedCommonName: acmeDomain,
			expectedAlgorithm:  x509.RSA,
		}},
		template: templateModel{
			Acme: map[string]static.CertificateResolver{
				"default": {ACME: &acme.Configuration{
					HTTPChallenge: &acme.HTTPChallenge{EntryPoint: "web"},
					KeyType:       "INVALID",
				}},
			},
		},
	}

	s.retrieveAcmeCertificate(c, testCase)
}

func (s *AcmeSuite) TestHTTP01OnHostRuleDefaultDynamicCertificatesWithWildcard(c *check.C) {
	testCase := acmeTestCase{
		traefikConfFilePath: "fixtures/acme/acme_tls.toml",
		subCases: []subCases{{
			host:               acmeDomain,
			expectedCommonName: wildcardDomain,
			expectedAlgorithm:  x509.RSA,
		}},
		template: templateModel{
			Acme: map[string]static.CertificateResolver{
				"default": {ACME: &acme.Configuration{
					HTTPChallenge: &acme.HTTPChallenge{EntryPoint: "web"},
				}},
			},
		},
	}

	s.retrieveAcmeCertificate(c, testCase)
}

func (s *AcmeSuite) TestHTTP01OnHostRuleDynamicCertificatesWithWildcard(c *check.C) {
	testCase := acmeTestCase{
		traefikConfFilePath: "fixtures/acme/acme_tls_dynamic.toml",
		subCases: []subCases{{
			host:               acmeDomain,
			expectedCommonName: wildcardDomain,
			expectedAlgorithm:  x509.RSA,
		}},
		template: templateModel{
			Acme: map[string]static.CertificateResolver{
				"default": {ACME: &acme.Configuration{
					HTTPChallenge: &acme.HTTPChallenge{EntryPoint: "web"},
				}},
			},
		},
	}

	s.retrieveAcmeCertificate(c, testCase)
}

func (s *AcmeSuite) TestTLSALPN01OnHostRuleTCP(c *check.C) {
	testCase := acmeTestCase{
		traefikConfFilePath: "fixtures/acme/acme_tcp.toml",
		subCases: []subCases{{
			host:               acmeDomain,
			expectedCommonName: acmeDomain,
			expectedAlgorithm:  x509.RSA,
		}},
		template: templateModel{
			Acme: map[string]static.CertificateResolver{
				"default": {ACME: &acme.Configuration{
					TLSChallenge: &acme.TLSChallenge{},
				}},
			},
		},
	}

	s.retrieveAcmeCertificate(c, testCase)
}

func (s *AcmeSuite) TestTLSALPN01OnHostRule(c *check.C) {
	testCase := acmeTestCase{
		traefikConfFilePath: "fixtures/acme/acme_base.toml",
		subCases: []subCases{{
			host:               acmeDomain,
			expectedCommonName: acmeDomain,
			expectedAlgorithm:  x509.RSA,
		}},
		template: templateModel{
			Acme: map[string]static.CertificateResolver{
				"default": {ACME: &acme.Configuration{
					TLSChallenge: &acme.TLSChallenge{},
				}},
			},
		},
	}

	s.retrieveAcmeCertificate(c, testCase)
}

func (s *AcmeSuite) TestTLSALPN01Domains(c *check.C) {
	testCase := acmeTestCase{
		traefikConfFilePath: "fixtures/acme/acme_domains.toml",
		subCases: []subCases{{
			host:               acmeDomain,
			expectedCommonName: acmeDomain,
			expectedAlgorithm:  x509.RSA,
		}},
		template: templateModel{
			Domains: []types.Domain{{
				Main: "traefik.acme.wtf",
			}},
			Acme: map[string]static.CertificateResolver{
				"default": {ACME: &acme.Configuration{
					TLSChallenge: &acme.TLSChallenge{},
				}},
			},
		},
	}

	s.retrieveAcmeCertificate(c, testCase)
}

func (s *AcmeSuite) TestTLSALPN01DomainsInSAN(c *check.C) {
	testCase := acmeTestCase{
		traefikConfFilePath: "fixtures/acme/acme_domains.toml",
		subCases: []subCases{{
			host:               acmeDomain,
			expectedCommonName: "acme.wtf",
			expectedAlgorithm:  x509.RSA,
		}},
		template: templateModel{
			Domains: []types.Domain{{
				Main: "acme.wtf",
				SANs: []string{"traefik.acme.wtf"},
			}},
			Acme: map[string]static.CertificateResolver{
				"default": {ACME: &acme.Configuration{
					TLSChallenge: &acme.TLSChallenge{},
				}},
			},
		},
	}

	s.retrieveAcmeCertificate(c, testCase)
}

// Test Let's encrypt down.
func (s *AcmeSuite) TestNoValidLetsEncryptServer(c *check.C) {
	file := s.adaptFile(c, "fixtures/acme/acme_base.toml", templateModel{
		Acme: map[string]static.CertificateResolver{
			"default": {ACME: &acme.Configuration{
				CAServer:      "http://wrongurl:4001/directory",
				HTTPChallenge: &acme.HTTPChallenge{EntryPoint: "web"},
			}},
		},
	})
	defer os.Remove(file)

	cmd, display := s.traefikCmd(withConfigFile(file))
	defer display(c)
	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer s.killCmd(cmd)

	// Expected traefik works
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 10*time.Second, try.StatusCodeIs(http.StatusOK))
	c.Assert(err, checker.IsNil)
}

// Doing an HTTPS request and test the response certificate.
func (s *AcmeSuite) retrieveAcmeCertificate(c *check.C, testCase acmeTestCase) {
	if len(testCase.template.PortHTTP) == 0 {
		testCase.template.PortHTTP = ":5002"
	}

	if len(testCase.template.PortHTTPS) == 0 {
		testCase.template.PortHTTPS = ":5001"
	}

	for _, value := range testCase.template.Acme {
		if len(value.ACME.CAServer) == 0 {
			value.ACME.CAServer = s.getAcmeURL()
		}
	}

	file := s.adaptFile(c, testCase.traefikConfFilePath, testCase.template)
	defer os.Remove(file)

	cmd, display := s.traefikCmd(withConfigFile(file))
	defer display(c)
	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer s.killCmd(cmd)
	// A real file is needed to have the right mode on acme.json file
	defer os.Remove("/tmp/acme.json")

	backend := startTestServer("9010", http.StatusOK, "")
	defer backend.Close()

	for _, sub := range testCase.subCases {
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}

		// wait for traefik (generating acme account take some seconds)
		err = try.Do(60*time.Second, func() error {
			_, errGet := client.Get("https://127.0.0.1:5001")
			return errGet
		})
		c.Assert(err, checker.IsNil)

		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
					ServerName:         sub.host,
				},
			},
		}

		req := testhelpers.MustNewRequest(http.MethodGet, "https://127.0.0.1:5001/", nil)
		req.Host = sub.host
		req.Header.Set("Host", sub.host)
		req.Header.Set("Accept", "*/*")

		var resp *http.Response

		// Retry to send a Request which uses the LE generated certificate
		err = try.Do(60*time.Second, func() error {
			resp, err = client.Do(req)

			// /!\ If connection is not closed, SSLHandshake will only be done during the first trial /!\
			req.Close = true

			if err != nil {
				return err
			}

			cn := resp.TLS.PeerCertificates[0].Subject.CommonName
			if cn != sub.expectedCommonName {
				return fmt.Errorf("domain %s found instead of %s", cn, sub.expectedCommonName)
			}

			return nil
		})

		c.Assert(err, checker.IsNil)
		c.Assert(resp.StatusCode, checker.Equals, http.StatusOK)
		// Check Domain into response certificate
		c.Assert(resp.TLS.PeerCertificates[0].Subject.CommonName, checker.Equals, sub.expectedCommonName)
		c.Assert(resp.TLS.PeerCertificates[0].PublicKeyAlgorithm, checker.Equals, sub.expectedAlgorithm)
	}
}
