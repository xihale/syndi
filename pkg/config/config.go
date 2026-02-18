package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds application configuration
type Config struct {
	// Server settings
	Port         string
	Env          string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	// Cache settings
	CacheType   string // "memory" or "redis"
	RedisURL    string
	CacheTTL    time.Duration
	MemoryCache int

	// Client settings
	UserAgent    string
	Timeout      time.Duration
	MaxRedirects int
	Proxy        string
	NoProxy      bool

	// Route specific
	DisableNSFW bool

	// Middleware options
	EnableCache bool
	AccessKey   string
	AllowOrigin string
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Port:         "1200",
		Env:          "production",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		CacheType:    "memory",
		RedisURL:     "redis://localhost:6379",
		CacheTTL:     15 * time.Minute,
		MemoryCache:  10000,
		UserAgent:    "RSSHub-Go/1.0 (+https://github.com/xihale/rsshub-go)",
		Timeout:      30 * time.Second,
		MaxRedirects: 10,
		Proxy:        "",
		NoProxy:      false,
		DisableNSFW:  false,
		EnableCache:  true,
		AccessKey:    "",
		AllowOrigin:  "*",
	}
}

// Load loads configuration from environment variables
func Load() *Config {
	cfg := DefaultConfig()

	if port := os.Getenv("PORT"); port != "" {
		cfg.Port = port
	}

	if env := os.Getenv("NODE_ENV"); env != "" {
		cfg.Env = env
	}

	if cacheType := os.Getenv("CACHE_TYPE"); cacheType != "" {
		cfg.CacheType = cacheType
	}

	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		cfg.RedisURL = redisURL
	}

	if ua := os.Getenv("USER_AGENT"); ua != "" {
		cfg.UserAgent = ua
	}

	if timeout := os.Getenv("TIMEOUT"); timeout != "" {
		if t, err := strconv.Atoi(timeout); err == nil {
			cfg.Timeout = time.Duration(t) * time.Second
		}
	}

	if proxy := os.Getenv("HTTP_PROXY"); proxy != "" {
		cfg.Proxy = proxy
	}

	if noProxy := os.Getenv("NO_PROXY"); noProxy != "" {
		cfg.NoProxy = true
	}

	if disableNSFW := os.Getenv("DISABLE_NSFW"); disableNSFW != "" {
		cfg.DisableNSFW = true
	}

	if enableCache := os.Getenv("ENABLE_CACHE"); enableCache != "" {
		cfg.EnableCache = enableCache == "true" || enableCache == "1"
	}

	if accessKey := os.Getenv("ACCESS_KEY"); accessKey != "" {
		cfg.AccessKey = accessKey
	}

	if allowOrigin := os.Getenv("ALLOW_ORIGIN"); allowOrigin != "" {
		cfg.AllowOrigin = allowOrigin
	}

	// Route-specific configs (prefixed with SITE_)
	// Example: TWITTER_COOKIE, GITHUB_TOKEN
	for _, env := range os.Environ() {
		// Routes can read these environment variables directly
		_ = env // placeholder
	}

	return cfg
}

// Get retrieves a configuration value for a route
func (c *Config) Get(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// IsProduction checks if running in production
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}
