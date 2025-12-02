//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Norgate-AV/smpc/internal/compiler"
	"github.com/Norgate-AV/smpc/internal/simpl"
	"github.com/Norgate-AV/smpc/internal/windows"
)

const (
	// Timeout for waiting for SIMPL Windows to appear
	windowAppearTimeout = 60 * time.Second
	// Timeout for window to become ready
	windowReadyTimeout = 30 * time.Second
	// UI settling time
	uiSettleTime = 5 * time.Second
)

// TestIntegration_SimpleCompile tests end-to-end compilation of a simple .smw file
func TestIntegration_SimpleCompile(t *testing.T) {
	if !windows.IsElevated() {
		t.Skip("Integration tests require administrator privileges")
	}

	// Get fixture path
	fixturePath := getFixturePath(t, "simple.smw")
	require.FileExists(t, fixturePath, "Fixture file should exist")

	// Run compilation
	result, cleanup := compileFile(t, fixturePath, false)
	defer cleanup()

	// Verify successful compilation
	assert.False(t, result.HasErrors, "Simple file should compile without errors")
	assert.Equal(t, 0, result.Errors, "Should have 0 errors")
	assert.GreaterOrEqual(t, result.Warnings, 0, "Warnings should be non-negative")
	assert.GreaterOrEqual(t, result.Notices, 0, "Notices should be non-negative")
	assert.Greater(t, result.CompileTime, 0.0, "Compile time should be positive")
}

// TestIntegration_CompileWithWarnings tests compilation of a file that produces warnings
func TestIntegration_CompileWithWarnings(t *testing.T) {
	if !windows.IsElevated() {
		t.Skip("Integration tests require administrator privileges")
	}

	fixturePath := getFixturePath(t, "with-warnings.smw")

	// Skip if fixture doesn't exist (optional fixture)
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skip("with-warnings.smw fixture not available")
	}

	result, cleanup := compileFile(t, fixturePath, false)
	defer cleanup()

	// Verify compilation with warnings
	assert.False(t, result.HasErrors, "Should compile successfully despite warnings")
	assert.Equal(t, 0, result.Errors, "Should have 0 errors")
	assert.Greater(t, result.Warnings, 0, "Should have at least 1 warning")
	assert.Len(t, result.WarningMessages, result.Warnings, "Warning count should match messages")
}

// TestIntegration_CompileWithErrors tests compilation of a file that produces errors
func TestIntegration_CompileWithErrors(t *testing.T) {
	if !windows.IsElevated() {
		t.Skip("Integration tests require administrator privileges")
	}

	fixturePath := getFixturePath(t, "with-errors.smw")

	// Skip if fixture doesn't exist (optional fixture)
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skip("with-errors.smw fixture not available")
	}

	result, cleanup := compileFile(t, fixturePath, false)
	defer cleanup()

	// Verify compilation failed with errors
	assert.True(t, result.HasErrors, "Should fail compilation with errors")
	assert.Greater(t, result.Errors, 0, "Should have at least 1 error")
	assert.Len(t, result.ErrorMessages, result.Errors, "Error count should match messages")
}

// TestIntegration_RecompileAll tests the recompile all functionality
func TestIntegration_RecompileAll(t *testing.T) {
	if !windows.IsElevated() {
		t.Skip("Integration tests require administrator privileges")
	}

	fixturePath := getFixturePath(t, "simple.smw")
	require.FileExists(t, fixturePath, "Fixture file should exist")

	// Run compilation with recompile all flag
	result, cleanup := compileFile(t, fixturePath, true)
	defer cleanup()

	// Verify successful compilation
	assert.False(t, result.HasErrors, "Recompile all should succeed")
	assert.Equal(t, 0, result.Errors, "Should have 0 errors")
}

// TestIntegration_NonExistentFile tests behavior with non-existent file
func TestIntegration_NonExistentFile(t *testing.T) {
	if !windows.IsElevated() {
		t.Skip("Integration tests require administrator privileges")
	}

	nonExistentPath := filepath.Join(os.TempDir(), "nonexistent.smw")

	// Ensure file doesn't exist
	os.Remove(nonExistentPath)

	ctx := context.Background()

	// Create real dependencies for integration test
	deps := compiler.NewDefaultDependencies()

	_, err := compiler.CompileWithDeps(compiler.CompileOptions{
		FilePath:     nonExistentPath,
		RecompileAll: false,
		Ctx:          ctx,
	}, deps)

	// Should fail - either during file opening or ShellExecute
	assert.Error(t, err, "Should return error for non-existent file")
}

// Helper Functions

// getFixturePath returns the absolute path to a test fixture
func getFixturePath(t *testing.T, filename string) string {
	// Get the current working directory
	cwd, err := os.Getwd()
	require.NoError(t, err, "Should get current directory")

	// Construct path to fixtures directory
	// Assuming tests run from project root
	fixturePath := filepath.Join(cwd, "test", "integration", "fixtures", filename)

	// If not found, try from test/integration directory
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		fixturePath = filepath.Join(cwd, "fixtures", filename)
	}

	return fixturePath
}

// compileFile performs end-to-end compilation and returns result with cleanup function
func compileFile(t *testing.T, filePath string, recompileAll bool) (*compiler.CompileResult, func()) {
	require.FileExists(t, filePath, "File should exist before compilation")

	// Convert to absolute path
	absPath, err := filepath.Abs(filePath)
	require.NoError(t, err, "Should resolve absolute path")

	// Create context for monitoring
	ctx, cancel := context.WithCancel(context.Background())

	// Start background window monitor
	go simpl.StartMonitoring(ctx)

	// Open file with SIMPL Windows
	t.Logf("Opening SIMPL Windows with file: %s", absPath)
	err = windows.ShellExecute(0, "runas", simpl.SIMPL_WINDOWS_PATH, absPath, "", 1)
	require.NoError(t, err, "Should launch SIMPL Windows")

	// Wait for process to start
	time.Sleep(500 * time.Millisecond)

	// Wait for window to appear
	t.Log("Waiting for SIMPL Windows to appear...")
	hwnd, found := simpl.WaitForAppear(windowAppearTimeout)
	require.True(t, found, "SIMPL Windows should appear within timeout")
	require.NotZero(t, hwnd, "Should have valid window handle")

	// Wait for window to be ready
	t.Log("Waiting for window to be ready...")
	ready := simpl.WaitForReady(hwnd, windowReadyTimeout)
	require.True(t, ready, "SIMPL Windows should be ready within timeout")

	// Allow UI to settle
	time.Sleep(uiSettleTime)

	// Get PID for cleanup
	var simplPid uint32

	// Cleanup function
	cleanup := func() {
		t.Log("Cleaning up SIMPL Windows...")
		cancel()
		if hwnd != 0 {
			simpl.Cleanup(hwnd)
		}
		// Give it time to close
		time.Sleep(1 * time.Second)
	}

	// Run compilation
	t.Log("Starting compilation...")

	// Create real dependencies for integration test
	deps := compiler.NewDefaultDependencies()

	result, err := compiler.CompileWithDeps(compiler.CompileOptions{
		FilePath:     absPath,
		RecompileAll: recompileAll,
		Hwnd:         hwnd,
		Ctx:          ctx,
		SimplPidPtr:  &simplPid,
	}, deps)

	// Note: We don't require NoError here because some tests expect compilation to fail
	if err != nil {
		t.Logf("Compilation returned error: %v", err)
	}

	require.NotNil(t, result, "Should always return a result")

	t.Logf("Compilation complete - Errors: %d, Warnings: %d, Notices: %d, Time: %.2fs",
		result.Errors, result.Warnings, result.Notices, result.CompileTime)

	return result, cleanup
}
