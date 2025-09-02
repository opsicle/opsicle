package models

import (
	"database/sql"
	"fmt"
)

type ListOrgInvitationsV1Output []OrgUserInvitation

type ListOrgInvitationsV1Opts struct {
	Db *sql.DB

	UserEmail *string
	UserId    *string
}

// ListOrgInvitationsV1 returns a list of invitations to organisations
// where the specified user email is pending account creation
func ListOrgInvitationsV1(opts ListOrgInvitationsV1Opts) (ListOrgInvitationsV1Output, error) {
	sqlSelector := "acceptor_id"
	sqlArgs := []any{}
	if opts.UserEmail != nil {
		sqlSelector = "acceptor_email"
		sqlArgs = append(sqlArgs, *opts.UserEmail)
	} else if opts.UserId != nil {
		sqlArgs = append(sqlArgs, *opts.UserId)
	} else {
		return nil, fmt.Errorf("models.ListOrgInvitationsV1: failed to receive valid input: %w", ErrorInvalidInput)
	}

	output := ListOrgInvitationsV1Output{}
	if err := executeMysqlSelects(mysqlQueryInput{
		Db: opts.Db,
		Stmt: fmt.Sprintf(`
			SELECT 
				o.id,
				o.name,
				o.code,
				oui.id,
				oui.join_code,
				oui.created_at,
				u.id,
				u.email
				FROM org_user_invitations oui
					JOIN orgs o ON o.id = oui.org_id
					JOIN users u ON u.id = oui.inviter_id
				WHERE oui.%s = ?
			`,
			sqlSelector,
		),
		Args:     sqlArgs,
		FnSource: "models.ListOrgInvitationsV1",
		ProcessRows: func(r *sql.Rows) error {
			var orgInvitation OrgUserInvitation
			if err := r.Scan(
				&orgInvitation.OrgId,
				&orgInvitation.OrgName,
				&orgInvitation.OrgCode,
				&orgInvitation.Id,
				&orgInvitation.JoinCode,
				&orgInvitation.CreatedAt,
				&orgInvitation.InviterId,
				&orgInvitation.InviterEmail,
			); err != nil {
				return err
			}
			output = append(output, orgInvitation)
			return nil
		},
	}); err != nil {
		return nil, err
	}
	return output, nil
}
