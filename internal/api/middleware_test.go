package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ryanoboyle/bb-stream/internal/config"
)

func TestCORSMiddleware(t *testing.T) {
	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test regular request
	req := httptest.NewRequest("GET", "/api/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Check CORS headers
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected Access-Control-Allow-Origin: *")
	}
	if rr.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Expected Access-Control-Allow-Methods header")
	}

	// Test preflight OPTIONS request
	req = httptest.NewRequest("OPTIONS", "/api/test", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("Expected status %d for OPTIONS, got %d", http.StatusNoContent, rr.Code)
	}
}

func TestAuthMiddleware_HealthCheck(t *testing.T) {
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Health check should bypass auth, got status %d", rr.Code)
	}
}

func TestAuthMiddleware_WebSocketUpgrade(t *testing.T) {
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("WebSocket should bypass auth, got status %d", rr.Code)
	}
}

func TestAuthMiddleware_LocalhostNoAuth(t *testing.T) {
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/buckets", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Localhost should be allowed without auth, got status %d", rr.Code)
	}
}

func TestAuthMiddleware_ValidAPIKey(t *testing.T) {
	// Ensure config is initialized
	_ = config.Get()
	// Set up config with API key
	config.SetAPIKey("test-secret-key")

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/buckets", nil)
	req.Header.Set("X-API-Key", "test-secret-key")
	req.RemoteAddr = "192.168.1.100:12345" // Non-localhost
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Valid API key should be accepted, got status %d", rr.Code)
	}

	// Clean up
	config.SetAPIKey("")
}

func TestAuthMiddleware_ValidBearerToken(t *testing.T) {
	_ = config.Get()
	config.SetAPIKey("bearer-secret")

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/buckets", nil)
	req.Header.Set("Authorization", "Bearer bearer-secret")
	req.RemoteAddr = "192.168.1.100:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Valid Bearer token should be accepted, got status %d", rr.Code)
	}

	config.SetAPIKey("")
}

func TestAuthMiddleware_InvalidAPIKey(t *testing.T) {
	_ = config.Get()
	config.SetAPIKey("correct-key")

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/buckets", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	req.RemoteAddr = "192.168.1.100:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Invalid API key should be rejected, got status %d", rr.Code)
	}

	config.SetAPIKey("")
}

func TestAuthMiddleware_NoKeyConfigured(t *testing.T) {
	_ = config.Get()
	config.SetAPIKey("")

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/buckets", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Should allow when no API key configured (backward compatibility)
	if rr.Code != http.StatusOK {
		t.Errorf("No API key configured should allow access, got status %d", rr.Code)
	}
}

func TestContentTypeJSON(t *testing.T) {
	handler := ContentTypeJSON(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type: application/json, got %s", contentType)
	}
}
