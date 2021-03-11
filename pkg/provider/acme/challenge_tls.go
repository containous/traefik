package acme

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/challenge/tlsalpn01"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/safe"
	traefiktls "github.com/traefik/traefik/v2/pkg/tls"
	"github.com/traefik/traefik/v2/pkg/types"
)

const providerNameALPN = "tlsalpn.acme"

// ChallengeTLSALPN TLSALPN challenge provider implements challenge.Provider.
type ChallengeTLSALPN struct {
	Timeout time.Duration

	chans   map[string]chan struct{}
	muChans sync.Mutex

	certs   map[string]*Certificate
	muCerts sync.Mutex

	configurationChan chan<- dynamic.Message
}

// NewChallengeTLSALPN creates a new ChallengeTLSALPN.
func NewChallengeTLSALPN(timeout time.Duration) *ChallengeTLSALPN {
	return &ChallengeTLSALPN{
		Timeout: timeout,
		chans:   make(map[string]chan struct{}),
		certs:   make(map[string]*Certificate),
	}
}

// Present presents a challenge to obtain new ACME certificate.
func (c *ChallengeTLSALPN) Present(domain, _, keyAuth string) error {
	logger := log.WithoutContext().WithField(log.ProviderName, providerNameALPN)
	logger.Debugf("TLS Challenge Present temp certificate for %s", domain)

	certPEMBlock, keyPEMBlock, err := tlsalpn01.ChallengeBlocks(domain, keyAuth)
	if err != nil {
		return err
	}

	cert := &Certificate{Certificate: certPEMBlock, Key: keyPEMBlock, Domain: types.Domain{Main: "TEMP-" + domain}}

	c.muChans.Lock()
	ch := make(chan struct{})
	c.chans[string(certPEMBlock)] = ch
	c.muChans.Unlock()

	c.muCerts.Lock()
	c.certs[keyAuth] = cert
	conf := createMessage(c.certs)
	c.muCerts.Unlock()

	c.configurationChan <- conf

	timer := time.NewTimer(c.Timeout)

	select {
	case t := <-timer.C:
		timer.Stop()

		c.muChans.Lock()
		c.cleanChan(string(certPEMBlock))
		c.muChans.Unlock()

		err = c.CleanUp(domain, "", keyAuth)
		if err != nil {
			logger.Errorf("Failed to clean up TLS challenge: %v", err)
		}

		return fmt.Errorf("timeout %s", t)
	case <-ch:
		// noop
		return nil
	}
}

// CleanUp cleans the challenges when certificate is obtained.
func (c *ChallengeTLSALPN) CleanUp(domain, _, keyAuth string) error {
	log.WithoutContext().WithField(log.ProviderName, providerNameALPN).
		Debugf("TLS Challenge CleanUp temp certificate for %s", domain)

	c.muCerts.Lock()
	delete(c.certs, keyAuth)
	conf := createMessage(c.certs)
	c.muCerts.Unlock()

	c.configurationChan <- conf

	return nil
}

// Init the provider.
func (c *ChallengeTLSALPN) Init() error {
	return nil
}

// Provide allows the provider to provide configurations to traefik using the given configuration channel.
func (c *ChallengeTLSALPN) Provide(configurationChan chan<- dynamic.Message, _ *safe.Pool) error {
	c.configurationChan = configurationChan

	return nil
}

// ListenConfiguration sets a new Configuration into the configurationChan.
func (c *ChallengeTLSALPN) ListenConfiguration(conf dynamic.Configuration) {
	c.muChans.Lock()

	for _, certificate := range conf.TLS.Certificates {
		if !containsACMETLS1(certificate.Stores) {
			continue
		}

		c.cleanChan(certificate.CertFile.String())
	}

	c.muChans.Unlock()
}

func (c *ChallengeTLSALPN) cleanChan(key string) {
	if _, ok := c.chans[key]; ok {
		close(c.chans[key])
		delete(c.chans, key)
	}
}

func createMessage(certs map[string]*Certificate) dynamic.Message {
	conf := dynamic.Message{
		ProviderName: providerNameALPN,
		Configuration: &dynamic.Configuration{
			HTTP: &dynamic.HTTPConfiguration{
				Routers:     map[string]*dynamic.Router{},
				Middlewares: map[string]*dynamic.Middleware{},
				Services:    map[string]*dynamic.Service{},
			},
			TLS: &dynamic.TLSConfiguration{},
		},
	}

	for _, cert := range certs {
		certConf := &traefiktls.CertAndStores{
			Certificate: traefiktls.Certificate{
				CertFile: traefiktls.FileOrContent(cert.Certificate),
				KeyFile:  traefiktls.FileOrContent(cert.Key),
			},
			Stores: []string{tlsalpn01.ACMETLS1Protocol},
		}
		conf.Configuration.TLS.Certificates = append(conf.Configuration.TLS.Certificates, certConf)
	}

	return conf
}

func containsACMETLS1(stores []string) bool {
	for _, store := range stores {
		if store == tlsalpn01.ACMETLS1Protocol {
			return true
		}
	}

	return false
}
