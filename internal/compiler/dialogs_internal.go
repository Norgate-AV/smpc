package compiler

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Norgate-AV/smpc/internal/interfaces"
	"github.com/Norgate-AV/smpc/internal/windows"
)

// DialogHandler handles dialog operations with injected dependencies
type DialogHandler struct {
	windowMgr     interfaces.WindowManager
	keyboard      interfaces.KeyboardInjector
	controlReader interfaces.ControlReader
}

func NewDialogHandler(windowMgr interfaces.WindowManager, keyboard interfaces.KeyboardInjector, controlReader interfaces.ControlReader) *DialogHandler {
	return &DialogHandler{
		windowMgr:     windowMgr,
		keyboard:      keyboard,
		controlReader: controlReader,
	}
}

func (dh *DialogHandler) HandleOperationComplete(pid uint32) error {
	if pid == 0 {
		return nil
	}

	slog.Info("Checking for 'Operation Complete' dialog...")
	slog.Debug("Checking for Operation Complete dialog")
	ev, ok := dh.windowMgr.WaitOnMonitor(3*time.Second,
		func(e windows.WindowEvent) bool { return strings.EqualFold(e.Title, "Operation Complete") },
		func(e windows.WindowEvent) bool {
			return strings.Contains(strings.ToLower(e.Title), "operation complete")
		},
	)

	if ok {
		slog.Debug("Operation Complete dialog detected", "title", ev.Title)
		slog.Info("Detected dialog: " + ev.Title)
		slog.Info("Dismissing 'Operation Complete' dialog...")
		dh.windowMgr.CloseWindow(ev.Hwnd, ev.Title)
		time.Sleep(500 * time.Millisecond)
		slog.Debug("Dialog dismissed")
	} else {
		slog.Debug("No Operation Complete dialog detected")
	}

	return nil
}

func (dh *DialogHandler) HandleIncompleteSymbols(pid uint32) error {
	if pid == 0 {
		return nil
	}

	slog.Info("Checking for 'Incomplete Symbols' error dialog...")
	ev, ok := dh.windowMgr.WaitOnMonitor(2*time.Second,
		func(e windows.WindowEvent) bool { return strings.EqualFold(e.Title, "Incomplete Symbols") },
		func(e windows.WindowEvent) bool { return strings.Contains(strings.ToLower(e.Title), "incomplete") },
	)

	if ok {
		slog.Error("ERROR: Incomplete Symbols detected", "title", ev.Title)
		slog.Info("The program contains incomplete symbols and cannot be compiled.")
		slog.Info("Please fix the incomplete symbols in SIMPL Windows before attempting to compile.")

		// Extract error details from the dialog
		childInfos := dh.windowMgr.CollectChildInfos(ev.Hwnd)
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

func (dh *DialogHandler) HandleConvertCompile(pid uint32) error {
	if pid == 0 {
		return nil
	}

	slog.Info("Watching for 'Convert/Compile' save prompt...")
	ev, ok := dh.windowMgr.WaitOnMonitor(5*time.Second,
		func(e windows.WindowEvent) bool { return strings.EqualFold(e.Title, "Convert/Compile") },
		func(e windows.WindowEvent) bool { return strings.Contains(strings.ToLower(e.Title), "convert/compile") },
	)

	if ok {
		slog.Info("Detected save prompt", "title", ev.Title)
		_ = dh.windowMgr.SetForeground(ev.Hwnd)
		time.Sleep(300 * time.Millisecond)
		_ = dh.keyboard.SendEnter()
		slog.Info("Auto-confirmed save prompt with 'Yes'")
	} else {
		slog.Debug("Save prompt not detected within timeout")
	}

	return nil
}

func (dh *DialogHandler) HandleCommentedOutSymbols(pid uint32) error {
	if pid == 0 {
		return nil
	}

	slog.Info("Watching for 'Commented out Symbols' dialog...")
	ev, ok := dh.windowMgr.WaitOnMonitor(5*time.Second,
		func(e windows.WindowEvent) bool {
			return strings.EqualFold(e.Title, "Commented out Symbols and/or Devices")
		},
		func(e windows.WindowEvent) bool { return strings.Contains(strings.ToLower(e.Title), "commented out") },
	)

	if ok {
		slog.Info("Detected dialog", "title", ev.Title)
		_ = dh.windowMgr.SetForeground(ev.Hwnd)
		time.Sleep(300 * time.Millisecond)
		_ = dh.keyboard.SendEnter()
		slog.Info("Auto-confirmed 'Commented out Symbols' dialog with 'Yes'")
	} else {
		slog.Debug("'Commented out Symbols' dialog not detected within timeout")
	}

	return nil
}

func (dh *DialogHandler) WaitForCompiling(pid uint32) error {
	if pid == 0 {
		return nil
	}

	slog.Info("Waiting for 'Compiling...' dialog...")
	ev, ok := dh.windowMgr.WaitOnMonitor(30*time.Second,
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

func (dh *DialogHandler) ParseCompileComplete(pid uint32) (hwnd uintptr, warnings, notices, errors int, compileTime float64, err error) {
	if pid == 0 {
		return 0, 0, 0, 0, 0, nil
	}

	slog.Info("Waiting for 'Compile Complete' dialog...")
	ev, ok := dh.windowMgr.WaitOnMonitor(5*time.Minute,
		func(e windows.WindowEvent) bool { return strings.EqualFold(e.Title, "Compile Complete") },
		func(e windows.WindowEvent) bool {
			return strings.Contains(strings.ToLower(e.Title), "compile complete")
		},
	)

	if !ok {
		return 0, 0, 0, 0, 0, fmt.Errorf("compilation timeout: did not detect 'Compile Complete' dialog within 5 minutes")
	}

	slog.Info("Detected", "title", ev.Title)
	hwnd = ev.Hwnd

	// Parse statistics from dialog children
	childInfos := dh.windowMgr.CollectChildInfos(ev.Hwnd)
	for _, ci := range childInfos {
		text := strings.ReplaceAll(ci.Text, "\r\n", "\n")
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if n, ok := ParseStatLine(line, "Program Warnings"); ok {
				warnings = n
			}
			if n, ok := ParseStatLine(line, "Program Notices"); ok {
				notices = n
			}
			if n, ok := ParseStatLine(line, "Program Errors"); ok {
				errors = n
			}
			if secs, ok := ParseCompileTimeLine(line); ok {
				compileTime = secs
			}
		}
	}

	return hwnd, warnings, notices, errors, compileTime, nil
}

func (dh *DialogHandler) ParseProgramCompilation(pid uint32) (warnings, notices, errors []string, err error) {
	if pid == 0 {
		return nil, nil, nil, nil
	}

	slog.Info("Waiting for 'Program Compilation' dialog...")
	ev, ok := dh.windowMgr.WaitOnMonitor(10*time.Second,
		func(e windows.WindowEvent) bool { return strings.EqualFold(e.Title, "Program Compilation") },
		func(e windows.WindowEvent) bool {
			return strings.Contains(strings.ToLower(e.Title), "program compilation")
		},
	)

	if !ok {
		slog.Debug("Program Compilation dialog not detected")
		return nil, nil, nil, nil
	}

	slog.Info("Detected", "title", ev.Title)
	childInfos := dh.windowMgr.CollectChildInfos(ev.Hwnd)

	// Extract messages from ListBox
	for _, ci := range childInfos {
		if ci.ClassName == "ListBox" && len(ci.Items) > 0 {
			for _, line := range ci.Items {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				lineUpper := strings.ToUpper(line)
				if strings.HasPrefix(lineUpper, "ERROR") {
					errors = append(errors, line)
				} else if strings.HasPrefix(lineUpper, "WARNING") {
					warnings = append(warnings, line)
				} else if strings.HasPrefix(lineUpper, "NOTICE") {
					notices = append(notices, line)
				} else {
					// Continuation of previous message
					if len(errors) > 0 {
						errors[len(errors)-1] += " " + line
					} else if len(warnings) > 0 {
						warnings[len(warnings)-1] += " " + line
					} else if len(notices) > 0 {
						notices[len(notices)-1] += " " + line
					}
				}
			}
		}
	}

	return warnings, notices, errors, nil
}

func (dh *DialogHandler) HandleConfirmation(pid uint32) error {
	if pid == 0 {
		return nil
	}

	ev, ok := dh.windowMgr.WaitOnMonitor(2*time.Second,
		func(e windows.WindowEvent) bool { return strings.EqualFold(e.Title, "Confirmation") },
		func(e windows.WindowEvent) bool { return strings.Contains(strings.ToLower(e.Title), "confirmation") },
	)

	if ok {
		slog.Info("Detected dialog (clicking 'No' to close without saving)", "title", ev.Title)
		if dh.controlReader.FindAndClickButton(ev.Hwnd, "&No") {
			slog.Debug("Successfully clicked 'No' button")
			time.Sleep(500 * time.Millisecond)
		} else {
			slog.Warn("Could not find 'No' button, trying to close dialog")
			dh.windowMgr.CloseWindow(ev.Hwnd, "Confirmation dialog")
			time.Sleep(500 * time.Millisecond)
		}
	}

	return nil
}
