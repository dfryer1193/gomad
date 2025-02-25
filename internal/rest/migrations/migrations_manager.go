package migrations

import (
	"context"
	"fmt"
	"github.com/dfryer1193/gomad/api"
	"github.com/dfryer1193/gomad/internal/data/repository"
	"github.com/dfryer1193/gomad/internal/data/repository/postgres"
	"sync"
)

type MigrationManager struct {
	databases  repository.DatabaseRepository
	migrations repository.MigrationRepository
}

var (
	manager *MigrationManager
	once    sync.Once
)

func GetMigrationsManager() *MigrationManager {
	once.Do(func() {
		manager = &MigrationManager{
			databases:  postgres.GetDatabaseRepository(),
			migrations: postgres.GetMigrationRepository(),
		}
	})

	return manager
}

func (mgr *MigrationManager) Close() {
	mgr.databases.Close()
	mgr.migrations.Close()
}

func (mgr *MigrationManager) ProcessMigrations(ctx context.Context, pending []api.MigrationProto) error {
	incomplete, err := mgr.filterCompleted(ctx, pending)
	if err != nil {
		return fmt.Errorf("failed to fetch migrations while processing migrations: %w", err)
	}

	err = mgr.migrations.BulkInsert(ctx, incomplete)
	if err != nil {
		return fmt.Errorf("failed to bulk insert migrations: %w", err)
	}
	return nil
}

func (mgr *MigrationManager) filterCompleted(ctx context.Context, pending []api.MigrationProto) ([]*api.MigrationProto, error) {
	sigMap := make(map[uint64]*api.MigrationProto)
	signatures := make([]uint64, 0, len(pending))
	for idx := range pending {
		sigMap[pending[idx].Signature] = &pending[idx]
		signatures = append(signatures, pending[idx].Signature)
	}

	existing, err := mgr.migrations.GetFilteredBySignature(ctx, signatures)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch migrations: %w", err)
	}

	for _, existingProto := range existing {
		if _, present := sigMap[existingProto.ID]; present {
			delete(sigMap, existingProto.ID)
		}
	}

	out := make([]*api.MigrationProto, 0, len(sigMap))
	for _, proto := range sigMap {
		out = append(out, proto)
	}

	return out, nil
}
