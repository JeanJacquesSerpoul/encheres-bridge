// Package server wires the router and runs the HTTP server.
package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"server_ai/internal/config"
	"server_ai/internal/handler"
	mw "server_ai/internal/middleware"
	"server_ai/internal/openrouter"
)

// Run wires up the router, starts the HTTP server, and handles graceful shutdown.
func Run(cfg *config.Config) error {
	client := openrouter.NewClient(cfg.OpenRouterAPIKey, cfg.MaxTokens, cfg.Timeout)

	chatHandler := handler.NewChatHandler(client, cfg.DefaultModel)
	modelsHandler := handler.NewModelsHandler(client)

	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(mw.NewCORS(cfg.CORSAllowedOrigins))

	// The bridge client polls /health before enabling its photo buttons.
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	})

	r.Post("/api/chat", chatHandler.ServeHTTP)
	r.Get("/api/models", modelsHandler.ServeHTTP)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
		// ReadTimeout must be short; WriteTimeout must exceed upstream timeout.
		ReadTimeout:  15 * time.Second,
		WriteTimeout: cfg.Timeout + 10*time.Second,
		IdleTimeout:  120 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	log.Printf("listening on :%s (provider: %s, default model: %s)", cfg.Port, config.Provider, cfg.DefaultModel)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
