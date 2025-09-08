package tls

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Key struct {
	Pem    []byte          `json:"pem" yaml:"pem"`
	RsaKey *rsa.PrivateKey `json:"-" yaml:"-"`
}

// Export writes the provided leaf key to prefix.{key} with safe perms.
// Returns full paths.
func (k *Key) Export(dirOrPrefix, name string) (keyPath string, err error) {
	if len(k.Pem) == 0 {
		return "", errors.New("leaf key PEM is empty")
	}
	prefix := ensurePrefix(dirOrPrefix)
	filename := fmt.Sprintf("%s.key", name)
	keyPath = filepath.Join(prefix, filename)
	if err = writePEMFile(keyPath, k.Pem, 0600); err != nil {
		return "", fmt.Errorf("write leaf key: %w", err)
	}
	return keyPath, nil
}

// LoadKey reads a key PEM from disk and returns a Key struct.
func LoadKey(keyPath string) (*Key, error) {
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read key: %w", err)
	}
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

	return &Key{
		Pem:    keyPEM,
		RsaKey: rsaKey,
	}, nil
}
