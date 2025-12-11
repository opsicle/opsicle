package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

type ValidateOrgTokenV1Opts struct {
	DatabaseConnection

	TokenId string
	Token   string
}

func ValidateOrgTokenV1(opts ValidateOrgTokenV1Opts) (*OrgToken, error) {
	if err := executeMysqlSelect(mysqlQueryInput{
		Db: opts.Db,
		Stmt: `
			SELECT
				ot.id,
				ot.org_id,
				ot.name,
				ot.api_key,
				orp.id,
				orp.resource,
				orp.allows,
				orp.denys
			FROM org_tokens ot
				LEFT JOIN org_token_roles otr ON otr.org_token_id = ot.id
				LEFT JOIN org_role_permissions orp ON orp.org_role_id = orls.id
			WHERE
				ot.id = ?
		`,
		Args:     []any{opts.TokenId},
		FnSource: "models.ValidateOrgTokenV1",
		ProcessRow: func(r *sql.Row) error {
			var x map[string]interface{}
			r.Scan(&x)
			v, _ := json.MarshalIndent(x, "", "  ")
			fmt.Println(string(v))
			return nil
		},
	}); err != nil {
		return nil, err
	}
	return &OrgToken{}, nil
}
