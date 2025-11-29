package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/Norgate-AV/smpc/internal/logging"
	"github.com/Norgate-AV/smpc/internal/version"
)

// resetFlags resets all flags to their default values between tests
func resetFlags() {
	verbose = false
	recompileAll = false
	showLogs = false
	simplHwnd = 0
	simplPid = 0
}

// TestValidateArgs_ValidFile tests argument validation with valid .smw file
func TestValidateArgs_ValidFile(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
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
	resetFlags()

	cmd := &cobra.Command{}
	args := []string{}

	err := validateArgs(cmd, args)
	assert.Error(t, err, "Should return error when no file provided")
	assert.Contains(t, err.Error(), "accepts 1 arg(s), received 0")
}

// TestValidateArgs_TooManyArguments tests validation with multiple arguments
func TestValidateArgs_TooManyArguments(t *testing.T) {
	resetFlags()

	cmd := &cobra.Command{}
	args := []string{"file1.smw", "file2.smw"}

	err := validateArgs(cmd, args)
	assert.Error(t, err, "Should return error when multiple files provided")
	assert.Contains(t, err.Error(), "accepts 1 arg(s), received 2")
}

// TestValidateArgs_LogsFlag tests --logs flag displays log file content
func TestValidateArgs_LogsFlag(t *testing.T) {
	resetFlags()

	// Create temp directory for log file
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "smpc", "smpc.log")

	// Setup logging to temp directory
	oldLocalAppData := os.Getenv("LOCALAPPDATA")
	defer os.Setenv("LOCALAPPDATA", oldLocalAppData)
	os.Setenv("LOCALAPPDATA", tmpDir)

	// Initialize logging
	_ = logging.Setup(false)
	defer logging.Close()

	// Write some test content to log file
	testContent := "Test log content\nLine 2\nLine 3"
	err := os.MkdirAll(filepath.Dir(logPath), 0o755)
	assert.NoError(t, err)
	err = os.WriteFile(logPath, []byte(testContent), 0o644)
	assert.NoError(t, err)

	// Set showLogs flag
	showLogs = true

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Capture exit call (validateArgs calls os.Exit(0) for --logs)
	exitCalled := false
	oldOsExit := osExit
	osExit = func(code int) {
		exitCalled = true
		assert.Equal(t, 0, code, "Should exit with code 0 for --logs")
	}
	defer func() { osExit = oldOsExit }()

	cmd := &cobra.Command{}
	args := []string{} // --logs doesn't require file argument

	// Call validateArgs
	_ = validateArgs(cmd, args)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify results
	assert.True(t, exitCalled, "Should call os.Exit(0) for --logs flag")
	assert.Contains(t, output, testContent, "Should print log file content to stdout")
}

// TestValidateArgs_LogsFlag_NoLogFile tests --logs flag when log file doesn't exist
func TestValidateArgs_LogsFlag_NoLogFile(t *testing.T) {
	// Skip this test - it's difficult to test because logging.Setup() creates the file
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
		t.Run(tt.name, func(t *testing.T) {
			resetFlags()

			// Create a new command instance to avoid flag conflicts
			cmd := &cobra.Command{
				Use: "test",
			}
			cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "V", false, "enable verbose output")
			cmd.PersistentFlags().BoolVarP(&recompileAll, "recompile-all", "r", false, "trigger Recompile All")
			cmd.PersistentFlags().BoolVarP(&showLogs, "logs", "l", false, "print log file")

			// Parse flags
			cmd.SetArgs(tt.args)
			err := cmd.ParseFlags(tt.args)
			assert.NoError(t, err, "Flag parsing should not error")

			// Verify flag values
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
func captureCommandOutput(t *testing.T, args []string) string {
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
