package api

import (
	"context"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/modelcontextprotocol/registry/internal/api/router"
	"github.com/modelcontextprotocol/registry/internal/config"
	"github.com/modelcontextprotocol/registry/internal/service"
	"github.com/modelcontextprotocol/registry/internal/telemetry"
)

// CORSMiddleware adds CORS headers to allow cross-origin requests
func CORSMiddleware(cfg *config.Config, next http.Handler) http.Handler {
	allowedOrigins := strings.Split(cfg.AllowedOrigins, ",")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		origin := r.Header.Get("Origin")
		if slices.Contains(allowedOrigins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Requested-With")
			w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
		}

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// TrailingSlashMiddleware redirects requests with trailing slashes to their canonical form
func TrailingSlashMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only redirect if the path is not "/" and ends with a "/"
		if r.URL.Path != "/" && strings.HasSuffix(r.URL.Path, "/") {
			// Create a copy of the URL and remove the trailing slash
			newURL := *r.URL
			newURL.Path = strings.TrimSuffix(r.URL.Path, "/")

			// Use 308 Permanent Redirect to preserve the request method
			http.Redirect(w, r, newURL.String(), http.StatusPermanentRedirect)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Server represents the HTTP server
type Server struct {
	config   *config.Config
	registry service.RegistryService
	humaAPI  huma.API
	server   *http.Server
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config, registryService service.RegistryService, metrics *telemetry.Metrics) *Server {
	// Create HTTP mux and Huma API
	mux := http.NewServeMux()

	api := router.NewHumaAPI(cfg, registryService, mux, metrics)

	// Wrap the mux with middleware
	handler := TrailingSlashMiddleware(CORSMiddleware(cfg, mux))

	server := &Server{
		config:   cfg,
		registry: registryService,
		humaAPI:  api,
		server: &http.Server{
			Addr:              cfg.ServerAddress,
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}

	return server
}

// Start begins listening for incoming HTTP requests
func (s *Server) Start() error {
	log.Printf("HTTP server starting on %s", s.config.ServerAddress)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
