package rest

import (
	"github.com/dfryer1193/gomad/internal/rest/hooks"
	"github.com/go-chi/chi/v5"
)

func SetupRoutes(router *chi.Mux) {
	hookHandler := hooks.GetHookManager()

	router.Route("/hooks/v1", func(r chi.Router) {
		r.Post("/push", hookHandler.HandlePush)
	})
	router.Route("/migrations/v1", func(r chi.Router) {
	})
}
