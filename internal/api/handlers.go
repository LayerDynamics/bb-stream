package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ryanoboyle/bb-stream/internal/b2"
	"github.com/ryanoboyle/bb-stream/internal/config"
	internalSync "github.com/ryanoboyle/bb-stream/internal/sync"
	"github.com/ryanoboyle/bb-stream/internal/watch"
	"github.com/ryanoboyle/bb-stream/pkg/errors"
	"github.com/ryanoboyle/bb-stream/pkg/logging"
)

// safeGo runs a function in a goroutine with panic recovery
func safeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logging.Logger().Error("goroutine panic recovered",
					"panic", r,
					"stack", string(debug.Stack()))
			}
		}()
		fn()
	}()
}

// Job TTL cleanup constants
const (
	jobTTL             = 1 * time.Hour    // How long to keep completed jobs
	jobCleanupInterval = 5 * time.Minute  // How often to clean up old jobs
)

// init starts the job cleanup goroutine
func init() {
	go func() {
		ticker := time.NewTicker(jobCleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			cleanupOldJobs()
		}
	}()
}

// cleanupOldJobs removes completed/stopped jobs older than jobTTL
func cleanupOldJobs() {
	now := time.Now()

	// Cleanup sync jobs
	syncJobsMu.Lock()
	for id, job := range syncJobs {
		if job.Status == "completed" || job.Status == "failed" {
			if !job.CompletedAt.IsZero() && now.Sub(job.CompletedAt) > jobTTL {
				delete(syncJobs, id)
			}
		}
	}
	syncJobsMu.Unlock()

	// Cleanup watch jobs
	watchJobsMu.Lock()
	for id, job := range watchJobs {
		if job.Status == "stopped" {
			if !job.StoppedAt.IsZero() && now.Sub(job.StoppedAt) > jobTTL {
				delete(watchJobs, id)
			}
		}
	}
	watchJobsMu.Unlock()
}

// JSON response helpers

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// handleError logs the error with context and sends a sanitized error response.
// The internal error is logged but not exposed to clients.
func handleError(w http.ResponseWriter, err error, status int, operation string, attrs ...any) {
	// Build log attributes
	logAttrs := []any{
		logging.Operation(operation),
		logging.Status(status),
		logging.Err(err),
	}
	logAttrs = append(logAttrs, attrs...)

	// Log the internal error
	logging.Logger().Error("request failed", logAttrs...)

	// Send sanitized error to client
	safeMessage := errors.Sanitize(err)
	respondError(w, status, safeMessage)
}

// Path validation helpers

// validatePath ensures a path is safe and does not escape the intended scope.
// It prevents path traversal attacks using sequences like "../" or absolute paths.
func validatePath(path string) (string, error) {
	// Clean the path first (resolves . and ..)
	cleaned := filepath.Clean(path)

	// Convert to forward slashes for consistent checking (B2 uses forward slashes)
	cleaned = filepath.ToSlash(cleaned)

	// Reject absolute paths
	if filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("absolute paths not allowed")
	}

	// Reject path traversal attempts
	if strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, "/../") || strings.HasSuffix(cleaned, "/..") {
		return "", fmt.Errorf("path traversal not allowed")
	}

	// Reject empty path
	if cleaned == "" || cleaned == "." {
		return "", fmt.Errorf("empty path not allowed")
	}

	return cleaned, nil
}

// validateBucketName ensures a bucket name contains only safe characters.
func validateBucketName(bucket string) error {
	if bucket == "" {
		return fmt.Errorf("bucket name required")
	}
	// B2 bucket names: 6-50 chars, a-z, 0-9, and hyphens (no leading/trailing hyphen)
	if len(bucket) < 6 || len(bucket) > 50 {
		return fmt.Errorf("bucket name must be 6-50 characters")
	}
	for _, r := range bucket {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return fmt.Errorf("bucket name contains invalid character")
		}
	}
	if bucket[0] == '-' || bucket[len(bucket)-1] == '-' {
		return fmt.Errorf("bucket name cannot start or end with hyphen")
	}
	return nil
}

// Auth handler

type AuthRequest struct {
	KeyID          string `json:"key_id"`
	ApplicationKey string `json:"application_key"`
}

func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Try to create a client with provided credentials
	ctx := r.Context()
	_, err := b2.New(ctx, req.KeyID, req.ApplicationKey)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "authenticated"})
}

// Bucket handlers

func (s *Server) handleListBuckets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	buckets, err := s.client.ListBucketInfo(ctx)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError, "list_buckets")
		return
	}

	respondJSON(w, http.StatusOK, buckets)
}

func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	bucketName := chi.URLParam(r, "name")
	prefix := r.URL.Query().Get("prefix")

	ctx := r.Context()
	objects, err := s.client.ListObjects(ctx, bucketName, prefix)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError, "list_files",
			logging.Bucket(bucketName))
		return
	}

	respondJSON(w, http.StatusOK, objects)
}

// Upload handlers

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		respondError(w, http.StatusBadRequest, "Failed to parse form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "No file provided")
		return
	}
	defer file.Close()

	bucket := r.URL.Query().Get("bucket")
	if err := validateBucketName(bucket); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		path = header.Filename
	}
	// Validate the path
	path, err = validatePath(path)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()
	result, err := s.client.UploadWithResult(ctx, bucket, path, file, header.Size, nil)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError, "upload",
			logging.Bucket(bucket), logging.Object(path))
		return
	}

	// Broadcast upload event
	s.BroadcastEvent("upload_complete", map[string]interface{}{
		"name": result.Name,
		"size": result.Size,
	})

	respondJSON(w, http.StatusOK, result)
}

func (s *Server) handleStreamUpload(w http.ResponseWriter, r *http.Request) {
	bucket := r.URL.Query().Get("bucket")
	if err := validateBucketName(bucket); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	path := r.URL.Query().Get("path")
	path, err := validatePath(path)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()
	err = s.client.StreamUpload(ctx, bucket, path, r.Body, nil)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError, "stream_upload",
			logging.Bucket(bucket), logging.Object(path))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "uploaded",
		"path":   path,
	})
}

// Download handlers

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	if err := validateBucketName(bucket); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	path, err := getPathFromURL(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()

	// Get object info for headers
	info, err := s.client.GetObjectInfo(ctx, bucket, path)
	if err != nil {
		handleError(w, err, http.StatusNotFound, "download",
			logging.Bucket(bucket), logging.Object(path))
		return
	}

	// Set headers
	w.Header().Set("Content-Type", info.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(path)))

	// Stream the file
	err = s.client.Download(ctx, bucket, path, w, nil)
	if err != nil {
		// Can't send error response after headers are sent
		return
	}
}

func (s *Server) handleStreamDownload(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	if err := validateBucketName(bucket); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	path, err := getPathFromURL(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()

	// Get object info
	info, err := s.client.GetObjectInfo(ctx, bucket, path)
	if err != nil {
		handleError(w, err, http.StatusNotFound, "stream_download",
			logging.Bucket(bucket), logging.Object(path))
		return
	}

	// Set headers for streaming
	w.Header().Set("Content-Type", info.ContentType)
	w.Header().Set("Transfer-Encoding", "chunked")

	// Use flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// Create a writer that flushes periodically
	flushWriter := &flushingWriter{w: w, f: flusher}

	err = s.client.StreamDownload(ctx, bucket, path, flushWriter, nil)
	if err != nil {
		return
	}
}

type flushingWriter struct {
	w       io.Writer
	f       http.Flusher
	written int
}

func (fw *flushingWriter) Write(p []byte) (int, error) {
	n, err := fw.w.Write(p)
	fw.written += n
	// Flush every 64KB
	if fw.written >= 65536 {
		fw.f.Flush()
		fw.written = 0
	}
	return n, err
}

// Sync handlers

var (
	syncJobs   = make(map[string]*SyncJob)
	syncJobsMu sync.RWMutex
)

type SyncJob struct {
	ID          string                   `json:"id"`
	Status      string                   `json:"status"`
	LocalPath   string                   `json:"local_path"`
	Bucket      string                   `json:"bucket"`
	Path        string                   `json:"path"`
	Direction   string                   `json:"direction"`
	StartTime   time.Time                `json:"start_time"`
	CompletedAt time.Time                `json:"completed_at,omitempty"`
	Progress    string                   `json:"progress,omitempty"`
	Result      *internalSync.SyncResult `json:"result,omitempty"`
}

type SyncRequest struct {
	LocalPath string `json:"local_path"`
	Bucket    string `json:"bucket"`
	Path      string `json:"path"`
	Direction string `json:"direction"` // "to_remote" or "to_local"
	DryRun    bool   `json:"dry_run"`
	Delete    bool   `json:"delete"`
}

func (s *Server) handleSyncStart(w http.ResponseWriter, r *http.Request) {
	var req SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.LocalPath == "" {
		respondError(w, http.StatusBadRequest, "local_path is required")
		return
	}
	if req.Bucket == "" {
		respondError(w, http.StatusBadRequest, "bucket is required")
		return
	}
	if req.Direction != "to_remote" && req.Direction != "to_local" {
		respondError(w, http.StatusBadRequest, "direction must be 'to_remote' or 'to_local'")
		return
	}

	// Generate job ID
	jobID := fmt.Sprintf("sync-%d", time.Now().UnixNano())

	// Create job
	job := &SyncJob{
		ID:        jobID,
		Status:    "running",
		LocalPath: req.LocalPath,
		Bucket:    req.Bucket,
		Path:      req.Path,
		Direction: req.Direction,
		StartTime: time.Now(),
	}

	syncJobsMu.Lock()
	syncJobs[jobID] = job
	syncJobsMu.Unlock()

	// Run sync in background with panic recovery
	safeGo(func() {
		opts := internalSync.DefaultSyncOptions()
		opts.DryRun = req.DryRun
		opts.Delete = req.Delete

		if req.Direction == "to_remote" {
			opts.Direction = internalSync.ToRemote
		} else {
			opts.Direction = internalSync.ToLocal
		}

		opts.ProgressCallback = func(status internalSync.SyncStatus) {
			syncJobsMu.Lock()
			job.Progress = fmt.Sprintf("%s: %s", status.Phase, status.CurrentFile)
			syncJobsMu.Unlock()

			s.BroadcastEvent("sync_progress", map[string]interface{}{
				"job_id":   jobID,
				"phase":    status.Phase,
				"file":     status.CurrentFile,
			})
		}

		syncer := internalSync.NewSyncer(s.client, opts)
		// Use background context since HTTP request context will be cancelled
		result, err := syncer.Sync(context.Background(), req.LocalPath, req.Bucket, req.Path)

		syncJobsMu.Lock()
		job.CompletedAt = time.Now()
		if err != nil {
			job.Status = "failed"
			job.Progress = err.Error()
			logging.Logger().Error("sync job failed",
				logging.JobID(jobID),
				logging.Bucket(req.Bucket),
				logging.Err(err))
		} else {
			job.Status = "completed"
			job.Result = result
			logging.Logger().Info("sync job completed",
				logging.JobID(jobID),
				logging.Bucket(req.Bucket),
				"uploaded", result.Uploaded,
				"downloaded", result.Downloaded,
				"deleted", result.Deleted)
		}
		syncJobsMu.Unlock()

		s.BroadcastEvent("sync_complete", map[string]interface{}{
			"job_id": jobID,
			"status": job.Status,
		})
	})

	respondJSON(w, http.StatusAccepted, map[string]string{
		"job_id": jobID,
		"status": "started",
	})
}

func (s *Server) handleSyncStatus(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")

	syncJobsMu.RLock()
	job, exists := syncJobs[jobID]
	syncJobsMu.RUnlock()

	if !exists {
		respondError(w, http.StatusNotFound, "Job not found")
		return
	}

	respondJSON(w, http.StatusOK, job)
}

// Watch handlers

var (
	watchJobs   = make(map[string]*WatchJob)
	watchJobsMu sync.RWMutex
)

type WatchJob struct {
	ID        string                `json:"id"`
	Status    string                `json:"status"`
	LocalPath string                `json:"local_path"`
	Bucket    string                `json:"bucket"`
	Path      string                `json:"path"`
	StartTime time.Time             `json:"start_time"`
	StoppedAt time.Time             `json:"stopped_at,omitempty"`
	uploader  *watch.AutoUploader
}

type WatchRequest struct {
	LocalPath string `json:"local_path"`
	Bucket    string `json:"bucket"`
	Path      string `json:"path"`
}

func (s *Server) handleWatchStart(w http.ResponseWriter, r *http.Request) {
	var req WatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.LocalPath == "" {
		respondError(w, http.StatusBadRequest, "local_path is required")
		return
	}
	if req.Bucket == "" {
		respondError(w, http.StatusBadRequest, "bucket is required")
		return
	}

	// Generate job ID
	jobID := fmt.Sprintf("watch-%d", time.Now().UnixNano())

	// Create auto uploader
	uploader, err := watch.NewAutoUploader(s.client, req.LocalPath, req.Bucket, req.Path, nil)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError, "watch_start",
			logging.Bucket(req.Bucket), logging.Path(req.LocalPath))
		return
	}

	uploader.OnUpload = func(path string, err error) {
		eventType := "watch_upload"
		data := map[string]interface{}{
			"job_id": jobID,
			"path":   path,
		}
		if err != nil {
			data["error"] = err.Error()
		}
		s.BroadcastEvent(eventType, data)
	}

	// Create job
	job := &WatchJob{
		ID:        jobID,
		Status:    "running",
		LocalPath: req.LocalPath,
		Bucket:    req.Bucket,
		Path:      req.Path,
		StartTime: time.Now(),
		uploader:  uploader,
	}

	watchJobsMu.Lock()
	watchJobs[jobID] = job
	watchJobsMu.Unlock()

	// Start watching - use background context since HTTP request will end
	go func() {
		_ = uploader.Start(context.Background())
	}()

	respondJSON(w, http.StatusOK, map[string]string{
		"job_id": jobID,
		"status": "started",
	})
}

func (s *Server) handleWatchStop(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JobID string `json:"job_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	watchJobsMu.Lock()
	job, exists := watchJobs[req.JobID]
	if exists {
		job.uploader.Stop()
		job.Status = "stopped"
		job.StoppedAt = time.Now()
		logging.Logger().Info("watch job stopped",
			logging.JobID(req.JobID),
			logging.Bucket(job.Bucket))
	}
	watchJobsMu.Unlock()

	if !exists {
		respondError(w, http.StatusNotFound, "Job not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"job_id": req.JobID,
		"status": "stopped",
	})
}

// Jobs handler

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	jobs := make([]interface{}, 0)

	syncJobsMu.RLock()
	for _, job := range syncJobs {
		jobs = append(jobs, map[string]interface{}{
			"id":     job.ID,
			"type":   "sync",
			"status": job.Status,
		})
	}
	syncJobsMu.RUnlock()

	watchJobsMu.RLock()
	for _, job := range watchJobs {
		jobs = append(jobs, map[string]interface{}{
			"id":     job.ID,
			"type":   "watch",
			"status": job.Status,
		})
	}
	watchJobsMu.RUnlock()

	respondJSON(w, http.StatusOK, jobs)
}

// Delete handler

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	bucket := chi.URLParam(r, "bucket")
	if err := validateBucketName(bucket); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	path, err := getPathFromURL(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()
	err = s.client.DeleteObject(ctx, bucket, path)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError, "delete",
			logging.Bucket(bucket), logging.Object(path))
		return
	}

	// Broadcast delete event
	s.BroadcastEvent("file_deleted", map[string]interface{}{
		"bucket": bucket,
		"path":   path,
	})

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "deleted",
		"path":   path,
	})
}

// URL parameter helper
func getPathFromURL(r *http.Request) (string, error) {
	path := chi.URLParam(r, "*")
	// Remove leading slash if present
	path = strings.TrimPrefix(path, "/")
	return validatePath(path)
}

// Config handlers

type ConfigResponse struct {
	KeyID         string `json:"key_id"`
	HasAppKey     bool   `json:"has_app_key"`
	DefaultBucket string `json:"default_bucket"`
	Configured    bool   `json:"configured"`
}

type ConfigRequest struct {
	KeyID          string `json:"key_id"`
	ApplicationKey string `json:"application_key"`
	DefaultBucket  string `json:"default_bucket"`
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := config.Get()

	resp := ConfigResponse{
		KeyID:         cfg.KeyID,
		HasAppKey:     cfg.ApplicationKey != "",
		DefaultBucket: cfg.DefaultBucket,
		Configured:    cfg.IsConfigured(),
	}

	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSetConfig(w http.ResponseWriter, r *http.Request) {
	var req ConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate credentials before saving
	if req.KeyID != "" && req.ApplicationKey != "" {
		ctx := r.Context()
		_, err := b2.New(ctx, req.KeyID, req.ApplicationKey)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid credentials: "+err.Error())
			return
		}
	}

	// Update config
	cfg := config.Get()
	if req.KeyID != "" {
		cfg.KeyID = req.KeyID
	}
	if req.ApplicationKey != "" {
		cfg.ApplicationKey = req.ApplicationKey
	}
	if req.DefaultBucket != "" {
		cfg.DefaultBucket = req.DefaultBucket
	}

	// Save config
	if err := config.Save(); err != nil {
		handleError(w, err, http.StatusInternalServerError, "save_config")
		return
	}

	// Re-initialize B2 client with new credentials
	if req.KeyID != "" && req.ApplicationKey != "" {
		newClient, err := b2.New(r.Context(), req.KeyID, req.ApplicationKey)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "create_client")
			return
		}
		s.client = newClient
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "saved",
	})
}
