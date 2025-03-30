package postgres

import (
	"context"
	"sync"

	"github.com/dfryer1193/gomad/internal/data/utils"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type secretRepository struct {
	pool *pgxpool.Pool
}

var (
	secretsRepo *secretRepository
	secretsOnce sync.Once
)

func GetSecretsRepository() *secretRepository {
	secretsOnce.Do(func() {
		connString, err := utils.BuildConnectionString("secrets")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to build connection string for secrets database")
		}

		pool, err := pgxpool.New(context.Background(), connString)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create connection pool for secrets database")
		}
		secretsRepo = &secretRepository{pool: pool}
	})

	return secretsRepo
}

func (r *secretRepository) InsertSecret(repoName string, secret string) (string, error) {
	query := `INSERT INTO webhook_secrets (repo_name, secret) VALUES ($1, $2) RETURNING secret`
	var savedSecret string
	err := r.pool.QueryRow(context.Background(), query, repoName, secret).Scan(&savedSecret)
	if err != nil {
		return "", err
	}

	return savedSecret, nil
}

func (r *secretRepository) GetSecret(repoName string) (string, error) {
	query := `SELECT secret FROM webhook_secrets WHERE repo_name = $1`
	var secret string
	err := r.pool.QueryRow(context.Background(), query, repoName).Scan(&secret)
	if err != nil {
		return "", err
	}

	return secret, nil
}

func (r *secretRepository) Close() {
	r.pool.Close()
}
