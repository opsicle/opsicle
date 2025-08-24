package models

import "database/sql"

type DatabaseConnection struct {
	Db *sql.DB
}
