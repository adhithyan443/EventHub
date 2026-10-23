package database

import (
	"context"
	"errors"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	ctx := context.Background()
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return err
	}

	driver, err := migratepostgres.WithConnection(
		ctx,
		conn,
		&migratepostgres.Config{},
	)
	if err != nil {
		_ = conn.Close()
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		_ = driver.Close()
		return err
	}
	defer func() {
		sourceErr, databaseErr := m.Close()
		if sourceErr != nil {
			slog.Warn("migration_source_close_failed", "error", sourceErr)
		}
		if databaseErr != nil {
			slog.Warn("migration_database_close_failed", "error", databaseErr)
		}
	}()

	slog.Info("database_migration_started")

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	slog.Info("database_migration_completed")
	return nil
}
