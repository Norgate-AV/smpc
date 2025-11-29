package compiler

import (
	"context"
)

// CompileResult holds the results of a compilation
type CompileResult struct {
	Warnings        int
	Notices         int
	Errors          int
	CompileTime     float64
	ErrorMessages   []string
	WarningMessages []string
	NoticeMessages  []string
	HasErrors       bool
}

// CompileOptions holds options for the compilation
type CompileOptions struct {
	FilePath     string
	RecompileAll bool
	Hwnd         uintptr
	Ctx          context.Context
	SimplPidPtr  *uint32 // Pointer to store PID for signal handlers
}

// Compile orchestrates the compilation process for a SIMPL Windows file
// This includes:
// - Handling pre-compilation dialogs
// - Triggering the compile
// - Monitoring compilation progress
// - Parsing results
// - Closing dialogs
//
// This function maintains backward compatibility by delegating to CompileWithDeps
// with default (real) dependencies.
func Compile(opts CompileOptions) (*CompileResult, error) {
	return CompileWithDeps(opts, NewDefaultDependencies())
}
