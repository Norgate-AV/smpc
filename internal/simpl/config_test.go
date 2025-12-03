package simpl

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSimplWindowsPath_DefaultPath(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv("SIMPL_WINDOWS_PATH")

	path := GetSimplWindowsPath()
	assert.Equal(t, DefaultSimplWindowsPath, path, "Should return default path when env var not set")
}

func TestGetSimplWindowsPath_EnvVarOverride(t *testing.T) {
	customPath := "D:\\Custom\\Path\\To\\smpwin.exe"

	// Set env var
	os.Setenv("SIMPL_WINDOWS_PATH", customPath)
	defer os.Unsetenv("SIMPL_WINDOWS_PATH")

	path := GetSimplWindowsPath()
	assert.Equal(t, customPath, path, "Should return env var path when set")
}

func TestGetSimplWindowsPath_EmptyEnvVar(t *testing.T) {
	// Set env var to empty string
	os.Setenv("SIMPL_WINDOWS_PATH", "")
	defer os.Unsetenv("SIMPL_WINDOWS_PATH")

	path := GetSimplWindowsPath()
	assert.Equal(t, DefaultSimplWindowsPath, path, "Should return default path when env var is empty")
}
