package compiler

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Norgate-AV/smpc/internal/windows"
)

// HandleOperationCompleteDialog checks for and dismisses the "Operation Complete" dialog
// that may appear after loading a file
func HandleOperationCompleteDialog(pid uint32) error {
	if pid == 0 || windows.MonitorCh == nil {
		return nil
	}

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

	return nil
}

// HandleIncompleteSymbolsDialog checks for the "Incomplete Symbols" error dialog
// Returns an error if the dialog is detected (this is a fatal error)
func HandleIncompleteSymbolsDialog(pid uint32) error {
	if pid == 0 || windows.MonitorCh == nil {
		return nil
	}

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

	return nil
}

// HandleConvertCompileDialog detects and auto-confirms the "Convert/Compile" save prompt
func HandleConvertCompileDialog(pid uint32) error {
	if pid == 0 || windows.MonitorCh == nil {
		return nil
	}

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

	return nil
}

// HandleCommentedOutSymbolsDialog detects and auto-confirms the "Commented out Symbols" dialog
func HandleCommentedOutSymbolsDialog(pid uint32) error {
	if pid == 0 || windows.MonitorCh == nil {
		return nil
	}

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

	return nil
}

// WaitForCompilingDialog waits for the "Compiling..." dialog to appear
func WaitForCompilingDialog(pid uint32) error {
	if pid == 0 || windows.MonitorCh == nil {
		return nil
	}

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

	return nil
}

// ParseCompileCompleteDialog waits for and parses the "Compile Complete" dialog
// Returns the hwnd of the dialog (for later closing) and the parsed statistics
func ParseCompileCompleteDialog(pid uint32) (uintptr, int, int, int, float64, error) {
	if pid == 0 || windows.MonitorCh == nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("PID not available or monitor channel not initialized")
	}

	slog.Info("Waiting for 'Compile Complete' dialog...")
	ev, ok := windows.WaitOnMonitor(5*time.Minute, // Increased timeout for large programs
		func(e windows.WindowEvent) bool { return strings.EqualFold(e.Title, "Compile Complete") },
		func(e windows.WindowEvent) bool {
			return strings.Contains(strings.ToLower(e.Title), "compile complete")
		},
	)

	if !ok {
		return 0, 0, 0, 0, 0, fmt.Errorf("compilation timeout: did not detect 'Compile Complete' dialog within 5 minutes")
	}

	slog.Info("Detected", "title", ev.Title)
	hwnd := ev.Hwnd
	childInfos := windows.CollectChildInfos(ev.Hwnd)
	slog.Debug("Child controls in dialog", "title", ev.Title)

	for _, ci := range childInfos {
		slog.Debug("Child control", "class", ci.ClassName, "text", ci.Text, "length", len(ci.Text))
	}

	// Parse stats from Compile Complete dialog
	var warnings, notices, errors int
	var compileTime float64

	for _, ci := range childInfos {
		text := strings.ReplaceAll(ci.Text, "\r\n", "\n")
		lines := strings.SplitSeq(text, "\n")
		for t := range lines {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			if n, ok := ParseStatLine(t, "Program Warnings"); ok {
				warnings = n
			}
			if n, ok := ParseStatLine(t, "Program Notices"); ok {
				notices = n
			}
			if n, ok := ParseStatLine(t, "Program Errors"); ok {
				errors = n
			}
			if secs, ok := ParseCompileTimeLine(t); ok {
				compileTime = secs
			}
		}
	}

	return hwnd, warnings, notices, errors, compileTime, nil
}

// ParseProgramCompilationDialog waits for and parses the "Program Compilation" dialog
// This dialog appears when there are warnings, notices, or errors
func ParseProgramCompilationDialog(pid uint32, warnings, notices, errors int) ([]string, []string, []string, error) {
	if pid == 0 || windows.MonitorCh == nil {
		return nil, nil, nil, nil
	}

	if warnings == 0 && notices == 0 && errors == 0 {
		return nil, nil, nil, nil
	}

	slog.Info("Waiting for 'Program Compilation' dialog...")
	ev, ok := windows.WaitOnMonitor(10*time.Second,
		func(e windows.WindowEvent) bool { return strings.EqualFold(e.Title, "Program Compilation") },
		func(e windows.WindowEvent) bool {
			return strings.Contains(strings.ToLower(e.Title), "program compilation")
		},
	)

	if !ok {
		slog.Debug("Program Compilation dialog not detected (may not have appeared)")
		return nil, nil, nil, nil
	}

	slog.Info("Detected", "title", ev.Title)
	childInfos := windows.CollectChildInfos(ev.Hwnd)
	slog.Debug("Child controls in dialog", "title", ev.Title)

	for _, ci := range childInfos {
		slog.Debug("Child control", "class", ci.ClassName, "text", ci.Text, "length", len(ci.Text))
	}

	warningMessages := []string{}
	noticeMessages := []string{}
	errorMessages := []string{}

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

	// Log the messages
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

	return errorMessages, warningMessages, noticeMessages, nil
}

// HandleConfirmationDialog handles the "Confirmation" dialog that may appear when closing
func HandleConfirmationDialog(pid uint32) error {
	if pid == 0 || windows.MonitorCh == nil {
		return nil
	}

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

	return nil
}
