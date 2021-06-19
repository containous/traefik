package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/lucas-clemente/quic-go/http3"
	"github.com/traefik/traefik/v2/pkg/config/static"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/tcp"
)

type http3server struct {
	*http3.Server

	http3conn net.PacketConn

	lock   sync.RWMutex
	getter func(info *tls.ClientHelloInfo) (*tls.Config, error)
}

func newHTTP3Server(ctx context.Context, configuration *static.EntryPoint, httpsServer *httpServer) (*http3server, error) {
	if configuration.HTTP3 == nil || !configuration.HTTP3.Enabled {
		return nil, nil
	}

	conn, err := net.ListenPacket("udp", configuration.GetAddress())
	if err != nil {
		return nil, fmt.Errorf("error while starting http3 listener: %w", err)
	}

	h3 := &http3server{
		http3conn: conn,
		getter: func(info *tls.ClientHelloInfo) (*tls.Config, error) {
			return nil, errors.New("no tls config")
		},
	}

	// fall back to the entry point's address if AdvertisedAs is not specified
	if configuration.HTTP3.AdvertisedAs == "" {
		configuration.HTTP3.AdvertisedAs = configuration.GetAddress()
	}

	h3.Server = &http3.Server{
		Server: &http.Server{
			Addr:         configuration.HTTP3.AdvertisedAs,
			Handler:      httpsServer.Server.(*http.Server).Handler,
			ErrorLog:     httpServerLogger,
			ReadTimeout:  time.Duration(configuration.Transport.RespondingTimeouts.ReadTimeout),
			WriteTimeout: time.Duration(configuration.Transport.RespondingTimeouts.WriteTimeout),
			IdleTimeout:  time.Duration(configuration.Transport.RespondingTimeouts.IdleTimeout),
			TLSConfig:    &tls.Config{GetConfigForClient: h3.getGetConfigForClient},
		},
	}

	previousHandler := httpsServer.Server.(*http.Server).Handler

	httpsServer.Server.(*http.Server).Handler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		err := h3.Server.SetQuicHeaders(rw.Header())
		if err != nil {
			log.FromContext(ctx).Errorf("failed to set HTTP3 headers: %v", err)
		}

		previousHandler.ServeHTTP(rw, req)
	})

	return h3, nil
}

func (e *http3server) Start() error {
	return e.Serve(e.http3conn)
}

func (e *http3server) Switch(rt *tcp.Router) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.getter = rt.GetTLSGetClientInfo()
}

func (e *http3server) getGetConfigForClient(info *tls.ClientHelloInfo) (*tls.Config, error) {
	e.lock.RLock()
	defer e.lock.RUnlock()

	return e.getter(info)
}
