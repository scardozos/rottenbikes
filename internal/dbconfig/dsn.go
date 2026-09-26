// Package dbconfig resolves the database connection string from the
// environment. Shared by the API server and the adminctl CLI so both use the
// same configuration (DATABASE_URL, or the DB_* variables with local dev
// defaults).
package dbconfig

import (
	"fmt"
	"os"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// DSN returns the Postgres connection string.
func DSN() string {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}

	user := getEnv("DB_USER", "rottenbikes")
	pass := getEnv("DB_PASSWORD", "rottenbikes")
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	dbname := getEnv("DB_NAME", "rottenbikes")
	sslmode := getEnv("DB_SSLMODE", "disable")

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, pass, host, port, dbname, sslmode)
}
