package mirror

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/traefik/traefik/v2/pkg/safe"
)

const defaultMaxBodySize int64 = -1

func TestMirroringOn100(t *testing.T) {
	var countMirror1, countMirror2 int32
	handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
	})
	pool := safe.NewPool(context.Background())
	mirror := New(handler, pool, defaultMaxBodySize)
	err := mirror.AddMirror(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		atomic.AddInt32(&countMirror1, 1)
	}), 10)
	assert.NoError(t, err)

	err = mirror.AddMirror(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		atomic.AddInt32(&countMirror2, 1)
	}), 50)
	assert.NoError(t, err)

	for i := 0; i < 100; i++ {
		mirror.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	}

	pool.Stop()

	val1 := atomic.LoadInt32(&countMirror1)
	val2 := atomic.LoadInt32(&countMirror2)
	assert.Equal(t, 10, int(val1))
	assert.Equal(t, 50, int(val2))
}

func TestMirroringOn10(t *testing.T) {
	var countMirror1, countMirror2 int32
	handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
	})
	pool := safe.NewPool(context.Background())
	mirror := New(handler, pool, defaultMaxBodySize)
	err := mirror.AddMirror(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		atomic.AddInt32(&countMirror1, 1)
	}), 10)
	assert.NoError(t, err)

	err = mirror.AddMirror(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		atomic.AddInt32(&countMirror2, 1)
	}), 50)
	assert.NoError(t, err)

	for i := 0; i < 10; i++ {
		mirror.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	}

	pool.Stop()

	val1 := atomic.LoadInt32(&countMirror1)
	val2 := atomic.LoadInt32(&countMirror2)
	assert.Equal(t, 1, int(val1))
	assert.Equal(t, 5, int(val2))
}

func TestInvalidPercent(t *testing.T) {
	mirror := New(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {}), safe.NewPool(context.Background()), defaultMaxBodySize)
	err := mirror.AddMirror(nil, -1)
	assert.Error(t, err)

	err = mirror.AddMirror(nil, 101)
	assert.Error(t, err)

	err = mirror.AddMirror(nil, 100)
	assert.NoError(t, err)

	err = mirror.AddMirror(nil, 0)
	assert.NoError(t, err)
}

func TestHijack(t *testing.T) {
	handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
	})
	pool := safe.NewPool(context.Background())
	mirror := New(handler, pool, defaultMaxBodySize)

	var mirrorRequest bool
	err := mirror.AddMirror(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		hijacker, ok := rw.(http.Hijacker)
		assert.Equal(t, true, ok)

		_, _, err := hijacker.Hijack()
		assert.Error(t, err)
		mirrorRequest = true
	}), 100)
	assert.NoError(t, err)

	mirror.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	pool.Stop()
	assert.Equal(t, true, mirrorRequest)
}

func TestFlush(t *testing.T) {
	handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
	})
	pool := safe.NewPool(context.Background())
	mirror := New(handler, pool, defaultMaxBodySize)

	var mirrorRequest bool
	err := mirror.AddMirror(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		hijacker, ok := rw.(http.Flusher)
		assert.Equal(t, true, ok)

		hijacker.Flush()

		mirrorRequest = true
	}), 100)
	assert.NoError(t, err)

	mirror.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	pool.Stop()
	assert.Equal(t, true, mirrorRequest)
}

func TestMirroringWithBody(t *testing.T) {
	const numMirrors = 10

	var (
		countMirror int32
		body        = []byte(`body`)
	)

	pool := safe.NewPool(context.Background())

	handler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		assert.NotNil(t, r.Body)
		bb, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, body, bb)
		rw.WriteHeader(http.StatusOK)
	})

	mirror := New(handler, pool, defaultMaxBodySize)

	for i := 0; i < numMirrors; i++ {
		err := mirror.AddMirror(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			assert.NotNil(t, r.Body)
			bb, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.Equal(t, body, bb)
			atomic.AddInt32(&countMirror, 1)
		}), 100)
		assert.NoError(t, err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(body))

	mirror.ServeHTTP(httptest.NewRecorder(), req)

	pool.Stop()

	val := atomic.LoadInt32(&countMirror)
	assert.Equal(t, numMirrors, int(val))
}

func TestCloneRequest(t *testing.T) {
	t.Run("http request body is nil", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "/", nil)
		assert.NoError(t, err)

		ctx := req.Context()
		rr, _, err := newReusableRequest(req, defaultMaxBodySize)
		assert.NoError(t, err)

		// first call
		cloned := rr.clone(ctx)
		assert.Equal(t, cloned, req)
		assert.Nil(t, cloned.Body)

		// second call
		cloned = rr.clone(ctx)
		assert.Equal(t, cloned, req)
		assert.Nil(t, cloned.Body)
	})

	t.Run("http request body is not nil", func(t *testing.T) {
		bb := []byte(`¯\_(ツ)_/¯`)
		contentLength := len(bb)

		buf := bytes.NewBuffer(bb)
		req, err := http.NewRequest(http.MethodPost, "/", buf)
		assert.NoError(t, err)

		ctx := req.Context()
		req.ContentLength = int64(contentLength)

		rr, _, err := newReusableRequest(req, defaultMaxBodySize)
		assert.NoError(t, err)

		// first call
		cloned := rr.clone(ctx)
		body, err := io.ReadAll(cloned.Body)
		assert.NoError(t, err)
		assert.Equal(t, bb, body)

		// second call
		cloned = rr.clone(ctx)
		body, err = io.ReadAll(cloned.Body)
		assert.NoError(t, err)
		assert.Equal(t, bb, body)
	})

	t.Run("failed case", func(t *testing.T) {
		bb := []byte(`1234567890`)
		buf := bytes.NewBuffer(bb)

		req, err := http.NewRequest(http.MethodPost, "/", buf)
		assert.NoError(t, err)

		_, expectedBytes, err := newReusableRequest(req, 2)
		assert.Error(t, err)
		assert.Equal(t, bb[:3], expectedBytes)
	})

	t.Run("valid case with maxBodySize", func(t *testing.T) {
		bb := []byte(`1234567890`)
		buf := bytes.NewBuffer(bb)

		req, err := http.NewRequest(http.MethodPost, "/", buf)
		assert.NoError(t, err)

		_, expectedBytes, err := newReusableRequest(req, 20)
		assert.NoError(t, err)
		assert.Nil(t, expectedBytes)
	})

	t.Run("no request given", func(t *testing.T) {
		_, _, err := newReusableRequest(nil, defaultMaxBodySize)
		assert.Error(t, err)
	})
}
