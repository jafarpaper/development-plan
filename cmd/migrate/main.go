package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/sirupsen/logrus"

	"activity-log-service/internal/infrastructure/config"
	"activity-log-service/internal/infrastructure/migration"
)

func main() {
	var (
		configPath     = flag.String("config", "configs/config.yaml", "Path to configuration file")
		migrationsPath = flag.String("migrations", "migrations", "Path to migrations directory")
		command        = flag.String("command", "up", "Migration command: up, down, status")
		targetVersion  = flag.Int("version", 0, "Target version for down migration")
	)
	flag.Parse()

	// Setup logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load config")
	}

	// Get database connection
	db, err := getDatabase(cfg)
	if err != nil {
		logger.WithError(err).Fatal("Failed to get database connection")
	}

	// Create migrator
	migrator := migration.NewMigrator(db, logger)
	ctx := context.Background()

	switch *command {
	case "up":
		logger.Info("Running migrations...")
		if err := migrator.Up(ctx, *migrationsPath); err != nil {
			logger.WithError(err).Fatal("Failed to run migrations")
		}
		logger.Info("Migrations completed successfully")

	case "down":
		if *targetVersion < 0 {
			logger.Fatal("Target version must be >= 0 for down migration")
		}
		logger.WithField("target_version", *targetVersion).Info("Rolling back migrations...")
		if err := migrator.Down(ctx, *migrationsPath, *targetVersion); err != nil {
			logger.WithError(err).Fatal("Failed to rollback migrations")
		}
		logger.Info("Rollback completed successfully")

	case "status":
		logger.Info("Checking migration status...")
		if err := showMigrationStatus(ctx, migrator, *migrationsPath); err != nil {
			logger.WithError(err).Fatal("Failed to get migration status")
		}

	default:
		logger.Fatalf("Unknown command: %s. Available commands: up, down, status", *command)
	}
}

func getDatabase(cfg *config.Config) (driver.Database, error) {
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{cfg.Arango.URL},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	client, err := driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: driver.BasicAuthentication(cfg.Arango.Username, cfg.Arango.Password),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	ctx := context.Background()
	db, err := client.Database(ctx, cfg.Arango.Database)
	if driver.IsNotFound(err) {
		db, err = client.CreateDatabase(ctx, cfg.Arango.Database, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create database: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return db, nil
}

func showMigrationStatus(ctx context.Context, migrator *migration.Migrator, migrationsPath string) error {
	migrations, err := migrator.LoadMigrations(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	appliedVersions, err := migrator.GetAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedSet := make(map[int]bool)
	for _, v := range appliedVersions {
		appliedSet[v] = true
	}

	fmt.Println("Migration Status:")
	fmt.Println("================")
	fmt.Printf("%-10s %-30s %-10s\n", "Version", "Name", "Status")
	fmt.Printf("%-10s %-30s %-10s\n", "-------", "----", "------")

	for _, m := range migrations {
		status := "Pending"
		if appliedSet[m.Version] {
			status = "Applied"
		}
		fmt.Printf("%-10d %-30s %-10s\n", m.Version, m.Name, status)
	}

	fmt.Printf("\nTotal migrations: %d\n", len(migrations))
	fmt.Printf("Applied migrations: %d\n", len(appliedVersions))
	fmt.Printf("Pending migrations: %d\n", len(migrations)-len(appliedVersions))

	return nil
}
