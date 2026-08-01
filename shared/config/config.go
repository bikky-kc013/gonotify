// Package config
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Env             string
	Port            string
	DatabaseURL     string
	LogRootPath     string
	LogLevel        string
	LogMaxSizeMB    int
	LogMaxAgeDays   int
	LogMaxBackups   int
	ShutdownTimeout time.Duration

	DBMaxIdleConns    int
	DBMaxOpenConns    int
	DBConnMaxLifetime time.Duration
	DBConnMaxIdleTime time.Duration

	CORSAllowedOrigins []string
}

func Load() (*Config, error) {
	env := os.Getenv("ENV")
	if env == "" {
		env = "local"
	}

	if env == "local" || env == "development" {
		err := godotenv.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: no .env file found, using system env vars: %v\n", err)
		}
	}

	cfg := &Config{
		Env:                env,
		Port:               getEnvOrDefault("PORT", ":8080"),
		DatabaseURL:        os.Getenv("DATABASE_CONN_URL"),
		LogRootPath:        getEnvOrDefault("LOG_ROOT_PATH", "./logs"),
		LogLevel:           getEnvOrDefault("LOG_LEVEL", "info"),
		LogMaxSizeMB:       getEnvInt("LOG_MAX_SIZE_MB", 100),
		LogMaxAgeDays:      getEnvInt("LOG_MAX_AGE_DAYS", 30),
		LogMaxBackups:      getEnvInt("LOG_MAX_BACKUPS", 5),
		ShutdownTimeout:    getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
		DBMaxIdleConns:     getEnvInt("DB_MAX_IDLE_CONNS", 10),
		DBMaxOpenConns:     getEnvInt("DB_MAX_OPEN_CONNS", 100),
		DBConnMaxLifetime:  getEnvDuration("DB_CONN_MAX_LIFETIME", time.Hour),
		DBConnMaxIdleTime:  getEnvDuration("DB_CONN_MAX_IDLE_TIME", 30*time.Minute),
		CORSAllowedOrigins: getEnvSlice("CORS_ALLOWED_ORIGINS", []string{"*"}),
	}

	if !strings.HasPrefix(cfg.Port, ":") {
		cfg.Port = ":" + cfg.Port
	}

	var errs []string
	if cfg.DatabaseURL == "" {
		errs = append(errs, "DATABASE_CONN_URL is required")
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("config validation failed:\n%s", strings.Join(errs, "\n"))
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return n
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return defaultVal
	}
	return d
}

func getEnvSlice(key string, defaultVal []string) []string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	parts := strings.Split(val, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
