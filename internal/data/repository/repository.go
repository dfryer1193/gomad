package repository

import (
	"github.com/dfryer1193/gomad/api"
)

type MigrationRepository interface {
	GetFilteredBySignature(signatures []uint64) ([]*api.Migration, error)
	BulkInsert(migrations []*api.MigrationProto) error
	Close()
}

type DatabaseRepository interface {
	CreateDatabase(dbName string, owner string) error
	DatabaseExists(dbName string) (bool, error)
	ListDatabases() ([]string, error)
	Close()
}
