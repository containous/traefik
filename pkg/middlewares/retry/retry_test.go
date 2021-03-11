package retry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/testhelpers"
)

func TestRetry(t *testing.T) {
	testCases := []struct {
		desc                  string
		config                dynamic.Retry
		wantRetryAttempts     int
		wantResponseStatus    int
		amountFaultyEndpoints int
	}{
		{
			desc:                  "no retry on success",
			config:                dynamic.Retry{Attempts: 5},
			wantRetryAttempts:     0,
			wantResponseStatus:    http.StatusOK,
			amountFaultyEndpoints: 0,
		},
		{
			desc:                  "no retry on success with backoff",
			config:                dynamic.Retry{Attempts: 5, InitialInterval: ptypes.Duration(time.Microsecond * 50)},
			wantRetryAttempts:     0,
			wantResponseStatus:    http.StatusOK,
			amountFaultyEndpoints: 0,
		},
		{
			desc:                  "no retry when max request attempts is one",
			config:                dynamic.Retry{Attempts: 1},
			wantRetryAttempts:     0,
			wantResponseStatus:    http.StatusBadGateway,
			amountFaultyEndpoints: 1,
		},
		{
			desc:                  "no retry when max request attempts is one with backoff",
			config:                dynamic.Retry{Attempts: 1, InitialInterval: ptypes.Duration(time.Microsecond * 50)},
			wantRetryAttempts:     0,
			wantResponseStatus:    http.StatusBadGateway,
			amountFaultyEndpoints: 1,
		},
		{
			desc:                  "one retry when one server is faulty",
			config:                dynamic.Retry{Attempts: 2},
			wantRetryAttempts:     1,
			wantResponseStatus:    http.StatusOK,
			amountFaultyEndpoints: 1,
		},
		{
			desc:                  "one retry when one server is faulty with backoff",
			config:                dynamic.Retry{Attempts: 2, InitialInterval: ptypes.Duration(time.Microsecond * 50)},
			wantRetryAttempts:     1,
			wantResponseStatus:    http.StatusOK,
			amountFaultyEndpoints: 1,
		},
		{
			desc:                  "two retries when two servers are faulty",
			config:                dynamic.Retry{Attempts: 3},
			wantRetryAttempts:     2,
			wantResponseStatus:    http.StatusOK,
			amountFaultyEndpoints: 2,
		},
		{
			desc:                  "two retries when two servers are faulty with backoff",
			config:                dynamic.Retry{Attempts: 3, InitialInterval: ptypes.Duration(time.Microsecond * 50)},
			wantRetryAttempts:     2,
			wantResponseStatus:    http.StatusOK,
			amountFaultyEndpoints: 2,
		},
		{
			desc:                  "max attempts exhausted delivers the 5xx response",
			config:                dynamic.Retry{Attempts: 3},
			wantRetryAttempts:     2,
			wantResponseStatus:    http.StatusBadGateway,
			amountFaultyEndpoints: 3,
		},
		{
			desc:                  "max attempts exhausted delivers the 5xx response with backoff",
			config:                dynamic.Retry{Attempts: 3, InitialInterval: ptypes.Duration(time.Microsecond * 50)},
			wantRetryAttempts:     2,
			wantResponseStatus:    http.StatusBadGateway,
			amountFaultyEndpoints: 3,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			retryAttemps := 0
			next := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				retryAttemps++

				if retryAttemps > test.amountFaultyEndpoints {
					// calls WroteHeaders on httptrace.
					_ = r.Write(io.Discard)

					rw.WriteHeader(http.StatusOK)
					return
				}

				rw.WriteHeader(http.StatusBadGateway)
			})

			retryListener := &countingRetryListener{}
			retry, err := New(context.Background(), next, test.config, retryListener, "traefikTest")
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "http://localhost:3000/ok", nil)

			retry.ServeHTTP(recorder, req)

			assert.Equal(t, test.wantResponseStatus, recorder.Code)
			assert.Equal(t, test.wantRetryAttempts, retryListener.timesCalled)
		})
	}
}

func TestRetryEmptyServerList(t *testing.T) {
	next := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusServiceUnavailable)
	})

	retryListener := &countingRetryListener{}
	retry, err := New(context.Background(), next, dynamic.Retry{Attempts: 3}, retryListener, "traefikTest")
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://localhost:3000/ok", nil)

	retry.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	assert.Equal(t, 0, retryListener.timesCalled)
}

func TestRetryListeners(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	retryListeners := Listeners{&countingRetryListener{}, &countingRetryListener{}}

	retryListeners.Retried(req, 1)
	retryListeners.Retried(req, 1)

	for _, retryListener := range retryListeners {
		listener := retryListener.(*countingRetryListener)
		if listener.timesCalled != 2 {
			t.Errorf("retry listener was called %d time(s), want %d time(s)", listener.timesCalled, 2)
		}
	}
}

func TestMultipleRetriesShouldNotLooseHeaders(t *testing.T) {
	attempt := 0
	expectedHeaderName := "X-Foo-Test-2"
	expectedHeaderValue := "bar"

	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		headerName := fmt.Sprintf("X-Foo-Test-%d", attempt)
		rw.Header().Add(headerName, expectedHeaderValue)
		if attempt < 2 {
			attempt++
			return
		}

		// Request has been successfully written to backend
		trace := httptrace.ContextClientTrace(req.Context())
		trace.WroteHeaders()

		// And we decide to answer to client
		rw.WriteHeader(http.StatusNoContent)
	})

	retry, err := New(context.Background(), next, dynamic.Retry{Attempts: 3}, &countingRetryListener{}, "traefikTest")
	require.NoError(t, err)

	responseRecorder := httptest.NewRecorder()
	retry.ServeHTTP(responseRecorder, testhelpers.MustNewRequest(http.MethodGet, "http://test", http.NoBody))

	headerValue := responseRecorder.Header().Get(expectedHeaderName)

	// Validate if we have the correct header
	if headerValue != expectedHeaderValue {
		t.Errorf("Expected to have %s for header %s, got %s", expectedHeaderValue, expectedHeaderName, headerValue)
	}

	// Validate that we don't have headers from previous attempts
	for i := 0; i < attempt; i++ {
		headerName := fmt.Sprintf("X-Foo-Test-%d", i)
		headerValue = responseRecorder.Header().Get("headerName")
		if headerValue != "" {
			t.Errorf("Expected no value for header %s, got %s", headerName, headerValue)
		}
	}
}

// countingRetryListener is a Listener implementation to count the times the Retried fn is called.
type countingRetryListener struct {
	timesCalled int
}

func (l *countingRetryListener) Retried(req *http.Request, attempt int) {
	l.timesCalled++
}

func TestRetryWithFlush(t *testing.T) {
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(200)
		_, err := rw.Write([]byte("FULL "))
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
		}
		rw.(http.Flusher).Flush()
		_, err = rw.Write([]byte("DATA"))
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
		}
	})

	retry, err := New(context.Background(), next, dynamic.Retry{Attempts: 1}, &countingRetryListener{}, "traefikTest")
	require.NoError(t, err)

	responseRecorder := httptest.NewRecorder()

	retry.ServeHTTP(responseRecorder, &http.Request{})

	assert.Equal(t, "FULL DATA", responseRecorder.Body.String())
}

func TestRetryWebsocket(t *testing.T) {
	testCases := []struct {
		desc                   string
		maxRequestAttempts     int
		expectedRetryAttempts  int
		expectedResponseStatus int
		expectedError          bool
		amountFaultyEndpoints  int
	}{
		{
			desc:                   "Switching ok after 2 retries",
			maxRequestAttempts:     3,
			expectedRetryAttempts:  2,
			amountFaultyEndpoints:  2,
			expectedResponseStatus: http.StatusSwitchingProtocols,
		},
		{
			desc:                   "Switching failed",
			maxRequestAttempts:     2,
			expectedRetryAttempts:  1,
			amountFaultyEndpoints:  2,
			expectedResponseStatus: http.StatusBadGateway,
			expectedError:          true,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			retryAttemps := 0
			next := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				retryAttemps++

				if retryAttemps > test.amountFaultyEndpoints {
					upgrader := websocket.Upgrader{}
					_, err := upgrader.Upgrade(rw, r, nil)
					if err != nil {
						http.Error(rw, err.Error(), http.StatusInternalServerError)
					}
					return
				}

				rw.WriteHeader(http.StatusBadGateway)
			})

			retryListener := &countingRetryListener{}
			retryH, err := New(context.Background(), next, dynamic.Retry{Attempts: test.maxRequestAttempts}, retryListener, "traefikTest")
			require.NoError(t, err)

			retryServer := httptest.NewServer(retryH)

			url := strings.Replace(retryServer.URL, "http", "ws", 1)
			_, response, err := websocket.DefaultDialer.Dial(url, nil)

			if !test.expectedError {
				require.NoError(t, err)
			}

			assert.Equal(t, test.expectedResponseStatus, response.StatusCode)
			assert.Equal(t, test.expectedRetryAttempts, retryListener.timesCalled)
		})
	}
}
