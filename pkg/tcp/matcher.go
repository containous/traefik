package tcp

import (
	"bufio"
	"net"
	"strings"

	"github.com/traefik/traefik/v2/pkg/types"
)

// Matcher is a matcher used to match connection properties.
type Matcher interface {
	Match(conn WriteCloser) bool
}

// ClientIP matches a connection based on the client IP.
type ClientIP struct {
	ip string
}

// NewClientIP returns a new clientIP with the specified IP.
func NewClientIP(ip string) *ClientIP {
	return &ClientIP{ip: ip}
}

// Match checks if the Remote Address matches the matcher IP.
func (c ClientIP) Match(conn WriteCloser) bool {
	host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	if len(host) == 0 {
		return false
	}

	return host == c.ip
}

// SNIHost matches the SNI Host of the connection.
type SNIHost struct {
	host string
}

// NewSNIHost returns a new SNIHost with the speficied host.
func NewSNIHost(host string) *SNIHost {
	return &SNIHost{host: host}
}

// Match checks if the SNI Host of the connection match the matcher host.
func (s SNIHost) Match(conn WriteCloser) bool {
	br := bufio.NewReader(conn)
	serverName, tls, _, err := clientHelloServerName(br)
	if err != nil {
		return false
	}
	if !tls {
		return false
	}
	serverName = types.CanonicalDomain(serverName)
	if serverName == "" {
		return false
	}

	return MatchDomain(serverName, s.host)
}

// MatchDomain return true if a domain matches the given domain.
func MatchDomain(domain, givenDomain string) bool {
	if domain == givenDomain {
		return true
	}

	for len(givenDomain) > 0 && givenDomain[len(givenDomain)-1] == '.' {
		givenDomain = givenDomain[:len(givenDomain)-1]
	}

	labels := strings.Split(domain, ".")
	for i := range labels {
		labels[i] = "*"
		candidate := strings.Join(labels, ".")
		if givenDomain == candidate {
			return true
		}
	}
	return false
}
