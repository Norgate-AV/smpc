package version_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Norgate-AV/smpc/internal/version"
)

func TestGetVersion(t *testing.T) {
	t.Parallel()

	v := version.GetVersion()
	assert.NotEmpty(t, v, "Version should not be empty")
}

func TestGetFullVersion(t *testing.T) {
	t.Parallel()

	full := version.GetFullVersion()
	assert.Contains(t, full, version.Version)
	assert.Contains(t, full, "commit:")
	assert.Contains(t, full, "built:")
}

func TestVersionFormat(t *testing.T) {
	t.Parallel()

	// Ensure version follows semantic versioning pattern
	v := version.GetVersion()
	if v != "dev" {
		assert.Regexp(t, `^v?\d+\.\d+\.\d+`, v, "Version should match semver pattern")
	}
}

func TestGetVersionReturnsVersionVariable(t *testing.T) {
	t.Parallel()

	// GetVersion should return exactly the Version variable
	assert.Equal(t, version.Version, version.GetVersion())
}

func TestGetFullVersionFormat(t *testing.T) {
	t.Parallel()

	// Verify the format of GetFullVersion matches expected pattern
	full := version.GetFullVersion()
	expected := version.Version + " (commit: " + version.Commit + ", built: " + version.Date + ")"
	assert.Equal(t, expected, full)
}
