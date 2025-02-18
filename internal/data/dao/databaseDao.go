package dao

import (
	"context"
	"fmt"
	"github.com/dfryer1193/gomad/internal/data/utils"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/jackc/pgx/v5"
)

// DatabaseDAO handles database management operations
type DatabaseDAO struct {
	pool *pgxpool.Pool
}

// NewDatabaseDAO creates a new DatabaseDAO instance using environment variables
func NewDatabaseDAO() *DatabaseDAO {
	connString, err := utils.BuildConnectionString("postgres") // Always connect to postgres database
	if err != nil {
		log.Fatal().Err(err).Msg("failed to build connection string for database management")
	}

	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create connection pool for database management with connection string " + connString)
	}

	return &DatabaseDAO{pool: pool}
}

// CreateDatabase creates a new database
func (dao *DatabaseDAO) CreateDatabase(ctx context.Context, dbName string, owner string) error {
	// Check if database already exists
	exists, err := dao.DatabaseExists(ctx, dbName)
	if err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if exists {
		return fmt.Errorf("database %s already exists", dbName)
	}

	// Create the database
	query := fmt.Sprintf("CREATE DATABASE %s", pgx.Identifier{dbName}.Sanitize())
	if owner != "" {
		query += fmt.Sprintf(" OWNER %s", pgx.Identifier{owner}.Sanitize())
	}

	_, err = dao.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	return nil
}

// DatabaseExists checks if a database exists
func (dao *DatabaseDAO) DatabaseExists(ctx context.Context, dbName string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(
        SELECT 1 FROM pg_database WHERE datname = $1
    )`

	err := dao.pool.QueryRow(ctx, query, dbName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check database existence: %w", err)
	}

	return exists, nil
}

// ListDatabases returns a list of all databases
func (dao *DatabaseDAO) ListDatabases(ctx context.Context) ([]string, error) {
	query := `SELECT datname FROM pg_database WHERE datistemplate = false`

	rows, err := dao.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query databases: %w", err)
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			return nil, fmt.Errorf("failed to scan database name: %w", err)
		}
		databases = append(databases, dbName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating database rows: %w", err)
	}

	return databases, nil
}

// Close closes the database connection pool
func (dao *DatabaseDAO) Close() {
	dao.pool.Close()
}
