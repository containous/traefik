package acme

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns"
	"github.com/go-acme/lego/v4/registration"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/rules"
	"github.com/traefik/traefik/v2/pkg/safe"
	traefiktls "github.com/traefik/traefik/v2/pkg/tls"
	"github.com/traefik/traefik/v2/pkg/types"
	"github.com/traefik/traefik/v2/pkg/version"
)

// oscpMustStaple enables OSCP stapling as from https://github.com/go-acme/lego/issues/270.
var oscpMustStaple = false

// Configuration holds ACME configuration provided by users.
type Configuration struct {
	Email          string `description:"Email address used for registration." json:"email,omitempty" toml:"email,omitempty" yaml:"email,omitempty"`
	CAServer       string `description:"CA server to use." json:"caServer,omitempty" toml:"caServer,omitempty" yaml:"caServer,omitempty"`
	PreferredChain string `description:"Preferred chain to use." json:"preferredChain,omitempty" toml:"preferredChain,omitempty" yaml:"preferredChain,omitempty" export:"true"`
	Storage        string `description:"Storage to use." json:"storage,omitempty" toml:"storage,omitempty" yaml:"storage,omitempty" export:"true"`
	KeyType        string `description:"KeyType used for generating certificate private key. Allow value 'EC256', 'EC384', 'RSA2048', 'RSA4096', 'RSA8192'." json:"keyType,omitempty" toml:"keyType,omitempty" yaml:"keyType,omitempty" export:"true"`
	EAB            *EAB   `description:"External Account Binding to use." json:"eab,omitempty" toml:"eab,omitempty" yaml:"eab,omitempty"`

	DNSChallenge  *DNSChallenge  `description:"Activate DNS-01 Challenge." json:"dnsChallenge,omitempty" toml:"dnsChallenge,omitempty" yaml:"dnsChallenge,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	HTTPChallenge *HTTPChallenge `description:"Activate HTTP-01 Challenge." json:"httpChallenge,omitempty" toml:"httpChallenge,omitempty" yaml:"httpChallenge,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	TLSChallenge  *TLSChallenge  `description:"Activate TLS-ALPN-01 Challenge." json:"tlsChallenge,omitempty" toml:"tlsChallenge,omitempty" yaml:"tlsChallenge,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
}

// SetDefaults sets the default values.
func (a *Configuration) SetDefaults() {
	a.CAServer = lego.LEDirectoryProduction
	a.Storage = "acme.json"
	a.KeyType = "RSA4096"
}

// CertAndStore allows mapping a TLS certificate to a TLS store.
type CertAndStore struct {
	Certificate
	Store string
}

// Certificate is a struct which contains all data needed from an ACME certificate.
type Certificate struct {
	Domain      types.Domain `json:"domain,omitempty" toml:"domain,omitempty" yaml:"domain,omitempty"`
	Certificate []byte       `json:"certificate,omitempty" toml:"certificate,omitempty" yaml:"certificate,omitempty"`
	Key         []byte       `json:"key,omitempty" toml:"key,omitempty" yaml:"key,omitempty"`
}

// EAB contains External Account Binding configuration.
type EAB struct {
	Kid         string `description:"Key identifier from External CA." json:"kid,omitempty" toml:"kid,omitempty" yaml:"kid,omitempty"`
	HmacEncoded string `description:"Base64 encoded HMAC key from External CA." json:"hmacEncoded,omitempty" toml:"hmacEncoded,omitempty" yaml:"hmacEncoded,omitempty"`
}

// DNSChallenge contains DNS challenge configuration.
type DNSChallenge struct {
	Provider                string          `description:"Use a DNS-01 based challenge provider rather than HTTPS." json:"provider,omitempty" toml:"provider,omitempty" yaml:"provider,omitempty" export:"true"`
	DelayBeforeCheck        ptypes.Duration `description:"Assume DNS propagates after a delay in seconds rather than finding and querying nameservers." json:"delayBeforeCheck,omitempty" toml:"delayBeforeCheck,omitempty" yaml:"delayBeforeCheck,omitempty" export:"true"`
	Resolvers               []string        `description:"Use following DNS servers to resolve the FQDN authority." json:"resolvers,omitempty" toml:"resolvers,omitempty" yaml:"resolvers,omitempty"`
	DisablePropagationCheck bool            `description:"Disable the DNS propagation checks before notifying ACME that the DNS challenge is ready. [not recommended]" json:"disablePropagationCheck,omitempty" toml:"disablePropagationCheck,omitempty" yaml:"disablePropagationCheck,omitempty" export:"true"`
}

// HTTPChallenge contains HTTP challenge configuration.
type HTTPChallenge struct {
	EntryPoint string `description:"HTTP challenge EntryPoint" json:"entryPoint,omitempty" toml:"entryPoint,omitempty" yaml:"entryPoint,omitempty"  export:"true"`
}

// TLSChallenge contains TLS challenge configuration.
type TLSChallenge struct{}

// Provider holds configurations of the provider.
type Provider struct {
	*Configuration
	ResolverName string
	Store        Store `json:"store,omitempty" toml:"store,omitempty" yaml:"store,omitempty"`

	TLSChallengeProvider  challenge.Provider
	HTTPChallengeProvider challenge.Provider

	certificates           []*CertAndStore
	account                *Account
	client                 *lego.Client
	certsChan              chan *CertAndStore
	configurationChan      chan<- dynamic.Message
	tlsManager             *traefiktls.Manager
	clientMutex            sync.Mutex
	configFromListenerChan chan dynamic.Configuration
	pool                   *safe.Pool
	resolvingDomains       map[string]struct{}
	resolvingDomainsMutex  sync.RWMutex
}

// SetTLSManager sets the tls manager to use.
func (p *Provider) SetTLSManager(tlsManager *traefiktls.Manager) {
	p.tlsManager = tlsManager
}

// SetConfigListenerChan initializes the configFromListenerChan.
func (p *Provider) SetConfigListenerChan(configFromListenerChan chan dynamic.Configuration) {
	p.configFromListenerChan = configFromListenerChan
}

// ListenConfiguration sets a new Configuration into the configFromListenerChan.
func (p *Provider) ListenConfiguration(config dynamic.Configuration) {
	p.configFromListenerChan <- config
}

// Init for compatibility reason the BaseProvider implements an empty Init.
func (p *Provider) Init() error {
	ctx := log.With(context.Background(), log.Str(log.ProviderName, p.ResolverName+".acme"))
	logger := log.FromContext(ctx)

	if len(p.Configuration.Storage) == 0 {
		return errors.New("unable to initialize ACME provider with no storage location for the certificates")
	}

	var err error
	p.account, err = p.Store.GetAccount(p.ResolverName)
	if err != nil {
		return fmt.Errorf("unable to get ACME account: %w", err)
	}

	// Reset Account if caServer changed, thus registration URI can be updated
	if p.account != nil && p.account.Registration != nil && !isAccountMatchingCaServer(ctx, p.account.Registration.URI, p.CAServer) {
		logger.Info("Account URI does not match the current CAServer. The account will be reset.")
		p.account = nil
	}

	p.certificates, err = p.Store.GetCertificates(p.ResolverName)
	if err != nil {
		return fmt.Errorf("unable to get ACME certificates : %w", err)
	}

	// Init the currently resolved domain map
	p.resolvingDomains = make(map[string]struct{})

	return nil
}

func isAccountMatchingCaServer(ctx context.Context, accountURI, serverURI string) bool {
	logger := log.FromContext(ctx)

	aru, err := url.Parse(accountURI)
	if err != nil {
		logger.Infof("Unable to parse account.Registration URL: %v", err)
		return false
	}

	cau, err := url.Parse(serverURI)
	if err != nil {
		logger.Infof("Unable to parse CAServer URL: %v", err)
		return false
	}

	return cau.Hostname() == aru.Hostname()
}

// Provide allows the file provider to provide configurations to traefik
// using the given Configuration channel.
func (p *Provider) Provide(configurationChan chan<- dynamic.Message, pool *safe.Pool) error {
	ctx := log.With(context.Background(), log.Str(log.ProviderName, p.ResolverName+".acme"))

	p.pool = pool

	p.watchCertificate(ctx)
	p.watchNewDomains(ctx)

	p.configurationChan = configurationChan
	p.refreshCertificates()

	p.renewCertificates(ctx)

	ticker := time.NewTicker(24 * time.Hour)
	pool.GoCtx(func(ctxPool context.Context) {
		for {
			select {
			case <-ticker.C:
				p.renewCertificates(ctx)
			case <-ctxPool.Done():
				ticker.Stop()
				return
			}
		}
	})

	return nil
}

func (p *Provider) getClient() (*lego.Client, error) {
	p.clientMutex.Lock()
	defer p.clientMutex.Unlock()

	ctx := log.With(context.Background(), log.Str(log.ProviderName, p.ResolverName+".acme"))
	logger := log.FromContext(ctx)

	if p.client != nil {
		return p.client, nil
	}

	account, err := p.initAccount(ctx)
	if err != nil {
		return nil, err
	}

	logger.Debug("Building ACME client...")

	caServer := lego.LEDirectoryProduction
	if len(p.CAServer) > 0 {
		caServer = p.CAServer
	}
	logger.Debug(caServer)

	config := lego.NewConfig(account)
	config.CADirURL = caServer
	config.Certificate.KeyType = GetKeyType(ctx, p.KeyType)
	config.UserAgent = fmt.Sprintf("containous-traefik/%s", version.Version)

	client, err := lego.NewClient(config)
	if err != nil {
		return nil, err
	}

	// New users will need to register; be sure to save it
	if account.GetRegistration() == nil {
		reg, errR := p.register(ctx, client)
		if errR != nil {
			return nil, errR
		}

		account.Registration = reg
	}

	// Save the account once before all the certificates generation/storing
	// No certificate can be generated if account is not initialized
	err = p.Store.SaveAccount(p.ResolverName, account)
	if err != nil {
		return nil, err
	}

	if (p.DNSChallenge == nil || len(p.DNSChallenge.Provider) == 0) &&
		(p.HTTPChallenge == nil || len(p.HTTPChallenge.EntryPoint) == 0) &&
		p.TLSChallenge == nil {
		return nil, errors.New("ACME challenge not specified, please select TLS or HTTP or DNS Challenge")
	}

	if p.DNSChallenge != nil && len(p.DNSChallenge.Provider) > 0 {
		logger.Debugf("Using DNS Challenge provider: %s", p.DNSChallenge.Provider)

		var provider challenge.Provider
		provider, err = dns.NewDNSChallengeProviderByName(p.DNSChallenge.Provider)
		if err != nil {
			return nil, err
		}

		err = client.Challenge.SetDNS01Provider(provider,
			dns01.CondOption(len(p.DNSChallenge.Resolvers) > 0, dns01.AddRecursiveNameservers(p.DNSChallenge.Resolvers)),
			dns01.WrapPreCheck(func(domain, fqdn, value string, check dns01.PreCheckFunc) (bool, error) {
				if p.DNSChallenge.DelayBeforeCheck > 0 {
					logger.Debugf("Delaying %d rather than validating DNS propagation now.", p.DNSChallenge.DelayBeforeCheck)
					time.Sleep(time.Duration(p.DNSChallenge.DelayBeforeCheck))
				}

				if p.DNSChallenge.DisablePropagationCheck {
					return true, nil
				}

				return check(fqdn, value)
			}),
		)
		if err != nil {
			return nil, err
		}
	}

	if p.HTTPChallenge != nil && len(p.HTTPChallenge.EntryPoint) > 0 {
		logger.Debug("Using HTTP Challenge provider.")

		err = client.Challenge.SetHTTP01Provider(p.HTTPChallengeProvider)
		if err != nil {
			return nil, err
		}
	}

	if p.TLSChallenge != nil {
		logger.Debug("Using TLS Challenge provider.")

		err = client.Challenge.SetTLSALPN01Provider(p.TLSChallengeProvider)
		if err != nil {
			return nil, err
		}
	}

	p.client = client
	return p.client, nil
}

func (p *Provider) initAccount(ctx context.Context) (*Account, error) {
	if p.account == nil || len(p.account.Email) == 0 {
		var err error
		p.account, err = NewAccount(ctx, p.Email, p.KeyType)
		if err != nil {
			return nil, err
		}
	}

	// Set the KeyType if not already defined in the account
	if len(p.account.KeyType) == 0 {
		p.account.KeyType = GetKeyType(ctx, p.KeyType)
	}

	return p.account, nil
}

func (p *Provider) register(ctx context.Context, client *lego.Client) (*registration.Resource, error) {
	logger := log.FromContext(ctx)

	if p.EAB != nil {
		logger.Info("Register with external account binding...")

		eabOptions := registration.RegisterEABOptions{TermsOfServiceAgreed: true, Kid: p.EAB.Kid, HmacEncoded: p.EAB.HmacEncoded}

		return client.Registration.RegisterWithExternalAccountBinding(eabOptions)
	}

	logger.Info("Register...")

	return client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
}

func (p *Provider) resolveDomains(ctx context.Context, domains []string, tlsStore string) {
	if len(domains) == 0 {
		log.FromContext(ctx).Debug("No domain parsed in provider ACME")
		return
	}

	log.FromContext(ctx).Debugf("Try to challenge certificate for domain %v found in HostSNI rule", domains)

	var domain types.Domain
	if len(domains) > 0 {
		domain = types.Domain{Main: domains[0]}
		if len(domains) > 1 {
			domain.SANs = domains[1:]
		}

		safe.Go(func() {
			if _, err := p.resolveCertificate(ctx, domain, tlsStore); err != nil {
				log.FromContext(ctx).Errorf("Unable to obtain ACME certificate for domains %q: %v", strings.Join(domains, ","), err)
			}
		})
	}
}

func (p *Provider) watchNewDomains(ctx context.Context) {
	p.pool.GoCtx(func(ctxPool context.Context) {
		for {
			select {
			case config := <-p.configFromListenerChan:
				if config.TCP != nil {
					for routerName, route := range config.TCP.Routers {
						if route.TLS == nil || route.TLS.CertResolver != p.ResolverName {
							continue
						}

						ctxRouter := log.With(ctx, log.Str(log.RouterName, routerName), log.Str(log.Rule, route.Rule))
						logger := log.FromContext(ctxRouter)

						if len(route.TLS.Domains) > 0 {
							for _, domain := range route.TLS.Domains {
								if domain.Main != dns01.UnFqdn(domain.Main) {
									logger.Warnf("FQDN detected, please remove the trailing dot: %s", domain.Main)
								}
								for _, san := range domain.SANs {
									if san != dns01.UnFqdn(san) {
										logger.Warnf("FQDN detected, please remove the trailing dot: %s", san)
									}
								}
							}

							domains := deleteUnnecessaryDomains(ctxRouter, route.TLS.Domains)
							for i := 0; i < len(domains); i++ {
								domain := domains[i]
								safe.Go(func() {
									if _, err := p.resolveCertificate(ctx, domain, traefiktls.DefaultTLSStoreName); err != nil {
										log.WithoutContext().WithField(log.ProviderName, p.ResolverName+".acme").
											Errorf("Unable to obtain ACME certificate for domains %q : %v", strings.Join(domain.ToStrArray(), ","), err)
									}
								})
							}
						} else {
							domains, err := rules.ParseHostSNI(route.Rule)
							if err != nil {
								logger.Errorf("Error parsing domains in provider ACME: %v", err)
								continue
							}
							p.resolveDomains(ctxRouter, domains, traefiktls.DefaultTLSStoreName)
						}
					}
				}

				for routerName, route := range config.HTTP.Routers {
					if route.TLS == nil || route.TLS.CertResolver != p.ResolverName {
						continue
					}

					ctxRouter := log.With(ctx, log.Str(log.RouterName, routerName), log.Str(log.Rule, route.Rule))

					if len(route.TLS.Domains) > 0 {
						domains := deleteUnnecessaryDomains(ctxRouter, route.TLS.Domains)
						for i := 0; i < len(domains); i++ {
							domain := domains[i]
							safe.Go(func() {
								if _, err := p.resolveCertificate(ctx, domain, traefiktls.DefaultTLSStoreName); err != nil {
									log.WithoutContext().WithField(log.ProviderName, p.ResolverName+".acme").
										Errorf("Unable to obtain ACME certificate for domains %q : %v", strings.Join(domain.ToStrArray(), ","), err)
								}
							})
						}
					} else {
						domains, err := rules.ParseDomains(route.Rule)
						if err != nil {
							log.FromContext(ctxRouter).Errorf("Error parsing domains in provider ACME: %v", err)
							continue
						}
						p.resolveDomains(ctxRouter, domains, traefiktls.DefaultTLSStoreName)
					}
				}
			case <-ctxPool.Done():
				return
			}
		}
	})
}

func (p *Provider) resolveCertificate(ctx context.Context, domain types.Domain, tlsStore string) (*certificate.Resource, error) {
	domains, err := p.getValidDomains(ctx, domain)
	if err != nil {
		return nil, err
	}

	// Check if provided certificates are not already in progress and lock them if needed
	uncheckedDomains := p.getUncheckedDomains(ctx, domains, tlsStore)
	if len(uncheckedDomains) == 0 {
		return nil, nil
	}

	defer p.removeResolvingDomains(uncheckedDomains)

	logger := log.FromContext(ctx)
	logger.Debugf("Loading ACME certificates %+v...", uncheckedDomains)

	client, err := p.getClient()
	if err != nil {
		return nil, fmt.Errorf("cannot get ACME client %w", err)
	}

	request := certificate.ObtainRequest{
		Domains:        domains,
		Bundle:         true,
		MustStaple:     oscpMustStaple,
		PreferredChain: p.PreferredChain,
	}

	cert, err := client.Certificate.Obtain(request)
	if err != nil {
		return nil, fmt.Errorf("unable to generate a certificate for the domains %v: %w", uncheckedDomains, err)
	}
	if cert == nil {
		return nil, fmt.Errorf("domains %v do not generate a certificate", uncheckedDomains)
	}
	if len(cert.Certificate) == 0 || len(cert.PrivateKey) == 0 {
		return nil, fmt.Errorf("domains %v generate certificate with no value: %v", uncheckedDomains, cert)
	}

	logger.Debugf("Certificates obtained for domains %+v", uncheckedDomains)

	if len(uncheckedDomains) > 1 {
		domain = types.Domain{Main: uncheckedDomains[0], SANs: uncheckedDomains[1:]}
	} else {
		domain = types.Domain{Main: uncheckedDomains[0]}
	}
	p.addCertificateForDomain(domain, cert.Certificate, cert.PrivateKey, tlsStore)

	return cert, nil
}

func (p *Provider) removeResolvingDomains(resolvingDomains []string) {
	p.resolvingDomainsMutex.Lock()
	defer p.resolvingDomainsMutex.Unlock()

	for _, domain := range resolvingDomains {
		delete(p.resolvingDomains, domain)
	}
}

func (p *Provider) addCertificateForDomain(domain types.Domain, certificate, key []byte, tlsStore string) {
	p.certsChan <- &CertAndStore{Certificate: Certificate{Certificate: certificate, Key: key, Domain: domain}, Store: tlsStore}
}

// deleteUnnecessaryDomains deletes from the configuration :
// - Duplicated domains
// - Domains which are checked by wildcard domain.
func deleteUnnecessaryDomains(ctx context.Context, domains []types.Domain) []types.Domain {
	var newDomains []types.Domain

	logger := log.FromContext(ctx)

	for idxDomainToCheck, domainToCheck := range domains {
		keepDomain := true

		for idxDomain, domain := range domains {
			if idxDomainToCheck == idxDomain {
				continue
			}

			if reflect.DeepEqual(domain, domainToCheck) {
				if idxDomainToCheck > idxDomain {
					logger.Warnf("The domain %v is duplicated in the configuration but will be process by ACME provider only once.", domainToCheck)
					keepDomain = false
				}
				break
			}

			// Check if CN or SANS to check already exists
			// or can not be checked by a wildcard
			var newDomainsToCheck []string
			for _, domainProcessed := range domainToCheck.ToStrArray() {
				if idxDomain < idxDomainToCheck && isDomainAlreadyChecked(domainProcessed, domain.ToStrArray()) {
					// The domain is duplicated in a CN
					logger.Warnf("Domain %q is duplicated in the configuration or validated by the domain %v. It will be processed once.", domainProcessed, domain)
					continue
				} else if domain.Main != domainProcessed && strings.HasPrefix(domain.Main, "*") && isDomainAlreadyChecked(domainProcessed, []string{domain.Main}) {
					// Check if a wildcard can validate the domain
					logger.Warnf("Domain %q will not be processed by ACME provider because it is validated by the wildcard %q", domainProcessed, domain.Main)
					continue
				}
				newDomainsToCheck = append(newDomainsToCheck, domainProcessed)
			}

			// Delete the domain if both Main and SANs can be validated by the wildcard domain
			// otherwise keep the unchecked values
			if newDomainsToCheck == nil {
				keepDomain = false
				break
			}
			domainToCheck.Set(newDomainsToCheck)
		}

		if keepDomain {
			newDomains = append(newDomains, domainToCheck)
		}
	}

	return newDomains
}

func (p *Provider) watchCertificate(ctx context.Context) {
	p.certsChan = make(chan *CertAndStore)

	p.pool.GoCtx(func(ctxPool context.Context) {
		for {
			select {
			case cert := <-p.certsChan:
				certUpdated := false
				for _, domainsCertificate := range p.certificates {
					if reflect.DeepEqual(cert.Domain, domainsCertificate.Certificate.Domain) {
						domainsCertificate.Certificate = cert.Certificate
						certUpdated = true
						break
					}
				}
				if !certUpdated {
					p.certificates = append(p.certificates, cert)
				}

				err := p.saveCertificates()
				if err != nil {
					log.FromContext(ctx).Error(err)
				}
			case <-ctxPool.Done():
				return
			}
		}
	})
}

func (p *Provider) saveCertificates() error {
	err := p.Store.SaveCertificates(p.ResolverName, p.certificates)

	p.refreshCertificates()

	return err
}

func (p *Provider) refreshCertificates() {
	conf := dynamic.Message{
		ProviderName: p.ResolverName + ".acme",
		Configuration: &dynamic.Configuration{
			HTTP: &dynamic.HTTPConfiguration{
				Routers:     map[string]*dynamic.Router{},
				Middlewares: map[string]*dynamic.Middleware{},
				Services:    map[string]*dynamic.Service{},
			},
			TLS: &dynamic.TLSConfiguration{},
		},
	}

	for _, cert := range p.certificates {
		certConf := &traefiktls.CertAndStores{
			Certificate: traefiktls.Certificate{
				CertFile: traefiktls.FileOrContent(cert.Certificate.Certificate),
				KeyFile:  traefiktls.FileOrContent(cert.Key),
			},
			Stores: []string{cert.Store},
		}
		conf.Configuration.TLS.Certificates = append(conf.Configuration.TLS.Certificates, certConf)
	}

	p.configurationChan <- conf
}

func (p *Provider) renewCertificates(ctx context.Context) {
	logger := log.FromContext(ctx)

	logger.Info("Testing certificate renew...")
	for _, cert := range p.certificates {
		crt, err := getX509Certificate(ctx, &cert.Certificate)
		// If there's an error, we assume the cert is broken, and needs update
		// <= 30 days left, renew certificate
		if err != nil || crt == nil || crt.NotAfter.Before(time.Now().Add(24*30*time.Hour)) {
			client, err := p.getClient()
			if err != nil {
				logger.Infof("Error renewing certificate from LE : %+v, %v", cert.Domain, err)
				continue
			}

			logger.Infof("Renewing certificate from LE : %+v", cert.Domain)

			renewedCert, err := client.Certificate.Renew(certificate.Resource{
				Domain:      cert.Domain.Main,
				PrivateKey:  cert.Key,
				Certificate: cert.Certificate.Certificate,
			}, true, oscpMustStaple, p.PreferredChain)
			if err != nil {
				logger.Errorf("Error renewing certificate from LE: %v, %v", cert.Domain, err)
				continue
			}

			if len(renewedCert.Certificate) == 0 || len(renewedCert.PrivateKey) == 0 {
				logger.Errorf("domains %v renew certificate with no value: %v", cert.Domain.ToStrArray(), cert)
				continue
			}

			p.addCertificateForDomain(cert.Domain, renewedCert.Certificate, renewedCert.PrivateKey, cert.Store)
		}
	}
}

// Get provided certificate which check a domains list (Main and SANs)
// from static and dynamic provided certificates.
func (p *Provider) getUncheckedDomains(ctx context.Context, domainsToCheck []string, tlsStore string) []string {
	p.resolvingDomainsMutex.Lock()
	defer p.resolvingDomainsMutex.Unlock()

	log.FromContext(ctx).Debugf("Looking for provided certificate(s) to validate %q...", domainsToCheck)

	allDomains := p.tlsManager.GetStore(tlsStore).GetAllDomains()

	// Get ACME certificates
	for _, cert := range p.certificates {
		allDomains = append(allDomains, strings.Join(cert.Domain.ToStrArray(), ","))
	}

	// Get currently resolved domains
	for domain := range p.resolvingDomains {
		allDomains = append(allDomains, domain)
	}

	uncheckedDomains := searchUncheckedDomains(ctx, domainsToCheck, allDomains)

	// Lock domains that will be resolved by this routine
	for _, domain := range uncheckedDomains {
		p.resolvingDomains[domain] = struct{}{}
	}

	return uncheckedDomains
}

func searchUncheckedDomains(ctx context.Context, domainsToCheck, existentDomains []string) []string {
	var uncheckedDomains []string
	for _, domainToCheck := range domainsToCheck {
		if !isDomainAlreadyChecked(domainToCheck, existentDomains) {
			uncheckedDomains = append(uncheckedDomains, domainToCheck)
		}
	}

	logger := log.FromContext(ctx)
	if len(uncheckedDomains) == 0 {
		logger.Debugf("No ACME certificate generation required for domains %q.", domainsToCheck)
	} else {
		logger.Debugf("Domains %q need ACME certificates generation for domains %q.", domainsToCheck, strings.Join(uncheckedDomains, ","))
	}
	return uncheckedDomains
}

func getX509Certificate(ctx context.Context, cert *Certificate) (*x509.Certificate, error) {
	logger := log.FromContext(ctx)

	tlsCert, err := tls.X509KeyPair(cert.Certificate, cert.Key)
	if err != nil {
		logger.Errorf("Failed to load TLS key pair from ACME certificate for domain %q (SAN : %q), certificate will be renewed : %v", cert.Domain.Main, strings.Join(cert.Domain.SANs, ","), err)
		return nil, err
	}

	crt := tlsCert.Leaf
	if crt == nil {
		crt, err = x509.ParseCertificate(tlsCert.Certificate[0])
		if err != nil {
			logger.Errorf("Failed to parse TLS key pair from ACME certificate for domain %q (SAN : %q), certificate will be renewed : %v", cert.Domain.Main, strings.Join(cert.Domain.SANs, ","), err)
		}
	}

	return crt, err
}

// getValidDomains checks if given domain is allowed to generate a ACME certificate and return it.
func (p *Provider) getValidDomains(ctx context.Context, domain types.Domain) ([]string, error) {
	domains := domain.ToStrArray()
	if len(domains) == 0 {
		return nil, errors.New("unable to generate a certificate in ACME provider when no domain is given")
	}

	if strings.HasPrefix(domain.Main, "*") {
		if p.DNSChallenge == nil {
			return nil, fmt.Errorf("unable to generate a wildcard certificate in ACME provider for domain %q : ACME needs a DNSChallenge", strings.Join(domains, ","))
		}

		if strings.HasPrefix(domain.Main, "*.*") {
			return nil, fmt.Errorf("unable to generate a wildcard certificate in ACME provider for domain %q : ACME does not allow '*.*' wildcard domain", strings.Join(domains, ","))
		}
	}

	var cleanDomains []string
	for _, domain := range domains {
		canonicalDomain := types.CanonicalDomain(domain)
		cleanDomain := dns01.UnFqdn(canonicalDomain)
		if canonicalDomain != cleanDomain {
			log.FromContext(ctx).Warnf("FQDN detected, please remove the trailing dot: %s", canonicalDomain)
		}
		cleanDomains = append(cleanDomains, cleanDomain)
	}

	return cleanDomains, nil
}

func isDomainAlreadyChecked(domainToCheck string, existentDomains []string) bool {
	for _, certDomains := range existentDomains {
		for _, certDomain := range strings.Split(certDomains, ",") {
			if types.MatchDomain(domainToCheck, certDomain) {
				return true
			}
		}
	}
	return false
}
