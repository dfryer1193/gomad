package dao

import (
	"context"
	"github.com/dfryer1193/gomad/api"
	"github.com/dfryer1193/gomad/internal/data/utils"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// MigrationDAO handles migration operations
type MigrationDAO struct {
	pool *pgxpool.Pool
}

// NewMigrationDAO creates a new MigrationDAO instance using environment variables
func NewMigrationDAO() *MigrationDAO {
	// Use migrations database for MigrationDAO
	connString, err := utils.BuildConnectionString("migrations")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to build connection string for migrations database")
	}

	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create connection pool for migrations database with connection string " + connString)
	}

	return &MigrationDAO{pool: pool}
}

// Create inserts a new migration record
func (dao *MigrationDAO) Create(ctx context.Context, m *api.Migration) error {
	query := `
        INSERT INTO migrations (namespace, "user", comment, ddl, completed)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id`

	return dao.pool.QueryRow(ctx, query,
		m.Namespace,
		m.User,
		m.Comment,
		m.DDL,
		m.Completed,
	).Scan(&m.ID)
}

// GetByNamespace retrieves all migrations for a given namespace
func (dao *MigrationDAO) GetByNamespace(ctx context.Context, namespace string) ([]*api.Migration, error) {
	query := `
        SELECT id, namespace, "user", comment, ddl, completed
        FROM migrations
        WHERE namespace = $1
        ORDER BY id`

	rows, err := dao.pool.Query(ctx, query, namespace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []*api.Migration
	for rows.Next() {
		m := &api.Migration{}
		err := rows.Scan(
			&m.ID,
			&m.Namespace,
			&m.User,
			&m.Comment,
			&m.DDL,
			&m.Completed,
		)
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, m)
	}

	return migrations, rows.Err()
}

// Close closes the database connection pool
func (dao *MigrationDAO) Close() {
	dao.pool.Close()
}
