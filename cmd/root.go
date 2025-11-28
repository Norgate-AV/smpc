package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Norgate-AV/smpc/internal/compiler"
	"github.com/Norgate-AV/smpc/internal/simpl"
	"github.com/Norgate-AV/smpc/internal/version"
	"github.com/Norgate-AV/smpc/internal/windows"
	"github.com/spf13/cobra"
)

var (
	verbose      bool
	recompileAll bool
	// Track SIMPL Windows for cleanup on interrupt
	simplHwnd uintptr
	simplPid  uint32
)

var RootCmd = &cobra.Command{
	Use:     "smpc <file-path>",
	Short:   "smpc - Automate compilation of .smw files",
	Version: version.GetVersion(),
	Args:    validateSmwFile,
	RunE:    Execute,
}

func init() {
	// Set custom version template to show full version info
	RootCmd.SetVersionTemplate(`{{printf "%s\n" .Version}}`)

	// Add flags
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "V", false, "enable verbose output")
	RootCmd.PersistentFlags().BoolVarP(&recompileAll, "recompile-all", "r", false, "trigger Recompile All (Alt+F12) instead of Compile (F12)")
}

// validateSmwFile validates that exactly one argument is provided and it has .smw extension
func validateSmwFile(cmd *cobra.Command, args []string) error {
	if err := cobra.ExactArgs(1)(cmd, args); err != nil {
		return err
	}

	if filepath.Ext(args[0]) != ".smw" {
		return fmt.Errorf("file must have .smw extension")
	}

	return nil
}

func Execute(cmd *cobra.Command, args []string) error {
	slog.Debug("Execute() called", "args", args)
	slog.Debug("Flags set", "verbose", verbose, "recompileAll", recompileAll)

	// Check if running as admin
	slog.Debug("Checking elevation status")
	if !windows.IsElevated() {
		slog.Info("This program requires administrator privileges")
		slog.Info("Relaunching as administrator")
		slog.Debug("Not elevated, relaunching as admin")

		if err := windows.RelaunchAsAdmin(); err != nil {
			slog.Error("RelaunchAsAdmin failed", "error", err)
			return fmt.Errorf("error relaunching as admin: %w", err)
		}

		// Exit this instance, the elevated one will continue
		slog.Debug("Relaunched successfully, exiting non-elevated instance")
		return nil
	}

	slog.Info("Running with administrator privileges ✓")
	slog.Debug("Running with administrator privileges")

	// Get the file path from the command arguments
	filePath := args[0]
	slog.Debug("Processing file", "path", filePath)

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("error resolving file path: %w", err)
	}

	// Start background window monitor to observe dialogs and window changes in real time
	slog.Debug("Creating context and starting monitor goroutine")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure the goroutine is cleaned up
	go simpl.StartMonitoring(ctx)
	slog.Debug("Monitor goroutine started")

	// Open the file with SIMPL Windows application using elevated privileges
	// SW_SHOWNORMAL = 1
	slog.Debug("Launching SIMPL Windows with file", "path", absPath)
	if err := windows.ShellExecute(0, "runas", simpl.SIMPL_WINDOWS_PATH, absPath, "", 1); err != nil {
		slog.Error("ShellExecute failed", "error", err)
		return fmt.Errorf("error opening file: %w", err)
	}
	slog.Debug("SIMPL Windows launched successfully")

	// At this point, the SIMPL Windows process should have started
	// Meaning, if we fail from any point onward, we need to make sure we
	// clean up by closing SIMPL Windows

	// Try to get the PID early so signal handlers can use it for cleanup
	// Give it a moment for the process to actually start
	time.Sleep(500 * time.Millisecond)
	earlyPid := simpl.GetPid()
	if earlyPid != 0 {
		simplPid = earlyPid
		slog.Debug("Early PID detection", "pid", earlyPid)
	}

	// Set up Windows console control handler to catch window close events
	// This is more reliable than signal handling on Windows
	windows.SetConsoleCtrlHandler(func(ctrlType uint32) uintptr {
		slog.Debug("Received console control event", "type", windows.GetCtrlTypeName(ctrlType), "code", ctrlType)
		slog.Info("Received console control event, cleaning up", "type", windows.GetCtrlTypeName(ctrlType))

		// Try to cleanup using hwnd if we have it
		if simplHwnd != 0 {
			slog.Debug("Cleaning up SIMPL Windows", "hwnd", simplHwnd)
			simpl.Cleanup(simplHwnd)
		} else if simplPid != 0 {
			// If we don't have hwnd yet but have PID, force terminate
			slog.Debug("Force terminating SIMPL Windows", "pid", simplPid)
			windows.TerminateProcess(simplPid)
		} else {
			// Last resort - try to find and kill any smpwin.exe process we may have started
			slog.Debug("Attempting to find and terminate SIMPL Windows process")
			pid := simpl.GetPid()
			if pid != 0 {
				slog.Debug("Found SIMPL Windows PID, terminating", "pid", pid)
				windows.TerminateProcess(pid)
			} else {
				slog.Debug("Could not find SIMPL Windows process to terminate")
			}
		}

		slog.Debug("Cleanup completed, exiting")

		// Must call os.Exit to actually terminate the process
		// Otherwise the handler returns and execution continues
		os.Exit(130)

		// Return TRUE to indicate we handled the event
		// (this won't actually be reached due to os.Exit, but required for the function signature)
		return 1
	})

	// Set up signal handler immediately to catch Ctrl+C during window wait
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		slog.Debug("Received signal", "signal", sig)
		slog.Info("Received interrupt signal, cleaning up")
		slog.Debug("Running cleanup due to interrupt")

		// Try to cleanup using hwnd if we have it
		if simplHwnd != 0 {
			slog.Debug("Cleaning up SIMPL Windows", "hwnd", simplHwnd)
			simpl.Cleanup(simplHwnd)
		} else if simplPid != 0 {
			// If we don't have hwnd yet but have PID, force terminate
			slog.Debug("Force terminating SIMPL Windows", "pid", simplPid)
			windows.TerminateProcess(simplPid)
		} else {
			// Last resort - try to find and kill any smpwin.exe process we may have started
			slog.Debug("Attempting to find and terminate SIMPL Windows process")
			pid := simpl.GetPid()
			if pid != 0 {
				slog.Debug("Found SIMPL Windows PID, terminating", "pid", pid)
				windows.TerminateProcess(pid)
			}
		}

		slog.Debug("Cleanup completed, exiting")
		os.Exit(130) // Standard exit code for Ctrl+C
	}()
	slog.Debug("Signal handler registered (early)")

	// Wait for the main window to appear (with a 1 minute timeout)
	slog.Info("Waiting for SIMPL Windows to fully launch...")
	slog.Debug("Waiting for SIMPL Windows window to appear")
	hwnd, found := simpl.WaitForAppear(60 * time.Second)
	if !found {
		slog.Error("Timeout waiting for window to appear")
		return fmt.Errorf("timed out waiting for SIMPL Windows window to appear")
	}
	slog.Debug("Window appeared", "hwnd", hwnd)

	// Store hwnd for signal handler cleanup
	simplHwnd = hwnd
	slog.Debug("Stored hwnd for signal handler", "hwnd", simplHwnd)

	// Set up deferred cleanup to ensure SIMPL Windows is closed on exit
	defer simpl.Cleanup(hwnd)
	slog.Debug("Cleanup deferred")

	// Wait for the window to be fully ready and responsive (with a 30 second timeout)
	slog.Debug("Waiting for window to be ready")
	if !simpl.WaitForReady(hwnd, 30*time.Second) {
		slog.Error("Window not responding properly")
		return fmt.Errorf("window appeared but is not responding properly")
	}
	slog.Debug("Window is ready")

	// Small extra delay to allow UI to finish settling
	slog.Info("Waiting a few extra seconds for UI to settle...")
	time.Sleep(5 * time.Second)

	slog.Info("Successfully opened file", "path", absPath)

	// Detect SIMPL Windows process PID for dialog monitoring
	slog.Debug("Getting SIMPL Windows process PID")
	pid := simpl.GetPid()
	if pid == 0 {
		slog.Warn("Could not determine PID")
		slog.Info("Warning: Could not determine SIMPL Windows process PID; dialog detection may be limited")
	} else {
		slog.Debug("SIMPL Windows PID detected", "pid", pid)
		simplPid = pid // Store for signal handler
	}

	// Check for "Operation Complete" dialog that may appear after loading the file
	// This dialog must be dismissed before we can send compile keystrokes
	if pid != 0 && windows.MonitorCh != nil {
		slog.Info("Checking for 'Operation Complete' dialog...")
		slog.Debug("Checking for Operation Complete dialog")
		ev, ok := windows.WaitOnMonitor(3*time.Second,
			func(e windows.WindowEvent) bool { return strings.EqualFold(e.Title, "Operation Complete") },
			func(e windows.WindowEvent) bool {
				return strings.Contains(strings.ToLower(e.Title), "operation complete")
			},
		)

		if ok {
			slog.Debug("Operation Complete dialog detected", "title", ev.Title)
			slog.Info("Detected dialog: " + ev.Title)
			slog.Info("Dismissing 'Operation Complete' dialog...")
			windows.CloseWindow(ev.Hwnd, ev.Title)
			time.Sleep(500 * time.Millisecond)
			slog.Debug("Dialog dismissed")
		} else {
			slog.Debug("No Operation Complete dialog detected")
		}
	}

	// Confirm elevation before sending keystrokes
	if windows.IsElevated() {
		slog.Debug("Process is elevated, proceeding with keystroke injection")
	} else {
		slog.Warn("Process is NOT elevated, keystroke injection may fail")
	}

	// Bring window to foreground and send F12 (compile)
	slog.Debug("Bringing window to foreground...")
	_ = windows.SetForeground(hwnd)

	slog.Info("Waiting for window to receive focus...")
	time.Sleep(1 * time.Second)

	// Use keybd_event (older API that works with SIMPL Windows)
	slog.Debug("Preparing to send keystroke")
	var keystrokeSent bool
	if recompileAll {
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

	if keystrokeSent {
		slog.Debug("Starting compile monitoring")
		// Detect "Incomplete Symbols" error dialog - this is a fatal error
		if pid != 0 && windows.MonitorCh != nil {
			slog.Info("Checking for 'Incomplete Symbols' error dialog...")
			ev, ok := windows.WaitOnMonitor(2*time.Second,
				func(e windows.WindowEvent) bool { return strings.EqualFold(e.Title, "Incomplete Symbols") },
				func(e windows.WindowEvent) bool { return strings.Contains(strings.ToLower(e.Title), "incomplete") },
			)

			if ok {
				slog.Error("ERROR: Incomplete Symbols detected", "title", ev.Title)
				slog.Info("The program contains incomplete symbols and cannot be compiled.")
				slog.Info("Please fix the incomplete symbols in SIMPL Windows before attempting to compile.")

				// Extract error details from the dialog
				childInfos := windows.CollectChildInfos(ev.Hwnd)
				for _, ci := range childInfos {
					if ci.ClassName == "Edit" && len(ci.Text) > 50 {
						slog.Info("Details", "text", ci.Text)
						break
					}
				}

				return fmt.Errorf("program contains incomplete symbols and cannot be compiled")
			}
		}

		// Detect save prompt ("Convert/Compile") via monitor channel and auto-confirm "Yes"
		if pid != 0 && windows.MonitorCh != nil {
			slog.Info("Watching for 'Convert/Compile' save prompt...")
			ev, ok := windows.WaitOnMonitor(5*time.Second,
				func(e windows.WindowEvent) bool { return strings.EqualFold(e.Title, "Convert/Compile") },
				func(e windows.WindowEvent) bool { return strings.Contains(strings.ToLower(e.Title), "convert/compile") },
			)

			if ok {
				slog.Info("Detected save prompt", "title", ev.Title)
				_ = windows.SetForeground(ev.Hwnd)
				time.Sleep(300 * time.Millisecond)
				_ = windows.SendEnter()
				slog.Info("Auto-confirmed save prompt with 'Yes'")
			} else {
				slog.Debug("Save prompt not detected within timeout")
			}
		}

		// Detect "Commented out Symbols and/or Devices" dialog and auto-confirm "Yes"
		if pid != 0 && windows.MonitorCh != nil {
			slog.Info("Watching for 'Commented out Symbols' dialog...")
			ev, ok := windows.WaitOnMonitor(5*time.Second,
				func(e windows.WindowEvent) bool {
					return strings.EqualFold(e.Title, "Commented out Symbols and/or Devices")
				},
				func(e windows.WindowEvent) bool { return strings.Contains(strings.ToLower(e.Title), "commented out") },
			)

			if ok {
				slog.Info("Detected dialog", "title", ev.Title)
				_ = windows.SetForeground(ev.Hwnd)
				time.Sleep(300 * time.Millisecond)
				_ = windows.SendEnter()
				slog.Info("Auto-confirmed 'Commented out Symbols' dialog with 'Yes'")
			} else {
				slog.Debug("'Commented out Symbols' dialog not detected within timeout")
			}
		}

		// Detect compile progress start ("Compiling...") via monitor channel
		if pid != 0 && windows.MonitorCh != nil {
			slog.Info("Waiting for 'Compiling...' dialog...")
			ev, ok := windows.WaitOnMonitor(30*time.Second,
				func(e windows.WindowEvent) bool { return strings.EqualFold(e.Title, "Compiling...") },
				func(e windows.WindowEvent) bool { return strings.Contains(strings.ToLower(e.Title), "compiling") },
			)

			if ok {
				slog.Info("Compile started", "title", ev.Title)
			} else {
				slog.Warn("Did not detect 'Compiling...' dialog within timeout")
			}
		}

		// Variables to store compile results
		var warnings, notices, errors int
		var compileTime float64
		warningMessages := []string{}
		noticeMessages := []string{}
		errorMessages := []string{}
		hasErrors := false
		var compileCompleteHwnd uintptr

		// Detect and parse Compile Complete dialog
		if pid != 0 && windows.MonitorCh != nil {
			slog.Info("Waiting for 'Compile Complete' dialog...")
			ev, ok := windows.WaitOnMonitor(5*time.Minute, // Increased timeout for large programs
				func(e windows.WindowEvent) bool { return strings.EqualFold(e.Title, "Compile Complete") },
				func(e windows.WindowEvent) bool {
					return strings.Contains(strings.ToLower(e.Title), "compile complete")
				},
			)

			if ok {
				slog.Info("Detected", "title", ev.Title)
				compileCompleteHwnd = ev.Hwnd // Store for later closing
				childInfos := windows.CollectChildInfos(ev.Hwnd)
				slog.Debug("Child controls in dialog", "title", ev.Title)

				for _, ci := range childInfos {
					slog.Debug("Child control", "class", ci.ClassName, "text", ci.Text, "length", len(ci.Text))
				} // Parse stats from Compile Complete dialog
				for _, ci := range childInfos {
					text := strings.ReplaceAll(ci.Text, "\r\n", "\n")
					lines := strings.SplitSeq(text, "\n")
					for t := range lines {
						t = strings.TrimSpace(t)
						if t == "" {
							continue
						}
						if n, ok := compiler.ParseStatLine(t, "Program Warnings"); ok {
							warnings = n
						}
						if n, ok := compiler.ParseStatLine(t, "Program Notices"); ok {
							notices = n
						}
						if n, ok := compiler.ParseStatLine(t, "Program Errors"); ok {
							errors = n
							if n > 0 {
								hasErrors = true
							}
						}
						if secs, ok := compiler.ParseCompileTimeLine(t); ok {
							compileTime = secs
						}
					}
				}
			} else {
				return fmt.Errorf("compilation timeout: did not detect 'Compile Complete' dialog within 5 minutes")
			}
		}

		// Detect and parse Program Compilation dialog (if warnings/notices/errors exist)
		if pid != 0 && windows.MonitorCh != nil && (warnings > 0 || notices > 0 || errors > 0) {
			slog.Info("Waiting for 'Program Compilation' dialog...")
			ev, ok := windows.WaitOnMonitor(10*time.Second,
				func(e windows.WindowEvent) bool { return strings.EqualFold(e.Title, "Program Compilation") },
				func(e windows.WindowEvent) bool {
					return strings.Contains(strings.ToLower(e.Title), "program compilation")
				},
			)

			if ok {
				slog.Info("Detected", "title", ev.Title)
				childInfos := windows.CollectChildInfos(ev.Hwnd)
				slog.Debug("Child controls in dialog", "title", ev.Title)

				for _, ci := range childInfos {
					slog.Debug("Child control", "class", ci.ClassName, "text", ci.Text, "length", len(ci.Text))
				}

				// Extract messages from ListBox and categorize them
				for _, ci := range childInfos {
					if ci.ClassName == "ListBox" && len(ci.Items) > 0 {
						// Use items directly instead of splitting text
						for _, line := range ci.Items {
							line = strings.TrimSpace(line)
							if line == "" {
								continue
							}
							// Categorize based on prefix
							lineUpper := strings.ToUpper(line)
							if strings.HasPrefix(lineUpper, "ERROR") {
								errorMessages = append(errorMessages, line)
								hasErrors = true
							} else if strings.HasPrefix(lineUpper, "WARNING") {
								warningMessages = append(warningMessages, line)
							} else if strings.HasPrefix(lineUpper, "NOTICE") {
								noticeMessages = append(noticeMessages, line)
							} else {
								// If it doesn't have a prefix, it's likely a continuation of the previous message
								// Append to the last message in the appropriate list
								if len(errorMessages) > 0 {
									errorMessages[len(errorMessages)-1] += " " + line
								} else if len(warningMessages) > 0 {
									warningMessages[len(warningMessages)-1] += " " + line
								} else if len(noticeMessages) > 0 {
									noticeMessages[len(noticeMessages)-1] += " " + line
								}
							}
						}
					}
				}

				if len(errorMessages) > 0 {
					slog.Info("Error messages:")
					for i, msg := range errorMessages {
						slog.Info("", "number", i+1, "message", msg)
					}
				}

				if len(warningMessages) > 0 {
					slog.Info("Warning messages:")
					for i, msg := range warningMessages {
						slog.Info("", "number", i+1, "message", msg)
					}
				}

				if len(noticeMessages) > 0 {
					slog.Info("Notice messages:")
					for i, msg := range noticeMessages {
						slog.Info("", "number", i+1, "message", msg)
					}
				}
			} else {
				slog.Debug("Program Compilation dialog not detected (may not have appeared)")
			}
		}

		// Close SIMPL Windows after successful compilation
		slog.Info("Closing dialogs and SIMPL Windows...")

		// First, close the "Compile Complete" dialog if it's still open
		if compileCompleteHwnd != 0 {
			windows.CloseWindow(compileCompleteHwnd, "Compile Complete dialog")
			time.Sleep(500 * time.Millisecond)
		}

		// Check for and handle "Confirmation" dialog that may appear when closing
		if pid != 0 && windows.MonitorCh != nil {
			ev, ok := windows.WaitOnMonitor(2*time.Second,
				func(e windows.WindowEvent) bool { return strings.EqualFold(e.Title, "Confirmation") },
				func(e windows.WindowEvent) bool { return strings.Contains(strings.ToLower(e.Title), "confirmation") },
			)

			if ok {
				slog.Info("Detected dialog (clicking 'No' to close without saving)", "title", ev.Title)
				// Find and click the "No" button directly
				if windows.FindAndClickButton(ev.Hwnd, "&No") {
					slog.Debug("Successfully clicked 'No' button")
					time.Sleep(500 * time.Millisecond)
				} else {
					slog.Warn("Could not find 'No' button, trying to close dialog")
					windows.CloseWindow(ev.Hwnd, "Confirmation dialog")
					time.Sleep(500 * time.Millisecond)
				}
			}
		}

		// Now close the main SIMPL Windows application (defer will also do this, but we do it here for clean output)
		if hwnd != 0 {
			windows.CloseWindow(hwnd, "SIMPL Windows")
			time.Sleep(1 * time.Second)
			slog.Info("SIMPL Windows closed successfully")
		}

		// Print final summary
		if pid != 0 && windows.MonitorCh != nil {
			slog.Info("=== Compile Summary ===")
			if errors > 0 {
				slog.Info("Errors", "count", errors)
			}
			slog.Info("Warnings", "count", warnings)
			slog.Info("Notices", "count", notices)
			slog.Info("Compile Time", "seconds", compileTime)
			slog.Info("=======================")
		}

		slog.Info("Press Enter to exit...")
		fmt.Scanln()

		// Exit with error code if compilation failed
		if hasErrors {
			slog.Error("Compilation failed", "errors", errors)
			return fmt.Errorf("compilation failed with %d error(s)", errors)
		}

		slog.Debug("Compilation completed successfully")
	}

	slog.Debug("Execute() completed successfully")
	return nil
}
