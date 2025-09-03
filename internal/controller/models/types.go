package models

import "database/sql"

type DatabaseConnection struct {
	Db *sql.DB
}

type DatabaseFunction string
