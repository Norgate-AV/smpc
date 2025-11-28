package compiler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Norgate-AV/smpc/internal/simpl"
	"github.com/Norgate-AV/smpc/internal/windows"
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
func Compile(opts CompileOptions) (*CompileResult, error) {
	result := &CompileResult{}

	// Detect SIMPL Windows process PID for dialog monitoring
	slog.Debug("Getting SIMPL Windows process PID")
	pid := simpl.GetPid()
	if pid == 0 {
		slog.Warn("Could not determine PID")
		slog.Info("Warning: Could not determine SIMPL Windows process PID; dialog detection may be limited")
	} else {
		slog.Debug("SIMPL Windows PID detected", "pid", pid)
		if opts.SimplPidPtr != nil {
			*opts.SimplPidPtr = pid // Store for signal handler
		}
	}

	// Handle "Operation Complete" dialog that may appear after loading the file
	if err := HandleOperationCompleteDialog(pid); err != nil {
		return nil, err
	}

	// Confirm elevation before sending keystrokes
	if windows.IsElevated() {
		slog.Debug("Process is elevated, proceeding with keystroke injection")
	} else {
		slog.Warn("Process is NOT elevated, keystroke injection may fail")
	}

	// Bring window to foreground and send compile keystroke
	slog.Debug("Bringing window to foreground")
	_ = windows.SetForeground(opts.Hwnd)

	slog.Info("Waiting for window to receive focus...")
	time.Sleep(1 * time.Second)

	// Send the appropriate keystroke to trigger compilation
	slog.Debug("Preparing to send keystroke")
	var keystrokeSent bool
	if opts.RecompileAll {
		slog.Info("Sending Alt+F12 keystroke to trigger Recompile All...")
		slog.Debug("Sending Alt+F12 keystroke")
		keystrokeSent = windows.SendAltF12()
		if keystrokeSent {
			slog.Info("Successfully sent Alt+F12 keystroke")
			slog.Debug("Alt+F12 sent successfully")
		} else {
			slog.Error("Failed to send Alt+F12")
		}
	} else {
		slog.Info("Sending F12 keystroke to trigger compile...")
		slog.Debug("Sending F12 keystroke")
		keystrokeSent = windows.SendF12()
		if keystrokeSent {
			slog.Info("Successfully sent F12 keystroke")
			slog.Debug("F12 sent successfully")
		} else {
			slog.Error("Failed to send F12")
		}
	}

	if !keystrokeSent {
		return nil, fmt.Errorf("failed to send compile keystroke")
	}

	slog.Debug("Starting compile monitoring")

	// Check for fatal "Incomplete Symbols" error
	if err := HandleIncompleteSymbolsDialog(pid); err != nil {
		return nil, err
	}

	// Handle save prompts and confirmations
	if err := HandleConvertCompileDialog(pid); err != nil {
		return nil, err
	}

	if err := HandleCommentedOutSymbolsDialog(pid); err != nil {
		return nil, err
	}

	// Wait for compilation to start
	if err := WaitForCompilingDialog(pid); err != nil {
		return nil, err
	}

	// Parse the Compile Complete dialog
	compileCompleteHwnd, warnings, notices, errors, compileTime, err := ParseCompileCompleteDialog(pid)
	if err != nil {
		return nil, err
	}

	result.Warnings = warnings
	result.Notices = notices
	result.Errors = errors
	result.CompileTime = compileTime
	result.HasErrors = errors > 0

	// Parse detailed messages from Program Compilation dialog
	errorMsgs, warningMsgs, noticeMsgs, err := ParseProgramCompilationDialog(pid, warnings, notices, errors)
	if err != nil {
		return nil, err
	}

	result.ErrorMessages = errorMsgs
	result.WarningMessages = warningMsgs
	result.NoticeMessages = noticeMsgs

	// If we got additional errors from the dialog, update hasErrors
	if len(errorMsgs) > 0 {
		result.HasErrors = true
	}

	// Close dialogs
	slog.Info("Closing dialogs and SIMPL Windows...")

	// First, close the "Compile Complete" dialog if it's still open
	if compileCompleteHwnd != 0 {
		windows.CloseWindow(compileCompleteHwnd, "Compile Complete dialog")
		time.Sleep(500 * time.Millisecond)
	}

	// Handle confirmation dialog when closing
	if err := HandleConfirmationDialog(pid); err != nil {
		return nil, err
	}

	// Now close the main SIMPL Windows application
	if opts.Hwnd != 0 {
		windows.CloseWindow(opts.Hwnd, "SIMPL Windows")
		time.Sleep(1 * time.Second)
		slog.Info("SIMPL Windows closed successfully")
	}

	// Print final summary
	if pid != 0 && windows.MonitorCh != nil {
		slog.Info("=== Compile Summary ===")
		if result.Errors > 0 {
			slog.Info("Errors", "count", result.Errors)
		}
		slog.Info("Warnings", "count", result.Warnings)
		slog.Info("Notices", "count", result.Notices)
		slog.Info("Compile Time", "seconds", result.CompileTime)
		slog.Info("=======================")
	}

	if result.HasErrors {
		return result, fmt.Errorf("compilation failed with %d error(s)", result.Errors)
	}

	slog.Debug("Compilation completed successfully")
	return result, nil
}
