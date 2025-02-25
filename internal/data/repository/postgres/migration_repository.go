package postgres

import (
	"context"
	"fmt"
	"github.com/dfryer1193/gomad/api"
	"github.com/dfryer1193/gomad/internal/data/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"reflect"
	"sync"
)

type MigrationRepository struct {
	pool *pgxpool.Pool
}

var (
	migrationRepository *MigrationRepository
	migrationOnce       sync.Once
)

func NewMigrationRepository() *MigrationRepository {
	migrationOnce.Do(func() {
		connString, err := utils.BuildConnectionString("migrations")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to build connection string for migrations database")
		}

		pool, err := pgxpool.New(context.Background(), connString)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create connection pool for migrations database")
		}
		migrationRepository = &MigrationRepository{pool: pool}
	})
	return migrationRepository
}

func (r *MigrationRepository) queryMigrations(ctx context.Context, query string, args ...any) ([]*api.Migration, error) {
	rows, err := r.pool.Query(ctx, query, args...)
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
			&m.CompletedAt,
		)
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return migrations, nil
}

func (r *MigrationRepository) GetFilteredBySignature(ctx context.Context, signatures []uint64) ([]*api.Migration, error) {
	if len(signatures) == 0 {
		return []*api.Migration{}, nil
	}

	query := `
		SELECT id, namespace, "user", comment, ddl, completedAt
		FROM migrations
		WHERE id = ANY($1)
		ORDER BY Created ASC`

	migrations, err := r.queryMigrations(ctx, query, signatures)
	if err != nil {
		return nil, err
	}

	return migrations, nil
}

func (r *MigrationRepository) BulkInsert(ctx context.Context, migrations []*api.MigrationProto) error {
	if len(migrations) == 0 {
		return nil
	}

	columns := []string{"id", "namespace", "user", "comment", "ddl", "createdAt", "shouldSkip"}
	rows := make([][]any, len(migrations))

	for i, m := range migrations {
		rows[i] = []any{
			m.Signature, // id from Signature field
			m.Namespace,
			m.User,
			m.Comment,
			m.DDL,
			m.CreatedAt,
			m.ShouldSkip,
		}
	}

	// Use CopyFrom for efficient bulk insert
	_, err := r.pool.CopyFrom(
		ctx,
		pgx.Identifier{"migrations"},
		columns,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("failed to bulk insert migrations: %w", err)
	}

	return nil
}
