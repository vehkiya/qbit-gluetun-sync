package logger

import (
	"log/slog"
	"os"
	"strings"
)

var internalLogger *slog.Logger

// Init initializes the global logger with the specified level.
func Init(levelStr string) {
	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	internalLogger = slog.New(handler)
}

// Ensure the logger falls back to standard log behavior if not initialized
func getDefault() *slog.Logger {
	if internalLogger != nil {
		return internalLogger
	}
	// Fallback during tests or early init
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

// Info logs an informational message.
func Info(msg string, args ...any) {
	getDefault().Info(msg, args...)
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	getDefault().Warn(msg, args...)
}

// Error logs an error message.
func Error(msg string, args ...any) {
	getDefault().Error(msg, args...)
}

// Debug logs a debug message.
func Debug(msg string, args ...any) {
	getDefault().Debug(msg, args...)
}

// Fatal logs an error and exits the program.
func Fatal(msg string, args ...any) {
	getDefault().Error(msg, args...)
	os.Exit(1)
}
