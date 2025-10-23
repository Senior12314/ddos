package database

import (
	"fmt"
	"time"

	"github.com/cloudnordsp/minecraft-protection/internal/config"
	"github.com/cloudnordsp/minecraft-protection/internal/storage"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// New creates a new database connection using GORM
func New(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.Database, cfg.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB for connection pool configuration
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.MaxLifetime)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// runMigrations runs database migrations using GORM AutoMigrate
func runMigrations(db *gorm.DB) error {
	// Auto-migrate all models
	err := db.AutoMigrate(
		&storage.User{},
		&storage.Organization{},
		&storage.ProtectedEndpoint{},
		&storage.Node{},
		&storage.IPWhitelist{},
		&storage.IPBlacklist{},
		&storage.IPRule{},
		&storage.Metric{},
		&storage.AuditLog{},
	)
	if err != nil {
		return fmt.Errorf("failed to auto-migrate: %w", err)
	}

	// Create additional indexes for performance
	if err := createIndexes(db); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// createIndexes creates additional indexes for better performance
func createIndexes(db *gorm.DB) error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_protected_endpoints_front ON protected_endpoints(front_ip, front_port)",
		"CREATE INDEX IF NOT EXISTS idx_protected_endpoints_active ON protected_endpoints(active)",
		"CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status)",
		"CREATE INDEX IF NOT EXISTS idx_nodes_last_seen ON nodes(last_seen)",
		"CREATE INDEX IF NOT EXISTS idx_ip_blacklist_expires_at ON ip_blacklists(expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_metrics_endpoint_id ON metrics(endpoint_id)",
		"CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource, resource_id)",
	}

	for _, index := range indexes {
		if err := db.Exec(index).Error; err != nil {
			return fmt.Errorf("failed to create index %s: %w", index, err)
		}
	}

	return nil
}