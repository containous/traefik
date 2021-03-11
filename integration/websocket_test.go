package integration

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/go-check/check"
	gorillawebsocket "github.com/gorilla/websocket"
	"github.com/traefik/traefik/v2/integration/try"
	checker "github.com/vdemeester/shakers"
	"golang.org/x/net/websocket"
)

// WebsocketSuite tests suite.
type WebsocketSuite struct{ BaseSuite }

func (s *WebsocketSuite) TestBase(c *check.C) {
	upgrader := gorillawebsocket.Upgrader{} // use default options

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				break
			}
			err = c.WriteMessage(mt, message)
			if err != nil {
				break
			}
		}
	}))

	file := s.adaptFile(c, "fixtures/websocket/config.toml", struct {
		WebsocketServer string
	}{
		WebsocketServer: srv.URL,
	})

	defer os.Remove(file)
	cmd, display := s.traefikCmd(withConfigFile(file), "--log.level=DEBUG")
	defer display(c)

	err := cmd.Start()
	c.Assert(err, check.IsNil)
	defer s.killCmd(cmd)

	// wait for traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 10*time.Second, try.BodyContains("127.0.0.1"))
	c.Assert(err, checker.IsNil)

	conn, _, err := gorillawebsocket.DefaultDialer.Dial("ws://127.0.0.1:8000/ws", nil)
	c.Assert(err, checker.IsNil)

	err = conn.WriteMessage(gorillawebsocket.TextMessage, []byte("OK"))
	c.Assert(err, checker.IsNil)

	_, msg, err := conn.ReadMessage()
	c.Assert(err, checker.IsNil)
	c.Assert(string(msg), checker.Equals, "OK")
}

func (s *WebsocketSuite) TestWrongOrigin(c *check.C) {
	upgrader := gorillawebsocket.Upgrader{} // use default options

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				break
			}
			err = c.WriteMessage(mt, message)
			if err != nil {
				break
			}
		}
	}))

	file := s.adaptFile(c, "fixtures/websocket/config.toml", struct {
		WebsocketServer string
	}{
		WebsocketServer: srv.URL,
	})

	defer os.Remove(file)
	cmd, display := s.traefikCmd(withConfigFile(file), "--log.level=DEBUG")
	defer display(c)

	err := cmd.Start()
	c.Assert(err, check.IsNil)
	defer s.killCmd(cmd)

	// wait for traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 10*time.Second, try.BodyContains("127.0.0.1"))
	c.Assert(err, checker.IsNil)

	config, err := websocket.NewConfig("ws://127.0.0.1:8000/ws", "ws://127.0.0.1:800")
	c.Assert(err, check.IsNil)

	conn, err := net.DialTimeout("tcp", "127.0.0.1:8000", time.Second)
	c.Assert(err, checker.IsNil)
	_, err = websocket.NewClient(config, conn)
	c.Assert(err, checker.NotNil)
	c.Assert(err, checker.ErrorMatches, "bad status")
}

func (s *WebsocketSuite) TestOrigin(c *check.C) {
	// use default options
	upgrader := gorillawebsocket.Upgrader{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				break
			}
			err = c.WriteMessage(mt, message)
			if err != nil {
				break
			}
		}
	}))

	file := s.adaptFile(c, "fixtures/websocket/config.toml", struct {
		WebsocketServer string
	}{
		WebsocketServer: srv.URL,
	})

	defer os.Remove(file)
	cmd, display := s.traefikCmd(withConfigFile(file), "--log.level=DEBUG")
	defer display(c)

	err := cmd.Start()
	c.Assert(err, check.IsNil)
	defer s.killCmd(cmd)

	// wait for traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 10*time.Second, try.BodyContains("127.0.0.1"))
	c.Assert(err, checker.IsNil)

	config, err := websocket.NewConfig("ws://127.0.0.1:8000/ws", "ws://127.0.0.1:8000")
	c.Assert(err, check.IsNil)

	conn, err := net.DialTimeout("tcp", "127.0.0.1:8000", time.Second)
	c.Assert(err, check.IsNil)
	client, err := websocket.NewClient(config, conn)
	c.Assert(err, checker.IsNil)

	n, err := client.Write([]byte("OK"))
	c.Assert(err, checker.IsNil)
	c.Assert(n, checker.Equals, 2)

	msg := make([]byte, 2)
	n, err = client.Read(msg)
	c.Assert(err, checker.IsNil)
	c.Assert(n, checker.Equals, 2)
	c.Assert(string(msg), checker.Equals, "OK")
}

func (s *WebsocketSuite) TestWrongOriginIgnoredByServer(c *check.C) {
	upgrader := gorillawebsocket.Upgrader{CheckOrigin: func(r *http.Request) bool {
		return true
	}}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				break
			}
			err = c.WriteMessage(mt, message)
			if err != nil {
				break
			}
		}
	}))

	file := s.adaptFile(c, "fixtures/websocket/config.toml", struct {
		WebsocketServer string
	}{
		WebsocketServer: srv.URL,
	})

	defer os.Remove(file)
	cmd, display := s.traefikCmd(withConfigFile(file), "--log.level=DEBUG")
	defer display(c)

	err := cmd.Start()
	c.Assert(err, check.IsNil)
	defer s.killCmd(cmd)

	// wait for traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 10*time.Second, try.BodyContains("127.0.0.1"))
	c.Assert(err, checker.IsNil)

	config, err := websocket.NewConfig("ws://127.0.0.1:8000/ws", "ws://127.0.0.1:80")
	c.Assert(err, check.IsNil)

	conn, err := net.DialTimeout("tcp", "127.0.0.1:8000", time.Second)
	c.Assert(err, checker.IsNil)
	client, err := websocket.NewClient(config, conn)
	c.Assert(err, checker.IsNil)

	n, err := client.Write([]byte("OK"))
	c.Assert(err, checker.IsNil)
	c.Assert(n, checker.Equals, 2)

	msg := make([]byte, 2)
	n, err = client.Read(msg)
	c.Assert(err, checker.IsNil)
	c.Assert(n, checker.Equals, 2)
	c.Assert(string(msg), checker.Equals, "OK")
}

func (s *WebsocketSuite) TestSSLTermination(c *check.C) {
	upgrader := gorillawebsocket.Upgrader{} // use default options

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				break
			}
			err = c.WriteMessage(mt, message)
			if err != nil {
				break
			}
		}
	}))
	file := s.adaptFile(c, "fixtures/websocket/config_https.toml", struct {
		WebsocketServer string
	}{
		WebsocketServer: srv.URL,
	})

	defer os.Remove(file)
	cmd, display := s.traefikCmd(withConfigFile(file), "--log.level=DEBUG")
	defer display(c)

	err := cmd.Start()
	c.Assert(err, check.IsNil)
	defer s.killCmd(cmd)

	// wait for traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 10*time.Second, try.BodyContains("127.0.0.1"))
	c.Assert(err, checker.IsNil)

	// Add client self-signed cert
	roots := x509.NewCertPool()
	certContent, err := os.ReadFile("./resources/tls/local.cert")
	c.Assert(err, checker.IsNil)
	roots.AppendCertsFromPEM(certContent)
	gorillawebsocket.DefaultDialer.TLSClientConfig = &tls.Config{
		RootCAs: roots,
	}
	conn, _, err := gorillawebsocket.DefaultDialer.Dial("wss://127.0.0.1:8000/ws", nil)
	c.Assert(err, checker.IsNil)

	err = conn.WriteMessage(gorillawebsocket.TextMessage, []byte("OK"))
	c.Assert(err, checker.IsNil)

	_, msg, err := conn.ReadMessage()
	c.Assert(err, checker.IsNil)
	c.Assert(string(msg), checker.Equals, "OK")
}

func (s *WebsocketSuite) TestBasicAuth(c *check.C) {
	upgrader := gorillawebsocket.Upgrader{} // use default options

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		user, password, _ := r.BasicAuth()
		c.Assert(user, check.Equals, "traefiker")
		c.Assert(password, check.Equals, "secret")

		for {
			mt, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			err = conn.WriteMessage(mt, message)
			if err != nil {
				break
			}
		}
	}))
	file := s.adaptFile(c, "fixtures/websocket/config.toml", struct {
		WebsocketServer string
	}{
		WebsocketServer: srv.URL,
	})

	defer os.Remove(file)
	cmd, display := s.traefikCmd(withConfigFile(file), "--log.level=DEBUG")
	defer display(c)

	err := cmd.Start()
	c.Assert(err, check.IsNil)
	defer s.killCmd(cmd)

	// wait for traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 10*time.Second, try.BodyContains("127.0.0.1"))
	c.Assert(err, checker.IsNil)

	config, err := websocket.NewConfig("ws://127.0.0.1:8000/ws", "ws://127.0.0.1:8000")
	auth := "traefiker:secret"
	config.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))

	c.Assert(err, check.IsNil)

	conn, err := net.DialTimeout("tcp", "127.0.0.1:8000", time.Second)
	c.Assert(err, checker.IsNil)
	client, err := websocket.NewClient(config, conn)
	c.Assert(err, checker.IsNil)

	n, err := client.Write([]byte("OK"))
	c.Assert(err, checker.IsNil)
	c.Assert(n, checker.Equals, 2)

	msg := make([]byte, 2)
	n, err = client.Read(msg)
	c.Assert(err, checker.IsNil)
	c.Assert(n, checker.Equals, 2)
	c.Assert(string(msg), checker.Equals, "OK")
}

func (s *WebsocketSuite) TestSpecificResponseFromBackend(c *check.C) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	file := s.adaptFile(c, "fixtures/websocket/config.toml", struct {
		WebsocketServer string
	}{
		WebsocketServer: srv.URL,
	})

	defer os.Remove(file)
	cmd, display := s.traefikCmd(withConfigFile(file), "--log.level=DEBUG")
	defer display(c)

	err := cmd.Start()
	c.Assert(err, check.IsNil)
	defer s.killCmd(cmd)

	// wait for traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 10*time.Second, try.BodyContains("127.0.0.1"))
	c.Assert(err, checker.IsNil)

	_, resp, err := gorillawebsocket.DefaultDialer.Dial("ws://127.0.0.1:8000/ws", nil)
	c.Assert(err, checker.NotNil)
	c.Assert(resp.StatusCode, check.Equals, http.StatusUnauthorized)
}

func (s *WebsocketSuite) TestURLWithURLEncodedChar(c *check.C) {
	upgrader := gorillawebsocket.Upgrader{} // use default options

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.EscapedPath(), check.Equals, "/ws/http%3A%2F%2Ftest")
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			mt, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			err = conn.WriteMessage(mt, message)
			if err != nil {
				break
			}
		}
	}))

	file := s.adaptFile(c, "fixtures/websocket/config.toml", struct {
		WebsocketServer string
	}{
		WebsocketServer: srv.URL,
	})

	defer os.Remove(file)
	cmd, display := s.traefikCmd(withConfigFile(file), "--log.level=DEBUG")
	defer display(c)

	err := cmd.Start()
	c.Assert(err, check.IsNil)
	defer s.killCmd(cmd)

	// wait for traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 10*time.Second, try.BodyContains("127.0.0.1"))
	c.Assert(err, checker.IsNil)

	conn, _, err := gorillawebsocket.DefaultDialer.Dial("ws://127.0.0.1:8000/ws/http%3A%2F%2Ftest", nil)
	c.Assert(err, checker.IsNil)

	err = conn.WriteMessage(gorillawebsocket.TextMessage, []byte("OK"))
	c.Assert(err, checker.IsNil)

	_, msg, err := conn.ReadMessage()
	c.Assert(err, checker.IsNil)
	c.Assert(string(msg), checker.Equals, "OK")
}

func (s *WebsocketSuite) TestSSLhttp2(c *check.C) {
	upgrader := gorillawebsocket.Upgrader{} // use default options

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				break
			}
			err = c.WriteMessage(mt, message)
			if err != nil {
				break
			}
		}
	}))

	ts.TLS = &tls.Config{}
	ts.TLS.NextProtos = append(ts.TLS.NextProtos, `h2`, `http/1.1`)
	ts.StartTLS()

	file := s.adaptFile(c, "fixtures/websocket/config_https.toml", struct {
		WebsocketServer string
	}{
		WebsocketServer: ts.URL,
	})

	defer os.Remove(file)
	cmd, display := s.traefikCmd(withConfigFile(file), "--log.level=DEBUG", "--accesslog")
	defer display(c)

	err := cmd.Start()
	c.Assert(err, check.IsNil)
	defer s.killCmd(cmd)

	// wait for traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 10*time.Second, try.BodyContains("127.0.0.1"))
	c.Assert(err, checker.IsNil)

	// Add client self-signed cert
	roots := x509.NewCertPool()
	certContent, err := os.ReadFile("./resources/tls/local.cert")
	c.Assert(err, checker.IsNil)
	roots.AppendCertsFromPEM(certContent)
	gorillawebsocket.DefaultDialer.TLSClientConfig = &tls.Config{
		RootCAs: roots,
	}
	conn, _, err := gorillawebsocket.DefaultDialer.Dial("wss://127.0.0.1:8000/echo", nil)
	c.Assert(err, checker.IsNil)

	err = conn.WriteMessage(gorillawebsocket.TextMessage, []byte("OK"))
	c.Assert(err, checker.IsNil)

	_, msg, err := conn.ReadMessage()
	c.Assert(err, checker.IsNil)
	c.Assert(string(msg), checker.Equals, "OK")
}

func (s *WebsocketSuite) TestHeaderAreForwared(c *check.C) {
	upgrader := gorillawebsocket.Upgrader{} // use default options

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Header.Get("X-Token"), check.Equals, "my-token")
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				break
			}
			err = c.WriteMessage(mt, message)
			if err != nil {
				break
			}
		}
	}))

	file := s.adaptFile(c, "fixtures/websocket/config.toml", struct {
		WebsocketServer string
	}{
		WebsocketServer: srv.URL,
	})

	defer os.Remove(file)
	cmd, display := s.traefikCmd(withConfigFile(file), "--log.level=DEBUG")
	defer display(c)

	err := cmd.Start()
	c.Assert(err, check.IsNil)
	defer s.killCmd(cmd)

	// wait for traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 10*time.Second, try.BodyContains("127.0.0.1"))
	c.Assert(err, checker.IsNil)

	headers := http.Header{}
	headers.Add("X-Token", "my-token")
	conn, _, err := gorillawebsocket.DefaultDialer.Dial("ws://127.0.0.1:8000/ws", headers)

	c.Assert(err, checker.IsNil)
	err = conn.WriteMessage(gorillawebsocket.TextMessage, []byte("OK"))
	c.Assert(err, checker.IsNil)

	_, msg, err := conn.ReadMessage()
	c.Assert(err, checker.IsNil)
	c.Assert(string(msg), checker.Equals, "OK")
}
