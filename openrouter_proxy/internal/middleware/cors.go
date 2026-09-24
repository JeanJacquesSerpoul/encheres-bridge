package middleware

import (
	"net/http"

	"github.com/rs/cors"
)

// NewCORS returns a CORS middleware configured with the provided origins.
// If no origins are provided, it falls back to "*".
func NewCORS(allowedOrigins []string) func(http.Handler) http.Handler {
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	})
	return c.Handler
}
