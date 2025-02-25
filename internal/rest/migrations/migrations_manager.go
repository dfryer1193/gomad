package migrations

import (
	"context"
	"fmt"
	"github.com/dfryer1193/gomad/api"
	"github.com/dfryer1193/gomad/internal/data/repository"
	"github.com/dfryer1193/gomad/internal/data/repository/postgres"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"maps"
	"net/http"
	"sync"
)

type MigrationManager struct {
	databases  repository.DatabaseRepository
	migrations repository.MigrationRepository
}

var (
	migrationsManager *MigrationManager
	once              sync.Once
)

func GetMigrationsManager() *MigrationManager {
	once.Do(func() {
		migrationsManager = &MigrationManager{
			databases:  postgres.NewDatabaseRepository(),
			migrations: postgres.NewMigrationRepository(),
		}
	})

	return migrationsManager
}

func (mgr *MigrationManager) Close() {
	mgr.databases.Close()
	mgr.migrations.Close()
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

func (mgr *MigrationManager) ProcessMigrations(ctx context.Context, pending []api.MigrationProto) error {
	incomplete, err := mgr.filterCompleted(ctx, pending)
	if err != nil {
		return fmt.Errorf("failed to fetch migrations while processing migrations: %w", err)
	}

	mgr.migrations.BulkInsert(incomplete)
	return nil
}

func (mgr *MigrationManager) GetDatabases(ctx *gin.Context) {
	dbs, err := mgr.databases.ListDatabases(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to list databases")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list databases"})
	}

	ctx.JSON(http.StatusOK, gin.H{"databases": dbs})
}

func (mgr *MigrationManager) GetMigrationsForDatabase(ctx *gin.Context) {
	dbName := ctx.Param("database")
	migrations, err := mgr.migrations.GetByNamespace(ctx, dbName)
	if err != nil {
		log.Err(err).Msg("failed to get migrations")
		ctx.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "failed to get migrations"},
		)
	}

	ctx.JSON(http.StatusOK, gin.H{"migrations": migrations})
}
