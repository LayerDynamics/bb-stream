package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRespondJSON(t *testing.T) {
	rr := httptest.NewRecorder()

	data := map[string]string{"message": "hello"}
	respondJSON(rr, http.StatusOK, data)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var result map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result["message"] != "hello" {
		t.Errorf("Expected message 'hello', got '%s'", result["message"])
	}
}

func TestRespondError(t *testing.T) {
	rr := httptest.NewRecorder()

	respondError(rr, http.StatusBadRequest, "something went wrong")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var result map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result["error"] != "something went wrong" {
		t.Errorf("Expected error 'something went wrong', got '%s'", result["error"])
	}
}

func TestHandleSyncStart_InvalidJSON(t *testing.T) {
	server := &Server{
		hub: NewWebSocketHub(),
	}

	req := httptest.NewRequest("POST", "/api/sync/start", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	server.handleSyncStart(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, rr.Code)
	}

	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["error"] != "Invalid request body" {
		t.Errorf("Expected 'Invalid request body' error, got '%s'", result["error"])
	}
}

func TestHandleSyncStart_MissingLocalPath(t *testing.T) {
	server := &Server{
		hub: NewWebSocketHub(),
	}

	body := `{"bucket": "test-bucket", "direction": "to_remote"}`
	req := httptest.NewRequest("POST", "/api/sync/start", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	server.handleSyncStart(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for missing local_path, got %d", http.StatusBadRequest, rr.Code)
	}

	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["error"] != "local_path is required" {
		t.Errorf("Expected 'local_path is required' error, got '%s'", result["error"])
	}
}

func TestHandleSyncStart_MissingBucket(t *testing.T) {
	server := &Server{
		hub: NewWebSocketHub(),
	}

	body := `{"local_path": "/tmp/test", "direction": "to_remote"}`
	req := httptest.NewRequest("POST", "/api/sync/start", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	server.handleSyncStart(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for missing bucket, got %d", http.StatusBadRequest, rr.Code)
	}

	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["error"] != "bucket is required" {
		t.Errorf("Expected 'bucket is required' error, got '%s'", result["error"])
	}
}

func TestHandleSyncStart_InvalidDirection(t *testing.T) {
	server := &Server{
		hub: NewWebSocketHub(),
	}

	body := `{"local_path": "/tmp/test", "bucket": "test-bucket", "direction": "invalid"}`
	req := httptest.NewRequest("POST", "/api/sync/start", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	server.handleSyncStart(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid direction, got %d", http.StatusBadRequest, rr.Code)
	}

	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["error"] != "direction must be 'to_remote' or 'to_local'" {
		t.Errorf("Expected direction error, got '%s'", result["error"])
	}
}

func TestHandleSyncStatus_NotFound(t *testing.T) {
	server := &Server{
		hub: NewWebSocketHub(),
	}

	// Set up chi context with URL param
	r := chi.NewRouter()
	r.Get("/api/sync/status/{id}", server.handleSyncStatus)

	req := httptest.NewRequest("GET", "/api/sync/status/nonexistent-job", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status %d for nonexistent job, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestHandleWatchStart_InvalidJSON(t *testing.T) {
	server := &Server{
		hub: NewWebSocketHub(),
	}

	req := httptest.NewRequest("POST", "/api/watch/start", bytes.NewBufferString("invalid"))
	rr := httptest.NewRecorder()

	server.handleWatchStart(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleWatchStart_MissingLocalPath(t *testing.T) {
	server := &Server{
		hub: NewWebSocketHub(),
	}

	body := `{"bucket": "test-bucket"}`
	req := httptest.NewRequest("POST", "/api/watch/start", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	server.handleWatchStart(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for missing local_path, got %d", http.StatusBadRequest, rr.Code)
	}

	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["error"] != "local_path is required" {
		t.Errorf("Expected 'local_path is required' error, got '%s'", result["error"])
	}
}

func TestHandleWatchStart_MissingBucket(t *testing.T) {
	server := &Server{
		hub: NewWebSocketHub(),
	}

	body := `{"local_path": "/tmp/test"}`
	req := httptest.NewRequest("POST", "/api/watch/start", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	server.handleWatchStart(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for missing bucket, got %d", http.StatusBadRequest, rr.Code)
	}

	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["error"] != "bucket is required" {
		t.Errorf("Expected 'bucket is required' error, got '%s'", result["error"])
	}
}

func TestHandleWatchStop_InvalidJSON(t *testing.T) {
	server := &Server{
		hub: NewWebSocketHub(),
	}

	req := httptest.NewRequest("POST", "/api/watch/stop", bytes.NewBufferString("invalid"))
	rr := httptest.NewRecorder()

	server.handleWatchStop(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleWatchStop_NotFound(t *testing.T) {
	server := &Server{
		hub: NewWebSocketHub(),
	}

	body := `{"job_id": "nonexistent-job"}`
	req := httptest.NewRequest("POST", "/api/watch/stop", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	server.handleWatchStop(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status %d for nonexistent job, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestHandleListJobs_Empty(t *testing.T) {
	server := &Server{
		hub: NewWebSocketHub(),
	}

	// Clear any existing jobs
	syncJobsMu.Lock()
	syncJobs = make(map[string]*SyncJob)
	syncJobsMu.Unlock()

	watchJobsMu.Lock()
	watchJobs = make(map[string]*WatchJob)
	watchJobsMu.Unlock()

	req := httptest.NewRequest("GET", "/api/jobs", nil)
	rr := httptest.NewRecorder()

	server.handleListJobs(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var result []interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty jobs list, got %d jobs", len(result))
	}
}

func TestHandleAuth_InvalidJSON(t *testing.T) {
	server := &Server{
		hub: NewWebSocketHub(),
	}

	req := httptest.NewRequest("POST", "/api/auth", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	server.handleAuth(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, rr.Code)
	}

	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["error"] != "Invalid request body" {
		t.Errorf("Expected 'Invalid request body' error, got '%s'", result["error"])
	}
}

func TestGetPathFromURL(t *testing.T) {
	tests := []struct {
		path      string
		expected  string
		expectErr bool
	}{
		{"/file.txt", "file.txt", false},
		{"/folder/file.txt", "folder/file.txt", false},
		{"/deep/nested/path/file.txt", "deep/nested/path/file.txt", false},
		{"/../../../etc/passwd", "", true},           // Path traversal
		{"/folder/../../../etc/passwd", "", true},    // Path traversal
	}

	for _, tt := range tests {
		r := chi.NewRouter()
		var extractedPath string
		var extractedErr error

		r.Get("/api/download/{bucket}/*", func(w http.ResponseWriter, r *http.Request) {
			extractedPath, extractedErr = getPathFromURL(r)
		})

		req := httptest.NewRequest("GET", "/api/download/mybucket"+tt.path, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if tt.expectErr {
			if extractedErr == nil {
				t.Errorf("getPathFromURL(%s) expected error, got none", tt.path)
			}
		} else {
			if extractedErr != nil {
				t.Errorf("getPathFromURL(%s) unexpected error: %v", tt.path, extractedErr)
			}
			if extractedPath != tt.expected {
				t.Errorf("getPathFromURL(%s) = %s, expected %s", tt.path, extractedPath, tt.expected)
			}
		}
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		path      string
		expected  string
		expectErr bool
	}{
		{"file.txt", "file.txt", false},
		{"folder/file.txt", "folder/file.txt", false},
		{"../etc/passwd", "", true},
		{"folder/../../../etc/passwd", "", true},
		{"/absolute/path", "", true},
		{"", "", true},
		{".", "", true},
	}

	for _, tt := range tests {
		result, err := validatePath(tt.path)
		if tt.expectErr {
			if err == nil {
				t.Errorf("validatePath(%q) expected error, got none", tt.path)
			}
		} else {
			if err != nil {
				t.Errorf("validatePath(%q) unexpected error: %v", tt.path, err)
			}
			if result != tt.expected {
				t.Errorf("validatePath(%q) = %q, expected %q", tt.path, result, tt.expected)
			}
		}
	}
}

func TestValidateBucketName(t *testing.T) {
	tests := []struct {
		bucket    string
		expectErr bool
	}{
		{"mybucket", false},
		{"my-bucket-123", false},
		{"bucket", false},
		{"", true},                    // Empty
		{"ab", true},                  // Too short
		{"MYBUCKET", true},            // Uppercase
		{"-mybucket", true},           // Leading hyphen
		{"mybucket-", true},           // Trailing hyphen
		{"my_bucket", true},           // Underscore not allowed
		{"my.bucket", true},           // Dot not allowed
	}

	for _, tt := range tests {
		err := validateBucketName(tt.bucket)
		if tt.expectErr && err == nil {
			t.Errorf("validateBucketName(%q) expected error, got none", tt.bucket)
		}
		if !tt.expectErr && err != nil {
			t.Errorf("validateBucketName(%q) unexpected error: %v", tt.bucket, err)
		}
	}
}

func TestSyncJobStruct(t *testing.T) {
	job := &SyncJob{
		ID:        "sync-123",
		Status:    "running",
		LocalPath: "/tmp/test",
		Bucket:    "test-bucket",
		Path:      "backup/",
		Direction: "to_remote",
	}

	// Test JSON serialization
	data, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("Failed to marshal SyncJob: %v", err)
	}

	var decoded SyncJob
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal SyncJob: %v", err)
	}

	if decoded.ID != job.ID {
		t.Errorf("Expected ID %s, got %s", job.ID, decoded.ID)
	}
	if decoded.Status != job.Status {
		t.Errorf("Expected Status %s, got %s", job.Status, decoded.Status)
	}
}

func TestWatchJobStruct(t *testing.T) {
	job := &WatchJob{
		ID:        "watch-123",
		Status:    "running",
		LocalPath: "/tmp/watch",
		Bucket:    "test-bucket",
		Path:      "uploads/",
	}

	// Test JSON serialization
	data, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("Failed to marshal WatchJob: %v", err)
	}

	var decoded WatchJob
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal WatchJob: %v", err)
	}

	if decoded.ID != job.ID {
		t.Errorf("Expected ID %s, got %s", job.ID, decoded.ID)
	}
	if decoded.Status != job.Status {
		t.Errorf("Expected Status %s, got %s", job.Status, decoded.Status)
	}
}

func TestSyncRequest_Validation(t *testing.T) {
	tests := []struct {
		name      string
		request   SyncRequest
		expectErr bool
	}{
		{
			name: "Valid to_remote request",
			request: SyncRequest{
				LocalPath: "/tmp/test",
				Bucket:    "my-bucket",
				Path:      "backup/",
				Direction: "to_remote",
			},
			expectErr: false,
		},
		{
			name: "Valid to_local request",
			request: SyncRequest{
				LocalPath: "/tmp/test",
				Bucket:    "my-bucket",
				Path:      "backup/",
				Direction: "to_local",
			},
			expectErr: false,
		},
		{
			name: "Valid with dry run",
			request: SyncRequest{
				LocalPath: "/tmp/test",
				Bucket:    "my-bucket",
				Direction: "to_remote",
				DryRun:    true,
			},
			expectErr: false,
		},
		{
			name: "Valid with delete",
			request: SyncRequest{
				LocalPath: "/tmp/test",
				Bucket:    "my-bucket",
				Direction: "to_remote",
				Delete:    true,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the struct serializes/deserializes correctly
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var decoded SyncRequest
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if decoded.LocalPath != tt.request.LocalPath {
				t.Errorf("LocalPath mismatch: got %s, want %s", decoded.LocalPath, tt.request.LocalPath)
			}
			if decoded.Bucket != tt.request.Bucket {
				t.Errorf("Bucket mismatch: got %s, want %s", decoded.Bucket, tt.request.Bucket)
			}
			if decoded.Direction != tt.request.Direction {
				t.Errorf("Direction mismatch: got %s, want %s", decoded.Direction, tt.request.Direction)
			}
		})
	}
}

func TestAuthRequest_Struct(t *testing.T) {
	req := AuthRequest{
		KeyID:          "test-key-id",
		ApplicationKey: "test-app-key",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal AuthRequest: %v", err)
	}

	var decoded AuthRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal AuthRequest: %v", err)
	}

	if decoded.KeyID != req.KeyID {
		t.Errorf("KeyID mismatch: got %s, want %s", decoded.KeyID, req.KeyID)
	}
	if decoded.ApplicationKey != req.ApplicationKey {
		t.Errorf("ApplicationKey mismatch: got %s, want %s", decoded.ApplicationKey, req.ApplicationKey)
	}
}

func TestFlushingWriter(t *testing.T) {
	// Create a mock flusher
	rr := httptest.NewRecorder()

	fw := &flushingWriter{
		w: rr,
		f: rr,
	}

	// Write some data
	data := make([]byte, 100)
	n, err := fw.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != 100 {
		t.Errorf("Expected to write 100 bytes, wrote %d", n)
	}
	if fw.written != 100 {
		t.Errorf("Expected written to be 100, got %d", fw.written)
	}

	// Write more data to trigger flush (>64KB)
	largeData := make([]byte, 65536)
	n, err = fw.Write(largeData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != 65536 {
		t.Errorf("Expected to write 65536 bytes, wrote %d", n)
	}
	// After flush, written should be reset to 0
	if fw.written != 0 {
		t.Errorf("Expected written to be reset to 0 after flush, got %d", fw.written)
	}
}
