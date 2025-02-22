package migrations

import (
	"github.com/dfryer1193/gomad/internal/data/repository"
	"github.com/dfryer1193/gomad/internal/data/repository/postgres"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
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
