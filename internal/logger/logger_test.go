package logger_test

import (
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Norgate-AV/smpc/internal/logger"
)

func TestNewLogger_DefaultOptions(t *testing.T) {
	// Set custom LOCALAPPDATA for testing
	tmpDir := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmpDir)

	log, err := logger.NewLogger(logger.LoggerOptions{})
	require.NoError(t, err)
	defer log.Close()

	assert.NotNil(t, log)

	logPath := log.GetLogPath()
	assert.NotEmpty(t, logPath)
	assert.Contains(t, logPath, "smpc.log")
	assert.True(t, filepath.IsAbs(logPath), "Log path should be absolute")
}

func TestNewLogger_CreatesLogDirectory(t *testing.T) {
	// Set custom LOCALAPPDATA for testing
	tmpDir := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmpDir)

	log, err := logger.NewLogger(logger.LoggerOptions{})
	require.NoError(t, err)
	defer log.Close()

	expectedDir := filepath.Join(tmpDir, "smpc")
	assert.DirExists(t, expectedDir)
}

func TestNewLogger_CustomLogDir(t *testing.T) {
	tmpDir := t.TempDir()

	log, err := logger.NewLogger(logger.LoggerOptions{
		LogDir: tmpDir,
	})
	require.NoError(t, err)
	defer log.Close()

	logPath := log.GetLogPath()
	expectedPath := filepath.Join(tmpDir, "smpc.log")
	assert.Equal(t, expectedPath, logPath)
}

func TestNewLogger_Verbose(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmpDir)

	// Test with verbose=true
	log, err := logger.NewLogger(logger.LoggerOptions{
		Verbose: true,
	})
	require.NoError(t, err)
	defer log.Close()

	assert.NotNil(t, log)
}

func TestNewLogger_NonVerbose(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmpDir)

	// Test with verbose=false
	log, err := logger.NewLogger(logger.LoggerOptions{
		Verbose: false,
	})
	require.NoError(t, err)
	defer log.Close()

	assert.NotNil(t, log)
}

func TestNewLogger_FallbackToUserProfile(t *testing.T) {
	// Clear LOCALAPPDATA and set USERPROFILE
	tmpDir := t.TempDir()
	t.Setenv("LOCALAPPDATA", "")
	t.Setenv("USERPROFILE", tmpDir)

	log, err := logger.NewLogger(logger.LoggerOptions{})
	require.NoError(t, err)
	defer log.Close()

	logPath := log.GetLogPath()
	assert.NotEmpty(t, logPath)

	// Should use USERPROFILE/AppData/Local/smpc/smpc.log
	expectedPath := filepath.Join(tmpDir, "AppData", "Local", "smpc", "smpc.log")
	assert.Equal(t, expectedPath, logPath)
}

func TestNewLogger_WithCompression(t *testing.T) {
	tmpDir := t.TempDir()

	log, err := logger.NewLogger(logger.LoggerOptions{
		LogDir:   tmpDir,
		Compress: true,
	})
	require.NoError(t, err)
	defer log.Close()

	assert.NotNil(t, log)
}

func TestLogger_Close(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmpDir)

	log, err := logger.NewLogger(logger.LoggerOptions{})
	require.NoError(t, err)

	// Close should not panic
	assert.NotPanics(t, func() {
		log.Close()
	})
}

func TestLogger_LogMethods(t *testing.T) {
	tmpDir := t.TempDir()

	log, err := logger.NewLogger(logger.LoggerOptions{
		LogDir: tmpDir,
	})
	require.NoError(t, err)
	defer log.Close()

	// Test that logging methods don't panic
	assert.NotPanics(t, func() {
		log.Debug("debug message", slog.String("key", "value"))
		log.Info("info message", slog.Int("count", 42))
		log.Warn("warn message", slog.Bool("flag", true))
		log.Error("error message", slog.Any("error", assert.AnError))
	})
}

func TestNoOpLogger(t *testing.T) {
	log := logger.NewNoOpLogger()
	assert.NotNil(t, log)

	// NoOp logger should not panic on any operations
	assert.NotPanics(t, func() {
		log.Debug("test")
		log.Info("test")
		log.Warn("test")
		log.Error("test")
	})
}
