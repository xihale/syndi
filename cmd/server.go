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

	"github.com/rsshub/go/internal/client"
	"github.com/rsshub/go/internal/middleware"
	handlercache "github.com/rsshub/go/internal/cache"
	"github.com/rsshub/go/pkg/cache"
	"github.com/rsshub/go/pkg/config"
	ctxpkg "github.com/rsshub/go/pkg/context"
	"github.com/rsshub/go/pkg/logger"
	"github.com/rsshub/go/pkg/models"
	"github.com/rsshub/go/pkg/registry"
	"github.com/rsshub/go/pkg/rss"
)

func main() {
	// Initialize logger
	cfg := config.Load()
	if err := logger.Init(cfg.Env); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting RSSHub Go", zap.String("port", cfg.Port))

	// Initialize cache
	var cacheInstance cache.Cache
	if cfg.CacheType == "redis" {
		logger.Warn("Redis cache not yet implemented, falling back to memory")
		cacheInstance = cache.NewMemoryCache(cfg.MemoryCache)
	} else {
		cacheInstance = cache.NewMemoryCache(cfg.MemoryCache)
	}

	// Initialize HTTP client
	httpClient := client.New(
		client.WithUserAgent(cfg.UserAgent),
		client.WithTimeout(cfg.Timeout),
	)

	// Create Gin engine with custom middleware stack
	engine := gin.New()

	// Apply middleware in order (outermost first)
	engine.Use(
		middleware.Recovery(),           // 1. Panic recovery (OUTERMOST)
		middleware.Logger(),             // 2. Request logging
		middleware.Header(cfg.CacheTTL), // 3. HTTP headers (CORS, ETag, Cache-Control)
		middleware.Parameter(),          // 4. Query parameter processing (INNERMOST)
		// Note: Cache middleware removed - using handler-level caching instead
	)

	// Register routes
	routeRegistry := registry.GetRegistry()
	registerRoutes(routeRegistry)

	// Setup routes on Gin
	setupGinRoutes(engine, routeRegistry, cacheInstance, httpClient)

	// Start server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      engine,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
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

// Register all routes (init-style for compatibility)
func registerRoutes(r *registry.Registry) {
	// Routes will be registered via init() functions in each route file
	// This is a placeholder - actual registration happens in route packages
}

func setupGinRoutes(engine *gin.Engine, routeRegistry *registry.Registry, cacheInstance cache.Cache, httpClient *client.Client) {
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

	// Main feed route with handler-level caching
	// Uses default TTL (15 minutes) and smart caching logic
	// Custom ShouldCache function prevents caching 404 errors
	mainRouteOpts := &handlercache.CachedHandlerOptions{
		ShouldCache: func(c *gin.Context, feed *models.Feed) bool {
			// Don't cache if there's an error code set (404, etc.)
		if errorCode, exists := c.Get("error_code"); exists && errorCode.(int) >= 400 {
				return false
			}
			// Use default logic for successful responses
			return handlercache.DefaultShouldCache(c, feed)
		},
	}

	engine.GET("/:namespace/:path", handlercache.Cached(cacheInstance, func(c *gin.Context) (*models.Feed, error) {
		namespace := c.Param("namespace")
		path := c.Param("path")
		fullPath := "/" + namespace + "/" + path

		// Check if route exists
		route := routeRegistry.GetRoute(fullPath)
		if route == nil {
			c.Set("error_code", http.StatusNotFound)
			return nil, fmt.Errorf("route not found: %s", fullPath)
		}

		// Create context for handler
		ctx := ctxpkg.NewContext(c.Writer, c.Request)
		ctx.SetClient(httpClient)
		ctx.SetCache(cacheInstance)

		// Call the route handler
		feed, err := route.Handler(ctx)
		if err != nil {
			return nil, err
		}

		// Store feed in context for parameter middleware
		c.Set("_rsshub_feed", feed)

		return feed, nil
	}, mainRouteOpts))

	// API route with shorter cache TTL (5 minutes for JSON data)
	apiRouteOpts := &handlercache.CachedHandlerOptions{
		TTL: 5 * time.Minute,
		ShouldCache: func(c *gin.Context, feed *models.Feed) bool {
			// Don't cache if there's an error code set (404, etc.)
			if errorCode, exists := c.Get("error_code"); exists && errorCode.(int) >= 400 {
				return false
			}
			// Use default logic for successful responses
			return handlercache.DefaultShouldCache(c, feed)
		},
	}

	engine.GET("/api/:namespace/:path", handlercache.Cached(cacheInstance, func(c *gin.Context) (*models.Feed, error) {
		namespace := c.Param("namespace")
		path := c.Param("path")
		fullPath := "/" + namespace + "/" + path

		route := routeRegistry.GetRoute(fullPath)
		if route == nil {
			c.Set("error_code", http.StatusNotFound)
			return nil, fmt.Errorf("route not found: %s", fullPath)
		}

		ctx := ctxpkg.NewContext(c.Writer, c.Request)
		ctx.SetClient(httpClient)
		ctx.SetCache(cacheInstance)

		feed, err := route.Handler(ctx)
		if err != nil {
			return nil, err
		}

		// Store feed in context for parameter middleware
		c.Set("_rsshub_feed", feed)

		return feed, nil
	}, apiRouteOpts))
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
