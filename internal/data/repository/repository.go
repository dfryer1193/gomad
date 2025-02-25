package repository

import (
	"context"
	"github.com/dfryer1193/gomad/api"
)

type MigrationRepository interface {
	Create(ctx context.Context, m *api.Migration) error
	GetFilteredBySignature(ctx context.Context, signatures []uint64) ([]*api.Migration, error)
	GetByNamespace(ctx context.Context, namespace string) ([]*api.Migration, error)
	Close()
}

type DatabaseRepository interface {
	CreateDatabase(ctx context.Context, dbName string, owner string) error
	DatabaseExists(ctx context.Context, dbName string) (bool, error)
	ListDatabases(ctx context.Context) ([]string, error)
	Close()
}
