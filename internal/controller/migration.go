package controller

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"math"
	"opsicle/internal/common"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type MigrateDatabaseOutput struct {
	PostMigrationVersion uint
	PreMigrationVersion  uint
	IsDatabaseDirty      bool
	VersionsApplied      []uint
}

type MigrateDatabaseOpts struct {
	Connection  *sql.DB
	Force       int
	Steps       *int
	ServiceLogs chan<- common.ServiceLog
}

func MigrateDatabase(opts MigrateDatabaseOpts) (*MigrateDatabaseOutput, error) {
	driver, err := mysql.WithInstance(opts.Connection, &mysql.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create mysql driver: %w", err)
	}
	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "established database connection")

	source, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to create iofs source: %w", err)
	}
	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "created migrations model")

	migrator, err := migrate.NewWithInstance("iofs", source, "mysql", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator instance: %w", err)
	}
	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "created migrator instance")

	output := &MigrateDatabaseOutput{}

	output.PreMigrationVersion, output.IsDatabaseDirty, err = migrator.Version()
	if err != nil {
		if !strings.Contains(err.Error(), "no migration") {
			return nil, fmt.Errorf("failed to get version of current migration: %w", err)
		}
	}
	if output.IsDatabaseDirty {
		if opts.Force != 0 {
			if err := migrator.Force(opts.Force); err != nil {
				return nil, fmt.Errorf("failed to force version[%v]: %w", opts.Force, err)
			}
			version, isDirty, _ := migrator.Version()
			output.PostMigrationVersion = version
			output.IsDatabaseDirty = isDirty
			return output, nil
		} else {
			return output, fmt.Errorf("failed to get a clean slate to run migrations on (current dirty version: %v)", output.IsDatabaseDirty)
		}
	}
	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "migrator version: %v (dirty: %v)", output.PreMigrationVersion, output.IsDatabaseDirty)
	direction := 1
	steps := 0
	if opts.Steps == nil {
		steps = -1
	} else {
		steps = *opts.Steps
		if steps < 0 {
			direction = -1
		}
		steps = int(math.Abs(float64(steps)))
	}
	isEndReached := false
	isFailed := false
	var migrationErr error
	for !isEndReached && steps != 0 {
		preVersion, preVersionIsDirty, _ := migrator.Version()
		if err := migrator.Steps(direction); err != nil {
			if strings.Contains(err.Error(), "file does not exist") {
				opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "file does not exist")
				isEndReached = true
			} else if errors.Is(err, migrate.ErrNoChange) {
				opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "no change detected")
				isEndReached = true
			} else {
				isFailed = true
				migrationErr = err
			}
		} else {
			if direction > 0 {
				version, isDirty, _ := migrator.Version()
				output.VersionsApplied = append(output.VersionsApplied, version)
				output.IsDatabaseDirty = isDirty
			} else if direction < 0 {
				output.VersionsApplied = append(output.VersionsApplied, preVersion)
				output.IsDatabaseDirty = preVersionIsDirty
			}
		}
		steps--
	}
	version, isDirty, _ := migrator.Version()
	output.PostMigrationVersion = version
	output.IsDatabaseDirty = isDirty
	if isFailed {
		return output, fmt.Errorf("failed to complete migrations: %w", migrationErr)
	}
	return output, nil
}
