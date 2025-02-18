package rest

import (
	"github.com/dfryer1193/gomad/internal/rest/migrations"
	"github.com/gin-gonic/gin"
)

func NewApi(router *gin.Engine) {
	migrationsManager := migrations.NewMigrationsManager()
	hookManager := NewHookManager()

	hookRouter := router.Group("/hooks/v1")
	{
		hookRouter.POST("/push", hookManager.HandlePush)
	}
	migrationsRouter := router.Group("/migrations/v1")
	{
		migrationsRouter.GET("/databases", migrationsManager.GetDatabases)
		migrationsRouter.GET("/databases/:database", migrationsManager.GetMigrationsForDatabase)
	}
}
