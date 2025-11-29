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
		{
			name:          "Parse with tabs",
			line:          "Program Warnings:\t5",
			prefix:        "Program Warnings",
			expectedValue: 5,
			expectedOk:    true,
		},
		{
			name:          "Parse with CRLF",
			line:          "Program Warnings: 3\r\n",
			prefix:        "Program Warnings",
			expectedValue: 3,
			expectedOk:    true,
		},
		{
			name:          "Parse with mixed whitespace",
			line:          "Program Errors  :\t\t10",
			prefix:        "Program Errors",
			expectedValue: 10,
			expectedOk:    true,
		},
		{
			name:          "Parse with trailing whitespace",
			line:          "Program Notices: 7  ",
			prefix:        "Program Notices",
			expectedValue: 7,
			expectedOk:    true,
		},
		// Edge cases: Malformed input
		{
			name:          "Malformed - negative number",
			line:          "Program Warnings: -5",
			prefix:        "Program Warnings",
			expectedValue: 0,
			expectedOk:    false,
		},
		{
			name:          "Malformed - decimal number",
			line:          "Program Warnings: 3.14",
			prefix:        "Program Warnings",
			expectedValue: 3, // Sscanf "%d" truncates decimal to int
			expectedOk:    true,
		},
		{
			name:          "Malformed - number with text",
			line:          "Program Warnings: 5abc",
			prefix:        "Program Warnings",
			expectedValue: 5, // Sscanf "%d" stops at 'a', successfully parses 5
			expectedOk:    true,
		},
		{
			name:          "Malformed - multiple colons",
			line:          "Program Warnings: : 5",
			prefix:        "Program Warnings",
			expectedValue: 0,
			expectedOk:    false,
		},
		// Edge cases: Unicode and special characters
		{
			name:          "Unicode - Chinese characters in value",
			line:          "Program Warnings: 中文",
			prefix:        "Program Warnings",
			expectedValue: 0,
			expectedOk:    false,
		},
		{
			name:          "Unicode - emoji in value",
			line:          "Program Warnings: 😊",
			prefix:        "Program Warnings",
			expectedValue: 0,
			expectedOk:    false,
		},
		{
			name:          "Special chars - parentheses",
			line:          "Program Warnings: (5)",
			prefix:        "Program Warnings",
			expectedValue: 0,
			expectedOk:    false,
		},
		// Edge cases: Very large numbers
		{
			name:          "Very large number - near int max",
			line:          "Program Warnings: 2147483647",
			prefix:        "Program Warnings",
			expectedValue: 2147483647,
			expectedOk:    true,
		},
		{
			name:          "Very large number - exceeds int32",
			line:          "Program Warnings: 9999999999",
			prefix:        "Program Warnings",
			expectedValue: 9999999999,
			expectedOk:    true,
		},
		// Edge cases: Boundary conditions
		{
			name:          "Boundary - zero with leading zeros",
			line:          "Program Warnings: 0000",
			prefix:        "Program Warnings",
			expectedValue: 0,
			expectedOk:    true,
		},
		{
			name:          "Boundary - single digit",
			line:          "Program Warnings: 1",
			prefix:        "Program Warnings",
			expectedValue: 1,
			expectedOk:    true,
		},
		{
			name:          "Boundary - only whitespace after colon",
			line:          "Program Warnings:   ",
			prefix:        "Program Warnings",
			expectedValue: 0,
			expectedOk:    false,
		},
		{
			name:          "Boundary - case sensitivity check",
			line:          "program warnings: 5",
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
		{
			name:          "Parse with tabs",
			line:          "Compile Time:\t1.5 seconds",
			expectedValue: 1.5,
			expectedOk:    true,
		},
		{
			name:          "Parse with CRLF",
			line:          "Compile Time: 2.25 seconds\r\n",
			expectedValue: 2.25,
			expectedOk:    true,
		},
		{
			name:          "Parse with mixed whitespace",
			line:          "Compile Time  :\t\t3.75   s",
			expectedValue: 3.75,
			expectedOk:    true,
		},
		{
			name:          "Parse very small time",
			line:          "Compile Time: 0.001 seconds",
			expectedValue: 0.001,
			expectedOk:    true,
		},
		// Edge cases: Malformed input
		{
			name:          "Malformed - negative time",
			line:          "Compile Time: -1.5 seconds",
			expectedValue: 0,
			expectedOk:    false,
		},
		{
			name:          "Malformed - text in time",
			line:          "Compile Time: one second",
			expectedValue: 0,
			expectedOk:    false,
		},
		{
			name:          "Malformed - multiple decimal points",
			line:          "Compile Time: 1.2.3 seconds",
			expectedValue: 1.2, // Regex matches "1.2.3", Sscanf "%f" stops at second decimal
			expectedOk:    true,
		},
		{
			name:          "Malformed - time with comma",
			line:          "Compile Time: 1,234 seconds",
			expectedValue: 1, // Regex [0-9.]+ matches "1", stops at comma
			expectedOk:    true,
		},
		// Edge cases: Unicode and special characters
		{
			name:          "Unicode - Chinese characters",
			line:          "Compile Time: 中文 seconds",
			expectedValue: 0,
			expectedOk:    false,
		},
		{
			name:          "Unicode - emoji",
			line:          "Compile Time: 😊 seconds",
			expectedValue: 0,
			expectedOk:    false,
		},
		// Edge cases: Very large numbers
		{
			name:          "Very large time - hours",
			line:          "Compile Time: 3600.0 seconds",
			expectedValue: 3600.0,
			expectedOk:    true,
		},
		{
			name:          "Very large time - days",
			line:          "Compile Time: 86400.5 seconds",
			expectedValue: 86400.5,
			expectedOk:    true,
		},
		// Edge cases: Boundary conditions
		{
			name:          "Boundary - exact zero",
			line:          "Compile Time: 0 seconds",
			expectedValue: 0.0,
			expectedOk:    true,
		},
		{
			name:          "Boundary - scientific notation (not supported)",
			line:          "Compile Time: 1.5e2 seconds",
			expectedValue: 1.5, // Regex matches "1.5e2", Sscanf "%f" stops at 'e'
			expectedOk:    true,
		},
		{
			name:          "Boundary - very high precision",
			line:          "Compile Time: 1.123456789 seconds",
			expectedValue: 1.123456789,
			expectedOk:    true,
		},
		{
			name:          "Boundary - only decimal point",
			line:          "Compile Time: . seconds",
			expectedValue: 0,
			expectedOk:    false,
		},
		{
			name:          "Boundary - leading decimal point",
			line:          "Compile Time: .5 seconds",
			expectedValue: 0.5,
			expectedOk:    true,
		},
		{
			name:          "Boundary - trailing decimal point",
			line:          "Compile Time: 5. seconds",
			expectedValue: 5.0,
			expectedOk:    true,
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
