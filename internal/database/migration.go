package database

import (
	"database/sql"
	"embed"
	"fmt"
	"opsicle/internal/common"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type MigrateOpts struct {
	Connection  *sql.DB
	Steps       int
	ServiceLogs chan<- common.ServiceLog
}

func MigrateMysql(opts MigrateOpts) error {
	driver, err := mysql.WithInstance(opts.Connection, &mysql.Config{})
	if err != nil {
		return fmt.Errorf("failed to create mysql driver: %w", err)
	}
	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "established database connection")

	source, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create iofs source: %w", err)
	}
	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "created migrations model")

	migrator, err := migrate.NewWithInstance("iofs", source, "mysql", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrator instance: %w", err)
	}
	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "created migrator instance")

	version, isDirty, err := migrator.Version()
	if err != nil {
		if !strings.Contains(err.Error(), "no migration") {
			return fmt.Errorf("failed to get version of current migration: %s", err)
		}
	}
	if isDirty {
		return fmt.Errorf("failed to get a clean slate to run migrations on (current dirty version: %v)", version)
	}
	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "migrator version: %v (dirty: %v)", version, isDirty)
	if opts.Steps != 0 {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "running %v steps of migrations", opts.Steps)
		if err := migrator.Steps(opts.Steps); err != nil {
			if strings.Contains(err.Error(), "no change") {
				opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "no change detected")
				return nil
			}
			return fmt.Errorf("failed to migrate %v steps: %s", opts.Steps, err)
		}
	} else {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "running all pending migrations")
		if err := migrator.Up(); err != nil {
			if strings.Contains(err.Error(), "no change") {
				opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "no change detected")
				return nil
			}
			return fmt.Errorf("failed to migrate: %s", err)
		}
	}

	return nil
}
