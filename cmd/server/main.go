package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/dfryer1193/mjolnir/router"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	r := router.New()

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", 80),
		Handler: r,
	}

	go func() {
		log.Info().Msg("Starting server on port :" + fmt.Sprint(80))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Info().Msg("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to shutdown server")
	}

	log.Info().Msg("Server stopped")
}
