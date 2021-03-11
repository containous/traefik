package server

import (
	"context"
	"errors"
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	"sync"
	"syscall"
	"time"

	"github.com/pires/go-proxyproto"
	"github.com/sirupsen/logrus"
	"github.com/traefik/traefik/v2/pkg/config/static"
	"github.com/traefik/traefik/v2/pkg/ip"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/middlewares"
	"github.com/traefik/traefik/v2/pkg/middlewares/forwardedheaders"
	"github.com/traefik/traefik/v2/pkg/safe"
	"github.com/traefik/traefik/v2/pkg/server/router"
	"github.com/traefik/traefik/v2/pkg/tcp"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var httpServerLogger = stdlog.New(log.WithoutContext().WriterLevel(logrus.DebugLevel), "", 0)

type httpForwarder struct {
	net.Listener
	connChan chan net.Conn
	errChan  chan error
}

func newHTTPForwarder(ln net.Listener) *httpForwarder {
	return &httpForwarder{
		Listener: ln,
		connChan: make(chan net.Conn),
		errChan:  make(chan error),
	}
}

// ServeTCP uses the connection to serve it later in "Accept".
func (h *httpForwarder) ServeTCP(conn tcp.WriteCloser) {
	h.connChan <- conn
}

// Accept retrieves a served connection in ServeTCP.
func (h *httpForwarder) Accept() (net.Conn, error) {
	select {
	case conn := <-h.connChan:
		return conn, nil
	case err := <-h.errChan:
		return nil, err
	}
}

// TCPEntryPoints holds a map of TCPEntryPoint (the entrypoint names being the keys).
type TCPEntryPoints map[string]*TCPEntryPoint

// NewTCPEntryPoints creates a new TCPEntryPoints.
func NewTCPEntryPoints(entryPointsConfig static.EntryPoints) (TCPEntryPoints, error) {
	serverEntryPointsTCP := make(TCPEntryPoints)
	for entryPointName, config := range entryPointsConfig {
		protocol, err := config.GetProtocol()
		if err != nil {
			return nil, fmt.Errorf("error while building entryPoint %s: %w", entryPointName, err)
		}

		if protocol != "tcp" {
			continue
		}

		ctx := log.With(context.Background(), log.Str(log.EntryPointName, entryPointName))

		serverEntryPointsTCP[entryPointName], err = NewTCPEntryPoint(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("error while building entryPoint %s: %w", entryPointName, err)
		}
	}
	return serverEntryPointsTCP, nil
}

// Start the server entry points.
func (eps TCPEntryPoints) Start() {
	for entryPointName, serverEntryPoint := range eps {
		ctx := log.With(context.Background(), log.Str(log.EntryPointName, entryPointName))
		go serverEntryPoint.Start(ctx)
	}
}

// Stop the server entry points.
func (eps TCPEntryPoints) Stop() {
	var wg sync.WaitGroup

	for epn, ep := range eps {
		wg.Add(1)

		go func(entryPointName string, entryPoint *TCPEntryPoint) {
			defer wg.Done()

			ctx := log.With(context.Background(), log.Str(log.EntryPointName, entryPointName))
			entryPoint.Shutdown(ctx)

			log.FromContext(ctx).Debugf("Entry point %s closed", entryPointName)
		}(epn, ep)
	}

	wg.Wait()
}

// Switch the TCP routers.
func (eps TCPEntryPoints) Switch(routersTCP map[string]*tcp.Router) {
	for entryPointName, rt := range routersTCP {
		eps[entryPointName].SwitchRouter(rt)
	}
}

// TCPEntryPoint is the TCP server.
type TCPEntryPoint struct {
	listener               net.Listener
	switcher               *tcp.HandlerSwitcher
	transportConfiguration *static.EntryPointsTransport
	tracker                *connectionTracker
	httpServer             *httpServer
	httpsServer            *httpServer

	http3Server *http3server
}

// NewTCPEntryPoint creates a new TCPEntryPoint.
func NewTCPEntryPoint(ctx context.Context, configuration *static.EntryPoint) (*TCPEntryPoint, error) {
	tracker := newConnectionTracker()

	listener, err := buildListener(ctx, configuration)
	if err != nil {
		return nil, fmt.Errorf("error preparing server: %w", err)
	}

	rt := &tcp.Router{}

	httpServer, err := createHTTPServer(ctx, listener, configuration, true)
	if err != nil {
		return nil, fmt.Errorf("error preparing httpServer: %w", err)
	}

	rt.HTTPForwarder(httpServer.Forwarder)

	httpsServer, err := createHTTPServer(ctx, listener, configuration, false)
	if err != nil {
		return nil, fmt.Errorf("error preparing httpsServer: %w", err)
	}

	h3server, err := newHTTP3Server(ctx, configuration, httpsServer)
	if err != nil {
		return nil, err
	}

	rt.HTTPSForwarder(httpsServer.Forwarder)

	tcpSwitcher := &tcp.HandlerSwitcher{}
	tcpSwitcher.Switch(rt)

	return &TCPEntryPoint{
		listener:               listener,
		switcher:               tcpSwitcher,
		transportConfiguration: configuration.Transport,
		tracker:                tracker,
		httpServer:             httpServer,
		httpsServer:            httpsServer,
		http3Server:            h3server,
	}, nil
}

// Start starts the TCP server.
func (e *TCPEntryPoint) Start(ctx context.Context) {
	logger := log.FromContext(ctx)
	logger.Debugf("Start TCP Server")

	if e.http3Server != nil {
		go func() { _ = e.http3Server.Start() }()
	}

	for {
		conn, err := e.listener.Accept()
		if err != nil {
			logger.Error(err)

			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Temporary() {
				continue
			}

			e.httpServer.Forwarder.errChan <- err
			e.httpsServer.Forwarder.errChan <- err

			return
		}

		writeCloser, err := writeCloser(conn)
		if err != nil {
			panic(err)
		}

		safe.Go(func() {
			// Enforce read/write deadlines at the connection level,
			// because when we're peeking the first byte to determine whether we are doing TLS,
			// the deadlines at the server level are not taken into account.
			if e.transportConfiguration.RespondingTimeouts.ReadTimeout > 0 {
				err := writeCloser.SetReadDeadline(time.Now().Add(time.Duration(e.transportConfiguration.RespondingTimeouts.ReadTimeout)))
				if err != nil {
					logger.Errorf("Error while setting read deadline: %v", err)
				}
			}

			if e.transportConfiguration.RespondingTimeouts.WriteTimeout > 0 {
				err = writeCloser.SetWriteDeadline(time.Now().Add(time.Duration(e.transportConfiguration.RespondingTimeouts.WriteTimeout)))
				if err != nil {
					logger.Errorf("Error while setting write deadline: %v", err)
				}
			}

			e.switcher.ServeTCP(newTrackedConnection(writeCloser, e.tracker))
		})
	}
}

// Shutdown stops the TCP connections.
func (e *TCPEntryPoint) Shutdown(ctx context.Context) {
	logger := log.FromContext(ctx)

	reqAcceptGraceTimeOut := time.Duration(e.transportConfiguration.LifeCycle.RequestAcceptGraceTimeout)
	if reqAcceptGraceTimeOut > 0 {
		logger.Infof("Waiting %s for incoming requests to cease", reqAcceptGraceTimeOut)
		time.Sleep(reqAcceptGraceTimeOut)
	}

	graceTimeOut := time.Duration(e.transportConfiguration.LifeCycle.GraceTimeOut)
	ctx, cancel := context.WithTimeout(ctx, graceTimeOut)
	logger.Debugf("Waiting %s seconds before killing connections.", graceTimeOut)

	var wg sync.WaitGroup

	shutdownServer := func(server stoppable) {
		defer wg.Done()
		err := server.Shutdown(ctx)
		if err == nil {
			return
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			logger.Debugf("Server failed to shutdown within deadline because: %s", err)
			if err = server.Close(); err != nil {
				logger.Error(err)
			}
			return
		}
		logger.Error(err)
		// We expect Close to fail again because Shutdown most likely failed when trying to close a listener.
		// We still call it however, to make sure that all connections get closed as well.
		server.Close()
	}

	if e.httpServer.Server != nil {
		wg.Add(1)
		go shutdownServer(e.httpServer.Server)
	}

	if e.httpsServer.Server != nil {
		wg.Add(1)
		go shutdownServer(e.httpsServer.Server)

		if e.http3Server != nil {
			wg.Add(1)
			go shutdownServer(e.http3Server)
		}
	}

	if e.tracker != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := e.tracker.Shutdown(ctx)
			if err == nil {
				return
			}
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				logger.Debugf("Server failed to shutdown before deadline because: %s", err)
			}
			e.tracker.Close()
		}()
	}

	wg.Wait()
	cancel()
}

// SwitchRouter switches the TCP router handler.
func (e *TCPEntryPoint) SwitchRouter(rt *tcp.Router) {
	rt.HTTPForwarder(e.httpServer.Forwarder)

	httpHandler := rt.GetHTTPHandler()
	if httpHandler == nil {
		httpHandler = router.BuildDefaultHTTPRouter()
	}

	e.httpServer.Switcher.UpdateHandler(httpHandler)

	rt.HTTPSForwarder(e.httpsServer.Forwarder)

	httpsHandler := rt.GetHTTPSHandler()
	if httpsHandler == nil {
		httpsHandler = router.BuildDefaultHTTPRouter()
	}

	e.httpsServer.Switcher.UpdateHandler(httpsHandler)

	e.switcher.Switch(rt)

	if e.http3Server != nil {
		e.http3Server.Switch(rt)
	}
}

// writeCloserWrapper wraps together a connection, and the concrete underlying
// connection type that was found to satisfy WriteCloser.
type writeCloserWrapper struct {
	net.Conn
	writeCloser tcp.WriteCloser
}

func (c *writeCloserWrapper) CloseWrite() error {
	return c.writeCloser.CloseWrite()
}

// writeCloser returns the given connection, augmented with the WriteCloser
// implementation, if any was found within the underlying conn.
func writeCloser(conn net.Conn) (tcp.WriteCloser, error) {
	switch typedConn := conn.(type) {
	case *proxyproto.Conn:
		underlying, ok := typedConn.TCPConn()
		if !ok {
			return nil, fmt.Errorf("underlying connection is not a tcp connection")
		}
		return &writeCloserWrapper{writeCloser: underlying, Conn: typedConn}, nil
	case *net.TCPConn:
		return typedConn, nil
	default:
		return nil, fmt.Errorf("unknown connection type %T", typedConn)
	}
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}

	if err := tc.SetKeepAlive(true); err != nil {
		return nil, err
	}

	if err := tc.SetKeepAlivePeriod(3 * time.Minute); err != nil {
		// Some systems, such as OpenBSD, have no user-settable per-socket TCP
		// keepalive options.
		if !errors.Is(err, syscall.ENOPROTOOPT) {
			return nil, err
		}
	}

	return tc, nil
}

func buildProxyProtocolListener(ctx context.Context, entryPoint *static.EntryPoint, listener net.Listener) (net.Listener, error) {
	proxyListener := &proxyproto.Listener{Listener: listener}

	if entryPoint.ProxyProtocol.Insecure {
		log.FromContext(ctx).Infof("Enabling ProxyProtocol without trusted IPs: Insecure")
		return proxyListener, nil
	}

	checker, err := ip.NewChecker(entryPoint.ProxyProtocol.TrustedIPs)
	if err != nil {
		return nil, err
	}

	proxyListener.Policy = func(upstream net.Addr) (proxyproto.Policy, error) {
		ipAddr, ok := upstream.(*net.TCPAddr)
		if !ok {
			return proxyproto.REJECT, fmt.Errorf("type error %v", upstream)
		}

		if !checker.ContainsIP(ipAddr.IP) {
			log.FromContext(ctx).Debugf("IP %s is not in trusted IPs list, ignoring ProxyProtocol Headers and bypass connection", ipAddr.IP)
			return proxyproto.IGNORE, nil
		}
		return proxyproto.USE, nil
	}

	log.FromContext(ctx).Infof("Enabling ProxyProtocol for trusted IPs %v", entryPoint.ProxyProtocol.TrustedIPs)

	return proxyListener, nil
}

func buildListener(ctx context.Context, entryPoint *static.EntryPoint) (net.Listener, error) {
	listener, err := net.Listen("tcp", entryPoint.GetAddress())
	if err != nil {
		return nil, fmt.Errorf("error opening listener: %w", err)
	}

	listener = tcpKeepAliveListener{listener.(*net.TCPListener)}

	if entryPoint.ProxyProtocol != nil {
		listener, err = buildProxyProtocolListener(ctx, entryPoint, listener)
		if err != nil {
			return nil, fmt.Errorf("error creating proxy protocol listener: %w", err)
		}
	}
	return listener, nil
}

func newConnectionTracker() *connectionTracker {
	return &connectionTracker{
		conns: make(map[net.Conn]struct{}),
	}
}

type connectionTracker struct {
	conns map[net.Conn]struct{}
	lock  sync.RWMutex
}

// AddConnection add a connection in the tracked connections list.
func (c *connectionTracker) AddConnection(conn net.Conn) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.conns[conn] = struct{}{}
}

// RemoveConnection remove a connection from the tracked connections list.
func (c *connectionTracker) RemoveConnection(conn net.Conn) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.conns, conn)
}

func (c *connectionTracker) isEmpty() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return len(c.conns) == 0
}

// Shutdown wait for the connection closing.
func (c *connectionTracker) Shutdown(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		if c.isEmpty() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// Close close all the connections in the tracked connections list.
func (c *connectionTracker) Close() {
	c.lock.Lock()
	defer c.lock.Unlock()
	for conn := range c.conns {
		if err := conn.Close(); err != nil {
			log.WithoutContext().Errorf("Error while closing connection: %v", err)
		}
		delete(c.conns, conn)
	}
}

type stoppable interface {
	Shutdown(context.Context) error
	Close() error
}

type stoppableServer interface {
	stoppable
	Serve(listener net.Listener) error
}

type httpServer struct {
	Server    stoppableServer
	Forwarder *httpForwarder
	Switcher  *middlewares.HTTPHandlerSwitcher
}

func createHTTPServer(ctx context.Context, ln net.Listener, configuration *static.EntryPoint, withH2c bool) (*httpServer, error) {
	httpSwitcher := middlewares.NewHandlerSwitcher(router.BuildDefaultHTTPRouter())

	var handler http.Handler
	var err error
	handler, err = forwardedheaders.NewXForwarded(
		configuration.ForwardedHeaders.Insecure,
		configuration.ForwardedHeaders.TrustedIPs,
		httpSwitcher)

	if err != nil {
		return nil, err
	}

	if withH2c {
		handler = h2c.NewHandler(handler, &http2.Server{})
	}

	serverHTTP := &http.Server{
		Handler:      handler,
		ErrorLog:     httpServerLogger,
		ReadTimeout:  time.Duration(configuration.Transport.RespondingTimeouts.ReadTimeout),
		WriteTimeout: time.Duration(configuration.Transport.RespondingTimeouts.WriteTimeout),
		IdleTimeout:  time.Duration(configuration.Transport.RespondingTimeouts.IdleTimeout),
	}

	listener := newHTTPForwarder(ln)
	go func() {
		err := serverHTTP.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.FromContext(ctx).Errorf("Error while starting server: %v", err)
		}
	}()
	return &httpServer{
		Server:    serverHTTP,
		Forwarder: listener,
		Switcher:  httpSwitcher,
	}, nil
}

func newTrackedConnection(conn tcp.WriteCloser, tracker *connectionTracker) *trackedConnection {
	tracker.AddConnection(conn)
	return &trackedConnection{
		WriteCloser: conn,
		tracker:     tracker,
	}
}

type trackedConnection struct {
	tracker *connectionTracker
	tcp.WriteCloser
}

func (t *trackedConnection) Close() error {
	t.tracker.RemoveConnection(t.WriteCloser)
	return t.WriteCloser.Close()
}
