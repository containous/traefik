package tcp

import (
	"io"
	"net"
	"time"

	"github.com/traefik/traefik/v2/pkg/log"
)

// Proxy forwards a TCP request to a TCP service.
type Proxy struct {
	address          string
	target           *net.TCPAddr
	terminationDelay time.Duration
}

// NewProxy creates a new Proxy.
func NewProxy(address string, terminationDelay time.Duration) (*Proxy, error) {
	return &Proxy{
		address:          address,
		terminationDelay: terminationDelay,
	}, nil
}

// ServeTCP forwards the connection to a service.
func (p *Proxy) ServeTCP(conn WriteCloser) {
	log.Debugf("Handling connection from %s", conn.RemoteAddr())

	// needed because of e.g. server.trackedConnection
	defer conn.Close()

	if p.target == nil || p.targetIsHostname() {
		tcpAddr, err := net.ResolveTCPAddr("tcp", p.address)
		if err != nil {
			log.Errorf("Error resolving tcp address: %v", err)
			return
		}
		p.target = tcpAddr
	}

	connBackend, err := net.DialTCP("tcp", nil, p.target)
	if err != nil {
		log.Errorf("Error while connection to backend: %v", err)
		return
	}

	// maybe not needed, but just in case
	defer connBackend.Close()

	errChan := make(chan error)
	go p.connCopy(conn, connBackend, errChan)
	go p.connCopy(connBackend, conn, errChan)

	err = <-errChan
	if err != nil {
		log.WithoutContext().Errorf("Error during connection: %v", err)
	}

	<-errChan
}

func (p Proxy) targetIsHostname() bool {
	if host, _, err := net.SplitHostPort(p.address); err == nil {
		return net.ParseIP(host) == nil
	}
	return false
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
