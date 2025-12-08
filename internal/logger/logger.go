// Package logger provides structured logging with file and console output.
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	// DefaultLogMaxSize is the default maximum size in megabytes before log rotation
	DefaultLogMaxSize = 2

	// DefaultLogMaxBackups is the default number of old log files to retain
	DefaultLogMaxBackups = 3

	// DefaultLogMaxAge is the default maximum number of days to retain old log files
	DefaultLogMaxAge = 28

	// LevelTrace is a custom log level below Debug, only logged to file
	LevelTrace = slog.LevelDebug - 4
)

// LoggerInterface defines the logging methods
type LoggerInterface interface {
	Trace(msg string, args ...any) // Only logs to file, never to console
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

	// File logger: structured text with all fields (including Trace level)
	fileLogger := slog.New(slog.NewTextHandler(lumberjackLogger, &slog.HandlerOptions{
		Level: LevelTrace, // Set to LevelTrace to capture all levels including Trace
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Replace "DEBUG-4" with "TRACE" in the level attribute
			if a.Key == slog.LevelKey && a.Value.Any().(slog.Level) == LevelTrace {
				a.Value = slog.StringValue("TRACE")
			}
			return a
		},
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
			// Log to stderr since we're closing the log file
			fmt.Fprintf(os.Stderr, "ERROR: Failed to close log file: %v\n", err)
		}
	}
}

// GetLogPath returns the path to the current log file
func (l *Logger) GetLogPath() string {
	return l.logPath
}

// Trace logs a trace message (file only, never to console)
func (l *Logger) Trace(msg string, args ...any) {
	l.file.Log(context.Background(), LevelTrace, msg, args...)
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
	// Trace level never goes to console
	if level == LevelTrace {
		return false
	}

	if !h.verbose && level == slog.LevelDebug {
		return false
	}

	return true
}

func (h *ConsoleHandler) Handle(_ context.Context, r slog.Record) error {
	var prefix string
	var colorFunc *color.Color

	switch r.Level {
	case slog.LevelError:
		prefix = "ERROR: "
		colorFunc = color.New(color.FgRed)
	case slog.LevelWarn:
		prefix = "WARNING: "
		colorFunc = color.New(color.FgYellow)
	case slog.LevelDebug:
		prefix = "VERBOSE: "
		colorFunc = color.New(color.FgCyan)
	}

	// Build the message with attributes
	// For Info level, include attributes UNLESS the message is an enumerated list item
	// (which starts with spaces and a number like "  1. ERROR...")
	// For other levels (DEBUG/VERBOSE, WARN, ERROR), always include attributes
	msg := r.Message

	// Determine if we should include attributes
	includeAttrs := r.NumAttrs() > 0
	if r.Level == slog.LevelInfo {
		// Don't include attributes for enumerated messages (starts with "  " followed by a digit)
		includeAttrs = includeAttrs && !isEnumeratedMessage(msg)
	}

	if includeAttrs {
		attrs := make([]string, 0, r.NumAttrs())

		r.Attrs(func(a slog.Attr) bool {
			attrs = append(attrs, fmt.Sprintf("%s=%v", a.Key, a.Value))
			return true
		})

		if len(attrs) > 0 {
			msg = fmt.Sprintf("%s %s", msg, joinAttrs(attrs))
		}
	}

	// Apply color if set, otherwise plain output
	if colorFunc != nil {
		if _, err := colorFunc.Fprintf(h.writer, "%s%s\n", prefix, msg); err != nil {
			// Ignore write errors to console
		}

		return nil
	}

	if _, err := fmt.Fprintf(h.writer, "%s%s\n", prefix, msg); err != nil {
		// Ignore write errors to console
	}

	return nil
}

// isEnumeratedMessage checks if a message is an enumerated list item
// (e.g., "  1. ERROR...", "  2. WARNING...")
func isEnumeratedMessage(msg string) bool {
	if len(msg) < 4 {
		return false
	}
	// Check for pattern: "  " followed by a digit
	return msg[0] == ' ' && msg[1] == ' ' && msg[2] >= '0' && msg[2] <= '9'
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

func (n *NoOpLogger) Trace(msg string, args ...any) {}
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
