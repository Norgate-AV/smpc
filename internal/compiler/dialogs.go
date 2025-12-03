package compiler

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Norgate-AV/smpc/internal/interfaces"
	"github.com/Norgate-AV/smpc/internal/logger"
	"github.com/Norgate-AV/smpc/internal/timeouts"
	"github.com/Norgate-AV/smpc/internal/windows"
)

// DialogHandler handles dialog operations with injected dependencies
type DialogHandler struct {
	log           logger.LoggerInterface
	windowMgr     interfaces.WindowManager
	keyboard      interfaces.KeyboardInjector
	controlReader interfaces.ControlReader
}

func NewDialogHandler(log logger.LoggerInterface, windowMgr interfaces.WindowManager, keyboard interfaces.KeyboardInjector, controlReader interfaces.ControlReader) *DialogHandler {
	return &DialogHandler{
		log:           log,
		windowMgr:     windowMgr,
		keyboard:      keyboard,
		controlReader: controlReader,
	}
}

// NewDialogHandlerWithAPI is a convenience constructor for production use with windows.WindowsAPI
func NewDialogHandlerWithAPI(log logger.LoggerInterface, api *windows.WindowsAPI) *DialogHandler {
	return NewDialogHandler(log, api, api, api)
}

// waitForDialog is a helper function that waits for a dialog by title and logs the result.
// It returns the dialog event and true if found, or a zero event and false if not found.
func (dh *DialogHandler) waitForDialog(title string, timeout time.Duration) (windows.WindowEvent, bool) {
	dh.log.Debug(fmt.Sprintf("Checking for '%s' dialog...", title))

	ev, ok := dh.windowMgr.WaitOnMonitor(timeout, func(e windows.WindowEvent) bool {
		return strings.EqualFold(e.Title, title)
	})

	if ok {
		dh.log.Debug(fmt.Sprintf("Detected '%s' dialog", ev.Title))
		dh.log.Debug("Dialog detected",
			slog.String("title", ev.Title),
			slog.Uint64("hwnd", uint64(ev.Hwnd)),
		)
	} else {
		dh.log.Debug(fmt.Sprintf("'%s' dialog not detected within timeout", title))
	}

	return ev, ok
}

func (dh *DialogHandler) HandleOperationComplete() error {
	ev, ok := dh.waitForDialog("Operation Complete", timeouts.DialogOperationCompleteTimeout)
	if ok {
		dh.windowMgr.CloseWindow(ev.Hwnd, ev.Title)
		time.Sleep(timeouts.WindowMessageDelay)
	}

	return nil
}

func (dh *DialogHandler) HandleIncompleteSymbols() error {
	ev, ok := dh.waitForDialog("Incomplete Symbols", timeouts.DialogIncompleteSymbolsTimeout)
	if ok {
		dh.log.Error("ERROR: Incomplete Symbols detected", slog.String("title", ev.Title))
		dh.log.Info("The program contains incomplete symbols and cannot be compiled.")
		dh.log.Info("Please fix the incomplete symbols in SIMPL Windows before attempting to compile.")

		// Extract error details from the dialog
		childInfos := dh.windowMgr.CollectChildInfos(ev.Hwnd)
		for _, ci := range childInfos {
			if ci.ClassName == "Edit" && len(ci.Text) > 50 {
				dh.log.Info("Details", slog.String("text", ci.Text))
				break
			}
		}

		return fmt.Errorf("program contains incomplete symbols and cannot be compiled")
	}

	return nil
}

func (dh *DialogHandler) HandleConvertCompile() error {
	ev, ok := dh.waitForDialog("Convert/Compile", timeouts.DialogConvertCompileTimeout)
	if ok {
		_ = dh.windowMgr.SetForeground(ev.Hwnd)
		time.Sleep(timeouts.DialogResponseDelay)
		dh.keyboard.SendEnter()
		dh.log.Info("Auto-confirmed save prompt")
	}

	return nil
}

func (dh *DialogHandler) HandleCommentedOutSymbols() error {
	ev, ok := dh.waitForDialog("Commented out Symbols and/or Devices", timeouts.DialogCommentedSymbolsTimeout)
	if ok {
		_ = dh.windowMgr.SetForeground(ev.Hwnd)
		time.Sleep(timeouts.DialogResponseDelay)
		dh.keyboard.SendEnter()
		dh.log.Info("Auto-confirmed commented symbols dialog")
	}

	return nil
}

func (dh *DialogHandler) WaitForCompiling() error {
	_, ok := dh.waitForDialog("Compiling...", timeouts.DialogCompilingTimeout)
	if !ok {
		dh.log.Warn("Did not detect 'Compiling...' dialog within timeout")
	}

	return nil
}

func (dh *DialogHandler) ParseCompileComplete() (hwnd uintptr, warnings, notices, errors int, compileTime float64, err error) {
	ev, ok := dh.waitForDialog("Compile Complete", 5*time.Minute)
	if !ok {
		return 0, 0, 0, 0, 0, fmt.Errorf("compilation timeout: did not detect 'Compile Complete' dialog within 5 minutes")
	}

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

func (dh *DialogHandler) ParseProgramCompilation() (warnings, notices, errors []string, err error) {
	ev, ok := dh.waitForDialog("Program Compilation", timeouts.DialogProgramCompilationTimeout)
	if !ok {
		return nil, nil, nil, nil
	}

	childInfos := dh.windowMgr.CollectChildInfos(ev.Hwnd)

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

			categorizeMessage(line, &errors, &warnings, &notices)
		}
	}

	return warnings, notices, errors, nil
}

func (dh *DialogHandler) HandleConfirmation() error {
	ev, ok := dh.waitForDialog("Confirmation", timeouts.DialogConfirmationTimeout)
	if ok {
		dh.log.Info("Handling confirmation dialog")

		if dh.controlReader.FindAndClickButton(ev.Hwnd, "&No") {
			dh.log.Debug("Successfully clicked 'No' button")
			time.Sleep(timeouts.WindowMessageDelay)
		} else {
			dh.log.Warn("Could not find 'No' button, trying to close dialog")
			dh.windowMgr.CloseWindow(ev.Hwnd, "Confirmation dialog")
			time.Sleep(timeouts.WindowMessageDelay)
		}
	}

	return nil
}

// categorizeMessage adds a message line to the appropriate category.
// If the line starts with ERROR/WARNING/NOTICE, it's a new message.
// Otherwise, it's a continuation of the previous message.
func categorizeMessage(line string, errors, warnings, notices *[]string) {
	lineUpper := strings.ToUpper(line)

	switch {
	case strings.HasPrefix(lineUpper, "ERROR"):
		*errors = append(*errors, line)
	case strings.HasPrefix(lineUpper, "WARNING"):
		*warnings = append(*warnings, line)
	case strings.HasPrefix(lineUpper, "NOTICE"):
		*notices = append(*notices, line)
	default:
		// Continuation of previous message
		appendContinuation(line, errors, warnings, notices)
	}
}

// appendContinuation appends a continuation line to the most recent message.
func appendContinuation(line string, errors, warnings, notices *[]string) {
	switch {
	case len(*errors) > 0:
		(*errors)[len(*errors)-1] += " " + line
	case len(*warnings) > 0:
		(*warnings)[len(*warnings)-1] += " " + line
	case len(*notices) > 0:
		(*notices)[len(*notices)-1] += " " + line
	}
}
