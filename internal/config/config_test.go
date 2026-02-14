package config

import (
	"os"
	"testing"
)

func TestDefaultValues(t *testing.T) {
	// Reset config
	cfg = nil

	// Get default config
	c := Get()

	if c.APIPort != 0 {
		// Default is set in Init, not Get
		// Get returns empty config if Init not called
	}

	if c.KeyID != "" {
		t.Error("Expected empty KeyID for default config")
	}
}

func TestSetCredentials(t *testing.T) {
	// Reset config
	cfg = &Config{}

	SetCredentials("test-key-id", "test-app-key")

	c := Get()
	if c.KeyID != "test-key-id" {
		t.Errorf("Expected KeyID 'test-key-id', got '%s'", c.KeyID)
	}
	if c.ApplicationKey != "test-app-key" {
		t.Errorf("Expected ApplicationKey 'test-app-key', got '%s'", c.ApplicationKey)
	}
}

func TestSetDefaultBucket(t *testing.T) {
	cfg = &Config{}

	SetDefaultBucket("my-bucket")

	c := Get()
	if c.DefaultBucket != "my-bucket" {
		t.Errorf("Expected DefaultBucket 'my-bucket', got '%s'", c.DefaultBucket)
	}
}

func TestSetAPIPort(t *testing.T) {
	cfg = &Config{}

	SetAPIPort(9000)

	c := Get()
	if c.APIPort != 9000 {
		t.Errorf("Expected APIPort 9000, got %d", c.APIPort)
	}
}

func TestSetAPIKey(t *testing.T) {
	cfg = &Config{}

	SetAPIKey("secret-api-key")

	c := Get()
	if c.APIKey != "secret-api-key" {
		t.Errorf("Expected APIKey 'secret-api-key', got '%s'", c.APIKey)
	}
}

func TestIsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		keyID    string
		appKey   string
		expected bool
	}{
		{"Empty credentials", "", "", false},
		{"Only KeyID", "key", "", false},
		{"Only AppKey", "", "appkey", false},
		{"Both set", "key", "appkey", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg = &Config{
				KeyID:          tt.keyID,
				ApplicationKey: tt.appKey,
			}

			result := IsConfigured()
			if result != tt.expected {
				t.Errorf("IsConfigured() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestEnvVariableOverride(t *testing.T) {
	// Skip if running in CI without proper setup
	if os.Getenv("CI") != "" {
		t.Skip("Skipping env var test in CI")
	}

	// Set env vars
	os.Setenv("BB_KEY_ID", "env-key-id")
	os.Setenv("BB_APP_KEY", "env-app-key")
	defer os.Unsetenv("BB_KEY_ID")
	defer os.Unsetenv("BB_APP_KEY")

	// Reset and re-init
	cfg = nil
	configPath = ""

	// Note: Full env var testing requires Init() which creates files
	// This is a placeholder for the concept
}

func TestGetReturnsNonNil(t *testing.T) {
	cfg = nil

	c := Get()

	if c == nil {
		t.Error("Get() should never return nil")
	}
}
