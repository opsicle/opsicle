package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type OrgUser struct {
	JoinedAt   time.Time `json:"joinedAt"`
	MemberType string    `json:"memberType"`
	OrgId      string    `json:"orgId"`
	OrgCode    string    `json:"orgCode"`
	OrgName    string    `json:"orgName"`
	UserId     string    `json:"userId"`
	UserEmail  string    `json:"userEmail"`
	UserType   string    `json:"userType"`
}

// Load loads an organisation user based on the `UserId` and `OrgId`,
// if these are empty or not UUIDs, this function will return an
// ErrorInvalidInput error
func (ou *OrgUser) LoadV1(opts DatabaseConnection) error {
	if _, err := uuid.Parse(ou.OrgId); err != nil {
		return fmt.Errorf("org id is not a uuid: %w", ErrorInvalidInput)
	} else if _, err := uuid.Parse(ou.UserId); err != nil {
		return fmt.Errorf("user id is not a uuid: %w", ErrorInvalidInput)
	}
	sqlStmt := `
	SELECT 
		ou.joined_at,
		ou.type,
		u.email,
		u.type,
		o.code,
		o.name,
		FROM org_users ou
			JOIN users u ON ou.user_id = u.id
			JOIN orgs o ON ou.org_id = o.id
		WHERE 
			ou.org_id = ?
			AND ou.user_id = ?
	`
	sqlArgs := []any{ou.OrgId, ou.UserId}
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return fmt.Errorf("models.OrgUser.LoadV1: failed to prepare statement: %w", ErrorStmtPreparationFailed)
	}

	res := stmt.QueryRow(sqlArgs...)
	if res.Err() != nil {
		return fmt.Errorf("models.OrgUser.LoadV1: failed to execute statement: %w", ErrorQueryFailed)
	}
	if err := res.Scan(
		&ou.JoinedAt,
		&ou.MemberType,
		&ou.UserEmail,
		&ou.UserType,
		&ou.OrgCode,
		&ou.OrgName,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrorNotFound
		}
		return fmt.Errorf("models.OrgUser.LoadV1: failed to load selected data into memory: %w", err)
	}
	return nil
}

type UpdateOrgUserFieldsV1 struct {
	Db *sql.DB

	FieldsToSet map[string]any
}

func (ou *OrgUser) UpdateFieldsV1(opts UpdateOrgUserFieldsV1) error {
	sqlArgs := []any{}
	fieldsToSet := []string{}
	for field, value := range opts.FieldsToSet {
		switch v := value.(type) {
		case string:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = ?", field))
			sqlArgs = append(sqlArgs, v)
		case []byte:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = ?", field))
			sqlArgs = append(sqlArgs, string(v))
		default:
			fieldsToSet = append(fieldsToSet, fmt.Sprintf("`%s` = ?", field))
			sqlArgs = append(sqlArgs, fmt.Sprintf("%v", v))
		}
	}
	sqlStmt := fmt.Sprintf(`
	UPDATE org_users
		SET %s
		WHERE org_id = ? AND user_id = ?`, strings.Join(fieldsToSet, ", "))
	sqlArgs = append(sqlArgs, ou.OrgId, ou.UserId)
	stmt, err := opts.Db.Prepare(sqlStmt)
	if err != nil {
		return fmt.Errorf("models.OrgUser.UpdateFieldsV1: failed to prepare insert statement: %w", ErrorStmtPreparationFailed)
	}
	results, err := stmt.Exec(sqlArgs...)
	if err != nil {
		return fmt.Errorf("models.OrgUser.UpdateFieldsV1: failed to execute statement: %w", ErrorQueryFailed)
	}
	rowsAffected, err := results.RowsAffected()
	if err != nil {
		return fmt.Errorf("models.OrgUser.UpdateFieldsV1: failed to get n(rows) affected: %w", ErrorRowsAffectedCheckFailed)
	}
	if rowsAffected != 1 {
		return fmt.Errorf("models.OrgUser.UpdateFieldsV1: n(rows) affected was wrong (got %v): %w", rowsAffected, ErrorRowsAffectedCheckFailed)
	}
	return nil
}
