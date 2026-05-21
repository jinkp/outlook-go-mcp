package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// New creates a structured JSON logger that writes to stderr.
// If logFile is non-empty, output is tee'd to that file as well.
func New(level string, logFile string) (*slog.Logger, error) {
	parsedLevel, err := parseLevel(level)
	if err != nil {
		return nil, err
	}

	var w io.Writer = os.Stderr

	if logFile != "" {
		f, err := openLogFile(logFile)
		if err != nil {
			// Log file failure is non-fatal: fall back to stderr only and warn.
			fmt.Fprintf(os.Stderr, "outlook-mcp: cannot open log file %q: %v — logging to stderr only\n", logFile, err)
		} else {
			w = io.MultiWriter(os.Stderr, f)
		}
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: parsedLevel})
	return slog.New(handler), nil
}

// NewPreBootstrap returns a minimal debug logger used before config is loaded.
// Always uses debug level so every startup event is captured.
// If logFile is non-empty, output is tee'd to that file as well (best-effort).
func NewPreBootstrap(logFile string) *slog.Logger {
	var w io.Writer = os.Stderr
	if logFile != "" {
		if f, err := openLogFile(logFile); err == nil {
			w = io.MultiWriter(os.Stderr, f)
		}
	}
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(handler)
}

// openLogFile opens (or creates) a log file for appending.
// Creates parent directories if they do not exist.
func openLogFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	return os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
}

func parseLevel(level string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid log level %q: must be one of debug, info, warn, error", level)
	}
}
