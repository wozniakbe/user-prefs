package main

import (
	"log/slog"
	"net/http"
)

// NewRouter registers all routes and wraps them with the middleware chain.
func NewRouter(h *PreferencesHandler, cfg Config, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	auth := JWTAuth(cfg.JWTSecret, cfg.JWTIssuer, cfg.DevBypassAuth)

	// Health check (no auth required)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Preferences CRUD
	mux.HandleFunc("GET /api/v1/users/{userId}/preferences", auth(h.GetAll))
	mux.HandleFunc("GET /api/v1/users/{userId}/preferences/{key}", auth(h.GetOne))
	mux.HandleFunc("PUT /api/v1/users/{userId}/preferences", auth(h.ReplaceAll))
	mux.HandleFunc("POST /api/v1/users/{userId}/preferences", auth(h.ReplaceAll))
	mux.HandleFunc("PATCH /api/v1/users/{userId}/preferences", auth(h.PatchPrefs))
	mux.HandleFunc("DELETE /api/v1/users/{userId}/preferences", auth(h.DeleteAll))
	mux.HandleFunc("DELETE /api/v1/users/{userId}/preferences/{key}", auth(h.DeleteOne))

	// Middleware chain: Recovery → CORS → RequestLogging → mux
	var handler http.Handler = mux
	handler = RequestLogging(logger)(handler)
	handler = CORS(cfg.CORSAllowOrigin)(handler)
	handler = Recovery(logger)(handler)

	return handler
}
