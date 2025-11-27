package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Norgate-AV/smpc/internal/compiler"
	"github.com/Norgate-AV/smpc/internal/simpl"
	"github.com/Norgate-AV/smpc/internal/windows"
)

func Execute() error {
	// Check if running as admin
	if !windows.IsElevated() {
		fmt.Println("This program requires administrator privileges.")
		fmt.Println("Relaunching as administrator...")

		if err := windows.RelaunchAsAdmin(); err != nil {
			return fmt.Errorf("error relaunching as admin: %w", err)
		}

		// Exit this instance, the elevated one will continue
		return nil
	}

	fmt.Println("Running with administrator privileges ✓")

	// Start background window monitor focused on SIMPL Windows process (if available)
	// It will help us observe dialogs and window changes in real time.
	go func() {
		// Try to obtain PID repeatedly until found, then monitor that PID
		var pid uint32

		for i := 0; i < 50 && pid == 0; i++ { // up to ~5s
			pid = simpl.GetPid()
			if pid == 0 {
				time.Sleep(100 * time.Millisecond)
			}
		}

		// Init channel
		windows.MonitorCh = make(chan windows.WindowEvent, 64)
		if pid == 0 {
			fmt.Println("[DEBUG] Window monitor falling back to all processes (SIMPL PID not found yet)")
			windows.StartWindowMonitor(0, 500*time.Millisecond)
		} else {
			fmt.Printf("[DEBUG] Window monitor targeting SIMPL PID %d\n", pid)
			windows.StartWindowMonitor(pid, 500*time.Millisecond)
		}
	}()

	// Check if a file path argument was provided
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: smpc <file-path>")
	}

	// Get the file path from the first command line argument
	filePath := os.Args[1]

	// Check if the file has .smw extension
	if filepath.Ext(filePath) != ".smw" {
		return fmt.Errorf("file must have .smw extension")
	}

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("error resolving file path: %w", err)
	}

	// Open the file with SIMPL Windows application using elevated privileges
	// SW_SHOWNORMAL = 1
	if err := windows.ShellExecute(0, "runas", simpl.SIMPL_WINDOWS_PATH, absPath, "", 1); err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}

	// Wait for the main window to appear (with a 1 minute timeout)
	fmt.Printf("Waiting for SIMPL Windows to fully launch...\n")
	hwnd, found := simpl.WaitForAppear(60 * time.Second)
	if !found {
		return fmt.Errorf("timed out waiting for SIMPL Windows window to appear")
	}

	// Wait for the window to be fully ready and responsive (with a 30 second timeout)
	if !simpl.WaitForReady(hwnd, 30*time.Second) {
		return fmt.Errorf("window appeared but is not responding properly")
	}

	// Small extra delay to allow UI to finish settling
	fmt.Println("Waiting a few extra seconds for UI to settle...")
	time.Sleep(5 * time.Second)

	fmt.Printf("Successfully opened file: %s\n", absPath)

	// Confirm elevation before sending keystrokes
	if windows.IsElevated() {
		fmt.Println("[DEBUG] Process is elevated, proceeding with keystroke injection")
	} else {
		fmt.Println("[DEBUG] WARNING - Process is NOT elevated, keystroke injection may fail")
	}

	// Bring window to foreground and send F12 (compile)
	_ = windows.SetForeground(hwnd)

	fmt.Println("Waiting for window to receive focus...")
	time.Sleep(1 * time.Second)

	// Use keybd_event (older API that works with SIMPL Windows)
	fmt.Println("Sending F12 keystroke to trigger compile...")
	if windows.SendF12() {
		fmt.Println("Successfully sent F12 keystroke")

		// Detect SIMPL Windows process PID
		pid := simpl.GetPid()
		if pid == 0 {
			fmt.Println("Warning: Could not determine SIMPL Windows process PID; dialog detection may be limited")
		}

		// Detect "Incomplete Symbols" error dialog - this is a fatal error
		if pid != 0 && windows.MonitorCh != nil {
			fmt.Println("Checking for 'Incomplete Symbols' error dialog...")
			ev, ok := windows.WaitOnMonitor(2*time.Second,
				func(e windows.WindowEvent) bool { return strings.EqualFold(e.Title, "Incomplete Symbols") },
				func(e windows.WindowEvent) bool { return strings.Contains(strings.ToLower(e.Title), "incomplete") },
			)

			if ok {
				fmt.Printf("\n*** ERROR: %s ***\n", ev.Title)
				fmt.Println("The program contains incomplete symbols and cannot be compiled.")
				fmt.Println("Please fix the incomplete symbols in SIMPL Windows before attempting to compile.")

				// Extract error details from the dialog
				childInfos := windows.CollectChildInfos(ev.Hwnd)
				for _, ci := range childInfos {
					if ci.ClassName == "Edit" && len(ci.Text) > 50 {
						fmt.Printf("\nDetails:\n%s\n", ci.Text)
						break
					}
				}

				return fmt.Errorf("program contains incomplete symbols and cannot be compiled")
			}
		}

		// Detect save prompt ("Convert/Compile") via monitor channel and auto-confirm "Yes"
		if pid != 0 && windows.MonitorCh != nil {
			fmt.Println("Watching for 'Convert/Compile' save prompt...")
			ev, ok := windows.WaitOnMonitor(5*time.Second,
				func(e windows.WindowEvent) bool { return strings.EqualFold(e.Title, "Convert/Compile") },
				func(e windows.WindowEvent) bool { return strings.Contains(strings.ToLower(e.Title), "convert/compile") },
			)

			if ok {
				fmt.Printf("Detected save prompt: %s\n", ev.Title)
				_ = windows.SetForeground(ev.Hwnd)
				time.Sleep(300 * time.Millisecond)
				_ = windows.SendEnter()
				fmt.Println("Auto-confirmed save prompt with 'Yes'")
			} else {
				fmt.Println("[DEBUG] Save prompt not detected within timeout")
			}
		}

		// Detect "Commented out Symbols and/or Devices" dialog and auto-confirm "Yes"
		if pid != 0 && windows.MonitorCh != nil {
			fmt.Println("Watching for 'Commented out Symbols' dialog...")
			ev, ok := windows.WaitOnMonitor(5*time.Second,
				func(e windows.WindowEvent) bool {
					return strings.EqualFold(e.Title, "Commented out Symbols and/or Devices")
				},
				func(e windows.WindowEvent) bool { return strings.Contains(strings.ToLower(e.Title), "commented out") },
			)

			if ok {
				fmt.Printf("Detected dialog: %s\n", ev.Title)
				_ = windows.SetForeground(ev.Hwnd)
				time.Sleep(300 * time.Millisecond)
				_ = windows.SendEnter()
				fmt.Println("Auto-confirmed 'Commented out Symbols' dialog with 'Yes'")
			} else {
				fmt.Println("[DEBUG] 'Commented out Symbols' dialog not detected within timeout")
			}
		}

		// Detect compile progress start ("Compiling...") via monitor channel
		if pid != 0 && windows.MonitorCh != nil {
			fmt.Println("Waiting for 'Compiling...' dialog...")
			ev, ok := windows.WaitOnMonitor(30*time.Second,
				func(e windows.WindowEvent) bool { return strings.EqualFold(e.Title, "Compiling...") },
				func(e windows.WindowEvent) bool { return strings.Contains(strings.ToLower(e.Title), "compiling") },
			)

			if ok {
				fmt.Printf("Compile started: %s\n", ev.Title)
			} else {
				fmt.Println("Warning: Did not detect 'Compiling...' dialog within timeout")
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
			fmt.Println("Waiting for 'Compile Complete' dialog...")
			ev, ok := windows.WaitOnMonitor(5*time.Minute, // Increased timeout for large programs
				func(e windows.WindowEvent) bool { return strings.EqualFold(e.Title, "Compile Complete") },
				func(e windows.WindowEvent) bool {
					return strings.Contains(strings.ToLower(e.Title), "compile complete")
				},
			)

			if ok {
				fmt.Printf("Detected: %s\n", ev.Title)
				compileCompleteHwnd = ev.Hwnd // Store for later closing
				childInfos := windows.CollectChildInfos(ev.Hwnd)
				fmt.Printf("[DEBUG] Child controls in %s dialog:\n", ev.Title)

				for _, ci := range childInfos {
					fmt.Printf("[DEBUG] class=%q text=%q (length=%d)\n", ci.ClassName, ci.Text, len(ci.Text))
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
			fmt.Println("Waiting for 'Program Compilation' dialog...")
			ev, ok := windows.WaitOnMonitor(10*time.Second,
				func(e windows.WindowEvent) bool { return strings.EqualFold(e.Title, "Program Compilation") },
				func(e windows.WindowEvent) bool {
					return strings.Contains(strings.ToLower(e.Title), "program compilation")
				},
			)

			if ok {
				fmt.Printf("Detected: %s\n", ev.Title)
				childInfos := windows.CollectChildInfos(ev.Hwnd)
				fmt.Printf("[DEBUG] Child controls in %s dialog:\n", ev.Title)

				for _, ci := range childInfos {
					fmt.Printf("[DEBUG] class=%q text=%q (length=%d)\n", ci.ClassName, ci.Text, len(ci.Text))
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
					fmt.Println("\nError messages:")
					for i, msg := range errorMessages {
						fmt.Printf("  %d. %s\n", i+1, msg)
					}
				}

				if len(warningMessages) > 0 {
					fmt.Println("\nWarning messages:")
					for i, msg := range warningMessages {
						fmt.Printf("  %d. %s\n", i+1, msg)
					}
				}

				if len(noticeMessages) > 0 {
					fmt.Println("\nNotice messages:")
					for i, msg := range noticeMessages {
						fmt.Printf("  %d. %s\n", i+1, msg)
					}
				}
			} else {
				fmt.Println("Note: Program Compilation dialog not detected (may not have appeared)")
			}
		}

		// Print final summary
		if pid != 0 && windows.MonitorCh != nil {
			fmt.Printf("\n=== Compile Summary ===\n")
			if errors > 0 {
				fmt.Printf("Errors: %d\n", errors)
			}
			fmt.Printf("Warnings: %d\n", warnings)
			fmt.Printf("Notices: %d\n", notices)
			fmt.Printf("Compile Time: %.2f seconds\n", compileTime)
			fmt.Println("=======================")
		}

		// Close SIMPL Windows after successful compilation
		fmt.Println("\nClosing dialogs and SIMPL Windows...")

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
				fmt.Printf("Detected dialog: %s (clicking 'No' to close without saving)\n", ev.Title)
				// Find and click the "No" button directly
				if windows.FindAndClickButton(ev.Hwnd, "&No") {
					fmt.Println("[DEBUG] Successfully clicked 'No' button")
					time.Sleep(500 * time.Millisecond)
				} else {
					fmt.Println("[DEBUG] WARNING: Could not find 'No' button, trying to close dialog")
					windows.CloseWindow(ev.Hwnd, "Confirmation dialog")
					time.Sleep(500 * time.Millisecond)
				}
			}
		}

		// Now close the main SIMPL Windows application
		if hwnd != 0 {
			windows.CloseWindow(hwnd, "SIMPL Windows")
			time.Sleep(1 * time.Second)
			fmt.Println("SIMPL Windows closed successfully")
		} else {
			fmt.Println("Warning: Could not close SIMPL Windows (main window handle not found)")
		}

		fmt.Println("Press Enter to exit...")
		fmt.Scanln()

		// Exit with error code if compilation failed
		if hasErrors {
			return fmt.Errorf("compilation failed with %d error(s)", errors)
		}
	}

	return nil
}
