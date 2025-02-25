package repository

import (
	"context"
	"github.com/dfryer1193/gomad/api"
)

type MigrationRepository interface {
	GetFilteredBySignature(ctx context.Context, signatures []uint64) ([]*api.Migration, error)
	BulkInsert(ctx context.Context, migrations []*api.MigrationProto) error
	Close()
}

type DatabaseRepository interface {
	CreateDatabase(ctx context.Context, dbName string, owner string) error
	DatabaseExists(ctx context.Context, dbName string) (bool, error)
	ListDatabases(ctx context.Context) ([]string, error)
	Close()
}
