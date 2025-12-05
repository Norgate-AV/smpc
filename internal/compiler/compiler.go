package compiler

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/Norgate-AV/smpc/internal/interfaces"
	"github.com/Norgate-AV/smpc/internal/logger"
	"github.com/Norgate-AV/smpc/internal/simpl"
	"github.com/Norgate-AV/smpc/internal/timeouts"
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
	SimplPidPtr  *uint32 // Pointer to store PID for signal handlers
}

// CompileDependencies holds all external dependencies for testing
type CompileDependencies struct {
	DialogHandler *DialogHandler
	ProcessMgr    interfaces.ProcessManager
	WindowMgr     interfaces.WindowManager
	Keyboard      interfaces.KeyboardInjector
}

// Compiler orchestrates the compilation process with injected dependencies
type Compiler struct {
	log  logger.LoggerInterface
	deps *CompileDependencies
}

// NewCompiler creates a new Compiler with the provided logger and default dependencies
func NewCompiler(log logger.LoggerInterface) *Compiler {
	windowsAPI := windows.NewWindowsAPI(log)
	simplAPI := simpl.SimplProcessAPI{}

	return &Compiler{
		log: log,
		deps: &CompileDependencies{
			DialogHandler: NewDialogHandlerWithAPI(log, windowsAPI),
			ProcessMgr:    simplAPI,
			WindowMgr:     windowsAPI,
			Keyboard:      windowsAPI,
		},
	}
}

// NewCompilerWithDeps creates a new Compiler with custom dependencies for testing
func NewCompilerWithDeps(log logger.LoggerInterface, deps *CompileDependencies) *Compiler {
	return &Compiler{
		log:  log,
		deps: deps,
	}
}

// Compile orchestrates the compilation process for a SIMPL Windows file
// This includes:
// - Handling pre-compilation dialogs
// - Triggering the compile
// - Monitoring compilation progress
// - Parsing results
// - Closing dialogs
func (c *Compiler) Compile(opts CompileOptions) (*CompileResult, error) {
	result := &CompileResult{}

	// Detect SIMPL Windows process PID for dialog monitoring
	c.log.Debug("Getting SIMPL Windows process PID")
	pid := c.deps.ProcessMgr.GetPid()
	if pid == 0 {
		c.log.Warn("Could not determine PID")
		c.log.Info("Warning: Could not determine SIMPL Windows process PID; dialog detection may be limited")
	} else {
		c.log.Debug("SIMPL Windows PID detected", slog.Uint64("pid", uint64(pid)))
		if opts.SimplPidPtr != nil {
			*opts.SimplPidPtr = pid // Store for signal handler
		}
	}

	// Handle "Operation Complete" dialog that may appear after loading the file
	// Only attempt dialog handling if we have a valid PID
	if pid != 0 {
		if err := c.deps.DialogHandler.HandleOperationComplete(); err != nil {
			return nil, err
		}
	}

	// Confirm elevation before sending keystrokes
	if c.deps.WindowMgr.IsElevated() {
		c.log.Debug("Process is elevated, proceeding with keystroke injection")
	} else {
		c.log.Warn("Process is NOT elevated, keystroke injection may fail")
	}

	// Bring window to foreground and send compile keystroke
	c.log.Debug("Bringing window to foreground")
	_ = c.deps.WindowMgr.SetForeground(opts.Hwnd)

	c.log.Debug("Waiting for window to receive focus...")
	time.Sleep(timeouts.FocusVerificationDelay)

	// Verify the window is in the foreground before sending keystrokes
	c.log.Debug("Verifying foreground window")
	verified := c.deps.WindowMgr.VerifyForegroundWindow(opts.Hwnd, pid)
	if !verified {
		c.log.Warn("Window verification failed, attempting to set foreground again")
		_ = c.deps.WindowMgr.SetForeground(opts.Hwnd)
		time.Sleep(timeouts.WindowMessageDelay)
		verified = c.deps.WindowMgr.VerifyForegroundWindow(opts.Hwnd, pid)
		if !verified {
			c.log.Error("Could not verify correct window is in foreground - keystrokes may not reach intended target")
		} else {
			c.log.Debug("Window verified on second attempt")
		}
	} else {
		c.log.Debug("Window verification successful")
	}

	// Send the appropriate keystroke to trigger compilation
	c.log.Debug("Preparing to send keystroke")

	var success bool
	if opts.RecompileAll {
		// c.log.Info("Triggering Recompile All (Alt+F12)")
		// Try SendInput first (modern API, atomic operation)
		success = c.deps.Keyboard.SendAltF12WithSendInput()
		if !success {
			c.log.Warn("SendAltF12WithSendInput failed, falling back to keybd_event")
			c.deps.Keyboard.SendAltF12()
		} else {
			c.log.Debug("SendAltF12WithSendInput succeeded")
		}
	} else {
		// c.log.Info("Triggering compile (F12)")
		// Try SendInput first (modern API, atomic operation)
		success = c.deps.Keyboard.SendF12WithSendInput()
		if !success {
			c.log.Warn("SendF12WithSendInput failed, falling back to keybd_event")
			c.deps.Keyboard.SendF12()
		} else {
			c.log.Debug("SendF12WithSendInput succeeded")
		}
	}

	c.log.Debug("Starting compile monitoring")

	// Only attempt dialog handling if we have a valid PID
	var compileCompleteHwnd uintptr
	var warnings, notices, errors int
	var compileTime float64

	if pid != 0 {
		// Check for fatal "Incomplete Symbols" error
		if err := c.deps.DialogHandler.HandleIncompleteSymbols(); err != nil {
			return nil, err
		}

		// Handle save prompts and confirmations
		if err := c.deps.DialogHandler.HandleConvertCompile(); err != nil {
			return nil, err
		}

		if err := c.deps.DialogHandler.HandleCommentedOutSymbols(); err != nil {
			return nil, err
		}

		// Wait for compilation to start
		if err := c.deps.DialogHandler.WaitForCompiling(); err != nil {
			return nil, err
		}

		// Parse the Compile Complete dialog
		var err error
		compileCompleteHwnd, warnings, notices, errors, compileTime, err = c.deps.DialogHandler.ParseCompileComplete()
		if err != nil {
			return nil, err
		}

		// Parse detailed messages if there are any warnings, notices, or errors
		if warnings > 0 || notices > 0 || errors > 0 {
			warningMsgs, noticeMsgs, errorMsgs, err := c.deps.DialogHandler.ParseProgramCompilation()
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

			// Log messages - only show header if there are actual messages to display
			if len(errorMsgs) > 0 {
				c.log.Info("")
				c.log.Info("Error messages:")
				for i, msg := range errorMsgs {
					c.log.Info(fmt.Sprintf("  %d. %s", i+1, msg),
						slog.Int("number", i+1),
						slog.String("message", msg),
					)
				}
			}

			if len(warningMsgs) > 0 {
				c.log.Info("")
				c.log.Info("Warning messages:")
				for i, msg := range warningMsgs {
					c.log.Info(fmt.Sprintf("  %d. %s", i+1, msg))
				}
			}

			if len(noticeMsgs) > 0 {
				c.log.Info("")
				c.log.Info("Notice messages:")
				for i, msg := range noticeMsgs {
					c.log.Info(fmt.Sprintf("  %d. %s", i+1, msg))
				}
			}

			// Add trailing blank line if any messages were displayed
			if len(errorMsgs) > 0 || len(warningMsgs) > 0 || len(noticeMsgs) > 0 {
				c.log.Info("")
			}
		}
	}

	result.Warnings = warnings
	result.Notices = notices
	result.Errors = errors
	result.CompileTime = compileTime
	result.HasErrors = errors > 0

	// Close dialogs
	c.log.Debug("Closing dialogs and SIMPL Windows...")

	// First, close the "Compile Complete" dialog if it's still open
	if compileCompleteHwnd != 0 {
		c.deps.WindowMgr.CloseWindow(compileCompleteHwnd, "Compile Complete dialog")
		time.Sleep(timeouts.StabilityCheckInterval)
	}

	// Handle confirmation dialog when closing
	if pid != 0 {
		if err := c.deps.DialogHandler.HandleConfirmation(); err != nil {
			return nil, err
		}
	}

	// Now close the main SIMPL Windows application
	if opts.Hwnd != 0 {
		c.deps.WindowMgr.CloseWindow(opts.Hwnd, "SIMPL Windows")
		time.Sleep(timeouts.CleanupDelay)
		// c.log.Info("SIMPL Windows closed")
	}

	if result.HasErrors {
		return result, fmt.Errorf("compilation failed with %d error(s)", result.Errors)
	}

	return result, nil
}
