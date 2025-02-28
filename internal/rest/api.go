package rest

import (
	"github.com/dfryer1193/gomad/internal/rest/hooks"
	mjolnirUtils "github.com/dfryer1193/mjolnir/utils"
	"github.com/go-chi/chi/v5"
)

func SetupRoutes(router *chi.Mux) {
	hookHandler := hooks.GetHookManager()

	router.Route("/hooks/v1", func(r chi.Router) {
		r.Post("/push", mjolnirUtils.ErrorHandler(hookHandler.HandlePush))
	})
	router.Route("/migrations/v1", func(r chi.Router) {
	})
}
