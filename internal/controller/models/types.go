package models

import "database/sql"

type DatabaseConnection struct {
	Db *sql.DB
}

type DatabaseFunction string

type UpdateFieldsV1 struct {
	Db *sql.DB

	FieldsToSet map[string]any
}
