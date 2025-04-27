package db

import (
	"fmt"
	"log"
	"time"

	"github.com/e-commerce/platform/internal/common/config"
	"github.com/e-commerce/platform/internal/common/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database is a wrapper around gorm.DB
type Database struct {
	*gorm.DB
}

// NewPostgresDB creates a new database connection
func NewPostgresDB(cfg *config.DatabaseConfig) (*Database, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=UTC",
		cfg.Host, cfg.Username, cfg.Password, cfg.DBName, cfg.Port)

	newLogger := logger.New(
		log.Default(),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)

	// SetMaxOpenConns sets the maximum number of open connections to the database
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeMinutes) * time.Minute)

	return &Database{db}, nil
}

// MigrateSchema creates or updates the database schema
func (db *Database) MigrateSchema() error {
	return db.AutoMigrate(
		&models.Product{},
		&models.Category{},
		&models.Brand{},
		&models.Seller{},
		&models.Image{},
		&models.Video{},
		&models.Variant{},
		&models.Attribute{},
		&models.AttributeValue{},
		&models.PriceHistory{},
		&models.StockHistory{},
		&models.UserFavorite{},
		&models.User{},
		&models.Notification{},
	)
}