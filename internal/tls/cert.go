package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

type Certificate struct {
	TLSCertificate  tls.Certificate   `json:"-" yaml:"-"`
	X509Certificate *x509.Certificate `json:"-" yaml:"-"`
	Pem             []byte            `json:"pem" yaml:"pem"`
}

// Export writes the provided cert to prefix.{crt} with safe perms.
// Returns full paths.
func (c *Certificate) Export(dirOrPrefix, name string) (crtPath string, err error) {
	if len(c.Pem) == 0 {
		return "", errors.New("cert pem is empty")
	}
	prefix := ensurePrefix(dirOrPrefix)
	filename := fmt.Sprintf("%s.crt", name)
	crtPath = filepath.Join(prefix, filename)
	if err = writePEMFile(crtPath, c.Pem, 0644); err != nil {
		return "", fmt.Errorf("write .crt: %w", err)
	}
	return crtPath, nil
}

type CertificateOptions struct {
	// Is the ID of the certificate that will be included in one of the
	// subject's names
	Id string `json:"id" yaml:"id"`

	// CommonName defines the CN field in the certificate's subject
	CommonName string `json:"commonName" yaml:"commonName"`

	// Country defines the C field in the certificate's subject
	Country string `json:"country" yaml:"country"`

	// Organization defines the O field's subject
	Organization []string `json:"organization" yaml:"organization"`

	// OrganizationalUnit defines the OU field in the certificate's subject,
	// applies only for leaf certificates
	OrganizationalUnit []string `json:"organizationalUnit" yaml:"organizationalUnit"`

	Names []pkix.AttributeTypeAndValue `json:"names" yaml:"names"`

	// IsClient specifies whether this is a client (it's a server if falsey),
	// does not apply for CA generation
	IsClient bool `json:"isClient" yaml:"isClient"`

	// DNSNames is a SANs field and applies only for leaf certificates
	DNSNames []string `json:"dnsNames" yaml:"dnsNames"`

	// IPs is a SANs field and applies only for leaf certificates
	IPs []net.IP `json:"ips"      yaml:"ips"`

	// NotBefore indicates when the certificate is valid from
	NotBefore time.Time `json:"notBefore" yaml:"notBefore"`

	// NotAfter indicates when the certificate is valid until
	NotAfter time.Time `json:"notAfter"  yaml:"notAfter"`

	// KeyBits is the number of bits in the generated RSA key (recommended to set to 4096 miniammyl)
	KeyBits int `json:"keyBits" yaml:"keyBits"`
}

func (o *CertificateOptions) ApplyDefaults() {
	if o.CommonName == "" {
		if o.IsClient {
			o.CommonName = "client"
		} else {
			o.CommonName = "server"
		}
	}
	if o.Country == "" {
		o.Country = "SG"
	}
	if len(o.Organization) == 0 {
		o.Organization = []string{"defaultOrg"}
	}
	if o.NotBefore.IsZero() {
		o.NotBefore = time.Now().Add(-1 * time.Minute)
	}
	if o.NotAfter.IsZero() {
		o.NotAfter = o.NotBefore.Add(180 * 24 * time.Hour)
	}
	if o.KeyBits == 0 {
		o.KeyBits = 2048
	}
}

// GenerateCertificateAuthority creates a self-signed CA certificate and key (RSA).
// Returns PEM-encoded cert and key, plus a parsed *x509.Certificate and crypto key usable for signing.
func GenerateCertificateAuthority(opts *CertificateOptions) (*Certificate, *Key, error) {
	opts.ApplyDefaults()

	caKey, err := rsa.GenerateKey(rand.Reader, opts.KeyBits)
	if err != nil {
		return nil, nil, fmt.Errorf("generate ca key: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: randomSerial(),
		Subject: pkix.Name{
			CommonName:         opts.CommonName,
			Organization:       opts.Organization,
			OrganizationalUnit: opts.OrganizationalUnit,
		},
		NotBefore:             opts.NotBefore,
		NotAfter:              opts.NotAfter,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("create ca cert: %w", err)
	}
	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	caKeyPEM, err := marshalPKCS8PEM(caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal ca key: %w", err)
	}
	tlsCert, err := tls.X509KeyPair(caCertPEM, caKeyPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("load tls cert: %w", err)
	}
	x509Cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, fmt.Errorf("parse ca cert: %w", err)
	}

	return &Certificate{
			TLSCertificate:  tlsCert,
			X509Certificate: x509Cert,
			Pem:             caCertPEM,
		}, &Key{
			RsaKey: caKey,
			Pem:    caKeyPEM,
		}, nil
}

// GenerateCertificate issues a leaf certificate signed by the provided CA.
// Returns PEM pair and a ready-to-use tls.Certificate.
func GenerateCertificate(opts *CertificateOptions, caCert *x509.Certificate, caKey *rsa.PrivateKey) (*Certificate, *Key, error) {
	if caCert == nil || caKey == nil {
		return nil, nil, errors.New("caCert and caKey are required")
	}
	opts.ApplyDefaults()

	key, err := rsa.GenerateKey(rand.Reader, opts.KeyBits)
	if err != nil {
		return nil, nil, fmt.Errorf("generate leaf key: %w", err)
	}

	eku := []x509.ExtKeyUsage{}
	if opts.IsClient {
		eku = append(eku, x509.ExtKeyUsageClientAuth)
	} else {
		eku = append(eku, x509.ExtKeyUsageServerAuth)
	}

	template := &x509.Certificate{
		SerialNumber: randomSerial(),
		Subject: pkix.Name{
			CommonName:         opts.CommonName,
			Organization:       opts.Organization,
			OrganizationalUnit: opts.OrganizationalUnit,
		},
		NotBefore:             opts.NotBefore,
		NotAfter:              opts.NotAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           eku,
		BasicConstraintsValid: true,
		DNSNames:              append([]string{}, opts.DNSNames...),
		IPAddresses:           append([]net.IP{}, opts.IPs...),
	}

	// For modern TLS, at least one SAN is recommended for servers.
	if !opts.IsClient && len(template.DNSNames) == 0 && len(template.IPAddresses) == 0 {
		return nil, nil, errors.New("server certificate needs at least one SAN (DNSNames or IPs)")
	}

	der, err := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("sign leaf: %w", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM, err := marshalPKCS8PEM(key)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal leaf key: %w", err)
	}
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("load tls cert: %w", err)
	}
	x509Cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, fmt.Errorf("parse leaf cert: %w", err)
	}

	certInstance := &Certificate{
		TLSCertificate:  tlsCert,
		X509Certificate: x509Cert,
		Pem:             certPEM,
	}
	keyInstance := &Key{
		Pem:    keyPEM,
		RsaKey: key,
	}

	return certInstance, keyInstance, nil
}

// LoadCertificate reads a certificate and key PEM from disk and returns a Certificate struct.
func LoadCertificate(certPath, keyPath string) (*Certificate, error) {
	// --- read PEM files ---
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read cert: %w", err)
	}
	key, err := LoadKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read key: %w", err)
	}

	// --- parse tls.Certificate (chain + key pair) ---
	tlsCert, err := tls.X509KeyPair(certPEM, key.Pem)
	if err != nil {
		return nil, fmt.Errorf("load x509 keypair: %w", err)
	}

	// --- parse leaf x509.Certificate ---
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil || certBlock.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("failed to decode PEM cert")
	}
	x509Cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse x509 cert: %w", err)
	}

	return &Certificate{
		TLSCertificate:  tlsCert,
		X509Certificate: x509Cert,
		Pem:             certPEM,
	}, nil
}
