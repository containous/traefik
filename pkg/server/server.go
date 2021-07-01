package server

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"time"

	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/metrics"
	"github.com/traefik/traefik/v2/pkg/middlewares/accesslog"
	"github.com/traefik/traefik/v2/pkg/safe"
	"github.com/traefik/traefik/v2/pkg/server/middleware"
)

// Server is the reverse-proxy/load-balancer engine.
type Server struct {
	watcher        *ConfigurationWatcher
	tcpEntryPoints TCPEntryPoints
	udpEntryPoints UDPEntryPoints
	chainBuilder   *middleware.ChainBuilder

	accessLoggerMiddleware *accesslog.Handler

	signals  chan os.Signal
	stopChan chan bool

	routinesPool *safe.Pool
}

// NewServer returns an initialized Server.
func NewServer(routinesPool *safe.Pool, entryPoints TCPEntryPoints, entryPointsUDP UDPEntryPoints, watcher *ConfigurationWatcher,
	chainBuilder *middleware.ChainBuilder, accessLoggerMiddleware *accesslog.Handler) *Server {
	srv := &Server{
		watcher:                watcher,
		tcpEntryPoints:         entryPoints,
		chainBuilder:           chainBuilder,
		accessLoggerMiddleware: accessLoggerMiddleware,
		signals:                make(chan os.Signal, 1),
		stopChan:               make(chan bool, 1),
		routinesPool:           routinesPool,
		udpEntryPoints:         entryPointsUDP,
	}

	srv.configureSignals()

	return srv
}

// Start starts the server and Stop/Close it when context is Done.
func (s *Server) Start(ctx context.Context) {
	go func() {
		<-ctx.Done()
		logger := log.FromContext(ctx)
		logger.Info("I have to go...")
		logger.Info("Stopping server gracefully")
		s.Stop()
	}()

	s.tcpEntryPoints.Start()
	s.udpEntryPoints.Start()
	s.watcher.start()

	s.routinesPool.GoCtx(s.listenSignals)
}

// Wait blocks until the server shutdown.
func (s *Server) Wait() {
	<-s.stopChan
}

// Stop stops the server.
func (s *Server) Stop() {
	defer log.WithoutContext().Info("Server stopped")

	s.tcpEntryPoints.Stop()
	s.udpEntryPoints.Stop()

	s.stopChan <- true
}

// Close destroys the server.
func (s *Server) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	go func(ctx context.Context) {
		<-ctx.Done()
		if errors.Is(ctx.Err(), context.Canceled) {
			return
		} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			panic("Timeout while stopping traefik, killing instance ✝")
		}
	}(ctx)

	stopMetricsClients()

	s.routinesPool.Stop()

	signal.Stop(s.signals)
	close(s.signals)

	close(s.stopChan)

	s.chainBuilder.Close()

	cancel()
}

func stopMetricsClients() {
	metrics.StopDatadog()
	metrics.StopStatsd()
	metrics.StopInfluxDB()
}
