package models

import "fmt"

// DeleteV1 deletes the organisation identified by its ID.
func (o *Org) DeleteV1(opts DatabaseConnection) error {
	if err := o.assertIdDefined(); err != nil {
		return err
	}
	if opts.Db == nil {
		return fmt.Errorf("missing db connection: %w", errorInputValidationFailed)
	}
	return executeMysqlDelete(mysqlQueryInput{
		Db:           opts.Db,
		Stmt:         `DELETE FROM orgs WHERE id = ?`,
		Args:         []any{o.GetId()},
		FnSource:     "models.Org.DeleteV1",
		RowsAffected: oneRowAffected,
	})
}
