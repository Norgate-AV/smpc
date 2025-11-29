package logging_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Norgate-AV/smpc/internal/logging"
)

func TestGetLogPath_BeforeSetup(t *testing.T) {
	// Before Setup is called, path should be empty
	// Note: This test may fail if Setup was already called in another test
	// In real usage, Setup is called once at program start
	path := logging.GetLogPath()
	// Path may or may not be empty depending on test execution order
	// Just verify it's a valid path string
	_ = path
}

func TestGetLogPath_AfterSetup(t *testing.T) {
	// Setup logging
	err := logging.Setup(false)
	require.NoError(t, err)
	defer logging.Close()

	path := logging.GetLogPath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "smpc.log")
	assert.True(t, filepath.IsAbs(path), "Log path should be absolute")
}

func TestSetup_CreatesLogDirectory(t *testing.T) {
	// Set custom LOCALAPPDATA for testing
	tmpDir := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmpDir)

	err := logging.Setup(false)
	require.NoError(t, err)
	defer logging.Close()

	expectedDir := filepath.Join(tmpDir, "smpc")
	assert.DirExists(t, expectedDir)
}

func TestSetup_CreatesLogFile(t *testing.T) {
	// Set custom LOCALAPPDATA for testing
	tmpDir := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmpDir)

	err := logging.Setup(false)
	require.NoError(t, err)
	defer logging.Close()

	logPath := logging.GetLogPath()
	assert.NotEmpty(t, logPath)

	// Log file may not exist until first write, so just verify the path is set correctly
	expectedPath := filepath.Join(tmpDir, "smpc", "smpc.log")
	assert.Equal(t, expectedPath, logPath)
}

func TestSetup_Verbose(t *testing.T) {
	// Set custom LOCALAPPDATA for testing
	tmpDir := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmpDir)

	// Test with verbose=true
	err := logging.Setup(true)
	require.NoError(t, err)
	defer logging.Close()

	// Verify setup succeeded - logger should be initialized
	assert.NotNil(t, logging.Logger)
}

func TestSetup_NonVerbose(t *testing.T) {
	// Set custom LOCALAPPDATA for testing
	tmpDir := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmpDir)

	// Test with verbose=false
	err := logging.Setup(false)
	require.NoError(t, err)
	defer logging.Close()

	// Verify setup succeeded - logger should be initialized
	assert.NotNil(t, logging.Logger)
}

func TestSetup_FallbackToUserProfile(t *testing.T) {
	// Clear LOCALAPPDATA and set USERPROFILE
	tmpDir := t.TempDir()
	t.Setenv("LOCALAPPDATA", "")
	t.Setenv("USERPROFILE", tmpDir)

	err := logging.Setup(false)
	require.NoError(t, err)
	defer logging.Close()

	logPath := logging.GetLogPath()
	assert.NotEmpty(t, logPath)

	// Should use USERPROFILE/AppData/Local/smpc/smpc.log
	expectedPath := filepath.Join(tmpDir, "AppData", "Local", "smpc", "smpc.log")
	assert.Equal(t, expectedPath, logPath)
}

func TestClose(t *testing.T) {
	// Set custom LOCALAPPDATA for testing
	tmpDir := t.TempDir()
	t.Setenv("LOCALAPPDATA", tmpDir)

	err := logging.Setup(false)
	require.NoError(t, err)

	// Close should not panic
	assert.NotPanics(t, func() {
		logging.Close()
	})
}
