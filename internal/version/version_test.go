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
	assert.Contains(t, full, version.GetVersion())
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

func TestGetCommit(t *testing.T) {
	t.Parallel()

	c := version.GetCommit()
	assert.NotEmpty(t, c, "Commit should not be empty")
}

func TestGetDate(t *testing.T) {
	t.Parallel()

	d := version.GetDate()
	assert.NotEmpty(t, d, "Date should not be empty")
}

func TestGetFullVersionFormat(t *testing.T) {
	t.Parallel()

	// Verify the format of GetFullVersion matches expected pattern
	full := version.GetFullVersion()
	expected := version.GetVersion() + " (commit: " + version.GetCommit() + ", built: " + version.GetDate() + ")"
	assert.Equal(t, expected, full)
}
