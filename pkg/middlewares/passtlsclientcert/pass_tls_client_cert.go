package passtlsclientcert

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/opentracing/opentracing-go/ext"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/middlewares"
	"github.com/traefik/traefik/v2/pkg/tracing"
)

const typeName = "PassClientTLSCert"

const (
	xForwardedTLSClientCert     = "X-Forwarded-Tls-Client-Cert"
	xForwardedTLSClientCertInfo = "X-Forwarded-Tls-Client-Cert-Info"
)

const (
	certSeparator     = ","
	fieldSeparator    = ";"
	subFieldSeparator = ","
)

var attributeTypeNames = map[string]string{
	"0.9.2342.19200300.100.1.25": "DC", // Domain component OID - RFC 2247
}

// DistinguishedNameOptions is a struct for specifying the configuration for the distinguished name info.
type DistinguishedNameOptions struct {
	CommonName             bool
	CountryName            bool
	DomainComponent        bool
	LocalityName           bool
	OrganizationName       bool
	OrganizationalUnitName bool
	SerialNumber           bool
	StateOrProvinceName    bool
}

func newDistinguishedNameOptions(info *dynamic.TLSCLientCertificateDNInfo) *DistinguishedNameOptions {
	if info == nil {
		return nil
	}

	return &DistinguishedNameOptions{
		CommonName:             info.CommonName,
		CountryName:            info.Country,
		DomainComponent:        info.DomainComponent,
		LocalityName:           info.Locality,
		OrganizationName:       info.Organization,
		OrganizationalUnitName: info.OrganizationalUnit,
		SerialNumber:           info.SerialNumber,
		StateOrProvinceName:    info.Province,
	}
}

// tlsClientCertificateInfo is a struct for specifying the configuration for the passTLSClientCert middleware.
type tlsClientCertificateInfo struct {
	notAfter     bool
	notBefore    bool
	sans         bool
	subject      *DistinguishedNameOptions
	issuer       *DistinguishedNameOptions
	serialNumber bool
}

func newTLSClientCertificateInfo(info *dynamic.TLSClientCertificateInfo) *tlsClientCertificateInfo {
	if info == nil {
		return nil
	}

	return &tlsClientCertificateInfo{
		issuer:       newDistinguishedNameOptions(info.Issuer),
		notAfter:     info.NotAfter,
		notBefore:    info.NotBefore,
		subject:      newDistinguishedNameOptions(info.Subject),
		serialNumber: info.SerialNumber,
		sans:         info.Sans,
	}
}

// passTLSClientCert is a middleware that helps setup a few tls info features.
type passTLSClientCert struct {
	next http.Handler
	name string
	pem  bool                      // pass the sanitized pem to the backend in a specific header
	info *tlsClientCertificateInfo // pass selected information from the client certificate
}

// New constructs a new PassTLSClientCert instance from supplied frontend header struct.
func New(ctx context.Context, next http.Handler, config dynamic.PassTLSClientCert, name string) (http.Handler, error) {
	log.FromContext(middlewares.GetLoggerCtx(ctx, name, typeName)).Debug("Creating middleware")

	return &passTLSClientCert{
		next: next,
		name: name,
		pem:  config.PEM,
		info: newTLSClientCertificateInfo(config.Info),
	}, nil
}

func (p *passTLSClientCert) GetTracingInformation() (string, ext.SpanKindEnum) {
	return p.name, tracing.SpanKindNoneEnum
}

func (p *passTLSClientCert) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	ctx := middlewares.GetLoggerCtx(req.Context(), p.name, typeName)
	logger := log.FromContext(ctx)

	if p.pem {
		if req.TLS != nil && len(req.TLS.PeerCertificates) > 0 {
			req.Header.Set(xForwardedTLSClientCert, getCertificates(ctx, req.TLS.PeerCertificates))
		} else {
			logger.Warn("Tried to extract a certificate on a request without mutual TLS")
		}
	}

	if p.info != nil {
		if req.TLS != nil && len(req.TLS.PeerCertificates) > 0 {
			headerContent := p.getCertInfo(ctx, req.TLS.PeerCertificates)
			req.Header.Set(xForwardedTLSClientCertInfo, url.QueryEscape(headerContent))
		} else {
			logger.Warn("Tried to extract a certificate on a request without mutual TLS")
		}
	}

	p.next.ServeHTTP(rw, req)
}

// getCertInfo Build a string with the wanted client certificates information
// - the `,` is used to separate certificates
// - the `;` is used to separate root fields
// - the value of root fields is always wrapped by double quote
// - if a field is empty, the field is ignored.
func (p *passTLSClientCert) getCertInfo(ctx context.Context, certs []*x509.Certificate) string {
	var headerValues []string

	for _, peerCert := range certs {
		var values []string

		if p.info != nil {
			subject := getDNInfo(ctx, p.info.subject, &peerCert.Subject)
			if subject != "" {
				values = append(values, fmt.Sprintf(`Subject="%s"`, strings.TrimSuffix(subject, subFieldSeparator)))
			}

			issuer := getDNInfo(ctx, p.info.issuer, &peerCert.Issuer)
			if issuer != "" {
				values = append(values, fmt.Sprintf(`Issuer="%s"`, strings.TrimSuffix(issuer, subFieldSeparator)))
			}

			if p.info.serialNumber && peerCert.SerialNumber != nil {
				sn := peerCert.SerialNumber.String()
				if sn != "" {
					values = append(values, fmt.Sprintf(`SerialNumber="%s"`, strings.TrimSuffix(sn, subFieldSeparator)))
				}
			}

			if p.info.notBefore {
				values = append(values, fmt.Sprintf(`NB="%d"`, uint64(peerCert.NotBefore.Unix())))
			}

			if p.info.notAfter {
				values = append(values, fmt.Sprintf(`NA="%d"`, uint64(peerCert.NotAfter.Unix())))
			}

			if p.info.sans {
				sans := getSANs(peerCert)
				if len(sans) > 0 {
					values = append(values, fmt.Sprintf(`SAN="%s"`, strings.Join(sans, subFieldSeparator)))
				}
			}
		}

		value := strings.Join(values, fieldSeparator)
		headerValues = append(headerValues, value)
	}

	return strings.Join(headerValues, certSeparator)
}

func getDNInfo(ctx context.Context, options *DistinguishedNameOptions, cs *pkix.Name) string {
	if options == nil {
		return ""
	}

	content := &strings.Builder{}

	// Manage non standard attributes
	for _, name := range cs.Names {
		// Domain Component - RFC 2247
		if options.DomainComponent && attributeTypeNames[name.Type.String()] == "DC" {
			content.WriteString(fmt.Sprintf("DC=%s%s", name.Value, subFieldSeparator))
		}
	}

	if options.CountryName {
		writeParts(ctx, content, cs.Country, "C")
	}

	if options.StateOrProvinceName {
		writeParts(ctx, content, cs.Province, "ST")
	}

	if options.LocalityName {
		writeParts(ctx, content, cs.Locality, "L")
	}

	if options.OrganizationName {
		writeParts(ctx, content, cs.Organization, "O")
	}

	if options.OrganizationalUnitName {
		writeParts(ctx, content, cs.OrganizationalUnit, "OU")
	}

	if options.SerialNumber {
		writePart(ctx, content, cs.SerialNumber, "SN")
	}

	if options.CommonName {
		writePart(ctx, content, cs.CommonName, "CN")
	}

	return content.String()
}

func writeParts(ctx context.Context, content io.StringWriter, entries []string, prefix string) {
	for _, entry := range entries {
		writePart(ctx, content, entry, prefix)
	}
}

func writePart(ctx context.Context, content io.StringWriter, entry, prefix string) {
	if len(entry) > 0 {
		_, err := content.WriteString(fmt.Sprintf("%s=%s%s", prefix, entry, subFieldSeparator))
		if err != nil {
			log.FromContext(ctx).Error(err)
		}
	}
}

// sanitize As we pass the raw certificates, remove the useless data and make it http request compliant.
func sanitize(cert []byte) string {
	cleaned := strings.NewReplacer(
		"-----BEGIN CERTIFICATE-----", "",
		"-----END CERTIFICATE-----", "",
		"\n", "",
	).Replace(string(cert))

	return url.QueryEscape(cleaned)
}

// getCertificates Build a string with the client certificates.
func getCertificates(ctx context.Context, certs []*x509.Certificate) string {
	var headerValues []string

	for _, peerCert := range certs {
		headerValues = append(headerValues, extractCertificate(ctx, peerCert))
	}

	return strings.Join(headerValues, certSeparator)
}

// extractCertificate extract the certificate from the request.
func extractCertificate(ctx context.Context, cert *x509.Certificate) string {
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	if certPEM == nil {
		log.FromContext(ctx).Error("Cannot extract the certificate content")
		return ""
	}

	return sanitize(certPEM)
}

// getSANs get the Subject Alternate Name values.
func getSANs(cert *x509.Certificate) []string {
	if cert == nil {
		return nil
	}

	var sans []string
	sans = append(sans, cert.DNSNames...)
	sans = append(sans, cert.EmailAddresses...)

	for _, ip := range cert.IPAddresses {
		sans = append(sans, ip.String())
	}

	for _, uri := range cert.URIs {
		sans = append(sans, uri.String())
	}

	return sans
}
