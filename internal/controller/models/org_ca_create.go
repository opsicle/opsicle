package models

import (
	"encoding/base64"
	"fmt"
	"opsicle/internal/tls"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CreateOrgCertificateAuthorityV1Input struct {
	DatabaseConnection
	CertOptions *tls.CertificateOptions
}

func (o *Org) CreateCertificateAuthorityV1(opts CreateOrgCertificateAuthorityV1Input) (*OrgCertificateAuthority, error) {
	if err := o.assertIdDefined(); err != nil {
		return nil, err
	}
	if opts.Db == nil {
		return nil, fmt.Errorf("missing db connection: %w", errorInputValidationFailed)
	}
	certOpts := opts.CertOptions
	if certOpts == nil {
		certOpts = &tls.CertificateOptions{}
	}
	if certOpts.Id == "" {
		certOpts.Id = o.GetId()
	}
	if certOpts.CommonName == "" {
		certOpts.CommonName = fmt.Sprintf("%s-ca", o.Code)
	}
	if len(certOpts.Organization) == 0 {
		certOpts.Organization = []string{o.Name}
	}
	certificate, key, err := tls.GenerateCertificateAuthority(certOpts)
	if err != nil {
		return nil, fmt.Errorf("generate certificate authority: %w", err)
	}
	certificateB64 := base64.StdEncoding.EncodeToString(certificate.Pem)
	privateKeyB64 := base64.StdEncoding.EncodeToString(key.Pem)
	caId := uuid.NewString()
	insertMap := map[string]any{
		"cert_b64":        certificateB64,
		"expires_at":      certificate.X509Certificate.NotAfter,
		"id":              caId,
		"is_deactivated":  false,
		"org_id":          o.GetId(),
		"private_key_b64": privateKeyB64,
	}
	fieldNames, fieldValues, fieldPlaceholders, err := parseInsertMap(insertMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse insert map: %w", err)
	}
	if err := executeMysqlInsert(mysqlQueryInput{
		Db: opts.Db,
		Stmt: fmt.Sprintf(
			`INSERT INTO org_ca (%s) VALUES (%s)`,
			strings.Join(fieldNames, ", "),
			strings.Join(fieldPlaceholders, ", "),
		),
		Args:         fieldValues,
		FnSource:     "models.Org.CreateCertificateAuthorityV1",
		RowsAffected: oneRowAffected,
	}); err != nil {
		return nil, err
	}
	createdAt := time.Now()
	ca := OrgCertificateAuthority{
		Id:             &caId,
		Org:            o,
		CertificateB64: certificateB64,
		PrivateKeyB64:  privateKeyB64,
		IsDeactivated:  false,
		CreatedAt:      createdAt,
		ExpiresAt:      certificate.X509Certificate.NotAfter,
	}
	return &ca, nil
}
