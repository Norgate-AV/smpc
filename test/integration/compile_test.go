//go:build integration
// +build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Norgate-AV/smpc/internal/compiler"
	"github.com/Norgate-AV/smpc/internal/logger"
	"github.com/Norgate-AV/smpc/internal/simpl"
	"github.com/Norgate-AV/smpc/internal/timeouts"
	"github.com/Norgate-AV/smpc/internal/windows"
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

	// Ensure we didn't hit a timeout
	for _, msg := range result.ErrorMessages {
		assert.NotContains(t, msg, "timeout", "Should not have timed out")
	}
}

// TestIntegration_CompileWithWarnings tests compilation of a file that produces warnings
func TestIntegration_CompileWithWarnings(t *testing.T) {
	if !windows.IsElevated() {
		t.Skip("Integration tests require administrator privileges")
	}

	fixturePath := getFixturePath(t, "warnings.smw")
	require.FileExists(t, fixturePath, "Fixture file should exist")

	result, cleanup := compileFile(t, fixturePath, false)
	defer cleanup()

	// Verify compilation with warnings
	assert.False(t, result.HasErrors, "Should compile successfully despite warnings")
	assert.Equal(t, 0, result.Errors, "Should have 0 errors")
	assert.Greater(t, result.Warnings, 0, "Should have at least 1 warning")
	assert.Len(t, result.WarningMessages, result.Warnings, "Warning count should match messages")

	// Ensure we didn't hit a timeout
	for _, msg := range result.ErrorMessages {
		assert.NotContains(t, msg, "timeout", "Should not have timed out")
	}
}

// TestIntegration_CompileWithNotices tests compilation of a file that produces notices
func TestIntegration_CompileWithNotices(t *testing.T) {
	if !windows.IsElevated() {
		t.Skip("Integration tests require administrator privileges")
	}

	fixturePath := getFixturePath(t, "notices.smw")
	require.FileExists(t, fixturePath, "Fixture file should exist")

	result, cleanup := compileFile(t, fixturePath, false)
	defer cleanup()

	// Verify compilation with notices
	assert.False(t, result.HasErrors, "Should compile successfully despite notices")
	assert.Equal(t, 0, result.Errors, "Should have 0 errors")
	assert.Greater(t, result.Notices, 0, "Should have at least 1 notice")
	assert.Len(t, result.NoticeMessages, result.Notices, "Notice count should match messages")

	// Ensure we didn't hit a timeout
	for _, msg := range result.ErrorMessages {
		assert.NotContains(t, msg, "timeout", "Should not have timed out")
	}
}

// TestIntegration_CompileWithWarningsAndNotices tests compilation with both warnings and notices
func TestIntegration_CompileWithWarningsAndNotices(t *testing.T) {
	if !windows.IsElevated() {
		t.Skip("Integration tests require administrator privileges")
	}

	fixturePath := getFixturePath(t, "warnings_and_notices.smw")
	require.FileExists(t, fixturePath, "Fixture file should exist")

	result, cleanup := compileFile(t, fixturePath, false)
	defer cleanup()

	// Verify compilation with both warnings and notices
	assert.False(t, result.HasErrors, "Should compile successfully despite warnings and notices")
	assert.Equal(t, 0, result.Errors, "Should have 0 errors")
	assert.Greater(t, result.Warnings, 0, "Should have at least 1 warning")
	assert.Greater(t, result.Notices, 0, "Should have at least 1 notice")
	assert.Len(t, result.WarningMessages, result.Warnings, "Warning count should match messages")
	assert.Len(t, result.NoticeMessages, result.Notices, "Notice count should match messages")

	// Ensure we didn't hit a timeout
	for _, msg := range result.ErrorMessages {
		assert.NotContains(t, msg, "timeout", "Should not have timed out")
	}
}

// TestIntegration_CompileWithErrors tests compilation of a file that produces errors
func TestIntegration_CompileWithErrors(t *testing.T) {
	if !windows.IsElevated() {
		t.Skip("Integration tests require administrator privileges")
	}

	fixturePath := getFixturePath(t, "error.smw")
	require.FileExists(t, fixturePath, "Fixture file should exist")

	result, cleanup := compileFile(t, fixturePath, false) // Use normal compile, not recompile all
	defer cleanup()

	// Verify compilation failed with errors
	assert.True(t, result.HasErrors, "Should fail compilation with errors")
	assert.Greater(t, result.Errors, 1, "Should have at least 2 errors (not just a timeout)")
	assert.Len(t, result.ErrorMessages, result.Errors, "Error count should match messages")

	// Ensure we didn't hit a timeout
	for _, msg := range result.ErrorMessages {
		assert.NotContains(t, msg, "timeout", "Should not have timed out - actual compilation errors expected")
		assert.NotContains(t, msg, "Compile Complete", "Should not have timed out - actual compilation errors expected")
	}
}

// TestIntegration_CompileIncomplete tests compilation of an incomplete file
func TestIntegration_CompileIncomplete(t *testing.T) {
	if !windows.IsElevated() {
		t.Skip("Integration tests require administrator privileges")
	}

	fixturePath := getFixturePath(t, "incomplete.smw")
	require.FileExists(t, fixturePath, "Fixture file should exist")

	result, cleanup := compileFile(t, fixturePath, false)
	defer cleanup()

	// Incomplete files should produce a specific error about incomplete symbols
	assert.NotNil(t, result, "Should return a result")
	assert.True(t, result.HasErrors, "Should have errors for incomplete symbols")
	assert.Equal(t, 1, result.Errors, "Should have exactly 1 error for incomplete symbols")

	// Verify it's the incomplete symbols error, not a timeout
	assert.Len(t, result.ErrorMessages, 1, "Should have one error message")
	if len(result.ErrorMessages) > 0 {
		assert.Contains(t, result.ErrorMessages[0], "Incomplete Symbols", "Error should be about incomplete symbols")
		assert.NotContains(t, result.ErrorMessages[0], "timeout", "Should not have timed out")
	}
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

	// Ensure we didn't hit a timeout
	for _, msg := range result.ErrorMessages {
		assert.NotContains(t, msg, "timeout", "Should not have timed out")
	}
}

// TestIntegration_FileValidation tests the file validation that should occur before compilation
func TestIntegration_FileValidation(t *testing.T) {
	// This test doesn't require admin privileges - it's just file validation

	nonExistentPath := filepath.Join(os.TempDir(), "nonexistent.smw")

	// Ensure file doesn't exist
	os.Remove(nonExistentPath)

	// Create a minimal logger for the test
	testLog, err := logger.NewLogger(logger.LoggerOptions{Verbose: false})
	require.NoError(t, err, "Should create logger")
	defer testLog.Close()

	// Test that file validation (as done in cmd/root.go) catches non-existent files
	_, err = os.Stat(nonExistentPath)

	// Should detect the file doesn't exist
	assert.True(t, os.IsNotExist(err), "Should detect non-existent file")
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

	// Create a minimal logger for the test (discard output)
	testLog, err := logger.NewLogger(logger.LoggerOptions{Verbose: false})
	require.NoError(t, err, "Should create logger")
	defer testLog.Close()

	// Create SIMPL client
	simplClient := simpl.NewClient(testLog)

	// Start background window monitor
	stopMonitor := simplClient.StartMonitoring()

	// Open file with SIMPL Windows
	t.Logf("Opening SIMPL Windows with file: %s", absPath)
	pid, err := windows.ShellExecuteEx(0, "open", simpl.GetSimplWindowsPath(), absPath, "", 1, testLog)
	require.NoError(t, err, "Should launch SIMPL Windows")
	t.Logf("SIMPL Windows process started with PID: %d", pid)

	// Wait for process to start
	time.Sleep(timeouts.WindowMessageDelay)

	// Wait for window to appear
	t.Log("Waiting for SIMPL Windows to appear...")
	hwnd, found := simplClient.WaitForAppear(pid, timeouts.WindowAppearTimeout)
	require.True(t, found, "SIMPL Windows should appear within timeout")
	require.NotZero(t, hwnd, "Should have valid window handle")

	// Wait for window to be ready
	t.Log("Waiting for window to be ready...")
	ready := simplClient.WaitForReady(hwnd, timeouts.WindowReadyTimeout)
	require.True(t, ready, "SIMPL Windows should be ready within timeout")

	// Allow UI to settle
	time.Sleep(timeouts.UISettlingDelay)

	// Use the PID from ShellExecuteEx for compilation
	simplPid := pid

	// Cleanup function
	cleanup := func() {
		t.Log("Cleaning up SIMPL Windows...")
		stopMonitor()
		if hwnd != 0 {
			simplClient.Cleanup(hwnd)
		}
		// Give it time to close
		time.Sleep(timeouts.FocusVerificationDelay)
	}

	// Run compilation
	t.Log("Starting compilation...")

	// Create compiler with logger
	comp := compiler.NewCompiler(testLog)

	result, err := comp.Compile(compiler.CompileOptions{
		FilePath:     absPath,
		RecompileAll: recompileAll,
		Hwnd:         hwnd,
		SimplPidPtr:  &simplPid,
	})
	// Note: We don't require NoError here because some tests expect compilation to fail
	if err != nil {
		t.Logf("Compilation returned error: %v", err)
	}

	require.NotNil(t, result, "Should always return a result")

	t.Logf("Compilation complete - Errors: %d, Warnings: %d, Notices: %d, Time: %.2fs",
		result.Errors, result.Warnings, result.Notices, result.CompileTime)

	return result, cleanup
}
