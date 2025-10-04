package models

// DeleteV1 deletes a user identified by a combination of the
// `UserId` and `OrgId`. If both those properties are not
// valid/populated, an error is thrown
func (ou *OrgUser) DeleteV1(opts DatabaseConnection) error {
	if err := ou.validate(); err != nil {
		return err
	}
	return executeMysqlDelete(mysqlQueryInput{
		Db:           opts.Db,
		Stmt:         `DELETE FROM org_users WHERE org_id = ? AND user_id = ?`,
		Args:         []any{ou.Org.GetId(), ou.User.GetId()},
		FnSource:     "models.OrgUser.DeleteV1",
		RowsAffected: oneRowAffected,
	})
}
