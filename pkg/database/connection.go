// Package database
package database

import (
	"database/sql"
	"fmt"

	"github.com/bikky-kc013/notification-system/pkg/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DBConnection struct {
	DB  *gorm.DB
	SQL *sql.DB
}

func NewDBConnection(cfg *config.Config) (*DBConnection, error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.DBConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.DBConnMaxIdleTime)

	return &DBConnection{DB: db, SQL: sqlDB}, nil
}

func (c *DBConnection) Close() error {
	return c.SQL.Close()
}

func (c *DBConnection) Ping() error {
	return c.SQL.Ping()
}
