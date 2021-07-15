package compress

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/klauspost/compress/gzhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/testhelpers"
)

const (
	acceptEncodingHeader  = "Accept-Encoding"
	contentEncodingHeader = "Content-Encoding"
	contentTypeHeader     = "Content-Type"
	varyHeader            = "Vary"
	gzipValue             = "gzip"
)

func TestShouldCompressWhenNoContentEncodingHeader(t *testing.T) {
	req := testhelpers.MustNewRequest(http.MethodGet, "http://localhost", nil)
	req.Header.Add(acceptEncodingHeader, gzipValue)

	baseBody := generateBytes(gzhttp.DefaultMinSize)

	next := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, err := rw.Write(baseBody)
		assert.NoError(t, err)
	})
	handler, err := New(context.Background(), next, dynamic.Compress{}, "testing")
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	assert.Equal(t, gzipValue, rw.Header().Get(contentEncodingHeader))
	assert.Equal(t, acceptEncodingHeader, rw.Header().Get(varyHeader))

	if assert.ObjectsAreEqualValues(rw.Body.Bytes(), baseBody) {
		assert.Fail(t, "expected a compressed body", "got %v", rw.Body.Bytes())
	}
}

func TestShouldNotCompressWhenContentEncodingHeader(t *testing.T) {
	req := testhelpers.MustNewRequest(http.MethodGet, "http://localhost", nil)
	req.Header.Add(acceptEncodingHeader, gzipValue)

	fakeCompressedBody := generateBytes(gzhttp.DefaultMinSize)
	next := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add(contentEncodingHeader, gzipValue)
		rw.Header().Add(varyHeader, acceptEncodingHeader)
		_, err := rw.Write(fakeCompressedBody)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
		}
	})
	handler, err := New(context.Background(), next, dynamic.Compress{}, "testing")
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	assert.Equal(t, gzipValue, rw.Header().Get(contentEncodingHeader))
	assert.Equal(t, acceptEncodingHeader, rw.Header().Get(varyHeader))

	assert.EqualValues(t, rw.Body.Bytes(), fakeCompressedBody)
}

func TestShouldNotCompressWhenNoAcceptEncodingHeader(t *testing.T) {
	req := testhelpers.MustNewRequest(http.MethodGet, "http://localhost", nil)

	fakeBody := generateBytes(gzhttp.DefaultMinSize)
	next := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, err := rw.Write(fakeBody)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
		}
	})
	handler, err := New(context.Background(), next, dynamic.Compress{}, "testing")
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	assert.Empty(t, rw.Header().Get(contentEncodingHeader))
	assert.EqualValues(t, rw.Body.Bytes(), fakeBody)
}

func TestShouldNotCompressWhenSpecificContentType(t *testing.T) {
	baseBody := generateBytes(gzhttp.DefaultMinSize)

	testCases := []struct {
		desc            string
		conf            dynamic.Compress
		reqContentType  string
		respContentType string
	}{
		{
			desc: "Exclude Request Content-Type",
			conf: dynamic.Compress{
				ExcludedContentTypes: []string{"text/event-stream"},
			},
			reqContentType: "text/event-stream",
		},
		{
			desc: "Exclude Response Content-Type",
			conf: dynamic.Compress{
				ExcludedContentTypes: []string{"text/event-stream"},
			},
			respContentType: "text/event-stream",
		},
		{
			desc:           "application/grpc",
			conf:           dynamic.Compress{},
			reqContentType: "application/grpc",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			req := testhelpers.MustNewRequest(http.MethodGet, "http://localhost", nil)
			req.Header.Add(acceptEncodingHeader, gzipValue)
			if test.reqContentType != "" {
				req.Header.Add(contentTypeHeader, test.reqContentType)
			}

			next := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				if len(test.respContentType) > 0 {
					rw.Header().Set(contentTypeHeader, test.respContentType)
				}

				_, err := rw.Write(baseBody)
				if err != nil {
					http.Error(rw, err.Error(), http.StatusInternalServerError)
				}
			})

			handler, err := New(context.Background(), next, test.conf, "test")
			require.NoError(t, err)

			rw := httptest.NewRecorder()
			handler.ServeHTTP(rw, req)

			assert.Empty(t, rw.Header().Get(acceptEncodingHeader))
			assert.Empty(t, rw.Header().Get(contentEncodingHeader))
			assert.EqualValues(t, rw.Body.Bytes(), baseBody)
		})
	}
}

func TestIntegrationShouldNotCompress(t *testing.T) {
	fakeCompressedBody := generateBytes(100000)

	testCases := []struct {
		name               string
		handler            http.Handler
		expectedStatusCode int
	}{
		{
			name: "when content already compressed",
			handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				rw.Header().Add(contentEncodingHeader, gzipValue)
				rw.Header().Add(varyHeader, acceptEncodingHeader)
				_, err := rw.Write(fakeCompressedBody)
				if err != nil {
					http.Error(rw, err.Error(), http.StatusInternalServerError)
				}
			}),
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "when content already compressed and status code Created",
			handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				rw.Header().Add(contentEncodingHeader, gzipValue)
				rw.Header().Add(varyHeader, acceptEncodingHeader)
				rw.WriteHeader(http.StatusCreated)
				_, err := rw.Write(fakeCompressedBody)
				if err != nil {
					http.Error(rw, err.Error(), http.StatusInternalServerError)
				}
			}),
			expectedStatusCode: http.StatusCreated,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			compress, err := New(context.Background(), test.handler, dynamic.Compress{}, "testing")
			require.NoError(t, err)

			ts := httptest.NewServer(compress)
			defer ts.Close()

			req := testhelpers.MustNewRequest(http.MethodGet, ts.URL, nil)
			req.Header.Add(acceptEncodingHeader, gzipValue)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)

			assert.Equal(t, test.expectedStatusCode, resp.StatusCode)

			assert.Equal(t, gzipValue, resp.Header.Get(contentEncodingHeader))
			assert.Equal(t, acceptEncodingHeader, resp.Header.Get(varyHeader))

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.EqualValues(t, fakeCompressedBody, body)
		})
	}
}

func TestShouldWriteHeaderWhenFlush(t *testing.T) {
	next := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add(contentEncodingHeader, gzipValue)
		rw.Header().Add(varyHeader, acceptEncodingHeader)
		rw.WriteHeader(http.StatusUnauthorized)
		rw.(http.Flusher).Flush()
		_, err := rw.Write([]byte("short"))
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
		}
	})
	handler, err := New(context.Background(), next, dynamic.Compress{}, "testing")
	require.NoError(t, err)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	req := testhelpers.MustNewRequest(http.MethodGet, ts.URL, nil)
	req.Header.Add(acceptEncodingHeader, gzipValue)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	assert.Equal(t, gzipValue, resp.Header.Get(contentEncodingHeader))
	assert.Equal(t, acceptEncodingHeader, resp.Header.Get(varyHeader))
}

func TestIntegrationShouldCompress(t *testing.T) {
	fakeBody := generateBytes(100000)

	testCases := []struct {
		name               string
		handler            http.Handler
		expectedStatusCode int
	}{
		{
			name: "when AcceptEncoding header is present",
			handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				_, err := rw.Write(fakeBody)
				if err != nil {
					http.Error(rw, err.Error(), http.StatusInternalServerError)
				}
			}),
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "when AcceptEncoding header is present and status code Created",
			handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				rw.WriteHeader(http.StatusCreated)
				_, err := rw.Write(fakeBody)
				if err != nil {
					http.Error(rw, err.Error(), http.StatusInternalServerError)
				}
			}),
			expectedStatusCode: http.StatusCreated,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			compress, err := New(context.Background(), test.handler, dynamic.Compress{}, "testing")
			require.NoError(t, err)

			ts := httptest.NewServer(compress)
			defer ts.Close()

			req := testhelpers.MustNewRequest(http.MethodGet, ts.URL, nil)
			req.Header.Add(acceptEncodingHeader, gzipValue)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)

			assert.Equal(t, test.expectedStatusCode, resp.StatusCode)

			assert.Equal(t, gzipValue, resp.Header.Get(contentEncodingHeader))
			assert.Equal(t, acceptEncodingHeader, resp.Header.Get(varyHeader))

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			if assert.ObjectsAreEqualValues(body, fakeBody) {
				assert.Fail(t, "expected a compressed body", "got %v", body)
			}
		})
	}
}

func BenchmarkCompress(b *testing.B) {
	testCases := []struct {
		name     string
		parallel bool
		size     int
	}{
		{
			name: "2k",
			size: 2048,
		},
		{
			name: "20k",
			size: 20480,
		},
		{
			name: "100k",
			size: 102400,
		},
		{
			name:     "2k parallel",
			parallel: true,
			size:     2048,
		},
		{
			name:     "20k parallel",
			parallel: true,
			size:     20480,
		},
		{
			name:     "100k parallel",
			parallel: true,
			size:     102400,
		},
	}

	for _, test := range testCases {
		b.Run(test.name, func(b *testing.B) {
			baseBody := generateBytes(test.size)

			next := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				_, err := rw.Write(baseBody)
				assert.NoError(b, err)
			})
			handler, _ := New(context.Background(), next, dynamic.Compress{}, "testing")

			req, _ := http.NewRequest("GET", "/whatever", nil)
			req.Header.Set("Accept-Encoding", "gzip")

			b.ReportAllocs()
			b.SetBytes(int64(test.size))
			if test.parallel {
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						runBenchmark(b, req, handler)
					}
				})
				return
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				runBenchmark(b, req, handler)
			}
		})
	}
}

func runBenchmark(b *testing.B, req *http.Request, handler http.Handler) {
	b.Helper()

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if code := res.Code; code != 200 {
		b.Fatalf("Expected 200 but got %d", code)
	}

	assert.Equal(b, gzipValue, res.Header().Get(contentEncodingHeader))
}

func generateBytes(length int) []byte {
	var value []byte
	for i := 0; i < length; i++ {
		value = append(value, 0x61+byte(i))
	}
	return value
}
