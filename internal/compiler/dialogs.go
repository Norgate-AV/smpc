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
