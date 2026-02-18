package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Port != "1200" {
		t.Errorf("expected default port '1200', got %s", cfg.Port)
	}

	if cfg.Env != "production" {
		t.Errorf("expected default env 'production', got %s", cfg.Env)
	}

	if cfg.CacheType != "memory" {
		t.Errorf("expected default cache type 'memory', got %s", cfg.CacheType)
	}

	if cfg.ReadTimeout != 30*time.Second {
		t.Errorf("expected default read timeout 30s, got %v", cfg.ReadTimeout)
	}

	if cfg.MemoryCache != 10000 {
		t.Errorf("expected default memory cache 10000, got %d", cfg.MemoryCache)
	}

	if !cfg.IsProduction() {
		t.Error("expected default config to be production")
	}
}

func TestLoad_FromEnv(t *testing.T) {
	// Save original env values
	origPort := os.Getenv("PORT")
	origEnv := os.Getenv("NODE_ENV")
	origCache := os.Getenv("CACHE_TYPE")
	origTimeout := os.Getenv("TIMEOUT")

	defer func() {
		os.Setenv("PORT", origPort)
		os.Setenv("NODE_ENV", origEnv)
		os.Setenv("CACHE_TYPE", origCache)
		os.Setenv("TIMEOUT", origTimeout)
	}()

	// Set test values
	os.Setenv("PORT", "8080")
	os.Setenv("NODE_ENV", "development")
	os.Setenv("CACHE_TYPE", "redis")
	os.Setenv("TIMEOUT", "60")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("expected port '8080', got %s", cfg.Port)
	}

	if cfg.Env != "development" {
		t.Errorf("expected env 'development', got %s", cfg.Env)
	}

	if cfg.CacheType != "redis" {
		t.Errorf("expected cache type 'redis', got %s", cfg.CacheType)
	}

	if cfg.Timeout != 60*time.Second {
		t.Errorf("expected timeout 60s, got %v", cfg.Timeout)
	}

	if cfg.IsProduction() {
		t.Error("expected development mode")
	}
}

func TestLoad_ProxySettings(t *testing.T) {
	origProxy := os.Getenv("HTTP_PROXY")
	origNoProxy := os.Getenv("NO_PROXY")

	defer func() {
		os.Setenv("HTTP_PROXY", origProxy)
		os.Setenv("NO_PROXY", origNoProxy)
	}()

	os.Setenv("HTTP_PROXY", "http://proxy.example.com:8080")
	os.Setenv("NO_PROXY", "1")

	cfg := Load()

	if cfg.Proxy != "http://proxy.example.com:8080" {
		t.Errorf("expected proxy 'http://proxy.example.com:8080', got %s", cfg.Proxy)
	}

	if !cfg.NoProxy {
		t.Error("expected NoProxy to be true")
	}
}

func TestLoad_DisableNSFW(t *testing.T) {
	origNSFW := os.Getenv("DISABLE_NSFW")
	defer os.Setenv("DISABLE_NSFW", origNSFW)

	os.Setenv("DISABLE_NSFW", "true")
	cfg := Load()

	if !cfg.DisableNSFW {
		t.Error("expected DisableNSFW to be true")
	}
}

func TestLoad_UserAgent(t *testing.T) {
	origUA := os.Getenv("USER_AGENT")
	defer os.Setenv("USER_AGENT", origUA)

	os.Setenv("USER_AGENT", "CustomBot/1.0")
	cfg := Load()

	if cfg.UserAgent != "CustomBot/1.0" {
		t.Errorf("expected user agent 'CustomBot/1.0', got %s", cfg.UserAgent)
	}
}

func TestLoad_RedisURL(t *testing.T) {
	origRedis := os.Getenv("REDIS_URL")
	defer os.Setenv("REDIS_URL", origRedis)

	os.Setenv("REDIS_URL", "redis://redis.example.com:6380")
	cfg := Load()

	if cfg.RedisURL != "redis://redis.example.com:6380" {
		t.Errorf("expected redis URL 'redis://redis.example.com:6380', got %s", cfg.RedisURL)
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
			cfg := &Config{Env: tt.env}
			if cfg.IsProduction() != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, cfg.IsProduction())
			}
		})
	}
}

func TestLoad_TimeoutParsing(t *testing.T) {
	tests := []struct {
		name     string
		timeout  string
		expected time.Duration
	}{
		{
			name:     "30 seconds",
			timeout:  "30",
			expected: 30 * time.Second,
		},
		{
			name:     "60 seconds",
			timeout:  "60",
			expected: 60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origTimeout := os.Getenv("TIMEOUT")
			defer os.Setenv("TIMEOUT", origTimeout)

			os.Setenv("TIMEOUT", tt.timeout)
			cfg := Load()

			if cfg.Timeout != tt.expected {
				t.Errorf("expected timeout %v, got %v", tt.expected, cfg.Timeout)
			}
		})
	}
}

func TestLoad_InvalidTimeout(t *testing.T) {
	origTimeout := os.Getenv("TIMEOUT")
	defer os.Setenv("TIMEOUT", origTimeout)

	os.Setenv("TIMEOUT", "invalid")
	cfg := Load()

	// Should keep default on invalid value
	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s on invalid input, got %v", cfg.Timeout)
	}
}

func TestLoad_EmptyEnv(t *testing.T) {
	// Clear all relevant env vars
	envs := []string{"PORT", "NODE_ENV", "CACHE_TYPE", "REDIS_URL", "USER_AGENT", "TIMEOUT", "HTTP_PROXY", "DISABLE_NSFW"}
	origValues := make(map[string]string)

	for _, env := range envs {
		origValues[env] = os.Getenv(env)
		os.Unsetenv(env)
	}

	defer func() {
		for env, val := range origValues {
			if val != "" {
				os.Setenv(env, val)
			}
		}
	}()

	cfg := Load()

	// Should have all defaults
	defaultCfg := DefaultConfig()
	if cfg.Port != defaultCfg.Port {
		t.Errorf("expected default port, got %s", cfg.Port)
	}

	if cfg.Env != defaultCfg.Env {
		t.Errorf("expected default env, got %s", cfg.Env)
	}

	if cfg.CacheType != defaultCfg.CacheType {
		t.Errorf("expected default cache type, got %s", cfg.CacheType)
	}
}
