package tcp

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type HandlerTCPFunc func(WriteCloser)

// ServeTCP calls f(conn).
func (f HandlerTCPFunc) ServeTCP(conn WriteCloser) {
	f(conn)
}

// A constructor for middleware
// that writes its own "tag" into the Conn and does nothing else.
// Useful in checking if a chain is behaving in the right order.
func tagMiddleware(tag string) Constructor {
	return func(h Handler) (Handler, error) {
		return HandlerTCPFunc(func(conn WriteCloser) {
			_, err := conn.Write([]byte(tag))
			if err != nil {
				panic("Unexpected")
			}
			h.ServeTCP(conn)
		}), nil
	}
}

var testApp = HandlerTCPFunc(func(conn WriteCloser) {
	_, err := conn.Write([]byte("app\n"))
	if err != nil {
		panic("unexpected")
	}
})

type myWriter struct {
	data []byte
}

func (mw *myWriter) Close() error {
	panic("implement me")
}

func (mw *myWriter) LocalAddr() net.Addr {
	panic("implement me")
}

func (mw *myWriter) RemoteAddr() net.Addr {
	panic("implement me")
}

func (mw *myWriter) SetDeadline(t time.Time) error {
	panic("implement me")
}

func (mw *myWriter) SetReadDeadline(t time.Time) error {
	panic("implement me")
}

func (mw *myWriter) SetWriteDeadline(t time.Time) error {
	panic("implement me")
}

func (mw *myWriter) Read(b []byte) (n int, err error) {
	panic("implement me")
}

func (mw *myWriter) Write(b []byte) (n int, err error) {
	mw.data = append(mw.data, b...)
	return len(mw.data), nil
}

func (mw *myWriter) CloseWrite() error {
	return nil
}

func TestNewChain(t *testing.T) {
	c1 := func(h Handler) (Handler, error) {
		return nil, nil
	}

	c2 := func(h Handler) (Handler, error) {
		return h, nil
	}

	slice := []Constructor{c1, c2}

	chain := NewChain(slice...)
	for k := range slice {
		assert.ObjectsAreEqual(chain.constructors[k], slice[k])
	}
}

func TestThenWorksWithNoMiddleware(t *testing.T) {
	handler, err := NewChain().Then(testApp)
	require.NoError(t, err)

	assert.ObjectsAreEqual(handler, testApp)
}

func TestThenTreatsNilAsError(t *testing.T) {
	handler, err := NewChain().Then(nil)
	require.Error(t, err)
	assert.Nil(t, handler)
}

func TestThenOrdersHandlersCorrectly(t *testing.T) {
	t1 := tagMiddleware("t1\n")
	t2 := tagMiddleware("t2\n")
	t3 := tagMiddleware("t3\n")

	chained, err := NewChain(t1, t2, t3).Then(testApp)
	require.NoError(t, err)

	conn := &myWriter{}
	chained.ServeTCP(conn)

	assert.Equal(t, "t1\nt2\nt3\napp\n", string(conn.data))
}

func TestAppendAddsHandlersCorrectly(t *testing.T) {
	chain := NewChain(tagMiddleware("t1\n"), tagMiddleware("t2\n"))
	newChain := chain.Append(tagMiddleware("t3\n"), tagMiddleware("t4\n"))

	assert.Len(t, chain.constructors, 2)
	assert.Len(t, newChain.constructors, 4)

	chained, err := newChain.Then(testApp)
	require.NoError(t, err)

	conn := &myWriter{}
	chained.ServeTCP(conn)

	assert.Equal(t, "t1\nt2\nt3\nt4\napp\n", string(conn.data))
}

func TestAppendRespectsImmutability(t *testing.T) {
	chain := NewChain(tagMiddleware(""))
	newChain := chain.Append(tagMiddleware(""))

	if &chain.constructors[0] == &newChain.constructors[0] {
		t.Error("Apppend does not respect immutability")
	}
}

func TestExtendAddsHandlersCorrectly(t *testing.T) {
	chain1 := NewChain(tagMiddleware("t1\n"), tagMiddleware("t2\n"))
	chain2 := NewChain(tagMiddleware("t3\n"), tagMiddleware("t4\n"))
	newChain := chain1.Extend(chain2)

	assert.Len(t, chain1.constructors, 2)
	assert.Len(t, chain2.constructors, 2)
	assert.Len(t, newChain.constructors, 4)

	chained, err := newChain.Then(testApp)
	require.NoError(t, err)

	conn := &myWriter{}
	chained.ServeTCP(conn)

	assert.Equal(t, "t1\nt2\nt3\nt4\napp\n", string(conn.data))
}

func TestExtendRespectsImmutability(t *testing.T) {
	chain := NewChain(tagMiddleware(""))
	newChain := chain.Extend(NewChain(tagMiddleware("")))

	if &chain.constructors[0] == &newChain.constructors[0] {
		t.Error("Extend does not respect immutability")
	}
}
