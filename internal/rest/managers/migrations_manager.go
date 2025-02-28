package managers

import (
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
	manager        *MigrationManager
	migrationsOnce sync.Once
)

func GetMigrationsManager() *MigrationManager {
	migrationsOnce.Do(func() {
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

func (mgr *MigrationManager) ProcessMigrations(pending []api.MigrationProto) error {
	incomplete, err := mgr.filterCompleted(pending)
	if err != nil {
		return fmt.Errorf("failed to fetch managers while processing managers: %w", err)
	}

	err = mgr.migrations.BulkInsert(incomplete)
	if err != nil {
		return fmt.Errorf("failed to bulk insert managers: %w", err)
	}
	return nil
}

func (mgr *MigrationManager) GetMigrationsForNamespace(namespace string) ([]*api.Migration, error) {
	migrations, err := mgr.migrations.GetAllForNamespace(namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch migrations for namespace %s : %w", namespace, err)
	}

	return migrations, nil
}

func (mgr *MigrationManager) GetMigrationById(id uint64) (*api.Migration, error) {
	migration, err := mgr.migrations.GetById(id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch migration id %d: %w", id, err)
	}

	return migration, nil
}

func (mgr *MigrationManager) filterCompleted(pending []api.MigrationProto) ([]*api.MigrationProto, error) {
	sigMap := make(map[uint64]*api.MigrationProto)
	signatures := make([]uint64, 0, len(pending))
	for idx := range pending {
		sigMap[pending[idx].Signature] = &pending[idx]
		signatures = append(signatures, pending[idx].Signature)
	}

	existing, err := mgr.migrations.GetFilteredBySignature(signatures)
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
