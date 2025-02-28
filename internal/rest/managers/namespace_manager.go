package managers

import (
	"fmt"
	"github.com/dfryer1193/gomad/internal/data/repository"
	"github.com/dfryer1193/gomad/internal/data/repository/postgres"
	"sync"
)

type NamespaceManager struct {
	dbRepo repository.DatabaseRepository
}

var (
	mgr           *NamespaceManager
	namespaceOnce sync.Once
)

func GetNamespaceManager() *NamespaceManager {
	namespaceOnce.Do(func() {
		mgr = &NamespaceManager{
			dbRepo: postgres.GetDatabaseRepository(),
		}
	})

	return mgr
}

func (mgr *NamespaceManager) GetNamespaces() ([]string, error) {
	namespaces, err := mgr.dbRepo.ListDatabases()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch namespaces: %w", err)
	}

	return namespaces, nil
}

func (mgr *NamespaceManager) Close() {
	mgr.dbRepo.Close()
}
