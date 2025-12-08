// Package compiler handles SIMPL Windows compilation orchestration and result parsing.
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

const (
	// Message type constants for parsing detailed messages
	msgTypeError   = "ERROR"
	msgTypeWarning = "WARNING"
	msgTypeNotice  = "NOTICE"

	// Dialog title constants
	dialogIncompleteSymbols   = "Incomplete Symbols"
	dialogConvertCompile      = "Convert/Compile"
	dialogCommentedOutSymbols = "Commented out Symbols and/or Devices"
	dialogCompiling           = "Compiling..."
	dialogCompileComplete     = "Compile Complete"
	dialogProgramCompilation  = "Program Compilation"
	dialogOperationComplete   = "Operation Complete"
	dialogConfirmation        = "Confirmation"
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
	FilePath                      string
	RecompileAll                  bool
	Hwnd                          uintptr
	SimplPid                      uint32        // Known PID from ShellExecuteEx (preferred over searching)
	SimplPidPtr                   *uint32       // Pointer to store PID for signal handlers
	SkipPreCompilationDialogCheck bool          // For testing - skip the pre-compilation dialog check
	CompilationTimeout            time.Duration // Override default timeout (0 = use default 5 minutes)
}

// CompileDependencies holds all external dependencies for testing
type CompileDependencies struct {
	ProcessMgr    interfaces.ProcessManager
	WindowMgr     interfaces.WindowManager
	Keyboard      interfaces.KeyboardInjector
	ControlReader interfaces.ControlReader
}

// Compiler orchestrates the compilation process with injected dependencies
type Compiler struct {
	log           logger.LoggerInterface
	processMgr    interfaces.ProcessManager
	windowMgr     interfaces.WindowManager
	keyboard      interfaces.KeyboardInjector
	controlReader interfaces.ControlReader
}

// NewCompiler creates a new Compiler with the provided logger and default dependencies
func NewCompiler(log logger.LoggerInterface) *Compiler {
	windowsAPI := windows.NewWindowsAPI(log)
	simplAPI := simpl.SimplProcessAPI{}

	return &Compiler{
		log:           log,
		processMgr:    simplAPI,
		windowMgr:     windowsAPI,
		keyboard:      windowsAPI,
		controlReader: windowsAPI,
	}
}

// NewCompilerWithDeps creates a new Compiler with custom dependencies for testing
func NewCompilerWithDeps(log logger.LoggerInterface, deps *CompileDependencies) *Compiler {
	return &Compiler{
		log:           log,
		processMgr:    deps.ProcessMgr,
		windowMgr:     deps.WindowMgr,
		keyboard:      deps.Keyboard,
		controlReader: deps.ControlReader,
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

	// Use the exact PID from ShellExecuteEx - no searching, no guessing
	pid := opts.SimplPid
	if pid == 0 {
		c.log.Warn("No PID provided - dialog monitoring will be disabled")
		c.log.Info("Warning: Could not determine SIMPL Windows process PID; dialog detection may be limited")
	} else {
		c.log.Debug("Using SIMPL Windows PID from launch", slog.Uint64("pid", uint64(pid)))
		if opts.SimplPidPtr != nil {
			*opts.SimplPidPtr = pid // Store for signal handler
		}
	}

	// Confirm elevation before sending keystrokes
	if c.windowMgr.IsElevated() {
		c.log.Debug("Process is elevated, proceeding with keystroke injection")
	} else {
		c.log.Warn("Process is NOT elevated, keystroke injection may fail")
	}

	// Bring window to foreground and send compile keystroke
	c.log.Debug("Bringing window to foreground")
	focusSuccess := c.windowMgr.SetForeground(opts.Hwnd)
	if !focusSuccess {
		c.log.Warn("SetForeground failed on first attempt, retrying...")
		time.Sleep(500 * time.Millisecond)

		focusSuccess = c.windowMgr.SetForeground(opts.Hwnd)
		if !focusSuccess {
			c.log.Error("Failed to bring window to foreground after retry")
			return &CompileResult{
				Errors:        1,
				HasErrors:     true,
				ErrorMessages: []string{"Failed to bring SIMPL Windows to foreground - cannot send keystrokes"},
			}, fmt.Errorf("failed to bring SIMPL Windows to foreground - cannot send keystrokes")
		}
	}

	time.Sleep(timeouts.FocusVerificationDelay)

	// Verify the window is in the foreground before sending keystrokes
	c.log.Debug("Verifying foreground window")
	verified := c.windowMgr.VerifyForegroundWindow(opts.Hwnd, pid)
	if !verified {
		c.log.Error("Could not verify correct window is in foreground")
		return &CompileResult{
			Errors:        1,
			HasErrors:     true,
			ErrorMessages: []string{"Wrong window in foreground - cannot safely send keystrokes"},
		}, fmt.Errorf("wrong window in foreground - cannot safely send keystrokes")
	}

	// Handle any pre-compilation dialogs (like "Operation Complete") that may be blocking
	// Skip this in test mode since tests send all events upfront
	if pid != 0 && !opts.SkipPreCompilationDialogCheck {
		if err := c.handlePreCompilationDialogs(); err != nil {
			c.log.Warn("Error handling pre-compilation dialogs", slog.Any("error", err))
		}
	}

	var success bool
	if opts.RecompileAll {
		// Try SendInput first (modern API, atomic operation)
		success = c.keyboard.SendAltF12WithSendInput()
		if !success {
			c.log.Warn("SendAltF12WithSendInput failed, falling back to keybd_event")
			c.keyboard.SendAltF12()
		} else {
			c.log.Debug("SendAltF12WithSendInput succeeded")
		}
	} else {
		// Try SendInput first (modern API, atomic operation)
		success = c.keyboard.SendF12WithSendInput()
		if !success {
			c.log.Warn("SendF12WithSendInput failed, falling back to keybd_event")
			c.keyboard.SendF12()
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
			// Return the result even on error so caller can see what happened
			return eventResult, err
		}

		// Copy event result into our result
		result = eventResult
	}

	// Close dialogs and handle post-compilation events
	c.log.Debug("Closing dialogs and SIMPL Windows...")

	// First, close the "Compile Complete" dialog if it's still open
	if compileCompleteHwnd != 0 {
		c.windowMgr.CloseWindow(compileCompleteHwnd, "Compile Complete dialog")
		time.Sleep(timeouts.StabilityCheckInterval)
	}

	// Close main window and handle any confirmation dialogs via events
	if opts.Hwnd != 0 {
		c.windowMgr.CloseWindow(opts.Hwnd, "SIMPL Windows")

		// Handle confirmation dialog that may appear when closing
		if pid != 0 {
			if err := c.handlePostCompilationEvents(); err != nil {
				// Return the result we have so far, even if cleanup failed
				return result, err
			}
		}

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
	// Use custom timeout if specified, otherwise use default 5 minutes
	compilationTimeout := timeouts.CompilationCompleteTimeout
	if opts.CompilationTimeout > 0 {
		compilationTimeout = opts.CompilationTimeout
	}
	timeout := time.NewTimer(compilationTimeout)
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
			case dialogIncompleteSymbols:
				// Fatal error - compilation cannot proceed
				c.log.Error("Incomplete Symbols detected", slog.String("title", ev.Title))
				c.log.Info("The program contains incomplete symbols and cannot be compiled.")
				c.log.Info("Please fix the incomplete symbols in SIMPL Windows before attempting to compile.")

				// Extract error details
				childInfos := c.windowMgr.CollectChildInfos(ev.Hwnd)
				for _, ci := range childInfos {
					if ci.ClassName == "Edit" && len(ci.Text) > 50 {
						c.log.Info("Details", slog.String("text", ci.Text))
						break
					}
				}

				// Close the dialog before returning
				c.windowMgr.CloseWindow(ev.Hwnd, "Incomplete Symbols dialog")

				// Return the SIMPL Windows hwnd so test cleanup can close it properly
				// Return a result indicating compilation failed
				return opts.Hwnd, &CompileResult{
					Errors:    1,
					HasErrors: true,
					ErrorMessages: []string{
						"Incomplete Symbols: The program contains incomplete symbols and cannot be compiled",
					},
				}, fmt.Errorf("program contains incomplete symbols and cannot be compiled")

			case dialogConvertCompile:
				// Save prompt - auto-confirm
				c.log.Debug("Handling 'Convert/Compile' dialog")
				_ = c.windowMgr.SetForeground(ev.Hwnd)
				time.Sleep(timeouts.DialogResponseDelay)
				c.keyboard.SendEnter()
				c.log.Info("Auto-confirmed save prompt")

			case dialogCommentedOutSymbols:
				// Confirmation dialog - auto-confirm
				c.log.Debug("Handling 'Commented out Symbols and/or Devices' dialog")
				_ = c.windowMgr.SetForeground(ev.Hwnd)
				time.Sleep(timeouts.DialogResponseDelay)
				c.keyboard.SendEnter()
				c.log.Info("Auto-confirmed commented symbols dialog")

			case dialogCompiling:
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

			case dialogCompileComplete:
				// Compilation finished - parse results
				if !compileCompleteDetected {
					c.log.Debug("Detected 'Compile Complete' dialog - parsing results")
					compileCompleteHwnd = ev.Hwnd

					// Parse statistics from dialog
					childInfos := c.windowMgr.CollectChildInfos(ev.Hwnd)
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

			case dialogProgramCompilation:
				// Detailed error/warning/notice messages
				if programCompHwnd == 0 {
					c.log.Debug("Detected 'Program Compilation' dialog")
					c.log.Info("Gathering details...")
					programCompHwnd = ev.Hwnd
				}

			case dialogOperationComplete:
				// Sometimes appears - close it
				c.log.Debug("Detected 'Operation Complete' dialog - closing")
				c.windowMgr.CloseWindow(ev.Hwnd, ev.Title)
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
			return opts.Hwnd, &CompileResult{
				Errors:    1,
				HasErrors: true,
				ErrorMessages: []string{
					"Compilation timeout: did not detect 'Compile Complete' dialog within 5 minutes",
				},
			}, fmt.Errorf("compilation timeout: did not detect 'Compile Complete' dialog within 5 minutes")
		}
	}
}

// parseDetailedMessages extracts error/warning/notice messages from Program Compilation dialog
func (c *Compiler) parseDetailedMessages(hwnd uintptr) (warnings, notices, errors []string) {
	childInfos := c.windowMgr.CollectChildInfos(hwnd)

	var lastType string // Track the type of the last message: "ERROR", "WARNING", or "NOTICE"

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
			case strings.HasPrefix(lineUpper, "ERROR\t") || strings.HasPrefix(lineUpper, "ERROR "):
				errors = append(errors, line)
				lastType = msgTypeError
			case strings.HasPrefix(lineUpper, "WARNING\t") || strings.HasPrefix(lineUpper, "WARNING "):
				warnings = append(warnings, line)
				lastType = msgTypeWarning
			case strings.HasPrefix(lineUpper, "NOTICE\t") || strings.HasPrefix(lineUpper, "NOTICE "):
				notices = append(notices, line)
				lastType = msgTypeNotice
			default:
				// Continuation of previous message - append to the last type that was seen
				switch lastType {
				case msgTypeError:
					if len(errors) > 0 {
						errors[len(errors)-1] += " " + line
					}
				case msgTypeWarning:
					if len(warnings) > 0 {
						warnings[len(warnings)-1] += " " + line
					}
				case msgTypeNotice:
					if len(notices) > 0 {
						notices[len(notices)-1] += " " + line
					}
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
				slog.String("type", "error"),
				slog.String("message", msg),
			)
		}
	}

	if len(warningMsgs) > 0 {
		c.log.Info("")
		c.log.Info("Warning messages:")
		for i, msg := range warningMsgs {
			c.log.Info(fmt.Sprintf("  %d. %s", i+1, msg),
				slog.Int("number", i+1),
				slog.String("type", "warning"),
				slog.String("message", msg),
			)
		}
	}

	if len(noticeMsgs) > 0 {
		c.log.Info("")
		c.log.Info("Notice messages:")
		for i, msg := range noticeMsgs {
			c.log.Info(fmt.Sprintf("  %d. %s", i+1, msg),
				slog.Int("number", i+1),
				slog.String("type", "notice"),
				slog.String("message", msg),
			)
		}
	}

	// Add trailing blank line if any messages were displayed
	if len(errorMsgs) > 0 || len(warningMsgs) > 0 || len(noticeMsgs) > 0 {
		c.log.Info("")
	}
}

// handlePreCompilationDialogs checks for and dismisses dialogs that may block compilation
// This includes "Operation Complete" dialog that can appear during SIMPL Windows startup
func (c *Compiler) handlePreCompilationDialogs() error {
	// Short timeout - check if there are any dialogs already present
	timeout := time.NewTimer(timeouts.WindowMessageDelay)
	defer timeout.Stop()

	for {
		select {
		case ev := <-windows.MonitorCh:
			c.log.Debug("Received pre-compilation event",
				slog.String("title", ev.Title),
				slog.Uint64("hwnd", uint64(ev.Hwnd)))

			// Handle dialogs that may block compilation
			switch ev.Title {
			case dialogOperationComplete:
				c.log.Debug("Detected 'Operation Complete' dialog - closing")
				c.log.Info("Handling pre-compilation 'Operation Complete' dialog")
				c.windowMgr.CloseWindow(ev.Hwnd, dialogOperationComplete)
				time.Sleep(timeouts.WindowMessageDelay)

			default:
				// Log but don't handle other dialogs here
				c.log.Trace("Ignoring pre-compilation dialog", slog.String("title", ev.Title))
			}

		case <-timeout.C:
			// Timeout is fine - no blocking dialogs present
			return nil
		}
	}
}

// handlePostCompilationEvents waits for and handles any post-compilation dialogs (like Confirmation)
func (c *Compiler) handlePostCompilationEvents() error {
	// Short timeout - if no confirmation dialog appears, that's fine
	timeout := time.NewTimer(timeouts.DialogConfirmationTimeout)
	defer timeout.Stop()

	select {
	case ev := <-windows.MonitorCh:
		c.log.Debug("Received post-compilation event",
			slog.String("title", ev.Title),
			slog.Uint64("hwnd", uint64(ev.Hwnd)))

		// Only handle Confirmation dialog here
		if ev.Title == dialogConfirmation {
			c.log.Debug("Detected 'Confirmation' dialog - clicking No")
			c.log.Info("Handling confirmation dialog")

			if c.controlReader.FindAndClickButton(ev.Hwnd, "&No") {
				c.log.Debug("Successfully clicked 'No' button")
				time.Sleep(timeouts.WindowMessageDelay)
			} else {
				c.log.Warn("Could not find 'No' button, trying to close dialog")
				c.windowMgr.CloseWindow(ev.Hwnd, "Confirmation dialog")
				time.Sleep(timeouts.WindowMessageDelay)
			}
		}

	case <-timeout.C:
		// Timeout is fine - dialog may not appear
	}

	return nil
}
