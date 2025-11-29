package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// CreateTempDir creates a temporary directory for testing
func CreateTempDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "smpc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// CreateTestSMWFile creates a minimal .smw file for testing
func CreateTestSMWFile(t *testing.T, dir string, name string) string {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("test content"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	return path
}
