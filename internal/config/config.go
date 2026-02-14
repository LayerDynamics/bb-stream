package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	KeyID          string `mapstructure:"key_id"`
	ApplicationKey string `mapstructure:"application_key"`
	DefaultBucket  string `mapstructure:"default_bucket"`
	APIPort        int    `mapstructure:"api_port"`
	APIKey         string `mapstructure:"api_key"`
}

var (
	cfg        *Config
	configPath string
)

// Init initializes the configuration system
func Init() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".config", "bb-stream")
	configPath = filepath.Join(configDir, "config.yaml")

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Set defaults
	viper.SetDefault("api_port", 8080)

	// Environment variable bindings
	viper.SetEnvPrefix("BB")
	viper.BindEnv("key_id", "BB_KEY_ID")
	viper.BindEnv("application_key", "BB_APP_KEY")
	viper.BindEnv("default_bucket", "BB_DEFAULT_BUCKET")
	viper.BindEnv("api_key", "BB_API_KEY")

	// Try to read config file (ignore error if doesn't exist)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Only return error if it's not a "file not found" error
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to read config: %w", err)
			}
		}
	}

	cfg = &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// Get returns the current configuration
func Get() *Config {
	if cfg == nil {
		cfg = &Config{}
	}
	return cfg
}

// Save writes the current configuration to disk
func Save() error {
	viper.Set("key_id", cfg.KeyID)
	viper.Set("application_key", cfg.ApplicationKey)
	viper.Set("default_bucket", cfg.DefaultBucket)
	viper.Set("api_port", cfg.APIPort)
	viper.Set("api_key", cfg.APIKey)

	return viper.WriteConfigAs(configPath)
}

// SetAPIKey updates the API key for authentication
func SetAPIKey(key string) {
	cfg.APIKey = key
}

// SetCredentials updates the B2 credentials
func SetCredentials(keyID, appKey string) {
	cfg.KeyID = keyID
	cfg.ApplicationKey = appKey
}

// SetDefaultBucket updates the default bucket
func SetDefaultBucket(bucket string) {
	cfg.DefaultBucket = bucket
}

// SetAPIPort updates the API server port
func SetAPIPort(port int) {
	cfg.APIPort = port
}

// GetConfigPath returns the path to the config file
func GetConfigPath() string {
	return configPath
}

// IsConfigured returns true if credentials are set (package level)
func IsConfigured() bool {
	return cfg.KeyID != "" && cfg.ApplicationKey != ""
}

// IsConfigured returns true if credentials are set (struct level)
func (c *Config) IsConfigured() bool {
	return c.KeyID != "" && c.ApplicationKey != ""
}
