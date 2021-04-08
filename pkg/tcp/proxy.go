package tcp

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/pires/go-proxyproto"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/log"
)

// Proxy forwards a TCP request to a TCP service.
type Proxy struct {
	address          string
	target           *net.TCPAddr
	terminationDelay time.Duration
	proxyProtocol    *dynamic.ProxyProtocol
	refreshTarget    bool
}

// NewProxy creates a new Proxy.
func NewProxy(address string, terminationDelay time.Duration, proxyProtocol *dynamic.ProxyProtocol) (*Proxy, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, err
	}

	if proxyProtocol != nil && (proxyProtocol.Version < 1 || proxyProtocol.Version > 2) {
		return nil, fmt.Errorf("unknown proxyProtocol version: %d", proxyProtocol.Version)
	}

	// enable the refresh of the target only if the address in not an IP
	refreshTarget := false
	if host, _, err := net.SplitHostPort(address); err == nil && net.ParseIP(host) == nil {
		refreshTarget = true
	}

	return &Proxy{
		address:          address,
		target:           tcpAddr,
		refreshTarget:    refreshTarget,
		terminationDelay: terminationDelay,
		proxyProtocol:    proxyProtocol,
	}, nil
}

// ServeTCP forwards the connection to a service.
func (p *Proxy) ServeTCP(conn WriteCloser) {
	log.WithoutContext().Debugf("Handling connection from %s", conn.RemoteAddr())

	// needed because of e.g. server.trackedConnection
	defer conn.Close()

	connBackend, err := p.dialBackend()
	if err != nil {
		log.WithoutContext().Errorf("Error while connecting to backend: %v", err)
		return
	}

	// maybe not needed, but just in case
	defer connBackend.Close()
	errChan := make(chan error)

	if p.proxyProtocol != nil && p.proxyProtocol.Version > 0 && p.proxyProtocol.Version < 3 {
		header := proxyproto.HeaderProxyFromAddrs(byte(p.proxyProtocol.Version), conn.RemoteAddr(), conn.LocalAddr())
		if _, err := header.WriteTo(connBackend); err != nil {
			log.WithoutContext().Errorf("Error while writing proxy protocol headers to backend connection: %v", err)
			return
		}
	}

	go p.connCopy(conn, connBackend, errChan)
	go p.connCopy(connBackend, conn, errChan)

	err = <-errChan
	if err != nil {
		log.WithoutContext().Errorf("Error during connection: %v", err)
	}

	<-errChan
}

func (p Proxy) dialBackend() (*net.TCPConn, error) {
	if !p.refreshTarget {
		return net.DialTCP("tcp", nil, p.target)
	}

	conn, err := net.Dial("tcp", p.address)
	if err != nil {
		return nil, err
	}

	return conn.(*net.TCPConn), nil
}

func (p Proxy) connCopy(dst, src WriteCloser, errCh chan error) {
	_, err := io.Copy(dst, src)
	errCh <- err

	errClose := dst.CloseWrite()
	if errClose != nil {
		log.WithoutContext().Debugf("Error while terminating connection: %v", errClose)
		return
	}

	if p.terminationDelay >= 0 {
		err := dst.SetReadDeadline(time.Now().Add(p.terminationDelay))
		if err != nil {
			log.WithoutContext().Debugf("Error while setting deadline: %v", err)
		}
	}
}
