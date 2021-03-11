package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/traefik/traefik/v2/pkg/testhelpers"
)

type staticTransport struct {
	res *http.Response
}

func (t *staticTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return t.res, nil
}

func BenchmarkProxy(b *testing.B) {
	res := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("")),
	}

	w := httptest.NewRecorder()
	req := testhelpers.MustNewRequest(http.MethodGet, "http://foo.bar/", nil)

	pool := newBufferPool()
	handler, _ := buildProxy(Bool(false), nil, &staticTransport{res}, pool)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(w, req)
	}
}
