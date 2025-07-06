package migration

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/arangodb/go-driver"
	"github.com/sirupsen/logrus"
)

type Migrator struct {
	db     driver.Database
	logger *logrus.Logger
}

type Migration struct {
	Version    int
	Name       string
	UpScript   string
	DownScript string
}

func NewMigrator(db driver.Database, logger *logrus.Logger) *Migrator {
	return &Migrator{
		db:     db,
		logger: logger,
	}
}

func (m *Migrator) LoadMigrations(migrationsPath string) ([]Migration, error) {
	migrations := make(map[int]Migration)

	err := filepath.WalkDir(migrationsPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		filename := d.Name()
		if !strings.HasSuffix(filename, ".aql") {
			return nil
		}

		// Parse migration filename: 001_migration_name.up.aql or 001_migration_name.down.aql
		parts := strings.Split(filename, "_")
		if len(parts) < 2 {
			return fmt.Errorf("invalid migration filename format: %s", filename)
		}

		versionStr := parts[0]
		version, err := strconv.Atoi(versionStr)
		if err != nil {
			return fmt.Errorf("invalid version number in filename %s: %w", filename, err)
		}

		name := strings.Join(parts[1:], "_")
		name = strings.TrimSuffix(name, ".up.aql")
		name = strings.TrimSuffix(name, ".down.aql")

		migration, exists := migrations[version]
		if !exists {
			migration = Migration{
				Version: version,
				Name:    name,
			}
		}

		content, err := fs.ReadFile(os.DirFS(migrationsPath), filename)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		if strings.Contains(filename, ".up.aql") {
			migration.UpScript = string(content)
		} else if strings.Contains(filename, ".down.aql") {
			migration.DownScript = string(content)
		}

		migrations[version] = migration
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	// Convert map to sorted slice
	var result []Migration
	for _, migration := range migrations {
		result = append(result, migration)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})

	return result, nil
}

func (m *Migrator) CreateMigrationsCollection(ctx context.Context) error {
	exists, err := m.db.CollectionExists(ctx, "migrations")
	if err != nil {
		return fmt.Errorf("failed to check if migrations collection exists: %w", err)
	}

	if !exists {
		_, err = m.db.CreateCollection(ctx, "migrations", nil)
		if err != nil {
			return fmt.Errorf("failed to create migrations collection: %w", err)
		}
		m.logger.Info("Created migrations collection")
	}

	return nil
}

func (m *Migrator) GetAppliedMigrations(ctx context.Context) ([]int, error) {
	_, err := m.db.Collection(ctx, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to get migrations collection: %w", err)
	}

	query := "FOR m IN migrations SORT m.version ASC RETURN m.version"
	cursor, err := m.db.Query(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer cursor.Close()

	var versions []int
	for cursor.HasMore() {
		var version int
		_, err := cursor.ReadDocument(ctx, &version)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration version: %w", err)
		}
		versions = append(versions, version)
	}

	return versions, nil
}

func (m *Migrator) RecordMigration(ctx context.Context, version int, name string) error {
	collection, err := m.db.Collection(ctx, "migrations")
	if err != nil {
		return fmt.Errorf("failed to get migrations collection: %w", err)
	}

	doc := map[string]interface{}{
		"_key":       fmt.Sprintf("%03d", version),
		"version":    version,
		"name":       name,
		"applied_at": time.Now(),
	}

	_, err = collection.CreateDocument(ctx, doc)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return nil
}

func (m *Migrator) RemoveMigrationRecord(ctx context.Context, version int) error {
	collection, err := m.db.Collection(ctx, "migrations")
	if err != nil {
		return fmt.Errorf("failed to get migrations collection: %w", err)
	}

	key := fmt.Sprintf("%03d", version)
	_, err = collection.RemoveDocument(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	return nil
}

func (m *Migrator) ExecuteAQL(ctx context.Context, script string) error {
	if strings.TrimSpace(script) == "" {
		return nil
	}

	cursor, err := m.db.Query(ctx, script, nil)
	if err != nil {
		return fmt.Errorf("failed to execute AQL script: %w", err)
	}
	defer cursor.Close()

	// Consume all results to ensure the query completes
	for cursor.HasMore() {
		var result interface{}
		cursor.ReadDocument(ctx, &result)
	}

	return nil
}

func (m *Migrator) Up(ctx context.Context, migrationsPath string) error {
	if err := m.CreateMigrationsCollection(ctx); err != nil {
		return err
	}

	migrations, err := m.LoadMigrations(migrationsPath)
	if err != nil {
		return err
	}

	appliedVersions, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	appliedSet := make(map[int]bool)
	for _, v := range appliedVersions {
		appliedSet[v] = true
	}

	for _, migration := range migrations {
		if appliedSet[migration.Version] {
			m.logger.WithFields(logrus.Fields{
				"version": migration.Version,
				"name":    migration.Name,
			}).Info("Migration already applied, skipping")
			continue
		}

		m.logger.WithFields(logrus.Fields{
			"version": migration.Version,
			"name":    migration.Name,
		}).Info("Applying migration")

		if err := m.ExecuteAQL(ctx, migration.UpScript); err != nil {
			return fmt.Errorf("failed to apply migration %d (%s): %w", migration.Version, migration.Name, err)
		}

		if err := m.RecordMigration(ctx, migration.Version, migration.Name); err != nil {
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		m.logger.WithFields(logrus.Fields{
			"version": migration.Version,
			"name":    migration.Name,
		}).Info("Migration applied successfully")
	}

	return nil
}

func (m *Migrator) Down(ctx context.Context, migrationsPath string, targetVersion int) error {
	migrations, err := m.LoadMigrations(migrationsPath)
	if err != nil {
		return err
	}

	appliedVersions, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	// Sort applied versions in descending order for rollback
	sort.Sort(sort.Reverse(sort.IntSlice(appliedVersions)))

	for _, version := range appliedVersions {
		if version <= targetVersion {
			break
		}

		// Find the migration with this version
		var migration *Migration
		for _, m := range migrations {
			if m.Version == version {
				migration = &m
				break
			}
		}

		if migration == nil {
			m.logger.WithField("version", version).Warn("Migration not found for rollback, skipping")
			continue
		}

		m.logger.WithFields(logrus.Fields{
			"version": migration.Version,
			"name":    migration.Name,
		}).Info("Rolling back migration")

		if err := m.ExecuteAQL(ctx, migration.DownScript); err != nil {
			return fmt.Errorf("failed to rollback migration %d (%s): %w", migration.Version, migration.Name, err)
		}

		if err := m.RemoveMigrationRecord(ctx, migration.Version); err != nil {
			return fmt.Errorf("failed to remove migration record %d: %w", migration.Version, err)
		}

		m.logger.WithFields(logrus.Fields{
			"version": migration.Version,
			"name":    migration.Name,
		}).Info("Migration rolled back successfully")
	}

	return nil
}
