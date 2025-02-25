// repository/postgres/database_repository.go
package postgres

import (
	"context"
	"fmt"
	"github.com/dfryer1193/gomad/internal/data/repository"
	"github.com/dfryer1193/gomad/internal/data/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"sync"
)

type databaseRepository struct {
	pool *pgxpool.Pool
}

var (
	dbRepo       *databaseRepository
	databaseOnce sync.Once
)

func GetDatabaseRepository() repository.DatabaseRepository {
	databaseOnce.Do(func() {
		connString, err := utils.BuildConnectionString("postgres")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to build connection string for database management")
		}

		pool, err := pgxpool.New(context.Background(), connString)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create connection pool for database management")
		}
		dbRepo = &databaseRepository{pool: pool}
	})

	return dbRepo
}

func (r *databaseRepository) CreateDatabase(ctx context.Context, dbName string, owner string) error {
	exists, err := r.DatabaseExists(ctx, dbName)
	if err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if exists {
		return fmt.Errorf("database %s already exists", dbName)
	}

	query := fmt.Sprintf("CREATE DATABASE %s", pgx.Identifier{dbName}.Sanitize())
	if owner != "" {
		query += fmt.Sprintf(" OWNER %s", pgx.Identifier{owner}.Sanitize())
	}

	_, err = r.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	return nil
}

func (r *databaseRepository) DatabaseExists(ctx context.Context, dbName string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(
        SELECT 1 FROM pg_database WHERE datname = $1
    )`

	err := r.pool.QueryRow(ctx, query, dbName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check database existence: %w", err)
	}

	return exists, nil
}

func (r *databaseRepository) ListDatabases(ctx context.Context) ([]string, error) {
	query := `SELECT datname FROM pg_database WHERE datistemplate = false`

	rows, err := r.pool.Query(ctx, query)
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

func (r *databaseRepository) Close() {
	r.pool.Close()
}
