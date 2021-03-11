package types

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/traefik/traefik/v2/pkg/log"
)

// ClientTLS holds TLS specific configurations as client
// CA, Cert and Key can be either path or file contents.
type ClientTLS struct {
	CA                 string `description:"TLS CA" json:"ca,omitempty" toml:"ca,omitempty" yaml:"ca,omitempty"`
	CAOptional         bool   `description:"TLS CA.Optional" json:"caOptional,omitempty" toml:"caOptional,omitempty" yaml:"caOptional,omitempty" export:"true"`
	Cert               string `description:"TLS cert" json:"cert,omitempty" toml:"cert,omitempty" yaml:"cert,omitempty"`
	Key                string `description:"TLS key" json:"key,omitempty" toml:"key,omitempty" yaml:"key,omitempty"`
	InsecureSkipVerify bool   `description:"TLS insecure skip verify" json:"insecureSkipVerify,omitempty" toml:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty" export:"true"`
}

// CreateTLSConfig creates a TLS config from ClientTLS structures.
func (clientTLS *ClientTLS) CreateTLSConfig(ctx context.Context) (*tls.Config, error) {
	if clientTLS == nil {
		log.FromContext(ctx).Warnf("clientTLS is nil")
		return nil, nil
	}

	caPool := x509.NewCertPool()
	clientAuth := tls.NoClientCert
	if clientTLS.CA != "" {
		var ca []byte
		if _, errCA := os.Stat(clientTLS.CA); errCA == nil {
			var err error
			ca, err = os.ReadFile(clientTLS.CA)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA. %w", err)
			}
		} else {
			ca = []byte(clientTLS.CA)
		}

		if !caPool.AppendCertsFromPEM(ca) {
			return nil, fmt.Errorf("failed to parse CA")
		}

		if clientTLS.CAOptional {
			clientAuth = tls.VerifyClientCertIfGiven
		} else {
			clientAuth = tls.RequireAndVerifyClientCert
		}
	}

	if !clientTLS.InsecureSkipVerify && (len(clientTLS.Cert) == 0 || len(clientTLS.Key) == 0) {
		return nil, fmt.Errorf("TLS Certificate or Key file must be set when TLS configuration is created")
	}

	cert := tls.Certificate{}
	_, errKeyIsFile := os.Stat(clientTLS.Key)

	if len(clientTLS.Cert) > 0 && len(clientTLS.Key) > 0 {
		var err error
		if _, errCertIsFile := os.Stat(clientTLS.Cert); errCertIsFile == nil {
			if errKeyIsFile == nil {
				cert, err = tls.LoadX509KeyPair(clientTLS.Cert, clientTLS.Key)
				if err != nil {
					return nil, fmt.Errorf("failed to load TLS keypair: %w", err)
				}
			} else {
				return nil, fmt.Errorf("TLS cert is a file, but tls key is not")
			}
		} else {
			if errKeyIsFile != nil {
				cert, err = tls.X509KeyPair([]byte(clientTLS.Cert), []byte(clientTLS.Key))
				if err != nil {
					return nil, fmt.Errorf("failed to load TLS keypair: %w", err)
				}
			} else {
				return nil, fmt.Errorf("TLS key is a file, but tls cert is not")
			}
		}
	}

	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caPool,
		InsecureSkipVerify: clientTLS.InsecureSkipVerify,
		ClientAuth:         clientAuth,
	}, nil
}
