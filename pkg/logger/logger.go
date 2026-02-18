package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Logger *zap.Logger
)

// Init initializes the logger
func Init(env string) error {
	var err error
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	if env == "development" {
		cfg = zap.NewDevelopmentConfig()
		cfg.Encoding = "console"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	Logger, err = cfg.Build()
	if err != nil {
		return err
	}

	return nil
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	if Logger != nil {
		Logger.Info(msg, fields...)
	}
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	if Logger != nil {
		Logger.Error(msg, fields...)
	}
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	if Logger != nil {
		Logger.Debug(msg, fields...)
	}
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	if Logger != nil {
		Logger.Warn(msg, fields...)
	}
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	if Logger != nil {
		Logger.Fatal(msg, fields...)
	}
}

// Sync flushes any buffered log entries
func Sync() error {
	if Logger != nil {
		return Logger.Sync()
	}
	return nil
}
