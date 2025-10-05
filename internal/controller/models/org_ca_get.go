package models

import (
	"database/sql"
	"errors"
	"fmt"
)

type LoadOrgCertificateAuthorityV1Opts struct {
	DatabaseConnection
}

func (o *Org) LoadCertificateAuthorityV1(opts LoadOrgCertificateAuthorityV1Opts) (*OrgCertificateAuthority, error) {
	if err := o.assertIdDefined(); err != nil {
		return nil, err
	}
	if opts.Db == nil {
		return nil, fmt.Errorf("missing db connection: %w", errorInputValidationFailed)
	}
	result := NewOrgCertificateAuthority()
	result.Org = o
	row := opts.Db.QueryRow(`
		SELECT
			id,
			cert_b64,
			private_key_b64,
			is_deactivated,
			created_at,
			expires_at
		FROM org_ca
		WHERE org_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, o.GetId())
	var (
		id            string
		isDeactivated bool
		createdAt     sql.NullTime
		expiresAt     sql.NullTime
	)
	if err := row.Scan(&id, &result.CertificateB64, &result.PrivateKeyB64, &isDeactivated, &createdAt, &expiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrorNotFound
		}
		return nil, fmt.Errorf("failed to load org certificate authority: %w", err)
	}
	result.Id = &id
	result.IsDeactivated = isDeactivated
	if createdAt.Valid {
		result.CreatedAt = createdAt.Time
	}
	if expiresAt.Valid {
		result.ExpiresAt = expiresAt.Time
	}
	return &result, nil
}
