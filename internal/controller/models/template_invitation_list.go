package models

import (
	"database/sql"
	"fmt"
)

type ListTemplateInvitationsV1Output []TemplateUserInvitation

type ListTemplateInvitationsV1Opts struct {
	Db *sql.DB

	UserEmail *string
	UserId    *string
}

// ListTemplateInvitationsV1 returns a list of invitations to organisations
// where the specified user email is pending account creation
func ListTemplateInvitationsV1(opts ListTemplateInvitationsV1Opts) (ListTemplateInvitationsV1Output, error) {
	sqlSelector := "acceptor_id"
	sqlArgs := []any{}
	if opts.UserEmail != nil {
		sqlSelector = "acceptor_email"
		sqlArgs = append(sqlArgs, *opts.UserEmail)
	} else if opts.UserId != nil {
		sqlArgs = append(sqlArgs, *opts.UserId)
	} else {
		return nil, fmt.Errorf("models.ListTemplateInvitationsV1: failed to receive valid input: %w", ErrorInvalidInput)
	}

	output := ListTemplateInvitationsV1Output{}
	if err := executeMysqlSelects(mysqlQueryInput{
		Db: opts.Db,
		Stmt: fmt.Sprintf(`
			SELECT 
				at.id,
				at.name,
				tui.id,
				u.id,
				u.email,
				tui.join_code,
				tui.can_view,
				tui.can_execute,
				tui.can_update,
				tui.can_delete,
				tui.can_invite,
				tui.created_at
				FROM automation_template_user_invitations tui
					JOIN automation_templates at ON at.id = tui.automation_template_id
					JOIN users u ON u.id = tui.inviter_id
				WHERE tui.%s = ?
			`,
			sqlSelector,
		),
		Args:     sqlArgs,
		FnSource: "models.ListTemplateInvitationsV1",
		ProcessRows: func(r *sql.Rows) error {
			var templateUserInvitation TemplateUserInvitation
			if err := r.Scan(
				&templateUserInvitation.TemplateId,
				&templateUserInvitation.TemplateName,
				&templateUserInvitation.Id,
				&templateUserInvitation.InviterId,
				&templateUserInvitation.InviterEmail,
				&templateUserInvitation.JoinCode,
				&templateUserInvitation.CanView,
				&templateUserInvitation.CanExecute,
				&templateUserInvitation.CanUpdate,
				&templateUserInvitation.CanDelete,
				&templateUserInvitation.CanInvite,
				&templateUserInvitation.CreatedAt,
			); err != nil {
				return err
			}
			output = append(output, templateUserInvitation)
			return nil
		},
	}); err != nil {
		return nil, err
	}
	return output, nil
}
