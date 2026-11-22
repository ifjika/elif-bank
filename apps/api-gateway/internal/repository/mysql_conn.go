package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

// ==============================================================================
// Repository: MySQL 8.0 / TiDB Cloud Serverless Connection Pool
// Related Documents: docs/TSD.md (Section 1) & .env.example
// ==============================================================================

type DBConfig struct {
	Host         string
	Port         string
	Database     string
	User         string
	Password     string
	SSLMode      bool
	MaxOpenConns int
	MaxIdleConns int
}

// NewMySQLConnection opens and verifies a connection pool to MySQL 8.0 or TiDB Cloud Serverless.
func NewMySQLConnection(ctx context.Context, cfg DBConfig) (*sql.DB, error) {
	if cfg.Host == "" || cfg.User == "" || cfg.Database == "" {
		return nil, fmt.Errorf("missing required database configuration (DB_HOST, DB_USER, DB_NAME)")
	}
	if cfg.Port == "" {
		cfg.Port = "4000" // Default TiDB Cloud Serverless port
	}

	mysqlCfg := mysql.NewConfig()
	mysqlCfg.User = cfg.User
	mysqlCfg.Passwd = cfg.Password
	mysqlCfg.Net = "tcp"
	mysqlCfg.Addr = fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	mysqlCfg.DBName = cfg.Database
	mysqlCfg.ParseTime = true
	mysqlCfg.Loc = time.UTC
	mysqlCfg.Collation = "utf8mb4_unicode_ci"

	// TiDB Cloud Serverless mandates TLS encryption
	if cfg.SSLMode {
		mysqlCfg.TLSConfig = "true"
	}

	db, err := sql.Open("mysql", mysqlCfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize mysql driver: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	} else {
		db.SetMaxOpenConns(25)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	} else {
		db.SetMaxIdleConns(10)
	}
	db.SetConnMaxLifetime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping MySQL/TiDB at %s: %w", mysqlCfg.Addr, err)
	}

	return db, nil
}
