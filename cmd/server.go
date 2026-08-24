package main

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	handlercache "github.com/xihale/rsshub-go/internal/cache"
	"github.com/xihale/rsshub-go/internal/client"
	"github.com/xihale/rsshub-go/internal/middleware"
	"github.com/xihale/rsshub-go/pkg/cache"
	"github.com/xihale/rsshub-go/pkg/config"
	ctxpkg "github.com/xihale/rsshub-go/pkg/context"
	"github.com/xihale/rsshub-go/pkg/docs"
	"github.com/xihale/rsshub-go/pkg/logger"
	"github.com/xihale/rsshub-go/pkg/models"
	"github.com/xihale/rsshub-go/pkg/registry"
	"github.com/xihale/rsshub-go/pkg/rss"
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

	// Register gob types for cache serialization
	gob.Register(&models.Feed{})
	gob.Register(&models.Item{})
	gob.Register(&models.Author{})

	logger.Info("Starting RSSHub Go", zap.String("port", cfg.GetPort()))

	// Initialize cache (two-tier: memory + badger)
	var cacheInstance cache.Cache
	badgerPath := cfg.GetBadgerPath()

	if cfg.GetCacheType() == "badger" {
		badgerCache, err := cache.NewBadgerCache(cfg.GetMemoryCacheSize(), badgerPath, cfg.GetCacheTTL(), cfg.GetCacheCleanupInterval())
		if err != nil {
			logger.Error("Failed to initialize badger cache, falling back to memory", zap.Error(err))
			cacheInstance = cache.NewMemoryCache(cfg.GetMemoryCacheSize())
			logger.Info("Using memory cache", zap.Int("size", cfg.GetMemoryCacheSize()))
		} else {
			cacheInstance = badgerCache
			logger.Info("Using two-tier cache (memory + badger)",
				zap.Int("memory_size", cfg.GetMemoryCacheSize()),
				zap.String("badger_path", badgerPath))
		}
	} else {
		cacheInstance = cache.NewMemoryCache(cfg.GetMemoryCacheSize())
		logger.Info("Using memory cache", zap.Int("size", cfg.GetMemoryCacheSize()))
	}

	// Initialize HTTP client
	clientOpts := []client.ClientOption{
		client.WithUserAgent(cfg.GetUserAgent()),
		client.WithTimeout(cfg.GetTimeout()),
		client.WithMaxRedirects(cfg.Client.MaxRedirects),
	}
	if cfg.Client.NoProxy {
		clientOpts = append(clientOpts, client.WithNoProxy())
	} else if proxy := cfg.GetProxy(); proxy != "" {
		clientOpts = append(clientOpts, client.WithProxy(proxy))
	}
	httpClient := client.New(clientOpts...)

	// Create Gin engine with custom middleware stack
	engine := gin.New()

	// Apply middleware in order (outermost first)
	engine.Use(
		middleware.Recovery(), // 1. Panic recovery (OUTERMOST)
		middleware.Logger(),   // 2. Request logging
		middleware.Header(cfg.GetCacheTTL(), cfg.GetAllowOrigin()), // 3. HTTP headers (CORS, ETag, Cache-Control)
		// Note: Parameter handling moved into handler-level caching
	)

	// Register routes explicitly via generated route package bootstrap.
	registerRoutePackages()

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

	// Close cache (badger needs to be closed properly)
	if err := cacheInstance.Close(); err != nil {
		logger.Error("Failed to close cache", zap.Error(err))
	}

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
		logger.Info("Documentation available at", zap.String("url", "http://localhost:"+cfg.GetPort()+"/"))
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
			// Extract all Gin path parameters into the route context.
			params := make(map[string]string, len(c.Params))
			for _, param := range c.Params {
				params[param.Key] = param.Value
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

		if cfg.GetEnableCache() {
			// Register route with handler-level caching.
			engine.GET(ginPath, handlercache.Cached(cacheInstance, handler, cachedHandlerOpts))
			continue
		}

		// Register route without caching.
		engine.GET(ginPath, func(c *gin.Context) {
			feed, err := handler(c)
			if err != nil {
				if errorCode, exists := c.Get("error_code"); exists {
					if code, ok := errorCode.(int); ok {
						c.JSON(code, gin.H{"error": err.Error()})
						return
					}
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.Header("X-Cache", "BYPASS")
			outputFeed(c, feed)
		})
	}
}

func outputFeed(c *gin.Context, feed *models.Feed) {
	processedFeed := &models.Feed{
		Title:       feed.Title,
		Link:        feed.Link,
		Description: feed.Description,
		Items:       middleware.ProcessFeed(c, feed.Items),
	}

	format := c.DefaultQuery("format", "rss")
	switch format {
	case "atom":
		outputAtom(c, processedFeed)
	case "json":
		data, err := json.MarshalIndent(processedFeed, "", "  ")
		if err != nil {
			logger.Error("Failed to generate JSON", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Data(http.StatusOK, "application/json; charset=utf-8", data)
	default:
		outputRSS(c, processedFeed)
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
