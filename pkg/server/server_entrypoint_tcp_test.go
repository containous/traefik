package server

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/config/static"
	"github.com/traefik/traefik/v2/pkg/tcp"
)

func TestShutdownHijacked(t *testing.T) {
	router := &tcp.Router{}
	router.HTTPHandler(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		conn, _, err := rw.(http.Hijacker).Hijack()
		require.NoError(t, err)

		resp := http.Response{StatusCode: http.StatusOK}
		err = resp.Write(conn)
		require.NoError(t, err)
	}))

	testShutdown(t, router)
}

func TestShutdownHTTP(t *testing.T) {
	router := &tcp.Router{}
	router.HTTPHandler(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		time.Sleep(time.Second)
	}))

	testShutdown(t, router)
}

func TestShutdownTCP(t *testing.T) {
	router := &tcp.Router{}
	router.AddCatchAllNoTLS(tcp.HandlerFunc(func(conn tcp.WriteCloser) {
		for {
			_, err := http.ReadRequest(bufio.NewReader(conn))

			if errors.Is(err, io.EOF) || (err != nil && errors.Is(err, net.ErrClosed)) {
				return
			}
			require.NoError(t, err)

			resp := http.Response{StatusCode: http.StatusOK}
			err = resp.Write(conn)
			require.NoError(t, err)
		}
	}))

	testShutdown(t, router)
}

func testShutdown(t *testing.T, router *tcp.Router) {
	t.Helper()

	epConfig := &static.EntryPointsTransport{}
	epConfig.SetDefaults()

	epConfig.LifeCycle.RequestAcceptGraceTimeout = 0
	epConfig.LifeCycle.GraceTimeOut = ptypes.Duration(5 * time.Second)
	epConfig.RespondingTimeouts.ReadTimeout = ptypes.Duration(5 * time.Second)
	epConfig.RespondingTimeouts.WriteTimeout = ptypes.Duration(5 * time.Second)

	entryPoint, err := NewTCPEntryPoint(context.Background(), &static.EntryPoint{
		// We explicitly use an IPV4 address because on Alpine, with an IPV6 address
		// there seems to be shenanigans related to properly cleaning up file descriptors
		Address:          "127.0.0.1:0",
		Transport:        epConfig,
		ForwardedHeaders: &static.ForwardedHeaders{},
	})
	require.NoError(t, err)

	conn, err := startEntrypoint(entryPoint, router)
	require.NoError(t, err)

	epAddr := entryPoint.listener.Addr().String()

	request, err := http.NewRequest(http.MethodHead, "http://127.0.0.1:8082", nil)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// We need to do a write on the conn before the shutdown to make it "exist".
	// Because the connection indeed exists as far as TCP is concerned,
	// but since we only pass it along to the HTTP server after at least one byte is peaked,
	// the HTTP server (and hence its shutdown) does not know about the connection until that first byte peaking.
	err = request.Write(conn)
	require.NoError(t, err)

	reader := bufio.NewReader(conn)
	// Wait for first byte in response.
	_, err = reader.Peek(1)
	require.NoError(t, err)

	go entryPoint.Shutdown(context.Background())

	// Make sure that new connections are not permitted anymore.
	// Note that this should be true not only after Shutdown has returned,
	// but technically also as early as the Shutdown has closed the listener,
	// i.e. during the shutdown and before the gracetime is over.
	var testOk bool
	for i := 0; i < 10; i++ {
		loopConn, err := net.Dial("tcp", epAddr)
		if err == nil {
			loopConn.Close()
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if !strings.HasSuffix(err.Error(), "connection refused") && !strings.HasSuffix(err.Error(), "reset by peer") {
			t.Fatalf(`unexpected error: got %v, wanted "connection refused" or "reset by peer"`, err)
		}
		testOk = true
		break
	}
	if !testOk {
		t.Fatal("entry point never closed")
	}

	// And make sure that the connection we had opened before shutting things down is still operational

	resp, err := http.ReadResponse(reader, request)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func startEntrypoint(entryPoint *TCPEntryPoint, router *tcp.Router) (net.Conn, error) {
	go entryPoint.Start(context.Background())

	entryPoint.SwitchRouter(router)

	for i := 0; i < 10; i++ {
		conn, err := net.Dial("tcp", entryPoint.listener.Addr().String())
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		return conn, err
	}

	return nil, errors.New("entry point never started")
}

func TestReadTimeoutWithoutFirstByte(t *testing.T) {
	epConfig := &static.EntryPointsTransport{}
	epConfig.SetDefaults()
	epConfig.RespondingTimeouts.ReadTimeout = ptypes.Duration(2 * time.Second)

	entryPoint, err := NewTCPEntryPoint(context.Background(), &static.EntryPoint{
		Address:          ":0",
		Transport:        epConfig,
		ForwardedHeaders: &static.ForwardedHeaders{},
	})
	require.NoError(t, err)

	router := &tcp.Router{}
	router.HTTPHandler(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))

	conn, err := startEntrypoint(entryPoint, router)
	require.NoError(t, err)

	errChan := make(chan error)

	go func() {
		b := make([]byte, 2048)
		_, err := conn.Read(b)
		errChan <- err
	}()

	select {
	case err := <-errChan:
		require.Equal(t, io.EOF, err)
	case <-time.Tick(5 * time.Second):
		t.Error("Timeout while read")
	}
}

func TestReadTimeoutWithFirstByte(t *testing.T) {
	epConfig := &static.EntryPointsTransport{}
	epConfig.SetDefaults()
	epConfig.RespondingTimeouts.ReadTimeout = ptypes.Duration(2 * time.Second)

	entryPoint, err := NewTCPEntryPoint(context.Background(), &static.EntryPoint{
		Address:          ":0",
		Transport:        epConfig,
		ForwardedHeaders: &static.ForwardedHeaders{},
	})
	require.NoError(t, err)

	router := &tcp.Router{}
	router.HTTPHandler(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))

	conn, err := startEntrypoint(entryPoint, router)
	require.NoError(t, err)

	_, err = conn.Write([]byte("GET /some HTTP/1.1\r\n"))
	require.NoError(t, err)

	errChan := make(chan error)

	go func() {
		b := make([]byte, 2048)
		_, err := conn.Read(b)
		errChan <- err
	}()

	select {
	case err := <-errChan:
		require.Equal(t, io.EOF, err)
	case <-time.Tick(5 * time.Second):
		t.Error("Timeout while read")
	}
}
