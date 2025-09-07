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
)

// ExportCertificate writes the provided leaf cert/key to prefix.{crt,key} with safe perms.
// Returns full paths.
func ExportCertificate(dirOrPrefix string, leaf *Certificate) (crtPath, keyPath string, err error) {
	if len(leaf.Pem) == 0 || len(leaf.KeyPem) == 0 {
		return "", "", errors.New("leaf cert/key PEM are empty")
	}
	prefix := ensurePrefix(dirOrPrefix)
	crtPath = filepath.Join(prefix, "cert.crt")
	keyPath = filepath.Join(prefix, "cert.key")
	if err = writePEMFile(crtPath, leaf.Pem, 0644); err != nil {
		return "", "", fmt.Errorf("write leaf crt: %w", err)
	}
	if err = writePEMFile(keyPath, leaf.KeyPem, 0600); err != nil {
		return "", "", fmt.Errorf("write leaf key: %w", err)
	}
	return crtPath, keyPath, nil
}

// GenerateCertificate issues a leaf certificate signed by the provided CA.
// Returns PEM pair and a ready-to-use tls.Certificate.
func GenerateCertificate(opts *CertificateOptions, caCert *x509.Certificate, caKey *rsa.PrivateKey) (*Certificate, error) {
	if caCert == nil || caKey == nil {
		return nil, errors.New("caCert and caKey are required")
	}
	opts.ApplyDefaults()

	key, err := rsa.GenerateKey(rand.Reader, opts.KeyBits)
	if err != nil {
		return nil, fmt.Errorf("generate leaf key: %w", err)
	}

	eku := []x509.ExtKeyUsage{}
	if opts.IsClient {
		eku = append(eku, x509.ExtKeyUsageClientAuth)
	} else {
		eku = append(eku, x509.ExtKeyUsageServerAuth)
	}

	template := &x509.Certificate{
		SerialNumber:          randomSerial(),
		Subject:               pkix.Name{CommonName: opts.CommonName, Organization: opts.Organization},
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
		return nil, errors.New("server certificate needs at least one SAN (DNSNames or IPs)")
	}

	der, err := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("sign leaf: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM, err := marshalPKCS8PEM(key)
	if err != nil {
		return nil, fmt.Errorf("marshal leaf key: %w", err)
	}
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
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
		Pem:             certPEM,
		KeyPem:          keyPEM,
		Key:             key,
	}, nil
}

// LoadCertificate reads a certificate and key PEM from disk and returns a Certificate struct.
func LoadCertificate(certPath, keyPath string) (*Certificate, error) {
	// --- read PEM files ---
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read cert: %w", err)
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read key: %w", err)
	}

	// --- parse tls.Certificate (chain + key pair) ---
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
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

	// --- parse rsa.PrivateKey ---
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode PEM key")
	}
	var rsaKey *rsa.PrivateKey
	switch keyBlock.Type {
	case "RSA PRIVATE KEY":
		rsaKey, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	case "PRIVATE KEY":
		parsedKey, err2 := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("parse PKCS8 key: %w", err2)
		}
		var ok bool
		rsaKey, ok = parsedKey.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
	default:
		return nil, fmt.Errorf("unsupported key type: %s", keyBlock.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("parse RSA key: %w", err)
	}

	return &Certificate{
		TLSCertificate:  tlsCert,
		X509Certificate: x509Cert,
		Pem:             certPEM,
		Key:             rsaKey,
		KeyPem:          keyPEM,
	}, nil
}
