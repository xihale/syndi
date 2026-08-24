package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"time"
)

// Config holds application configuration
type Config struct {
	// Server settings
	Server struct {
		Port         string        `yaml:"port"`
		Env          string        `yaml:"env"`
		ReadTimeout  time.Duration `yaml:"read_timeout"`
		WriteTimeout time.Duration `yaml:"write_timeout"`
		IdleTimeout  time.Duration `yaml:"idle_timeout"`
	} `yaml:"server"`
	// Cache settings
	Cache struct {
		Type   string `yaml:"type"` // "memory" or "badger"
		Badger struct {
			Path string `yaml:"path"`
		} `yaml:"badger"`
		CleanupInterval time.Duration `yaml:"cleanup_interval"`
		TTL             time.Duration `yaml:"ttl"`
		MemorySize      int           `yaml:"memory_size"`
	} `yaml:"cache"`
	// Client settings
	Client struct {
		UserAgent    string        `yaml:"user_agent"`
		Timeout      time.Duration `yaml:"timeout"`
		MaxRedirects int           `yaml:"max_redirects"`
		Proxy        string        `yaml:"proxy"`
		NoProxy      bool          `yaml:"no_proxy"`
	} `yaml:"client"`
	// Route specific
	Routes struct {
		DisableNSFW bool `yaml:"disable_nsfw"`
	} `yaml:"routes"`
	// Middleware options
	Middleware struct {
		EnableCache bool   `yaml:"enable_cache"`
		AccessKey   string `yaml:"access_key"`
		AllowOrigin string `yaml:"allow_origin"`
	} `yaml:"middleware"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	cfg := &Config{}
	// Server defaults
	cfg.Server.Port = "1200"
	cfg.Server.Env = "production"
	cfg.Server.ReadTimeout = 30 * time.Second
	cfg.Server.WriteTimeout = 30 * time.Second
	cfg.Server.IdleTimeout = 120 * time.Second
	// Cache defaults
	cfg.Cache.Type = "memory"
	cfg.Cache.Badger.Path = "./data/cache"
	cfg.Cache.TTL = 15 * time.Minute
	cfg.Cache.CleanupInterval = 5 * time.Minute
	cfg.Cache.MemorySize = 10000
	// Client defaults
	cfg.Client.UserAgent = "RSSHub-Go/1.0 (+https://github.com/xihale/rsshub-go)"
	cfg.Client.Timeout = 30 * time.Second
	cfg.Client.MaxRedirects = 10
	cfg.Client.Proxy = ""
	cfg.Client.NoProxy = false
	// Route defaults
	cfg.Routes.DisableNSFW = false
	// Middleware defaults
	cfg.Middleware.EnableCache = true
	cfg.Middleware.AccessKey = ""
	cfg.Middleware.AllowOrigin = "*"
	return cfg
}

// Load loads configuration from a YAML file
// If configPath is empty, it tries to find config.yaml in:
// 1. ./config.yaml (current directory)
// 2. /etc/rsshub-go/config.yaml (system-wide config)
func Load(configPath string) (*Config, error) {
	cfg := DefaultConfig()
	// If no path specified, try default locations
	if configPath == "" {
		// Try environment variable first
		if envPath := os.Getenv("RSSHUB_CONFIG"); envPath != "" {
			configPath = envPath
		} else {
			// Try default locations in order of precedence
			for _, path := range []string{"config.yaml", "/etc/rsshub-go/config.yaml"} {
				if _, err := os.Stat(path); err == nil {
					configPath = path
					break
				}
			}
		}
	}
	// If we found a config file, load it
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
		}
	}
	return cfg, nil
}

// LoadOrPanic loads configuration and panics on error
func LoadOrPanic(configPath string) *Config {
	cfg, err := Load(configPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to load configuration: %v", err))
	}
	return cfg
}

// GetPort returns the server port (for backward compatibility)
func (c *Config) GetPort() string {
	return c.Server.Port
}

// GetEnv returns the environment (for backward compatibility)
func (c *Config) GetEnv() string {
	return c.Server.Env
}

// GetCacheTTL returns the cache TTL (for backward compatibility)
func (c *Config) GetCacheTTL() time.Duration {
	return c.Cache.TTL
}

// GetCacheType returns the cache type (for backward compatibility)
func (c *Config) GetCacheType() string {
	return c.Cache.Type
}

// GetMemoryCacheSize returns the memory cache size (for backward compatibility)
func (c *Config) GetMemoryCacheSize() int {
	return c.Cache.MemorySize
}

// GetBadgerPath returns the BadgerDB path (for backward compatibility)
func (c *Config) GetBadgerPath() string {
	return c.Cache.Badger.Path
}

// GetCacheCleanupInterval returns the cleanup interval for expired cache entries (for backward compatibility)
func (c *Config) GetCacheCleanupInterval() time.Duration {
	return c.Cache.CleanupInterval
}

// GetUserAgent returns the user agent (for backward compatibility)
func (c *Config) GetUserAgent() string {
	return c.Client.UserAgent
}

// GetTimeout returns the HTTP client timeout (for backward compatibility)
func (c *Config) GetTimeout() time.Duration {
	return c.Client.Timeout
}

// GetProxy returns the proxy URL (for backward compatibility)
func (c *Config) GetProxy() string {
	return c.Client.Proxy
}

// GetDisableNSFW returns whether NSFW routes are disabled (for backward compatibility)
func (c *Config) GetDisableNSFW() bool {
	return c.Routes.DisableNSFW
}

// GetEnableCache returns whether caching is enabled (for backward compatibility)
func (c *Config) GetEnableCache() bool {
	return c.Middleware.EnableCache
}

// GetAccessKey returns the access key (for backward compatibility)
func (c *Config) GetAccessKey() string {
	return c.Middleware.AccessKey
}

// GetAllowOrigin returns the allowed origin (for backward compatibility)
func (c *Config) GetAllowOrigin() string {
	return c.Middleware.AllowOrigin
}

// Get retrieves a configuration value for a route
// This checks environment variables for route-specific configs
func (c *Config) Get(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// IsProduction checks if running in production
func (c *Config) IsProduction() bool {
	return c.Server.Env == "production"
}

// GetConfigPath returns the absolute path to the config file
func GetConfigPath(configPath string) (string, error) {
	if configPath == "" {
		if envPath := os.Getenv("RSSHUB_CONFIG"); envPath != "" {
			configPath = envPath
		} else {
			// Return first existing default config
			for _, path := range []string{"config.yaml", "/etc/rsshub-go/config.yaml"} {
				if _, err := os.Stat(path); err == nil {
					return filepath.Abs(path)
				}
			}
			return "", fmt.Errorf("no config file found, tried config.yaml and /etc/rsshub-go/config.yaml")
		}
	}
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for %s: %w", configPath, err)
	}
	if _, err := os.Stat(absPath); err != nil {
		return "", fmt.Errorf("config file not found: %s", absPath)
	}
	return absPath, nil
}
