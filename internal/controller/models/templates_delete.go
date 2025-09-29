package models

func (t *Template) DeleteV1(opts DatabaseConnection) error {
	if err := t.validate(); err != nil {
		return err
	}
	return executeMysqlDelete(mysqlQueryInput{
		Db:           opts.Db,
		Stmt:         `DELETE FROM templates WHERE id = ?`,
		Args:         []any{t.GetId()},
		FnSource:     "models.Template.DeleteV1",
		RowsAffected: oneRowAffected,
	})
}
