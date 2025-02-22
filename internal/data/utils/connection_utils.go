package utils

import (
	"fmt"
	"os"
	"strconv"
)

// Helper functions

// BuildConnectionString constructs a connection string from environment variables
func BuildConnectionString(dbName string) (string, error) {
	host := getEnvOrDefault("DB_HOST", "localhost")
	portStr := getEnvOrDefault("DB_PORT", "5432")
	user := getEnvOrDefault("DB_USER", "postgres")
	password := os.Getenv("DB_PASSWORD") // No default for password

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", fmt.Errorf("invalid port number: %w", err)
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s",
		user,
		password,
		host,
		port,
		dbName,
	), nil
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
