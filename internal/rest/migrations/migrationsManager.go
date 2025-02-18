package migrations

import (
	"github.com/dfryer1193/gomad/internal/data/dao"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"net/http"
)

type Manager struct {
	dbDao         *dao.DatabaseDAO
	migrationsDao *dao.MigrationDAO
}

func NewMigrationsManager() *Manager {
	return &Manager{
		dbDao:         dao.NewDatabaseDAO(),
		migrationsDao: dao.NewMigrationDAO(),
	}
}

func (mgr *Manager) GetDatabases(ctx *gin.Context) {
	dbs, err := mgr.dbDao.ListDatabases(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to list databases")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list databases"})
	}

	ctx.JSON(http.StatusOK, gin.H{"databases": dbs})
}

func (mgr *Manager) GetMigrationsForDatabase(ctx *gin.Context) {
	dbName := ctx.Param("database")
	migrations, err := mgr.migrationsDao.GetByNamespace(ctx, dbName)
	if err != nil {
		log.Err(err).Msg("failed to get migrations")
		ctx.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "failed to get migrations"},
		)
	}

	ctx.JSON(http.StatusOK, gin.H{"migrations": migrations})
}
