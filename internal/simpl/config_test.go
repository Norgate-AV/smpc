package simpl

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSimplWindowsPath_DefaultPath(t *testing.T) {
	// Cannot use t.Parallel() - modifies environment variables

	// Ensure env var is not set
	os.Unsetenv("SIMPL_WINDOWS_PATH")

	path := GetSimplWindowsPath()
	assert.Equal(t, DefaultSimplWindowsPath, path, "Should return default path when env var not set")
}

func TestGetSimplWindowsPath_EnvVarOverride(t *testing.T) {
	// Cannot use t.Parallel() - modifies environment variables

	customPath := "D:\\Custom\\Path\\To\\smpwin.exe"

	// Set env var
	os.Setenv("SIMPL_WINDOWS_PATH", customPath)
	defer os.Unsetenv("SIMPL_WINDOWS_PATH")

	path := GetSimplWindowsPath()
	assert.Equal(t, customPath, path, "Should return env var path when set")
}

func TestGetSimplWindowsPath_EmptyEnvVar(t *testing.T) {
	// Cannot use t.Parallel() - modifies environment variables

	// Set env var to empty string
	os.Setenv("SIMPL_WINDOWS_PATH", "")
	defer os.Unsetenv("SIMPL_WINDOWS_PATH")

	path := GetSimplWindowsPath()
	assert.Equal(t, DefaultSimplWindowsPath, path, "Should return default path when env var is empty")
}

func TestValidateSimplWindowsInstallation_DefaultPathNotFound(t *testing.T) {
	// Cannot use t.Parallel() - modifies environment variables

	// Most test environments won't have SIMPL Windows installed
	os.Unsetenv("SIMPL_WINDOWS_PATH")

	err := ValidateSimplWindowsInstallation()
	// On systems without SIMPL Windows, we expect an error
	if err != nil {
		assert.Contains(t, err.Error(), "SIMPL Windows not found at default path")
		assert.Contains(t, err.Error(), DefaultSimplWindowsPath)
	}
	// Note: If SIMPL Windows IS installed, err will be nil, which is also valid
}

func TestValidateSimplWindowsInstallation_CustomPathNotFound(t *testing.T) {
	// Cannot use t.Parallel() - modifies environment variables

	nonExistentPath := "Z:\\NonExistent\\Path\\smpwin.exe"

	os.Setenv("SIMPL_WINDOWS_PATH", nonExistentPath)
	defer os.Unsetenv("SIMPL_WINDOWS_PATH")

	err := ValidateSimplWindowsInstallation()

	assert.Error(t, err, "Should return error when custom path does not exist")
	assert.Contains(t, err.Error(), "SIMPL Windows not found at custom path")
	assert.Contains(t, err.Error(), nonExistentPath)
	assert.Contains(t, err.Error(), "SIMPL_WINDOWS_PATH")
}
