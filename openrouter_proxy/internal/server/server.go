// Package server wires the router and runs the HTTP server.
package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
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

	chatHandler := handler.NewChatHandler(client, cfg.DefaultModel, cfg.AllowedModels)

	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	// RealIP believes X-Forwarded-For: only a reverse proxy that rewrites it
	// may be trusted, or any caller would choose the IP it is limited under.
	if cfg.TrustProxy {
		r.Use(chimw.RealIP)
	}
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(mw.SecurityHeaders)
	r.Use(mw.NewCORS(cfg.CORSAllowedOrigins))

	// The bridge client polls /health before enabling its photo buttons.
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	})

	r.Group(func(r chi.Router) {
		if cfg.RateLimitPerMin > 0 {
			r.Use(mw.NewRateLimiter(cfg.RateLimitPerMin, cfg.RateLimitBurst).Handler)
		}
		r.Post("/api/chat", chatHandler.ServeHTTP)
		// The client never lists models; the route only exists on request.
		if cfg.EnableModelsEndpoint {
			r.Get("/api/models", handler.NewModelsHandler(client).ServeHTTP)
		}
	})

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
		// ReadTimeout must be short; WriteTimeout must exceed upstream timeout.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      cfg.Timeout + 10*time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    16 << 10,
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

	for _, o := range cfg.CORSAllowedOrigins {
		if o == "*" {
			log.Println("warning: CORS_ORIGINS is *, any web page may call this server; set it to the client's origin in production")
		}
	}
	if cfg.RateLimitPerMin == 0 {
		log.Println("warning: RATE_LIMIT_PER_MIN is 0, /api/* is not rate-limited")
	}
	log.Printf("allowed models: %s", strings.Join(cfg.AllowedModels, ", "))
	log.Printf("listening on :%s (provider: %s, default model: %s)", cfg.Port, config.Provider, cfg.DefaultModel)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
