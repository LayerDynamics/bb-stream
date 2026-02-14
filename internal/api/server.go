package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ryanoboyle/bb-stream/internal/b2"
	"github.com/ryanoboyle/bb-stream/pkg/logging"
)

// Version constants
const (
	Version    = "0.1.0"
	APIVersion = 1
)

// Server is the HTTP API server
type Server struct {
	client     *b2.Client
	router     chi.Router
	httpServer *http.Server
	port       int
	hub        *WebSocketHub
	shutdown   chan struct{}
	wg         sync.WaitGroup
	startTime  time.Time
}

// NewServer creates a new API server
func NewServer(client *b2.Client, port int) *Server {
	s := &Server{
		client:    client,
		port:      port,
		hub:       NewWebSocketHub(),
		shutdown:  make(chan struct{}),
		startTime: time.Now(),
	}

	s.setupRouter()
	return s
}

// setupRouter configures the Chi router with all routes
func (s *Server) setupRouter() {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(SecurityHeadersMiddleware)
	r.Use(CORSMiddleware)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Version and status
		r.Get("/version", s.handleVersion)
		r.Get("/status", s.handleStatus)

		// Auth
		r.Post("/auth", s.handleAuth)

		// Buckets
		r.Get("/buckets", s.handleListBuckets)
		r.Get("/buckets/{name}/files", s.handleListFiles)

		// Upload
		r.Post("/upload", s.handleUpload)
		r.Post("/upload/stream", s.handleStreamUpload)

		// Download
		r.Get("/download/{bucket}/*", s.handleDownload)
		r.Get("/stream/{bucket}/*", s.handleStreamDownload)

		// Delete
		r.Delete("/delete/{bucket}/*", s.handleDelete)

		// Sync
		r.Post("/sync/start", s.handleSyncStart)
		r.Get("/sync/status/{id}", s.handleSyncStatus)

		// Watch
		r.Post("/watch/start", s.handleWatchStart)
		r.Post("/watch/stop", s.handleWatchStop)

		// Jobs
		r.Get("/jobs", s.handleListJobs)

		// WebSocket
		r.Get("/ws", s.handleWebSocket)

		// Config
		r.Get("/config", s.handleGetConfig)
		r.Post("/config", s.handleSetConfig)
	})

	s.router = r
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.router,
	}

	// Start WebSocket hub
	go s.hub.Run()

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	logging.Logger().Info("starting graceful shutdown")

	// Signal shutdown
	close(s.shutdown)

	// Stop WebSocket hub
	s.hub.Stop()

	// Stop all watch jobs
	stopAllWatchJobs()

	// Wait for background work with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logging.Logger().Info("background work completed")
	case <-ctx.Done():
		logging.Logger().Warn("shutdown timeout, some background work may be interrupted")
	}

	// Shutdown HTTP server
	return s.httpServer.Shutdown(ctx)
}

// stopAllWatchJobs stops all running watch jobs
func stopAllWatchJobs() {
	watchJobsMu.Lock()
	defer watchJobsMu.Unlock()

	for id, job := range watchJobs {
		if job.Status == "running" {
			job.uploader.Stop()
			job.Status = "stopped"
			job.StoppedAt = time.Now()
			logging.Logger().Info("stopped watch job during shutdown", logging.JobID(id))
		}
	}
}

// GetRouter returns the router (for testing)
func (s *Server) GetRouter() chi.Router {
	return s.router
}

// GetHub returns the WebSocket hub
func (s *Server) GetHub() *WebSocketHub {
	return s.hub
}

// BroadcastEvent sends an event to all connected WebSocket clients
func (s *Server) BroadcastEvent(eventType string, data interface{}) {
	s.hub.Broadcast(Event{
		Type: eventType,
		Data: data,
	})
}

// handleVersion returns version information
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"version":     Version,
		"api_version": APIVersion,
	})
}

// handleStatus returns server status information
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	// Count active jobs
	syncJobsMu.RLock()
	activeSyncJobs := 0
	for _, job := range syncJobs {
		if job.Status == "running" {
			activeSyncJobs++
		}
	}
	syncJobsMu.RUnlock()

	watchJobsMu.RLock()
	activeWatchJobs := 0
	for _, job := range watchJobs {
		if job.Status == "running" {
			activeWatchJobs++
		}
	}
	watchJobsMu.RUnlock()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"version":           Version,
		"api_version":       APIVersion,
		"uptime_seconds":    int64(time.Since(s.startTime).Seconds()),
		"active_sync_jobs":  activeSyncJobs,
		"active_watch_jobs": activeWatchJobs,
		"websocket_clients": s.hub.ClientCount(),
	})
}
