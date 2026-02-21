package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/xihale/rsshub-go/internal/client"
	"github.com/xihale/rsshub-go/internal/middleware"
	handlercache "github.com/xihale/rsshub-go/internal/cache"
	"github.com/xihale/rsshub-go/pkg/cache"
	"github.com/xihale/rsshub-go/pkg/config"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/docs"
	"github.com/xihale/rsshub-go/pkg/logger"
	"github.com/xihale/rsshub-go/pkg/models"
	"github.com/xihale/rsshub-go/pkg/registry"
	"github.com/xihale/rsshub-go/pkg/rss"

	// Import route packages to trigger init() registration
	_ "github.com/xihale/rsshub-go/routes/github"
	_ "github.com/xihale/rsshub-go/routes/hackernews"
	_ "github.com/xihale/rsshub-go/routes/npm"
	_ "github.com/xihale/rsshub-go/routes/reddit"
	_ "github.com/xihale/rsshub-go/routes/techne98"
)

func main() {
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		fmt.Println("Using default configuration. Create a config.yaml file to customize settings.")
		cfg = config.DefaultConfig()
	}

	// Initialize logger
	if err := logger.Init(cfg.GetEnv()); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting RSSHub Go", zap.String("port", cfg.GetPort()))

	// Initialize cache
	var cacheInstance cache.Cache
	if cfg.GetCacheType() == "redis" {
		logger.Warn("Redis cache not yet implemented, falling back to memory")
		cacheInstance = cache.NewMemoryCache(cfg.GetMemoryCacheSize())
	} else {
		cacheInstance = cache.NewMemoryCache(cfg.GetMemoryCacheSize())
	}

	// Initialize HTTP client
	httpClient := client.New(
		client.WithUserAgent(cfg.GetUserAgent()),
		client.WithTimeout(cfg.GetTimeout()),
	)

	// Create Gin engine with custom middleware stack
	engine := gin.New()

	// Apply middleware in order (outermost first)
	engine.Use(
		middleware.Recovery(),                // 1. Panic recovery (OUTERMOST)
		middleware.Logger(),                  // 2. Request logging
		middleware.Header(cfg.GetCacheTTL()), // 3. HTTP headers (CORS, ETag, Cache-Control)
		// Note: Parameter handling moved into handler-level caching
	)

	// Register routes (auto-registered via init() in route packages)
	routeRegistry := registry.GetRegistry()

	// Log registered routes for debugging
	allRoutes := routeRegistry.GetAllRoutes()
	logger.Info("Registered routes", zap.Int("count", len(allRoutes)))
	for _, route := range allRoutes {
		logger.Info("Route", zap.String("path", route.Path), zap.String("name", route.Name))
	}

	// Setup routes on Gin
	setupGinRoutes(engine, routeRegistry, cacheInstance, httpClient, cfg)

	// Start server
	server := &http.Server{
		Addr:         ":" + cfg.GetPort(),
		Handler:      engine,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Graceful shutdown
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exiting")
}

// setupGinRoutes registers all routes from the registry directly with Gin
func setupGinRoutes(engine *gin.Engine, routeRegistry *registry.Registry, cacheInstance cache.Cache, httpClient *client.Client, cfg *config.Config) {
	// Initialize and register documentation routes
	docsHandler, err := docs.NewHandler()
	if err != nil {
		logger.Warn("Failed to initialize docs handler", zap.Error(err))
	} else {
		docsHandler.RegisterRoutes(engine)
		logger.Info("Documentation available at", zap.String("url", "http://localhost:"+cfg.GetPort()+"/docs"))
	}

	// Health check - no caching (always fresh status)
	engine.GET("/status", func(c *gin.Context) {
		feed := &models.Feed{
			Title:       "RSSHub Go",
			Link:        c.Request.URL.String(),
			Description: "RSS feed generation in Go",
			Items: []models.Item{
				{
					Title:   "Server is running",
					Link:    "/status",
					GUID:    "status-1",
					PubDate: time.Now(),
				},
			},
		}

		outputRSS(c, feed)
	})

	// Get all routes from registry
	allRoutes := routeRegistry.GetAllRoutes()

	// Register each route with Gin using parameterized patterns
	for _, route := range allRoutes {
		// Determine TTL for this route (use route-specific or default)
		ttl := cfg.GetCacheTTL()
		if route.CacheTTL != nil {
			ttl = *route.CacheTTL
		}

		// Create cache options for this specific route
		cachedHandlerOpts := &handlercache.CachedHandlerOptions{
			KeyGenerator: handlercache.DefaultKeyGenerator,
			TTL:          ttl,
			ETagEnabled:  true,
			ShouldCache: func(c *gin.Context, feed *models.Feed) bool {
				if errorCode, exists := c.Get("error_code"); exists && errorCode.(int) >= 400 {
					return false
				}
				return handlercache.DefaultShouldCache(c, feed)
			},
		}
		// Convert route path to Gin pattern (e.g., "/github/repos/:username" stays the same)
		// The route already contains :param syntax which Gin understands
		ginPath := route.Path

		// Create handler wrapper
		handler := func(c *gin.Context) (*models.Feed, error) {
			// Extract path parameters into context
			params := make(map[string]string)
			for _, param := range route.Parameters {
				params[param.Name] = c.Param(param.Name)
			}

			// Create custom context
			ctx := ctxpkg.NewContext(c.Writer, c.Request)
			ctx.SetParams(params)
			ctx.SetClient(httpClient)
			ctx.SetCache(cacheInstance)

			// Call route handler
			feed, err := route.Handler(ctx)
			if err != nil {
				return nil, err
			}

			// Store feed for parameter middleware
			c.Set("_rsshub_feed", feed)

			return feed, nil
		}

		// Register RSS route with caching
		engine.GET(ginPath, handlercache.Cached(cacheInstance, handler, cachedHandlerOpts))
	}
}

func outputRSS(c *gin.Context, feed *models.Feed) {
	data, err := rss.GenerateRSS(feed)
	if err != nil {
		logger.Error("Failed to generate RSS", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "application/rss+xml; charset=utf-8")
	c.String(http.StatusOK, string(data))
}

func outputAtom(c *gin.Context, feed *models.Feed) {
	data, err := rss.GenerateAtom(feed)
	if err != nil {
		logger.Error("Failed to generate Atom", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "application/atom+xml; charset=utf-8")
	c.String(http.StatusOK, string(data))
}
