package rest

import (
	"github.com/go-chi/chi/v5"
)

func NewApi(router *chi.Mux) {
	hookManager := NewHookManager()

	router.Route("/hooks/v1", func(r chi.Router) {
		r.Post("/push", hookManager.HandlePush)
	})
	router.Route("/migrations/v1", func(r chi.Router) {
	})
}
