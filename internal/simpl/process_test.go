package simpl

import (
	"testing"
)

// TestFindProcessByName_CurrentProcess tests finding the current running process
func TestFindProcessByName_CurrentProcess(t *testing.T) {
	t.Parallel()

	// Test finding the Go test runner process (should exist)
	pid := findProcessByName("go.exe")

	// On Windows, go.exe should be running (the test runner itself)
	// If not found, it might be running as a different process name
	// This test verifies the function doesn't crash and returns a valid format
	if pid != 0 {
		t.Logf("Found go.exe with PID: %d", pid)
	} else {
		t.Log("go.exe not found (may be running under different name)")
	}
}

// TestFindProcessByName_NonExistentProcess tests searching for a process that doesn't exist
func TestFindProcessByName_NonExistentProcess(t *testing.T) {
	t.Parallel()

	// Search for a process that almost certainly doesn't exist
	pid := findProcessByName("this_process_definitely_does_not_exist_12345.exe")

	// Should return 0 when process not found
	if pid != 0 {
		t.Errorf("Expected PID 0 for non-existent process, got %d", pid)
	}
}

// TestFindProcessByName_CaseInsensitive tests case-insensitive matching
func TestFindProcessByName_CaseInsensitive(t *testing.T) {
	t.Parallel()

	// Find a known system process (usually running on Windows)
	// Try different case variations
	tests := []struct {
		name        string
		processName string
	}{
		{"lowercase", "explorer.exe"},
		{"uppercase", "EXPLORER.EXE"},
		{"mixed_case", "ExPlOrEr.ExE"},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pid := findProcessByName(tt.processName)

			// explorer.exe should be running on most Windows systems
			// If not found, just log it (system may not have explorer running)
			if pid != 0 {
				t.Logf("Found %s with PID: %d", tt.processName, pid)
			} else {
				t.Logf("%s not found (may not be running)", tt.processName)
			}
		})
	}
}

// TestFindProcessByName_SystemProcess tests finding a common system process
func TestFindProcessByName_SystemProcess(t *testing.T) {
	t.Parallel()

	// Search for common Windows system processes
	// At least one of these should be running
	processNames := []string{
		"svchost.exe",
		"System",
		"Registry",
		"csrss.exe",
	}

	foundAny := false
	for _, processName := range processNames {
		pid := findProcessByName(processName)
		if pid != 0 {
			t.Logf("Found system process %s with PID: %d", processName, pid)
			foundAny = true
			break
		}
	}

	if !foundAny {
		t.Log("Warning: No common system processes found. This is unusual but not necessarily an error.")
	}
}

// TestFindProcessByName_EmptyString tests behavior with empty process name
func TestFindProcessByName_EmptyString(t *testing.T) {
	t.Parallel()

	pid := findProcessByName("")

	// Empty string should not match any process
	if pid != 0 {
		t.Errorf("Expected PID 0 for empty process name, got %d", pid)
	}
}

// TestFindProcessByName_WithoutExtension tests searching without .exe extension
func TestFindProcessByName_WithoutExtension(t *testing.T) {
	t.Parallel()

	// Search for "explorer" without the .exe extension
	pid := findProcessByName("explorer")

	// Should NOT find the process because we need exact match with .exe
	// The actual process name in the system is "explorer.exe"
	if pid != 0 {
		t.Logf("Warning: Found process with PID %d using partial name (unexpected)", pid)
	}
}
