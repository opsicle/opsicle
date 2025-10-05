package models

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"opsicle/internal/tls"
)

type OrgCertificateAuthority struct {
	Id             *string   `json:"id" yaml:"id"`
	Org            *Org      `json:"org" yaml:"org"`
	CertificateB64 string    `json:"certificateB64" yaml:"certificateB64"`
	PrivateKeyB64  string    `json:"privateKeyB64" yaml:"privateKeyB64"`
	IsDeactivated  bool      `json:"isDeactivated" yaml:"isDeactivated"`
	CreatedAt      time.Time `json:"createdAt" yaml:"createdAt"`
	ExpiresAt      time.Time `json:"expiresAt" yaml:"expiresAt"`
}

func NewOrgCertificateAuthority() OrgCertificateAuthority {
	return OrgCertificateAuthority{
		Org: &Org{},
	}
}

func (oca OrgCertificateAuthority) GetId() string {
	if oca.Id == nil {
		return ""
	}
	return *oca.Id
}

func (oca OrgCertificateAuthority) GetOrg() *Org {
	if oca.Org == nil {
		return &Org{}
	}
	return oca.Org
}

func (oca OrgCertificateAuthority) GetCryptoMaterials() (*x509.Certificate, *rsa.PrivateKey, error) {
	certBytes, err := base64.StdEncoding.DecodeString(oca.CertificateB64)
	if err != nil {
		return nil, nil, fmt.Errorf("decode certificate: %w", err)
	}
	certBlock, _ := pem.Decode(certBytes)
	if certBlock == nil {
		return nil, nil, errors.New("failed to decode certificate pem block")
	}
	caCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse certificate: %w", err)
	}

	keyBytes, err := base64.StdEncoding.DecodeString(oca.PrivateKeyB64)
	if err != nil {
		return nil, nil, fmt.Errorf("decode private key: %w", err)
	}
	keyBlock, _ := pem.Decode(keyBytes)
	if keyBlock == nil {
		return nil, nil, errors.New("failed to decode private key pem block")
	}
	var rsaKey *rsa.PrivateKey
	switch keyBlock.Type {
	case "PRIVATE KEY":
		parsedKey, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("parse pkcs8 private key: %w", err)
		}
		var ok bool
		rsaKey, ok = parsedKey.(*rsa.PrivateKey)
		if !ok {
			return nil, nil, errors.New("private key is not rsa")
		}
	case "RSA PRIVATE KEY":
		parsedKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("parse pkcs1 private key: %w", err)
		}
		rsaKey = parsedKey
	default:
		return nil, nil, fmt.Errorf("unsupported private key type: %s", keyBlock.Type)
	}

	return caCert, rsaKey, nil
}

type RegenerateOrgCertificateAuthorityV1Input struct {
	DatabaseConnection
	CertOptions *tls.CertificateOptions
}

func (oca *OrgCertificateAuthority) RegenerateV1(opts RegenerateOrgCertificateAuthorityV1Input) error {
	if oca == nil || oca.Id == nil {
		return fmt.Errorf("org certificate authority undefined: %w", errorInputValidationFailed)
	}
	if opts.Db == nil {
		return fmt.Errorf("missing db connection: %w", errorInputValidationFailed)
	}
	certOpts := opts.CertOptions
	if certOpts == nil {
		certOpts = &tls.CertificateOptions{}
	}
	if certOpts.Id == "" {
		if oca.Org != nil && oca.Org.Id != nil {
			certOpts.Id = *oca.Org.Id
		} else {
			certOpts.Id = *oca.Id
		}
	}
	if oca.Org != nil && certOpts.CommonName == "" {
		certOpts.CommonName = fmt.Sprintf("%s-ca", oca.Org.Code)
	}
	if oca.Org != nil && len(certOpts.Organization) == 0 {
		certOpts.Organization = []string{oca.Org.Name}
	}
	certificate, key, err := tls.GenerateCertificateAuthority(certOpts)
	if err != nil {
		return fmt.Errorf("generate certificate authority: %w", err)
	}
	certificateB64 := base64.StdEncoding.EncodeToString(certificate.Pem)
	privateKeyB64 := base64.StdEncoding.EncodeToString(key.Pem)
	updateMap := map[string]any{
		"cert_b64":        certificateB64,
		"expires_at":      certificate.X509Certificate.NotAfter,
		"is_deactivated":  false,
		"private_key_b64": privateKeyB64,
	}
	fieldNames, fieldSetters, fieldValues, err := parseUpdateMap(updateMap)
	if err != nil {
		return fmt.Errorf("failed to parse update map: %w", err)
	}
	fieldValues = append(fieldValues, *oca.Id)
	if err := executeMysqlUpdate(mysqlQueryInput{
		Db: opts.Db,
		Stmt: fmt.Sprintf(
			`UPDATE org_ca SET %s WHERE id = ?`,
			strings.Join(fieldSetters, ", "),
		),
		Args:         fieldValues,
		FnSource:     fmt.Sprintf("models.OrgCertificateAuthority.RegenerateV1[%s]", strings.Join(fieldNames, ",")),
		RowsAffected: oneRowAffected,
	}); err != nil {
		return err
	}
	oca.CertificateB64 = certificateB64
	oca.PrivateKeyB64 = privateKeyB64
	oca.IsDeactivated = false
	oca.ExpiresAt = certificate.X509Certificate.NotAfter
	return nil
}
