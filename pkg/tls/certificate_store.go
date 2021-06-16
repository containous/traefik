package tls

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/safe"
)

// CertificateStore store for dynamic certificates.
type CertificateStore struct {
	DynamicCerts       *safe.Safe
	DefaultCertificate *tls.Certificate
	CertCache          *cache.Cache
}

// NewCertificateStore create a store for dynamic certificates.
func NewCertificateStore() *CertificateStore {
	return &CertificateStore{
		DynamicCerts: &safe.Safe{},
		CertCache:    cache.New(1*time.Hour, 10*time.Minute),
	}
}

func (c CertificateStore) getDefaultCertificateDomains() []string {
	var allCerts []string

	if c.DefaultCertificate == nil {
		return allCerts
	}

	x509Cert, err := x509.ParseCertificate(c.DefaultCertificate.Certificate[0])
	if err != nil {
		log.WithoutContext().Errorf("Could not parse default certificate: %v", err)
		return allCerts
	}

	if len(x509Cert.Subject.CommonName) > 0 {
		allCerts = append(allCerts, x509Cert.Subject.CommonName)
	}

	allCerts = append(allCerts, x509Cert.DNSNames...)

	for _, ipSan := range x509Cert.IPAddresses {
		allCerts = append(allCerts, ipSan.String())
	}

	return allCerts
}

// GetAllDomains return a slice with all the certificate domain.
func (c CertificateStore) GetAllDomains() []string {
	allDomains := c.getDefaultCertificateDomains()

	// Get dynamic certificates
	if c.DynamicCerts != nil && c.DynamicCerts.Get() != nil {
		for domain := range c.DynamicCerts.Get().(map[string]*tls.Certificate) {
			allDomains = append(allDomains, domain)
		}
	}

	return allDomains
}

// GetBestCertificate returns the best match certificate, and caches the response.
func (c *CertificateStore) GetBestCertificate(clientHello *tls.ClientHelloInfo) *tls.Certificate {
	if c == nil {
		return nil
	}
	domainToCheck := strings.ToLower(strings.TrimSpace(clientHello.ServerName))
	if len(domainToCheck) == 0 {
		// If no ServerName is provided, Check for local IP address matches
		host, _, err := net.SplitHostPort(clientHello.Conn.LocalAddr().String())
		if err != nil {
			log.Debugf("Could not split host/port: %v", err)
		}
		domainToCheck = strings.TrimSpace(host)
	}

	if cert, ok := c.CertCache.Get(domainToCheck); ok {
		return cert.(*tls.Certificate)
	}

	matchedCerts := map[string]*tls.Certificate{}
	if c.DynamicCerts != nil && c.DynamicCerts.Get() != nil {
		for domains, cert := range c.DynamicCerts.Get().(map[string]*tls.Certificate) {
			for _, certDomain := range strings.Split(domains, ",") {
				if MatchDomain(domainToCheck, certDomain) {
					matchedCerts[certDomain] = cert
				}
			}
		}
	}

	if len(matchedCerts) > 0 {
		// sort map by keys
		keys := make([]string, 0, len(matchedCerts))
		for k := range matchedCerts {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// cache best match
		c.CertCache.SetDefault(domainToCheck, matchedCerts[keys[len(keys)-1]])
		return matchedCerts[keys[len(keys)-1]]
	}

	return nil
}

// ResetCache clears the cache in the store.
func (c CertificateStore) ResetCache() {
	if c.CertCache != nil {
		c.CertCache.Flush()
	}
}

// MatchDomain return true if a domain match the cert domain.
func MatchDomain(domain, certDomain string) bool {
	if domain == certDomain {
		return true
	}

	for len(certDomain) > 0 && certDomain[len(certDomain)-1] == '.' {
		certDomain = certDomain[:len(certDomain)-1]
	}

	labels := strings.Split(domain, ".")
	for i := range labels {
		labels[i] = "*"
		candidate := strings.Join(labels, ".")
		if certDomain == candidate {
			return true
		}
	}
	return false
}
