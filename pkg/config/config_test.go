package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Server.Port != "1200" {
		t.Errorf("expected default port '1200', got %s", cfg.Server.Port)
	}

	if cfg.Server.Env != "production" {
		t.Errorf("expected default env 'production', got %s", cfg.Server.Env)
	}

	if cfg.Cache.Type != "memory" {
		t.Errorf("expected default cache type 'memory', got %s", cfg.Cache.Type)
	}

	if cfg.Server.ReadTimeout != 30*time.Second {
		t.Errorf("expected default read timeout 30s, got %v", cfg.Server.ReadTimeout)
	}

	if cfg.Cache.MemorySize != 10000 {
		t.Errorf("expected default memory cache 10000, got %d", cfg.Cache.MemorySize)
	}

	if !cfg.IsProduction() {
		t.Error("expected default config to be production")
	}

	// Test getters for backward compatibility
	if cfg.GetPort() != "1200" {
		t.Errorf("GetPort() expected '1200', got %s", cfg.GetPort())
	}

	if cfg.GetEnv() != "production" {
		t.Errorf("GetEnv() expected 'production', got %s", cfg.GetEnv())
	}
}

func TestLoad_FromYAML(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := `
server:
  port: "8080"
  env: "development"
  read_timeout: 60s
  write_timeout: 60s
  idle_timeout: 300s

cache:
  type: "memory"
  ttl: 30m
  memory_size: 5000

client:
  user_agent: "CustomBot/1.0"
  timeout: 45s
  max_redirects: 5
  proxy: "http://proxy.example.com:8080"
  no_proxy: true

routes:
  disable_nsfw: true

middleware:
  enable_cache: false
  access_key: "secret123"
  allow_origin: "https://example.com"
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify server settings
	if cfg.Server.Port != "8080" {
		t.Errorf("expected port '8080', got %s", cfg.Server.Port)
	}

	if cfg.Server.Env != "development" {
		t.Errorf("expected env 'development', got %s", cfg.Server.Env)
	}

	if cfg.Server.ReadTimeout != 60*time.Second {
		t.Errorf("expected read timeout 60s, got %v", cfg.Server.ReadTimeout)
	}

	// Verify cache settings
	if cfg.Cache.Type != "memory" {
		t.Errorf("expected cache type 'memory', got %s", cfg.Cache.Type)
	}



	if cfg.Cache.TTL != 30*time.Minute {
		t.Errorf("expected cache TTL 30m, got %v", cfg.Cache.TTL)
	}

	if cfg.Cache.MemorySize != 5000 {
		t.Errorf("expected memory cache size 5000, got %d", cfg.Cache.MemorySize)
	}

	// Verify client settings
	if cfg.Client.UserAgent != "CustomBot/1.0" {
		t.Errorf("expected user agent 'CustomBot/1.0', got %s", cfg.Client.UserAgent)
	}

	if cfg.Client.Timeout != 45*time.Second {
		t.Errorf("expected timeout 45s, got %v", cfg.Client.Timeout)
	}

	if cfg.Client.MaxRedirects != 5 {
		t.Errorf("expected max redirects 5, got %d", cfg.Client.MaxRedirects)
	}

	if cfg.Client.Proxy != "http://proxy.example.com:8080" {
		t.Errorf("expected proxy 'http://proxy.example.com:8080', got %s", cfg.Client.Proxy)
	}

	if !cfg.Client.NoProxy {
		t.Error("expected NoProxy to be true")
	}

	// Verify route settings
	if !cfg.Routes.DisableNSFW {
		t.Error("expected DisableNSFW to be true")
	}

	// Verify middleware settings
	if cfg.Middleware.EnableCache {
		t.Error("expected EnableCache to be false")
	}

	if cfg.Middleware.AccessKey != "secret123" {
		t.Errorf("expected access key 'secret123', got %s", cfg.Middleware.AccessKey)
	}

	if cfg.Middleware.AllowOrigin != "https://example.com" {
		t.Errorf("expected allow origin 'https://example.com', got %s", cfg.Middleware.AllowOrigin)
	}

	// Test backward compatibility getters
	if cfg.GetPort() != "8080" {
		t.Errorf("GetPort() expected '8080', got %s", cfg.GetPort())
	}

	if cfg.IsProduction() {
		t.Error("expected development mode, but IsProduction() returned true")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	// When a specific file path is provided but doesn't exist, it should error
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error when specified config file not found, got nil")
	}

	// When no path is provided and no default files exist, should return default config
	// Create a temp directory with no config files
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("expected no error when no config files found, got %v", err)
	}

	// Should return default config
	defaultCfg := DefaultConfig()
	if cfg.Server.Port != defaultCfg.Server.Port {
		t.Errorf("expected default port when file not found, got %s", cfg.Server.Port)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	invalidContent := `
server:
  port: "8080"
  env: [invalid yaml structure
`

	if err := os.WriteFile(configFile, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := Load(configFile)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestLoad_EnvVariableOverride(t *testing.T) {
	// Test that RSSHUB_CONFIG environment variable works
	origConfig := os.Getenv("RSSHUB_CONFIG")
	defer os.Setenv("RSSHUB_CONFIG", origConfig)

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "custom-config.yaml")
	configContent := `
server:
  port: "9999"
  env: "test"
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	os.Setenv("RSSHUB_CONFIG", configFile)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Server.Port != "9999" {
		t.Errorf("expected port '9999' from env var, got %s", cfg.Server.Port)
	}
}

func TestConfig_Get(t *testing.T) {
	cfg := DefaultConfig()

	// Test default value
	val := cfg.Get("NON_EXISTENT_KEY", "default")
	if val != "default" {
		t.Errorf("expected default value 'default', got %s", val)
	}

	// Test with environment variable
	origVal := os.Getenv("TEST_KEY")
	defer os.Setenv("TEST_KEY", origVal)

	os.Setenv("TEST_KEY", "test_value")
	val = cfg.Get("TEST_KEY", "default")
	if val != "test_value" {
		t.Errorf("expected 'test_value', got %s", val)
	}
}

func TestConfig_IsProduction(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected bool
	}{
		{
			name:     "production",
			env:      "production",
			expected: true,
		},
		{
			name:     "development",
			env:      "development",
			expected: false,
		},
		{
			name:     "test",
			env:      "test",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			cfg.Server.Env = tt.env
			if cfg.IsProduction() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, cfg.IsProduction())
			}
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Test existing file
	if err := os.WriteFile(configFile, []byte("server:\n  port: \"8080\""), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	path, err := GetConfigPath(configFile)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if path != configFile {
		t.Errorf("expected path %s, got %s", configFile, path)
	}
}

func TestGetConfigPath_NotFound(t *testing.T) {
	_, err := GetConfigPath("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestLoadOrPanic(t *testing.T) {
	// Test successful load
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := `
server:
  port: "8080"
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := LoadOrPanic(configFile)
	if cfg.Server.Port != "8080" {
		t.Errorf("expected port '8080', got %s", cfg.Server.Port)
	}

	// Test panic on invalid file
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid config, but didn't panic")
		}
	}()

	LoadOrPanic("/nonexistent/config.yaml")
}
