// Package logger provides structured logging using zap
package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.SugaredLogger

func init() {
	log = NewLogger()
}

// NewLogger creates a new logger with configuration from environment
func NewLogger() *zap.SugaredLogger {
	level := getLogLevel()

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Development:      isDevelopment(),
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	if isDevelopment() {
		config.Encoding = "console"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	logger, err := config.Build()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}

	return logger.Sugar()
}

// Get returns the global logger instance
func Get() *zap.SugaredLogger {
	return log
}

// Debug logs a debug message
func Debug(msg string, fields ...interface{}) {
	log.Debugw(msg, fields...)
}

// Info logs an info message
func Info(msg string, fields ...interface{}) {
	log.Infow(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...interface{}) {
	log.Warnw(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...interface{}) {
	log.Errorw(msg, fields...)
}

// DPanic logs a panic message
func DPanic(msg string, fields ...interface{}) {
	log.DPanicw(msg, fields...)
}

// Panic logs a panic message
func Panic(msg string, fields ...interface{}) {
	log.Panicw(msg, fields...)
}

// Fatal logs a fatal message
func Fatal(msg string, fields ...interface{}) {
	log.Fatalw(msg, fields...)
}

// Sync flushes any buffered log entries
func Sync() error {
	return log.Desugar().Sync()
}

func getLogLevel() zapcore.Level {
	levelStr := os.Getenv("LOG_LEVEL")
	if levelStr == "" {
		return zapcore.WarnLevel
	}

	var level zapcore.Level
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		return zapcore.WarnLevel
	}

	return level
}

func isDevelopment() bool {
	return os.Getenv("ENV") == "development" || os.Getenv("NODE_ENV") == "development"
}
