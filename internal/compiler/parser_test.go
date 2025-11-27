package compiler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseStatLine(t *testing.T) {
	tests := []struct {
		name          string
		line          string
		prefix        string
		expectedValue int
		expectedOk    bool
	}{
		{
			name:          "Parse warnings count",
			line:          "Program Warnings: 1",
			prefix:        "Program Warnings",
			expectedValue: 1,
			expectedOk:    true,
		},
		{
			name:          "Parse errors count",
			line:          "Program Errors: 5",
			prefix:        "Program Errors",
			expectedValue: 5,
			expectedOk:    true,
		},
		{
			name:          "Parse zero count",
			line:          "Program Warnings: 0",
			prefix:        "Program Warnings",
			expectedValue: 0,
			expectedOk:    true,
		},
		{
			name:          "Parse with extra spaces",
			line:          "Program Warnings  :   42",
			prefix:        "Program Warnings",
			expectedValue: 42,
			expectedOk:    true,
		},
		{
			name:          "Parse large number",
			line:          "Program Errors: 999",
			prefix:        "Program Errors",
			expectedValue: 999,
			expectedOk:    true,
		},
		{
			name:          "No match - wrong prefix",
			line:          "Program Warnings: 1",
			prefix:        "Build Errors",
			expectedValue: 0,
			expectedOk:    false,
		},
		{
			name:          "No match - missing colon",
			line:          "Program Warnings 1",
			prefix:        "Program Warnings",
			expectedValue: 0,
			expectedOk:    false,
		},
		{
			name:          "No match - non-numeric value",
			line:          "Program Warnings: abc",
			prefix:        "Program Warnings",
			expectedValue: 0,
			expectedOk:    false,
		},
		{
			name:          "No match - empty line",
			line:          "",
			prefix:        "Program Warnings",
			expectedValue: 0,
			expectedOk:    false,
		},
		{
			name:          "No match - prefix in middle of line",
			line:          "Total Program Warnings: 5",
			prefix:        "Program Warnings",
			expectedValue: 0,
			expectedOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, ok := ParseStatLine(tt.line, tt.prefix)
			assert.Equal(t, tt.expectedOk, ok, "ok value mismatch")
			assert.Equal(t, tt.expectedValue, value, "parsed value mismatch")
		})
	}
}

func TestParseCompileTimeLine(t *testing.T) {
	tests := []struct {
		name          string
		line          string
		expectedValue float64
		expectedOk    bool
	}{
		{
			name:          "Parse seconds with 'seconds' suffix",
			line:          "Compile Time: 0.23 seconds",
			expectedValue: 0.23,
			expectedOk:    true,
		},
		{
			name:          "Parse seconds with 's' suffix",
			line:          "Compile Time: 1.5 s",
			expectedValue: 1.5,
			expectedOk:    true,
		},
		{
			name:          "Parse seconds without suffix",
			line:          "Compile Time: 2.75",
			expectedValue: 2.75,
			expectedOk:    true,
		},
		{
			name:          "Parse integer time",
			line:          "Compile Time: 3 seconds",
			expectedValue: 3.0,
			expectedOk:    true,
		},
		{
			name:          "Parse zero time",
			line:          "Compile Time: 0.00 seconds",
			expectedValue: 0.0,
			expectedOk:    true,
		},
		{
			name:          "Parse with extra spaces",
			line:          "Compile Time  :   5.42   seconds",
			expectedValue: 5.42,
			expectedOk:    true,
		},
		{
			name:          "Parse large time",
			line:          "Compile Time: 123.456 seconds",
			expectedValue: 123.456,
			expectedOk:    true,
		},
		{
			name:          "No match - wrong prefix",
			line:          "Build Time: 0.23 seconds",
			expectedValue: 0,
			expectedOk:    false,
		},
		{
			name:          "No match - missing colon",
			line:          "Compile Time 0.23 seconds",
			expectedValue: 0,
			expectedOk:    false,
		},
		{
			name:          "No match - non-numeric value",
			line:          "Compile Time: abc seconds",
			expectedValue: 0,
			expectedOk:    false,
		},
		{
			name:          "No match - empty line",
			line:          "",
			expectedValue: 0,
			expectedOk:    false,
		},
		{
			name:          "No match - prefix in middle of line",
			line:          "Total Compile Time: 0.23 seconds",
			expectedValue: 0,
			expectedOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, ok := ParseCompileTimeLine(tt.line)
			assert.Equal(t, tt.expectedOk, ok, "ok value mismatch")
			if tt.expectedOk {
				assert.InDelta(t, tt.expectedValue, value, 0.0001, "parsed value mismatch")
			} else {
				assert.Equal(t, tt.expectedValue, value, "parsed value should be zero for non-match")
			}
		})
	}
}
