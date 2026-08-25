package routeutils

import (
	"time"

	"go.uber.org/zap"
)

// LogRouteStart logs handler invocation
func LogRouteStart(logger *zap.Logger, routeName string, params map[string]string) {
	if logger == nil {
		return
	}

	fields := []zap.Field{zap.String("route", routeName)}
	for k, v := range params {
		fields = append(fields, zap.String("param_"+k, v))
	}

	logger.Debug("Route handler started", fields...)
}

// LogRouteSuccess logs successful completion with duration and item count
func LogRouteSuccess(logger *zap.Logger, routeName string, duration time.Duration, itemCount int) {
	if logger == nil {
		return
	}

	logger.Info("Route handler completed",
		zap.String("route", routeName),
		zap.Duration("duration", duration),
		zap.Int("item_count", itemCount),
	)
}

// LogRouteError logs handler failure
func LogRouteError(logger *zap.Logger, routeName string, err error, params map[string]string) {
	if logger == nil {
		return
	}

	fields := []zap.Field{
		zap.String("route", routeName),
		zap.Error(err),
	}
	for k, v := range params {
		fields = append(fields, zap.String("param_"+k, v))
	}

	logger.Error("Route handler failed", fields...)
}

// LogFetchStart logs external HTTP request
func LogFetchStart(logger *zap.Logger, url string) {
	if logger == nil {
		return
	}

	logger.Debug("Fetching URL", zap.String("url", url))
}

// LogFetchSuccess logs successful fetch with status and bytes
func LogFetchSuccess(logger *zap.Logger, url string, statusCode int, bytes int) {
	if logger == nil {
		return
	}

	logger.Debug("Fetch successful",
		zap.String("url", url),
		zap.Int("status", statusCode),
		zap.Int("bytes", bytes),
	)
}

// LogFetchError logs fetch failure
func LogFetchError(logger *zap.Logger, url string, err error, statusCode int) {
	if logger == nil {
		return
	}

	fields := []zap.Field{
		zap.String("url", url),
		zap.Error(err),
	}
	if statusCode > 0 {
		fields = append(fields, zap.Int("status", statusCode))
	}

	logger.Error("Fetch failed", fields...)
}

// WithTiming returns a completion function that logs duration
// Usage: complete := routeutils.WithTiming(logger, "route_name"); defer complete()
func WithTiming(logger *zap.Logger, routeName string) func() {
	start := time.Now()
	return func() {
		if logger == nil {
			return
		}
		duration := time.Since(start)
		logger.Debug("Route timing",
			zap.String("route", routeName),
			zap.Duration("duration", duration),
		)
	}
}

// LogCacheHit logs a cache hit
func LogCacheHit(logger *zap.Logger, key string) {
	if logger == nil {
		return
	}
	logger.Debug("Cache hit", zap.String("key", key))
}

// LogCacheMiss logs a cache miss
func LogCacheMiss(logger *zap.Logger, key string) {
	if logger == nil {
		return
	}
	logger.Debug("Cache miss", zap.String("key", key))
}

// LogCacheSet logs a cache set operation
func LogCacheSet(logger *zap.Logger, key string, ttl time.Duration) {
	if logger == nil {
		return
	}
	logger.Debug("Cache set",
		zap.String("key", key),
		zap.Duration("ttl", ttl),
	)
}

// LogParseError logs a parsing error
func LogParseError(logger *zap.Logger, url string, err error) {
	if logger == nil {
		return
	}
	logger.Error("Failed to parse response",
		zap.String("url", url),
		zap.Error(err),
	)
}

// LogDebug logs a debug message
func LogDebug(logger *zap.Logger, msg string, fields ...zap.Field) {
	if logger == nil {
		return
	}
	logger.Debug(msg, fields...)
}

// LogInfo logs an info message
func LogInfo(logger *zap.Logger, msg string, fields ...zap.Field) {
	if logger == nil {
		return
	}
	logger.Info(msg, fields...)
}

// LogWarn logs a warning message
func LogWarn(logger *zap.Logger, msg string, fields ...zap.Field) {
	if logger == nil {
		return
	}
	logger.Warn(msg, fields...)
}

// LogError logs an error message
func LogError(logger *zap.Logger, msg string, fields ...zap.Field) {
	if logger == nil {
		return
	}
	logger.Error(msg, fields...)
}
