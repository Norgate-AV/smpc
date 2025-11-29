package compiler_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Norgate-AV/smpc/internal/compiler"
)

func TestCompileResult_HasErrors(t *testing.T) {
	tests := []struct {
		name      string
		result    compiler.CompileResult
		hasErrors bool
	}{
		{
			name:      "No errors",
			result:    compiler.CompileResult{Errors: 0, HasErrors: false},
			hasErrors: false,
		},
		{
			name:      "Has errors",
			result:    compiler.CompileResult{Errors: 5, HasErrors: true},
			hasErrors: true,
		},
		{
			name:      "Warnings only",
			result:    compiler.CompileResult{Warnings: 3, Errors: 0, HasErrors: false},
			hasErrors: false,
		},
		{
			name:      "Notices only",
			result:    compiler.CompileResult{Notices: 2, Errors: 0, HasErrors: false},
			hasErrors: false,
		},
		{
			name: "Mixed warnings and errors",
			result: compiler.CompileResult{
				Warnings:  3,
				Notices:   1,
				Errors:    2,
				HasErrors: true,
			},
			hasErrors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.hasErrors, tt.result.HasErrors)
		})
	}
}

func TestCompileResult_MessageCounts(t *testing.T) {
	result := compiler.CompileResult{
		ErrorMessages:   []string{"error1", "error2"},
		WarningMessages: []string{"warn1"},
		NoticeMessages:  []string{"notice1", "notice2", "notice3"},
	}

	assert.Len(t, result.ErrorMessages, 2)
	assert.Len(t, result.WarningMessages, 1)
	assert.Len(t, result.NoticeMessages, 3)
}

func TestCompileResult_EmptyMessages(t *testing.T) {
	result := compiler.CompileResult{
		Errors:          0,
		Warnings:        0,
		Notices:         0,
		ErrorMessages:   []string{},
		WarningMessages: []string{},
		NoticeMessages:  []string{},
		HasErrors:       false,
	}

	assert.Len(t, result.ErrorMessages, 0)
	assert.Len(t, result.WarningMessages, 0)
	assert.Len(t, result.NoticeMessages, 0)
	assert.False(t, result.HasErrors)
	assert.Equal(t, 0, result.Errors)
	assert.Equal(t, 0, result.Warnings)
	assert.Equal(t, 0, result.Notices)
}

func TestCompileResult_CompileTime(t *testing.T) {
	result := compiler.CompileResult{
		CompileTime: 2.5,
	}

	assert.Equal(t, 2.5, result.CompileTime)
	assert.Greater(t, result.CompileTime, 0.0)
}

func TestCompileResult_CountsMatchMessages(t *testing.T) {
	result := compiler.CompileResult{
		Errors:   2,
		Warnings: 3,
		Notices:  1,
		ErrorMessages: []string{
			"ERROR: Undefined variable X",
			"ERROR: Missing semicolon",
		},
		WarningMessages: []string{
			"WARNING: Unused variable Y",
			"WARNING: Deprecated function",
			"WARNING: Unreachable code",
		},
		NoticeMessages: []string{
			"NOTICE: Compilation successful",
		},
		HasErrors: true,
	}

	assert.Equal(t, result.Errors, len(result.ErrorMessages))
	assert.Equal(t, result.Warnings, len(result.WarningMessages))
	assert.Equal(t, result.Notices, len(result.NoticeMessages))
}

func TestCompileOptions_RequiredFields(t *testing.T) {
	opts := compiler.CompileOptions{
		FilePath:     "test.smw",
		RecompileAll: false,
		Hwnd:         uintptr(12345),
	}

	assert.Equal(t, "test.smw", opts.FilePath)
	assert.False(t, opts.RecompileAll)
	assert.Equal(t, uintptr(12345), opts.Hwnd)
}

func TestCompileOptions_RecompileAllFlag(t *testing.T) {
	tests := []struct {
		name         string
		recompileAll bool
	}{
		{name: "Normal compile", recompileAll: false},
		{name: "Recompile all", recompileAll: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := compiler.CompileOptions{
				RecompileAll: tt.recompileAll,
			}
			assert.Equal(t, tt.recompileAll, opts.RecompileAll)
		})
	}
}

func TestCompileOptions_SimplPidPtr(t *testing.T) {
	var pid uint32
	opts := compiler.CompileOptions{
		SimplPidPtr: &pid,
	}

	// Verify pointer is set
	assert.NotNil(t, opts.SimplPidPtr)

	// Verify we can write to it
	*opts.SimplPidPtr = 12345
	assert.Equal(t, uint32(12345), pid)
}
