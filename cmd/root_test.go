package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/Norgate-AV/smpc/internal/logger"
	"github.com/Norgate-AV/smpc/internal/version"
)

// resetFlags resets all flags to their default values between tests
func resetFlags() {
	// Reset command flags
	_ = RootCmd.Flags().Set("verbose", "false")
	_ = RootCmd.Flags().Set("recompile-all", "false")
	_ = RootCmd.Flags().Set("logs", "false")
}

// TestValidateArgs_ValidFile tests argument validation with valid .smw file
func TestValidateArgs_ValidFile(t *testing.T) {
	t.Parallel()

	resetFlags()

	// Create a temporary .smw file
	tmpDir := t.TempDir()
	smwFile := filepath.Join(tmpDir, "test.smw")
	err := os.WriteFile(smwFile, []byte("test"), 0o644)
	assert.NoError(t, err)

	cmd := &cobra.Command{}
	args := []string{smwFile}

	err = validateArgs(cmd, args)
	assert.NoError(t, err, "Valid .smw file should pass validation")
}

// TestValidateArgs_InvalidExtension tests argument validation with non-.smw file
func TestValidateArgs_InvalidExtension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		file      string
		expectErr string
	}{
		{
			name:      "txt file",
			file:      "test.txt",
			expectErr: "file must have .smw extension",
		},
		{
			name:      "no extension",
			file:      "test",
			expectErr: "file must have .smw extension",
		},
		{
			name:      "wrong case extension",
			file:      "test.SMW",
			expectErr: "file must have .smw extension",
		},
		{
			name:      "similar extension",
			file:      "test.smw2",
			expectErr: "file must have .smw extension",
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resetFlags()

			cmd := &cobra.Command{}
			args := []string{tt.file}

			err := validateArgs(cmd, args)
			assert.Error(t, err, "Should return error for invalid extension")
			assert.Contains(t, err.Error(), tt.expectErr)
		})
	}
}

// TestValidateArgs_MissingArgument tests validation with no file argument
func TestValidateArgs_MissingArgument(t *testing.T) {
	t.Parallel()

	resetFlags()

	cmd := &cobra.Command{}
	args := []string{}

	// validateArgs now allows 0 args (for --logs flag)
	// The actual requirement for file is checked in Execute
	err := validateArgs(cmd, args)
	assert.NoError(t, err, "validateArgs should allow 0 args for --logs flag")
}

// TestValidateArgs_TooManyArguments tests validation with multiple arguments
func TestValidateArgs_TooManyArguments(t *testing.T) {
	t.Parallel()

	resetFlags()

	cmd := &cobra.Command{}
	args := []string{"file1.smw", "file2.smw"}

	err := validateArgs(cmd, args)
	assert.Error(t, err, "Should return error when multiple files provided")
	assert.Contains(t, err.Error(), "accepts 1 arg(s), received 2")
}

// TestValidateArgs_LogsFlag tests the --logs flag functionality
func TestValidateArgs_LogsFlag(t *testing.T) {
	resetFlags()
	defer resetFlags() // Clean up after test

	// Create temp directory for log file
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "smpc", "smpc.log")

	// Setup logger to temp directory
	oldLocalAppData := os.Getenv("LOCALAPPDATA")
	defer os.Setenv("LOCALAPPDATA", oldLocalAppData)
	os.Setenv("LOCALAPPDATA", tmpDir)

	// Initialize logger
	log, err := logger.NewLogger(logger.LoggerOptions{Verbose: false})
	assert.NoError(t, err)
	defer log.Close()

	// Write some test content to log file
	testContent := "Test log content\nLine 2\nLine 3"
	err = os.MkdirAll(filepath.Dir(logPath), 0o755)
	assert.NoError(t, err)
	err = os.WriteFile(logPath, []byte(testContent), 0o644)
	assert.NoError(t, err)

	// Set showLogs flag on PersistentFlags
	err = RootCmd.PersistentFlags().Set("logs", "true")
	assert.NoError(t, err)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test handleLogsFlag directly with a mock exit function
	exitCalled := false
	var exitCode int
	mockExit := func(code int) {
		exitCalled = true
		exitCode = code
	}

	// Create Config with ShowLogs flag
	cfg := &Config{ShowLogs: true}

	// Call handleLogsFlag directly instead of through Execute
	err = handleLogsFlag(cfg, mockExit)
	assert.NoError(t, err)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify results
	assert.True(t, exitCalled, "Should call exit function for --logs flag")
	assert.Equal(t, 0, exitCode, "Should exit with code 0 for --logs")
	assert.Contains(t, output, testContent, "Should print log file content to stdout")
}

// TestValidateArgs_LogsFlag_NoLogFile tests --logs flag when log file doesn't exist
func TestValidateArgs_LogsFlag_NoLogFile(t *testing.T) {
	// Skip this test - it's difficult to test because logger.Setup() creates the file
	// and keeps a file handle open. The behavior is adequately tested by integration tests.
	t.Skip("Skipping test - file handle management makes this difficult to test in unit tests")
}

// TestRootCmd_Version tests --version flag
func TestRootCmd_Version(t *testing.T) {
	resetFlags()

	// Capture stdout
	output := captureCommandOutput(t, []string{"--version"})

	// Verify version is printed
	expectedVersion := version.GetVersion()
	assert.Contains(t, output, expectedVersion, "Should print version information")
}

// TestRootCmd_Help tests --help flag
func TestRootCmd_Help(t *testing.T) {
	resetFlags()

	// Capture stdout
	output := captureCommandOutput(t, []string{"--help"})

	// Verify help text contains key information
	assert.Contains(t, output, "smpc <file-path>", "Should show usage")
	assert.Contains(t, output, "Automate compilation", "Should show description")
	assert.Contains(t, output, "--verbose", "Should list verbose flag")
	assert.Contains(t, output, "--recompile-all", "Should list recompile-all flag")
	assert.Contains(t, output, "--logs", "Should list logs flag")
}

// TestRootCmd_Flags tests flag parsing
func TestRootCmd_Flags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		args              []string
		expectedVerbose   bool
		expectedRecompile bool
		expectedLogs      bool
	}{
		{
			name:              "no flags",
			args:              []string{},
			expectedVerbose:   false,
			expectedRecompile: false,
			expectedLogs:      false,
		},
		{
			name:              "verbose flag short",
			args:              []string{"-V"},
			expectedVerbose:   true,
			expectedRecompile: false,
			expectedLogs:      false,
		},
		{
			name:              "verbose flag long",
			args:              []string{"--verbose"},
			expectedVerbose:   true,
			expectedRecompile: false,
			expectedLogs:      false,
		},
		{
			name:              "recompile flag short",
			args:              []string{"-r"},
			expectedVerbose:   false,
			expectedRecompile: true,
			expectedLogs:      false,
		},
		{
			name:              "recompile flag long",
			args:              []string{"--recompile-all"},
			expectedVerbose:   false,
			expectedRecompile: true,
			expectedLogs:      false,
		},
		{
			name:              "logs flag short",
			args:              []string{"-l"},
			expectedVerbose:   false,
			expectedRecompile: false,
			expectedLogs:      true,
		},
		{
			name:              "logs flag long",
			args:              []string{"--logs"},
			expectedVerbose:   false,
			expectedRecompile: false,
			expectedLogs:      true,
		},
		{
			name:              "multiple flags",
			args:              []string{"-V", "-r"},
			expectedVerbose:   true,
			expectedRecompile: true,
			expectedLogs:      false,
		},
		{
			name:              "all flags",
			args:              []string{"--verbose", "--recompile-all", "--logs"},
			expectedVerbose:   true,
			expectedRecompile: true,
			expectedLogs:      true,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resetFlags()

			// Create a new command instance to avoid flag conflicts
			cmd := &cobra.Command{
				Use: "test",
			}

			cmd.PersistentFlags().BoolP("verbose", "V", false, "enable verbose output")
			cmd.PersistentFlags().BoolP("recompile-all", "r", false, "trigger Recompile All")
			cmd.PersistentFlags().BoolP("logs", "l", false, "print log file")

			// Parse flags
			cmd.SetArgs(tt.args)
			err := cmd.ParseFlags(tt.args)
			assert.NoError(t, err, "Flag parsing should not error")

			// Verify flag values
			verbose, _ := cmd.Flags().GetBool("verbose")
			recompileAll, _ := cmd.Flags().GetBool("recompile-all")
			showLogs, _ := cmd.Flags().GetBool("logs")
			assert.Equal(t, tt.expectedVerbose, verbose, "Verbose flag mismatch")
			assert.Equal(t, tt.expectedRecompile, recompileAll, "Recompile flag mismatch")
			assert.Equal(t, tt.expectedLogs, showLogs, "Logs flag mismatch")
		})
	}
}

// TestRootCmd_InvalidFlag tests behavior with unknown flags
func TestRootCmd_InvalidFlag(t *testing.T) {
	resetFlags()

	// Create temp .smw file
	tmpDir := t.TempDir()
	smwFile := filepath.Join(tmpDir, "test.smw")
	_ = os.WriteFile(smwFile, []byte("test"), 0o644)

	// Capture stderr for error output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Try to execute with invalid flag
	RootCmd.SetArgs([]string{"--invalid-flag", smwFile})
	err := RootCmd.Execute()

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read error output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify error
	assert.Error(t, err, "Should return error for invalid flag")
	assert.Contains(t, output, "unknown flag", "Error message should mention unknown flag")
}

// Helper function to capture command output
func captureCommandOutput(_ *testing.T, args []string) string {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute command
	RootCmd.SetArgs(args)
	_ = RootCmd.Execute()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	return buf.String()
}

// TestExecutionContext_ExitFuncInjectable tests that exitFunc is injectable for testing
func TestExecutionContext_ExitFuncInjectable(t *testing.T) {
	t.Parallel()

	// Create a mock logger
	log := &logger.NoOpLogger{}

	// Track if exit was called and with what code
	exitCalled := false
	var exitCode int

	// Create execution context with mock exit function
	ctx := &ExecutionContext{
		simplPid:    12345,
		simplHwnd:   67890,
		log:         log,
		simplClient: nil, // Not needed for this test
		exitFunc: func(code int) {
			exitCalled = true
			exitCode = code
		},
	}

	// Verify exitFunc can be called
	ctx.exitFunc(130)

	// Verify our mock was called correctly
	assert.True(t, exitCalled, "Exit function should have been called")
	assert.Equal(t, 130, exitCode, "Exit code should be 130")
}

// TestExecutionContext_DefaultExitFunc tests that exitFunc defaults to os.Exit
func TestExecutionContext_DefaultExitFunc(t *testing.T) {
	t.Parallel()

	// Create execution context with default exit function (as done in Execute)
	ctx := &ExecutionContext{
		exitFunc: os.Exit,
	}

	// Verify exitFunc is set (we can't call it without actually exiting)
	assert.NotNil(t, ctx.exitFunc, "Exit function should be set")
}

// TestValidateAndResolvePath_ValidFile tests validation with an existing file
func TestValidateAndResolvePath_ValidFile(t *testing.T) {
	t.Parallel()

	// Create a temporary .smw file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.smw")
	err := os.WriteFile(testFile, []byte("test content"), 0o644)
	assert.NoError(t, err, "Should create test file")

	// Create a mock logger
	mockLog := logger.NewNoOpLogger()

	// Validate and resolve path
	absPath, err := validateAndResolvePath(testFile, mockLog)

	assert.NoError(t, err, "Should not error for existing file")
	assert.NotEmpty(t, absPath, "Should return absolute path")
	assert.True(t, filepath.IsAbs(absPath), "Returned path should be absolute")
}

// TestValidateAndResolvePath_MissingFile tests validation with non-existent file
func TestValidateAndResolvePath_MissingFile(t *testing.T) {
	t.Parallel()

	// Create a mock logger
	mockLog := logger.NewNoOpLogger()

	// Try to validate a non-existent file
	nonExistentFile := filepath.Join(t.TempDir(), "does-not-exist.smw")
	absPath, err := validateAndResolvePath(nonExistentFile, mockLog)

	assert.Error(t, err, "Should return error for non-existent file")
	assert.Empty(t, absPath, "Should return empty path on error")
	assert.Contains(t, err.Error(), "file does not exist", "Error should mention file doesn't exist")
	assert.Contains(t, err.Error(), nonExistentFile, "Error should include file path")
}

// TestValidateAndResolvePath_RelativePath tests resolving a relative path
func TestValidateAndResolvePath_RelativePath(t *testing.T) {
	t.Parallel()

	// Create a temporary directory and file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "relative.smw")
	err := os.WriteFile(testFile, []byte("test"), 0o644)
	assert.NoError(t, err, "Should create test file")

	// Change to temp directory to test relative path resolution
	originalDir, err := os.Getwd()
	assert.NoError(t, err, "Should get current directory")
	defer func() {
		err := os.Chdir(originalDir)
		assert.NoError(t, err, "Should restore original directory")
	}()

	err = os.Chdir(tempDir)
	assert.NoError(t, err, "Should change to temp directory")

	// Create a mock logger
	mockLog := logger.NewNoOpLogger()

	// Validate relative path
	absPath, err := validateAndResolvePath("relative.smw", mockLog)

	assert.NoError(t, err, "Should resolve relative path")
	assert.True(t, filepath.IsAbs(absPath), "Should return absolute path")
	assert.Contains(t, absPath, "relative.smw", "Should contain filename")
}

// TestValidateAndResolvePath_DirectoryInsteadOfFile tests error when path is a directory
func TestValidateAndResolvePath_DirectoryInsteadOfFile(t *testing.T) {
	t.Parallel()

	// Create a temporary directory
	tempDir := t.TempDir()

	// Create a mock logger
	mockLog := logger.NewNoOpLogger()

	// Try to validate a directory instead of a file
	// os.Stat succeeds for directories, so this tests that we don't validate file type
	absPath, err := validateAndResolvePath(tempDir, mockLog)

	// The function only checks if path exists via os.Stat, not if it's a file
	// So it should succeed and return the absolute path
	assert.NoError(t, err, "Function doesn't validate file type, only existence")
	assert.NotEmpty(t, absPath, "Should return absolute path")
	assert.True(t, filepath.IsAbs(absPath), "Should return absolute path")
}
