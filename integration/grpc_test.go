package integration

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/go-check/check"
	"github.com/traefik/traefik/v2/integration/helloworld"
	"github.com/traefik/traefik/v2/integration/try"
	"github.com/traefik/traefik/v2/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	LocalhostCert []byte
	LocalhostKey  []byte
)

const randCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

// GRPCSuite tests suite.
type GRPCSuite struct{ BaseSuite }

type myserver struct {
	stopStreamExample chan bool
}

func (s *GRPCSuite) SetUpSuite(c *check.C) {
	var err error
	LocalhostCert, err = os.ReadFile("./resources/tls/local.cert")
	c.Assert(err, check.IsNil)
	LocalhostKey, err = os.ReadFile("./resources/tls/local.key")
	c.Assert(err, check.IsNil)
}

func (s *myserver) SayHello(ctx context.Context, in *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	return &helloworld.HelloReply{Message: "Hello " + in.Name}, nil
}

func (s *myserver) StreamExample(in *helloworld.StreamExampleRequest, server helloworld.Greeter_StreamExampleServer) error {
	data := make([]byte, 512)
	for i := range data {
		data[i] = randCharset[rand.Intn(len(randCharset))]
	}

	if err := server.Send(&helloworld.StreamExampleReply{Data: string(data)}); err != nil {
		log.WithoutContext().Error(err)
	}

	<-s.stopStreamExample
	return nil
}

func startGRPCServer(lis net.Listener, server *myserver) error {
	cert, err := tls.X509KeyPair(LocalhostCert, LocalhostKey)
	if err != nil {
		return err
	}

	creds := credentials.NewServerTLSFromCert(&cert)
	serverOption := grpc.Creds(creds)

	s := grpc.NewServer(serverOption)
	defer s.Stop()

	helloworld.RegisterGreeterServer(s, server)
	return s.Serve(lis)
}

func starth2cGRPCServer(lis net.Listener, server *myserver) error {
	s := grpc.NewServer()
	defer s.Stop()

	helloworld.RegisterGreeterServer(s, server)
	return s.Serve(lis)
}

func getHelloClientGRPC() (helloworld.GreeterClient, func() error, error) {
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(LocalhostCert)
	credsClient := credentials.NewClientTLSFromCert(roots, "")
	conn, err := grpc.Dial("127.0.0.1:4443", grpc.WithTransportCredentials(credsClient))
	if err != nil {
		return nil, func() error { return nil }, err
	}
	return helloworld.NewGreeterClient(conn), conn.Close, nil
}

func getHelloClientGRPCh2c() (helloworld.GreeterClient, func() error, error) {
	conn, err := grpc.Dial("127.0.0.1:8081", grpc.WithInsecure())
	if err != nil {
		return nil, func() error { return nil }, err
	}
	return helloworld.NewGreeterClient(conn), conn.Close, nil
}

func callHelloClientGRPC(name string, secure bool) (string, error) {
	var client helloworld.GreeterClient
	var closer func() error
	var err error

	if secure {
		client, closer, err = getHelloClientGRPC()
	} else {
		client, closer, err = getHelloClientGRPCh2c()
	}
	defer func() { _ = closer() }()

	if err != nil {
		return "", err
	}
	r, err := client.SayHello(context.Background(), &helloworld.HelloRequest{Name: name})
	if err != nil {
		return "", err
	}
	return r.Message, nil
}

func callStreamExampleClientGRPC() (helloworld.Greeter_StreamExampleClient, func() error, error) {
	client, closer, err := getHelloClientGRPC()
	if err != nil {
		return nil, closer, err
	}
	t, err := client.StreamExample(context.Background(), &helloworld.StreamExampleRequest{})
	if err != nil {
		return nil, closer, err
	}

	return t, closer, nil
}

func (s *GRPCSuite) TestGRPC(c *check.C) {
	lis, err := net.Listen("tcp", ":0")
	c.Assert(err, check.IsNil)
	_, port, err := net.SplitHostPort(lis.Addr().String())
	c.Assert(err, check.IsNil)

	go func() {
		err := startGRPCServer(lis, &myserver{})
		c.Log(err)
		c.Assert(err, check.IsNil)
	}()

	file := s.adaptFile(c, "fixtures/grpc/config.toml", struct {
		CertContent    string
		KeyContent     string
		GRPCServerPort string
	}{
		CertContent:    string(LocalhostCert),
		KeyContent:     string(LocalhostKey),
		GRPCServerPort: port,
	})

	defer os.Remove(file)
	cmd, display := s.traefikCmd(withConfigFile(file))
	defer display(c)

	err = cmd.Start()
	c.Assert(err, check.IsNil)
	defer s.killCmd(cmd)

	// wait for Traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 1*time.Second, try.BodyContains("Host(`127.0.0.1`)"))
	c.Assert(err, check.IsNil)

	var response string
	err = try.Do(1*time.Second, func() error {
		response, err = callHelloClientGRPC("World", true)
		return err
	})
	c.Assert(err, check.IsNil)
	c.Assert(response, check.Equals, "Hello World")
}

func (s *GRPCSuite) TestGRPCh2c(c *check.C) {
	lis, err := net.Listen("tcp", ":0")
	c.Assert(err, check.IsNil)
	_, port, err := net.SplitHostPort(lis.Addr().String())
	c.Assert(err, check.IsNil)

	go func() {
		err := starth2cGRPCServer(lis, &myserver{})
		c.Log(err)
		c.Assert(err, check.IsNil)
	}()

	file := s.adaptFile(c, "fixtures/grpc/config_h2c.toml", struct {
		GRPCServerPort string
	}{
		GRPCServerPort: port,
	})

	defer os.Remove(file)
	cmd, display := s.traefikCmd(withConfigFile(file))
	defer display(c)

	err = cmd.Start()
	c.Assert(err, check.IsNil)
	defer s.killCmd(cmd)

	// wait for Traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 1*time.Second, try.BodyContains("Host(`127.0.0.1`)"))
	c.Assert(err, check.IsNil)

	var response string
	err = try.Do(1*time.Second, func() error {
		response, err = callHelloClientGRPC("World", false)
		return err
	})
	c.Assert(err, check.IsNil)
	c.Assert(response, check.Equals, "Hello World")
}

func (s *GRPCSuite) TestGRPCh2cTermination(c *check.C) {
	lis, err := net.Listen("tcp", ":0")
	c.Assert(err, check.IsNil)
	_, port, err := net.SplitHostPort(lis.Addr().String())
	c.Assert(err, check.IsNil)

	go func() {
		err := starth2cGRPCServer(lis, &myserver{})
		c.Log(err)
		c.Assert(err, check.IsNil)
	}()

	file := s.adaptFile(c, "fixtures/grpc/config_h2c_termination.toml", struct {
		CertContent    string
		KeyContent     string
		GRPCServerPort string
	}{
		CertContent:    string(LocalhostCert),
		KeyContent:     string(LocalhostKey),
		GRPCServerPort: port,
	})

	defer os.Remove(file)
	cmd, display := s.traefikCmd(withConfigFile(file))
	defer display(c)

	err = cmd.Start()
	c.Assert(err, check.IsNil)
	defer s.killCmd(cmd)

	// wait for Traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 1*time.Second, try.BodyContains("Host(`127.0.0.1`)"))
	c.Assert(err, check.IsNil)

	var response string
	err = try.Do(1*time.Second, func() error {
		response, err = callHelloClientGRPC("World", true)
		return err
	})
	c.Assert(err, check.IsNil)
	c.Assert(response, check.Equals, "Hello World")
}

func (s *GRPCSuite) TestGRPCInsecure(c *check.C) {
	lis, err := net.Listen("tcp", ":0")
	c.Assert(err, check.IsNil)
	_, port, err := net.SplitHostPort(lis.Addr().String())
	c.Assert(err, check.IsNil)

	go func() {
		err := startGRPCServer(lis, &myserver{})
		c.Log(err)
		c.Assert(err, check.IsNil)
	}()

	file := s.adaptFile(c, "fixtures/grpc/config_insecure.toml", struct {
		CertContent    string
		KeyContent     string
		GRPCServerPort string
	}{
		CertContent:    string(LocalhostCert),
		KeyContent:     string(LocalhostKey),
		GRPCServerPort: port,
	})

	defer os.Remove(file)
	cmd, display := s.traefikCmd(withConfigFile(file))
	defer display(c)

	err = cmd.Start()
	c.Assert(err, check.IsNil)
	defer s.killCmd(cmd)

	// wait for Traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 1*time.Second, try.BodyContains("Host(`127.0.0.1`)"))
	c.Assert(err, check.IsNil)

	var response string
	err = try.Do(1*time.Second, func() error {
		response, err = callHelloClientGRPC("World", true)
		return err
	})
	c.Assert(err, check.IsNil)
	c.Assert(response, check.Equals, "Hello World")
}

func (s *GRPCSuite) TestGRPCBuffer(c *check.C) {
	stopStreamExample := make(chan bool)
	defer func() { stopStreamExample <- true }()
	lis, err := net.Listen("tcp", ":0")
	c.Assert(err, check.IsNil)
	_, port, err := net.SplitHostPort(lis.Addr().String())
	c.Assert(err, check.IsNil)

	go func() {
		err := startGRPCServer(lis, &myserver{
			stopStreamExample: stopStreamExample,
		})
		c.Log(err)
		c.Assert(err, check.IsNil)
	}()

	file := s.adaptFile(c, "fixtures/grpc/config.toml", struct {
		CertContent    string
		KeyContent     string
		GRPCServerPort string
	}{
		CertContent:    string(LocalhostCert),
		KeyContent:     string(LocalhostKey),
		GRPCServerPort: port,
	})

	defer os.Remove(file)
	cmd, display := s.traefikCmd(withConfigFile(file))
	defer display(c)

	err = cmd.Start()
	c.Assert(err, check.IsNil)
	defer s.killCmd(cmd)

	// wait for Traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 1*time.Second, try.BodyContains("Host(`127.0.0.1`)"))
	c.Assert(err, check.IsNil)
	var client helloworld.Greeter_StreamExampleClient
	client, closer, err := callStreamExampleClientGRPC()
	defer func() { _ = closer() }()
	c.Assert(err, check.IsNil)

	received := make(chan bool)
	go func() {
		tr, err := client.Recv()
		c.Assert(err, check.IsNil)
		c.Assert(len(tr.Data), check.Equals, 512)
		received <- true
	}()

	err = try.Do(10*time.Second, func() error {
		select {
		case <-received:
			return nil
		default:
			return errors.New("failed to receive stream data")
		}
	})
	c.Assert(err, check.IsNil)
}

func (s *GRPCSuite) TestGRPCBufferWithFlushInterval(c *check.C) {
	stopStreamExample := make(chan bool)
	lis, err := net.Listen("tcp", ":0")
	c.Assert(err, check.IsNil)
	_, port, err := net.SplitHostPort(lis.Addr().String())
	c.Assert(err, check.IsNil)

	go func() {
		err := startGRPCServer(lis, &myserver{
			stopStreamExample: stopStreamExample,
		})
		c.Log(err)
		c.Assert(err, check.IsNil)
	}()

	file := s.adaptFile(c, "fixtures/grpc/config.toml", struct {
		CertContent    string
		KeyContent     string
		GRPCServerPort string
	}{
		CertContent:    string(LocalhostCert),
		KeyContent:     string(LocalhostKey),
		GRPCServerPort: port,
	})
	defer os.Remove(file)

	cmd, display := s.traefikCmd(withConfigFile(file))
	defer display(c)
	err = cmd.Start()
	c.Assert(err, check.IsNil)
	defer s.killCmd(cmd)

	// wait for Traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 1*time.Second, try.BodyContains("Host(`127.0.0.1`)"))
	c.Assert(err, check.IsNil)

	var client helloworld.Greeter_StreamExampleClient
	client, closer, err := callStreamExampleClientGRPC()
	defer func() {
		_ = closer()
		stopStreamExample <- true
	}()
	c.Assert(err, check.IsNil)

	received := make(chan bool)
	go func() {
		tr, err := client.Recv()
		c.Assert(err, check.IsNil)
		c.Assert(len(tr.Data), check.Equals, 512)
		received <- true
	}()

	err = try.Do(100*time.Millisecond, func() error {
		select {
		case <-received:
			return nil
		default:
			return errors.New("failed to receive stream data")
		}
	})
	c.Assert(err, check.IsNil)
}

func (s *GRPCSuite) TestGRPCWithRetry(c *check.C) {
	lis, err := net.Listen("tcp", ":0")
	c.Assert(err, check.IsNil)
	_, port, err := net.SplitHostPort(lis.Addr().String())
	c.Assert(err, check.IsNil)

	go func() {
		err := startGRPCServer(lis, &myserver{})
		c.Log(err)
		c.Assert(err, check.IsNil)
	}()

	file := s.adaptFile(c, "fixtures/grpc/config_retry.toml", struct {
		CertContent    string
		KeyContent     string
		GRPCServerPort string
	}{
		CertContent:    string(LocalhostCert),
		KeyContent:     string(LocalhostKey),
		GRPCServerPort: port,
	})

	defer os.Remove(file)
	cmd, display := s.traefikCmd(withConfigFile(file))
	defer display(c)

	err = cmd.Start()
	c.Assert(err, check.IsNil)
	defer s.killCmd(cmd)

	// wait for Traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/rawdata", 1*time.Second, try.BodyContains("Host(`127.0.0.1`)"))
	c.Assert(err, check.IsNil)

	var response string
	err = try.Do(1*time.Second, func() error {
		response, err = callHelloClientGRPC("World", true)
		return err
	})
	c.Assert(err, check.IsNil)
	c.Assert(response, check.Equals, "Hello World")
}
