package rest

import (
	"github.com/dfryer1193/gomad/internal/rest/handlers"
	mjolnirUtils "github.com/dfryer1193/mjolnir/utils"
	"github.com/go-chi/chi/v5"
)

func SetupRoutes(router *chi.Mux) {
	hookHandler := handlers.GetHookHandler()
	migrationsHandler := handlers.GetMigrationHandler()

	router.Route("/handlers/v1", func(r chi.Router) {
		r.Post("/push", mjolnirUtils.ErrorHandler(hookHandler.HandlePush))
	})

	router.Route("/namespaces/v1", func(r chi.Router) {
		r.Get("/", mjolnirUtils.ErrorHandler(migrationsHandler.GetNamespaces))
		r.Get("/:namespace/managers", mjolnirUtils.ErrorHandler(migrationsHandler.GetMigrationsForNamespace))
		r.Get("/:namespace/migrations/:migrationId", mjolnirUtils.ErrorHandler(migrationsHandler.GetMigrationById))
		// TODO: Write the handlers required for frontend
		// POST /<namespace>/managers/<migrationId>/execute
	})
}
