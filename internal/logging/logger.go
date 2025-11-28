package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

var (
	Logger           *slog.Logger
	lumberjackLogger *lumberjack.Logger
	logPath          string
)

// ConsoleHandler is a custom slog handler for clean console output
type ConsoleHandler struct {
	verbose bool
}

func (h *ConsoleHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *ConsoleHandler) Handle(_ context.Context, r slog.Record) error {
	// Only show certain messages in console based on level
	switch r.Level {
	case slog.LevelError:
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", r.Message)
	case slog.LevelWarn:
		fmt.Fprintf(os.Stderr, "WARNING: %s\n", r.Message)
	case slog.LevelInfo:
		// Regular user-facing messages - no prefix
		fmt.Println(r.Message)
	case slog.LevelDebug:
		// Only show debug messages if verbose is enabled
		if h.verbose {
			fmt.Printf("[DEBUG] %s\n", r.Message)
		}
	}

	return nil
}

func (h *ConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *ConsoleHandler) WithGroup(name string) slog.Handler {
	return h
}

// Setup initializes the logging system with both file and console handlers
func Setup(verbose bool) error {
	// Set up log file in %LOCALAPPDATA%\smpc\smpc.log
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
	}

	logDir := filepath.Join(localAppData, "smpc")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return fmt.Errorf("could not create log directory: %w", err)
	}

	logPath = filepath.Join(logDir, "smpc.log")

	// Set up lumberjack for log rotation
	lumberjackLogger = &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    10,   // megabytes - rotate after 10MB
		MaxBackups: 3,    // keep 3 old log files
		MaxAge:     28,   // days - delete logs older than 28 days
		Compress:   true, // compress rotated files with gzip
	}

	// Create a multi-writer that writes to both file and console
	fileHandler := slog.NewTextHandler(lumberjackLogger, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Log everything to file
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Add microsecond precision to file logs
			if a.Key == slog.TimeKey {
				return slog.String(slog.TimeKey, a.Value.Time().Format("2006/01/02 15:04:05.000000"))
			}

			return a
		},
	})

	consoleHandler := &ConsoleHandler{verbose: verbose}

	// Create a handler that writes to both
	multiHandler := &MultiHandler{
		handlers: []slog.Handler{fileHandler, consoleHandler},
	}

	Logger = slog.New(multiHandler)

	// Set as the global default logger so slog.Info(), slog.Debug() etc. work everywhere
	slog.SetDefault(Logger)

	// Show log file location to user
	fmt.Printf("Debug log: %s\n", logPath)
	slog.Info("=== SMPC started ===")

	return nil
}

// GetLogPath returns the path to the current log file
func GetLogPath() string {
	return logPath
}

// Close closes the log file and flushes any buffered data
func Close() {
	if lumberjackLogger != nil {
		lumberjackLogger.Close()
	}
}

// MultiHandler sends logs to multiple handlers
type MultiHandler struct {
	handlers []slog.Handler
}

func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}

	return false
}

func (h *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if err := handler.Handle(ctx, r); err != nil {
			return err
		}
	}

	return nil
}

func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))

	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}

	return &MultiHandler{handlers: newHandlers}
}

func (h *MultiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))

	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}

	return &MultiHandler{handlers: newHandlers}
}
