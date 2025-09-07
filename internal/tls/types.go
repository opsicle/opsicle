package tls

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"net"
	"time"
)

type CertificateAuthorityOptions struct {
	// Subject / metadata
	CommonName   string   `json:"commonName" yaml:"commonName"`
	Organization []string `json:"organization" yaml:"organization"`

	// Validity window
	NotBefore time.Time `json:"notBefore" yaml:"notBefore"`
	NotAfter  time.Time `json:"notAfter"  yaml:"notAfter"`

	// Key parameters
	KeyBits int `json:"keyBits" yaml:"keyBits"`
}

func (o *CertificateAuthorityOptions) ApplyDefaults() {
	if o.CommonName == "" {
		o.CommonName = "default-ca-common-name"
	}
	if len(o.Organization) == 0 {
		o.Organization = []string{"Default CA Organization"}
	}
	if o.NotBefore.IsZero() {
		o.NotBefore = time.Now().Add(-1 * time.Minute)
	}
	if o.NotAfter.IsZero() {
		o.NotAfter = o.NotBefore.Add(365 * 24 * time.Hour)
	}
	if o.KeyBits == 0 {
		o.KeyBits = 4096
	}
}

type CertificateOptions struct {
	Id string `json:"id" yaml:"id"`

	// Subject / metadata
	CommonName   string   `json:"commonName" yaml:"commonName"`
	Organization []string `json:"organization" yaml:"organization"`

	// Usage
	IsClient bool `json:"isClient" yaml:"isClient"` // if false => server

	// SANs
	DNSNames []string `json:"dnsNames" yaml:"dnsNames"`
	IPs      []net.IP `json:"ips"      yaml:"ips"`

	// Validity window
	NotBefore time.Time `json:"notBefore" yaml:"notBefore"`
	NotAfter  time.Time `json:"notAfter"  yaml:"notAfter"`

	// Key parameters
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
	if len(o.Organization) == 0 {
		o.Organization = []string{"Dev"}
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

type Certificate struct {
	TLSCertificate  tls.Certificate   `json:"-" yaml:"-"`
	X509Certificate *x509.Certificate `json:"-" yaml:"-"`
	Pem             []byte            `json:"pem" yaml:"pem"`
	Key             *rsa.PrivateKey   `json:"-" yaml:"-"`
	KeyPem          []byte            `json:"keyPem"  yaml:"keyPem"`
}
