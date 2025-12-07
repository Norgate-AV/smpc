// Package logger provides structured logging with file and console output.
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	// DefaultLogMaxSize is the default maximum size in megabytes before log rotation
	DefaultLogMaxSize = 10

	// DefaultLogMaxBackups is the default number of old log files to retain
	DefaultLogMaxBackups = 3

	// DefaultLogMaxAge is the default maximum number of days to retain old log files
	DefaultLogMaxAge = 28
)

// LoggerInterface defines the logging methods
type LoggerInterface interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Close()
	GetLogPath() string
}

// LoggerOptions configures the logger
type LoggerOptions struct {
	Verbose    bool
	LogDir     string // If empty, uses %LOCALAPPDATA%\smpc
	MaxSize    int    // Max size in megabytes before rotation (default: 10)
	MaxBackups int    // Max number of old log files to keep (default: 3)
	MaxAge     int    // Max days to keep old log files (default: 28)
	Compress   bool   // Whether to compress rotated logs (default: true)
}

// GetLogPath returns the path where logs will be written based on options
func GetLogPath(opts LoggerOptions) string {
	// Determine log directory
	logDir := opts.LogDir
	if logDir == "" {
		localAppData := os.Getenv("LOCALAPPDATA")

		if localAppData == "" {
			localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}

		logDir = filepath.Join(localAppData, "smpc")
	}

	return filepath.Join(logDir, "smpc.log")
}

// PrintLogFile prints the current log file to the provided writer
// If writer is nil, prints to stdout. Returns error if log file doesn't exist or can't be read.
func PrintLogFile(w io.Writer, opts LoggerOptions) error {
	if w == nil {
		w = os.Stdout
	}

	logPath := GetLogPath(opts)

	file, err := os.Open(logPath)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", logPath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Ignore close errors on read-only file
		}
	}()

	if _, err := io.Copy(w, file); err != nil {
		return fmt.Errorf("failed to read log file: %w", err)
	}

	return nil
}

// Logger handles dual output logging (file + console)
type Logger struct {
	file             *slog.Logger
	console          *slog.Logger
	lumberjackLogger *lumberjack.Logger
	logPath          string
}

// NewLogger creates a new logger instance
func NewLogger(opts LoggerOptions) (*Logger, error) {
	// Set defaults
	if opts.MaxSize == 0 {
		opts.MaxSize = DefaultLogMaxSize
	}

	if opts.MaxBackups == 0 {
		opts.MaxBackups = DefaultLogMaxBackups
	}

	if opts.MaxAge == 0 {
		opts.MaxAge = DefaultLogMaxAge
	}

	// Get log path and ensure directory exists
	logPath := GetLogPath(opts)
	logDir := filepath.Dir(logPath)

	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("could not create log directory: %w", err)
	}

	// Set up lumberjack for log rotation
	lumberjackLogger := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    opts.MaxSize,
		MaxBackups: opts.MaxBackups,
		MaxAge:     opts.MaxAge,
		Compress:   opts.Compress,
	}

	// File logger: structured text with all fields
	fileLogger := slog.New(slog.NewTextHandler(lumberjackLogger, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Console logger: clean output without timestamps
	consoleHandler := &ConsoleHandler{
		writer:  os.Stdout,
		verbose: opts.Verbose,
	}

	consoleLogger := slog.New(consoleHandler)

	logger := &Logger{
		file:             fileLogger,
		console:          consoleLogger,
		lumberjackLogger: lumberjackLogger,
		logPath:          logPath,
	}

	return logger, nil
}

// Close closes the log file and flushes any buffered data
func (l *Logger) Close() {
	if l.lumberjackLogger != nil {
		if err := l.lumberjackLogger.Close(); err != nil {
			// Log close errors but don't fail
		}
	}
}

// GetLogPath returns the path to the current log file
func (l *Logger) GetLogPath() string {
	return l.logPath
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...any) {
	l.file.Debug(msg, args...)
	l.console.Debug(msg, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...any) {
	l.file.Info(msg, args...)
	l.console.Info(msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...any) {
	l.file.Warn(msg, args...)
	l.console.Warn(msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...any) {
	l.file.Error(msg, args...)
	l.console.Error(msg, args...)
}

// ConsoleHandler is a simple handler that outputs clean messages to console
type ConsoleHandler struct {
	writer  io.Writer
	verbose bool
}

func (h *ConsoleHandler) Enabled(_ context.Context, level slog.Level) bool {
	if !h.verbose && level == slog.LevelDebug {
		return false
	}

	return true
}

func (h *ConsoleHandler) Handle(_ context.Context, r slog.Record) error {
	var prefix string
	switch r.Level {
	case slog.LevelError:
		prefix = "ERROR: "
	case slog.LevelWarn:
		prefix = "WARNING: "
	case slog.LevelDebug:
		prefix = "[DEBUG] "
	}

	// Build the message with attributes
	msg := r.Message
	if r.NumAttrs() > 0 {
		attrs := make([]string, 0, r.NumAttrs())

		r.Attrs(func(a slog.Attr) bool {
			attrs = append(attrs, fmt.Sprintf("%s=%v", a.Key, a.Value))
			return true
		})

		if len(attrs) > 0 {
			msg = fmt.Sprintf("%s %s", msg, joinAttrs(attrs))
		}
	}

	if _, err := fmt.Fprintf(h.writer, "%s%s\n", prefix, msg); err != nil {
		// Ignore write errors to console
	}
	return nil
}

// joinAttrs joins attributes with spaces
func joinAttrs(attrs []string) string {
	if len(attrs) == 0 {
		return ""
	}

	result := attrs[0]
	for i := 1; i < len(attrs); i++ {
		result += " " + attrs[i]
	}

	return result
}

func (h *ConsoleHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *ConsoleHandler) WithGroup(_ string) slog.Handler {
	return h
}

// NoOpLogger is a logger that does nothing - useful for tests
type NoOpLogger struct{}

func (n *NoOpLogger) Debug(msg string, args ...any) {}
func (n *NoOpLogger) Info(msg string, args ...any)  {}
func (n *NoOpLogger) Warn(msg string, args ...any)  {}
func (n *NoOpLogger) Error(msg string, args ...any) {}
func (n *NoOpLogger) Close()                        {}
func (n *NoOpLogger) GetLogPath() string            { return "" }

// NewNoOpLogger creates a new no-op logger for testing
func NewNoOpLogger() *NoOpLogger {
	return &NoOpLogger{}
}
