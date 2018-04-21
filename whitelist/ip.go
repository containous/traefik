package whitelist

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/containous/traefik/log"
	"github.com/sirupsen/logrus"
)

const (
	// XForwardedFor Header name
	XForwardedFor = "X-Forwarded-For"
)

// IP allows to check that addresses are in a white list
type IP struct {
	whiteListsIPs    []*net.IP
	whiteListsNet    []*net.IPNet
	insecure         bool
	useXForwardedFor bool
}

// NewIP builds a new IP given a list of CIDR-Strings to white list
func NewIP(whiteList []string, insecure bool, useXForwardedFor bool) (*IP, error) {
	if len(whiteList) == 0 && !insecure {
		return nil, errors.New("no white list provided")
	}

	ip := IP{
		insecure:         insecure,
		useXForwardedFor: useXForwardedFor,
	}

	if !insecure {
		for _, ipMask := range whiteList {
			if ipAddr := net.ParseIP(ipMask); ipAddr != nil {
				ip.whiteListsIPs = append(ip.whiteListsIPs, &ipAddr)
			} else {
				_, ipAddr, err := net.ParseCIDR(ipMask)
				if err != nil {
					return nil, fmt.Errorf("parsing CIDR white list %s: %v", ipAddr, err)
				}
				ip.whiteListsNet = append(ip.whiteListsNet, ipAddr)
			}
		}
	}

	return &ip, nil
}

// IsAuthorized checks if provided request is authorized by the white list
func (ip *IP) IsAuthorized(req *http.Request) (bool, error) {
	if ip.insecure {
		return true, nil
	}

	var invalidMatches []string

	if ip.useXForwardedFor {
		xFFs := req.Header[XForwardedFor]
		if len(xFFs) > 0 {
			for _, xFF := range xFFs {
				ok, err := ip.contains(parseHost(xFF))
				if err != nil {
					return false, err
				}

				if ok {
					return ok, nil
				}

				invalidMatches = append(invalidMatches, xFF)
			}
		}
	}

	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return false, err
	}

	ok, err := ip.contains(host)
	if err != nil {
		return ok, err
	}

	if !ok {
		if log.GetLevel() == logrus.DebugLevel {
			invalidMatches = append(invalidMatches, req.RemoteAddr)
			log.Debugf("%q matched none of the white list", strings.Join(invalidMatches, ", "))
		}
	}

	return ok, err
}

// contains checks if provided address is in the white list
func (ip *IP) contains(addr string) (bool, error) {
	ipAddr, err := parseIP(addr)
	if err != nil {
		return false, fmt.Errorf("unable to parse address: %s: %s", addr, err)
	}

	return ip.ContainsIP(ipAddr), nil
}

// ContainsIP checks if provided address is in the white list
func (ip *IP) ContainsIP(addr net.IP) bool {
	if ip.insecure {
		return true
	}

	for _, whiteListIP := range ip.whiteListsIPs {
		if whiteListIP.Equal(addr) {
			return true
		}
	}

	for _, whiteListNet := range ip.whiteListsNet {
		if whiteListNet.Contains(addr) {
			return true
		}
	}

	return false
}

func parseIP(addr string) (net.IP, error) {
	userIP := net.ParseIP(addr)
	if userIP == nil {
		return nil, fmt.Errorf("can't parse IP from address %s", addr)
	}

	return userIP, nil
}

func parseHost(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}
