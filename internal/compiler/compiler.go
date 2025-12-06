package compiler

import (
	"fmt"
	"log/slog"
	"strings"
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
		// Try SendInput first (modern API, atomic operation)
		success = c.deps.Keyboard.SendAltF12WithSendInput()
		if !success {
			c.log.Warn("SendAltF12WithSendInput failed, falling back to keybd_event")
			c.deps.Keyboard.SendAltF12()
		} else {
			c.log.Debug("SendAltF12WithSendInput succeeded")
		}
	} else {
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

	if pid != 0 {
		// Use event-driven dialog handling
		var err error
		var eventResult *CompileResult
		compileCompleteHwnd, eventResult, err = c.handleCompilationEvents(opts)
		if err != nil {
			return nil, err
		}

		// Copy event result into our result
		result = eventResult
	}

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
	}

	if result.HasErrors {
		return result, fmt.Errorf("compilation failed with %d error(s)", result.Errors)
	}

	return result, nil
}

// handleCompilationEvents uses an event-driven approach to respond to dialogs as they appear
func (c *Compiler) handleCompilationEvents(opts CompileOptions) (uintptr, *CompileResult, error) {
	// Maximum time to wait for compilation to complete
	timeout := time.NewTimer(timeouts.CompilationCompleteTimeout)
	defer timeout.Stop()

	result := &CompileResult{}

	// Track what we've seen and what we're waiting for
	var (
		compilingDetected       bool
		compileCompleteDetected bool
		compileCompleteHwnd     uintptr
		programCompHwnd         uintptr
	)

	c.log.Debug("Entering event-driven dialog monitoring loop")

	// Event loop - respond to dialogs as they appear in real-time
	for {
		select {
		case ev := <-windows.MonitorCh:
			c.log.Debug("Received window event",
				slog.String("title", ev.Title),
				slog.Uint64("hwnd", uint64(ev.Hwnd)),
			)

			// Handle each dialog type as it appears
			switch ev.Title {
			case "Incomplete Symbols":
				// Fatal error - compilation cannot proceed
				c.log.Error("ERROR: Incomplete Symbols detected", slog.String("title", ev.Title))
				c.log.Info("The program contains incomplete symbols and cannot be compiled.")
				c.log.Info("Please fix the incomplete symbols in SIMPL Windows before attempting to compile.")

				// Extract error details
				childInfos := c.deps.WindowMgr.CollectChildInfos(ev.Hwnd)
				for _, ci := range childInfos {
					if ci.ClassName == "Edit" && len(ci.Text) > 50 {
						c.log.Info("Details", slog.String("text", ci.Text))
						break
					}
				}

				return 0, nil, fmt.Errorf("program contains incomplete symbols and cannot be compiled")

			case "Convert/Compile":
				// Save prompt - auto-confirm
				c.log.Debug("Handling 'Convert/Compile' dialog")
				_ = c.deps.WindowMgr.SetForeground(ev.Hwnd)
				time.Sleep(timeouts.DialogResponseDelay)
				c.deps.Keyboard.SendEnter()
				c.log.Info("Auto-confirmed save prompt")

			case "Commented out Symbols and/or Devices":
				// Confirmation dialog - auto-confirm
				c.log.Debug("Handling 'Commented out Symbols and/or Devices' dialog")
				_ = c.deps.WindowMgr.SetForeground(ev.Hwnd)
				time.Sleep(timeouts.DialogResponseDelay)
				c.deps.Keyboard.SendEnter()
				c.log.Info("Auto-confirmed commented symbols dialog")

			case "Compiling...":
				// Compilation in progress
				if !compilingDetected {
					c.log.Debug("Detected 'Compiling...' dialog")

					if opts.RecompileAll {
						c.log.Info("Compiling program... (Recompile All)")
					} else {
						c.log.Info("Compiling program...")
					}

					compilingDetected = true
				}

			case "Compile Complete":
				// Compilation finished - parse results
				if !compileCompleteDetected {
					c.log.Debug("Detected 'Compile Complete' dialog - parsing results")
					compileCompleteHwnd = ev.Hwnd

					// Parse statistics from dialog
					childInfos := c.deps.WindowMgr.CollectChildInfos(ev.Hwnd)
					for _, ci := range childInfos {
						text := strings.ReplaceAll(ci.Text, "\r\n", "\n")
						lines := strings.Split(text, "\n")

						for _, line := range lines {
							line = strings.TrimSpace(line)
							if line == "" {
								continue
							}

							if n, ok := ParseStatLine(line, "Program Warnings"); ok {
								result.Warnings = n
							}

							if n, ok := ParseStatLine(line, "Program Notices"); ok {
								result.Notices = n
							}

							if n, ok := ParseStatLine(line, "Program Errors"); ok {
								result.Errors = n
							}

							if secs, ok := ParseCompileTimeLine(line); ok {
								result.CompileTime = secs
							}
						}
					}

					compileCompleteDetected = true
				}

			case "Program Compilation":
				// Detailed error/warning/notice messages
				if programCompHwnd == 0 {
					c.log.Debug("Detected 'Program Compilation' dialog")
					programCompHwnd = ev.Hwnd
				}

			case "Operation Complete":
				// Sometimes appears - close it
				c.log.Debug("Detected 'Operation Complete' dialog - closing")
				c.deps.WindowMgr.CloseWindow(ev.Hwnd, ev.Title)
				time.Sleep(timeouts.WindowMessageDelay)
			}

			// If we have both "Compile Complete" and (optionally) "Program Compilation", we're done
			if compileCompleteDetected {
				// If there are warnings/notices/errors, wait briefly for Program Compilation dialog
				if (result.Warnings > 0 || result.Notices > 0 || result.Errors > 0) && programCompHwnd == 0 {
					time.Sleep(500 * time.Millisecond)
					continue
				}

				// Parse detailed messages if we have the Program Compilation dialog
				if programCompHwnd != 0 {
					result.WarningMessages, result.NoticeMessages, result.ErrorMessages = c.parseDetailedMessages(programCompHwnd)

					// Log the messages
					c.logCompilationMessages(result.ErrorMessages, result.WarningMessages, result.NoticeMessages)
				}

				// Set HasErrors flag
				result.HasErrors = result.Errors > 0 || len(result.ErrorMessages) > 0

				// Compilation complete
				return compileCompleteHwnd, result, nil
			}

		case <-timeout.C:
			c.log.Error("Compilation timeout: did not complete within 5 minutes")
			return 0, nil, fmt.Errorf("compilation timeout: did not detect 'Compile Complete' dialog within 5 minutes")
		}
	}
}

// parseDetailedMessages extracts error/warning/notice messages from Program Compilation dialog
func (c *Compiler) parseDetailedMessages(hwnd uintptr) (warnings, notices, errors []string) {
	childInfos := c.deps.WindowMgr.CollectChildInfos(hwnd)

	// Extract messages from ListBox
	for _, ci := range childInfos {
		if ci.ClassName != "ListBox" || len(ci.Items) == 0 {
			continue
		}

		for _, line := range ci.Items {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			lineUpper := strings.ToUpper(line)
			switch {
			case strings.HasPrefix(lineUpper, "ERROR"):
				errors = append(errors, line)
			case strings.HasPrefix(lineUpper, "WARNING"):
				warnings = append(warnings, line)
			case strings.HasPrefix(lineUpper, "NOTICE"):
				notices = append(notices, line)
			default:
				// Continuation of previous message
				switch {
				case len(errors) > 0:
					errors[len(errors)-1] += " " + line
				case len(warnings) > 0:
					warnings[len(warnings)-1] += " " + line
				case len(notices) > 0:
					notices[len(notices)-1] += " " + line
				}
			}
		}
	}

	return warnings, notices, errors
}

// logCompilationMessages logs error/warning/notice messages with proper formatting
func (c *Compiler) logCompilationMessages(errorMsgs, warningMsgs, noticeMsgs []string) {
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
