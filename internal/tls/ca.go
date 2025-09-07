package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

// ExportCertificateAuthority writes the CA cert/key to prefix.{crt,key} with safe perms.
// Returns full paths.
func ExportCertificateAuthority(dirOrPrefix string, ca *Certificate) (crtPath, keyPath string, err error) {
	prefix := ensurePrefix(dirOrPrefix)
	crtPath = filepath.Join(prefix, "ca.crt")
	keyPath = filepath.Join(prefix, "ca.key")
	if err = writePEMFile(crtPath, ca.Pem, 0644); err != nil {
		return "", "", fmt.Errorf("write CA crt: %w", err)
	}
	if err = writePEMFile(keyPath, ca.KeyPem, 0600); err != nil {
		return "", "", fmt.Errorf("write CA key: %w", err)
	}
	return crtPath, keyPath, nil
}

// GenerateCertificateAuthority creates a self-signed CA certificate and key (RSA).
// Returns PEM-encoded cert and key, plus a parsed *x509.Certificate and crypto key usable for signing.
func GenerateCertificateAuthority(opts *CertificateAuthorityOptions) (ca *Certificate, err error) {
	opts.ApplyDefaults()

	caKey, err := rsa.GenerateKey(rand.Reader, opts.KeyBits)
	if err != nil {
		return nil, fmt.Errorf("generate CA key: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber:          randomSerial(),
		Subject:               pkix.Name{CommonName: opts.CommonName, Organization: opts.Organization},
		NotBefore:             opts.NotBefore,
		NotAfter:              opts.NotAfter,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("create CA cert: %w", err)
	}
	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	caKeyPEM, err := marshalPKCS8PEM(caKey)
	if err != nil {
		return nil, fmt.Errorf("marshal CA key: %w", err)
	}
	tlsCert, err := tls.X509KeyPair(caCertPEM, caKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("load tls cert: %w", err)
	}
	x509Cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("parse CA cert: %w", err)
	}

	return &Certificate{
		TLSCertificate:  tlsCert,
		X509Certificate: x509Cert,
		Pem:             caCertPEM,
		KeyPem:          caKeyPEM,
		Key:             caKey,
	}, nil
}

func LoadCertificateAuthority(caCertPath, caKeyPath string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// --- load cert ---
	certPEM, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read cert: %w", err)
	}
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, nil, fmt.Errorf("failed to decode PEM certificate")
	}
	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse cert: %w", err)
	}

	// --- load key ---
	keyPEM, err := os.ReadFile(caKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read key: %w", err)
	}
	block, _ = pem.Decode(keyPEM)
	if block == nil {
		return nil, nil, fmt.Errorf("failed to decode PEM key")
	}

	var caKey *rsa.PrivateKey
	switch block.Type {
	case "RSA PRIVATE KEY":
		caKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		// PKCS#8 (may be RSA or ECDSA; assert type)
		key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, nil, fmt.Errorf("parse pkcs8 key: %w", err2)
		}
		var ok bool
		caKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, nil, fmt.Errorf("not an RSA key")
		}
	default:
		return nil, nil, fmt.Errorf("unsupported key type %q", block.Type)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("parse key: %w", err)
	}

	return caCert, caKey, nil
}
