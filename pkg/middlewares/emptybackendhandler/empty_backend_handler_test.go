package emptybackendhandler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/traefik/traefik/v2/pkg/testhelpers"
	"github.com/vulcand/oxy/roundrobin"
)

func TestEmptyBackendHandler(t *testing.T) {
	testCases := []struct {
		amountServer       int
		expectedStatusCode int
	}{
		{
			amountServer:       0,
			expectedStatusCode: http.StatusServiceUnavailable,
		},
		{
			amountServer:       1,
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(fmt.Sprintf("amount servers %d", test.amountServer), func(t *testing.T) {
			t.Parallel()

			handler := New(&healthCheckLoadBalancer{amountServer: test.amountServer})

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)

			handler.ServeHTTP(recorder, req)

			assert.Equal(t, test.expectedStatusCode, recorder.Result().StatusCode)
		})
	}
}

type healthCheckLoadBalancer struct {
	amountServer int
}

func (lb *healthCheckLoadBalancer) RegisterStatusUpdater(fn func(up bool)) error {
	return nil
}

func (lb *healthCheckLoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (lb *healthCheckLoadBalancer) Servers() []*url.URL {
	servers := make([]*url.URL, lb.amountServer)
	for i := 0; i < lb.amountServer; i++ {
		servers = append(servers, testhelpers.MustParseURL("http://localhost"))
	}
	return servers
}

func (lb *healthCheckLoadBalancer) RemoveServer(u *url.URL) error {
	return nil
}

func (lb *healthCheckLoadBalancer) UpsertServer(u *url.URL, options ...roundrobin.ServerOption) error {
	return nil
}

func (lb *healthCheckLoadBalancer) ServerWeight(u *url.URL) (int, bool) {
	return 0, false
}

func (lb *healthCheckLoadBalancer) NextServer() (*url.URL, error) {
	return nil, nil
}

func (lb *healthCheckLoadBalancer) Next() http.Handler {
	return nil
}
