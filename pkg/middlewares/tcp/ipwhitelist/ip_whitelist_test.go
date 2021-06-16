package tcpipwhitelist

import (
	"context"
	"io/ioutil"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/tcp"
)

func TestNewIPWhiteLister(t *testing.T) {
	testCases := []struct {
		desc          string
		whiteList     dynamic.TCPIPWhiteList
		expectedError bool
	}{
		{
			desc:          "Empty config",
			whiteList:     dynamic.TCPIPWhiteList{},
			expectedError: true,
		},
		{
			desc: "invalid IP",
			whiteList: dynamic.TCPIPWhiteList{
				SourceRange: []string{"foo"},
			},
			expectedError: true,
		},
		{
			desc: "valid IP",
			whiteList: dynamic.TCPIPWhiteList{
				SourceRange: []string{"10.10.10.10"},
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			next := tcp.HandlerFunc(func(conn tcp.WriteCloser) {})
			whiteLister, err := New(context.Background(), next, test.whiteList, "traefikTest")

			if test.expectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, whiteLister)
			}
		})
	}
}

func TestIPWhiteLister_ServeHTTP(t *testing.T) {
	testCases := []struct {
		desc       string
		whiteList  dynamic.TCPIPWhiteList
		remoteAddr string
		expected   string
	}{
		{
			desc: "authorized with remote address",
			whiteList: dynamic.TCPIPWhiteList{
				SourceRange: []string{"20.20.20.20"},
			},
			remoteAddr: "20.20.20.20:1234",
			expected:   "OK",
		},
		{
			desc: "non authorized with remote address",
			whiteList: dynamic.TCPIPWhiteList{
				SourceRange: []string{"20.20.20.20"},
			},
			remoteAddr: "20.20.20.21:1234",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			next := tcp.HandlerFunc(func(conn tcp.WriteCloser) {
				write, err := conn.Write([]byte("OK"))
				require.NoError(t, err)
				assert.Equal(t, 2, write)

				err = conn.Close()
				require.NoError(t, err)
			})

			whiteLister, err := New(context.Background(), next, test.whiteList, "traefikTest")
			require.NoError(t, err)

			server, client := net.Pipe()

			go func() {
				whiteLister.ServeTCP(&contextWriteCloser{client, addr{test.remoteAddr}})
			}()

			read, err := ioutil.ReadAll(server)
			require.NoError(t, err)

			assert.Equal(t, test.expected, string(read))
		})
	}
}

type contextWriteCloser struct {
	net.Conn
	addr
}

type addr struct {
	remoteAddr string
}

func (a addr) Network() string {
	panic("implement me")
}

func (a addr) String() string {
	return a.remoteAddr
}

func (c contextWriteCloser) CloseWrite() error {
	panic("implement me")
}

func (c contextWriteCloser) RemoteAddr() net.Addr { return c.addr }

func (c contextWriteCloser) Context() context.Context {
	return context.Background()
}
