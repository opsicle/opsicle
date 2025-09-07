package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

func ensurePrefix(dirOrPrefix string) string {
	ext := filepath.Ext(dirOrPrefix)
	if ext != "" {
		// If user passed a file with extension, strip it to use as prefix.
		dirOrPrefix = stringsTrimSuffix(dirOrPrefix, ext)
	}
	dir := filepath.Dir(dirOrPrefix)
	if dir == "." || dir == "/" || dir == "" {
		dir = "."
	}
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func marshalPKCS8PEM(key *rsa.PrivateKey) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil
}

func randomSerial() *big.Int {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	s, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return big.NewInt(time.Now().UnixNano())
	}
	return s
}

func stringsTrimSuffix(s, suffix string) string {
	if len(suffix) == 0 || len(s) < len(suffix) {
		return s
	}
	if s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}

func writePEMFile(path string, data []byte, mode os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
