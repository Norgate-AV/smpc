//go:build windows

package windows

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildArgsString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "no args",
			args:     []string{},
			expected: "",
		},
		{
			name:     "single arg without spaces",
			args:     []string{"--verbose"},
			expected: "--verbose",
		},
		{
			name:     "multiple args without spaces",
			args:     []string{"--verbose", "--recompile-all"},
			expected: "--verbose --recompile-all",
		},
		{
			name:     "path with spaces is quoted",
			args:     []string{`C:\Users\Name\Kingston University\file.smw`},
			expected: `"C:\Users\Name\Kingston University\file.smw"`,
		},
		{
			name:     "path with spaces plus flags",
			args:     []string{`C:\Users\Name\Kingston University\file.smw`, "--verbose"},
			expected: `"C:\Users\Name\Kingston University\file.smw" --verbose`,
		},
		{
			name:     "path without spaces is not quoted",
			args:     []string{`C:\Users\Name\Projects\file.smw`, "--verbose"},
			expected: `C:\Users\Name\Projects\file.smw --verbose`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := buildArgsString(tt.args)
			assert.Equal(t, tt.expected, result)
		})
	}
}
