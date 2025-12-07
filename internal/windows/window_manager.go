//go:build windows

package windows

import (
	"log/slog"
	"strings"
	"time"
	"unsafe"

	"github.com/Norgate-AV/smpc/internal/logger"
	"github.com/Norgate-AV/smpc/internal/timeouts"
)

// windowManager implements the WindowManager interface
type windowManager struct {
	log logger.LoggerInterface
}

// newWindowManager creates a new window manager
func newWindowManager(log logger.LoggerInterface) *windowManager {
	return &windowManager{log: log}
}

// CloseWindow sends a WM_CLOSE message to the specified window
func (w *windowManager) CloseWindow(hwnd uintptr, title string) {
	w.log.Debug("Closing window", slog.String("title", title))

	ret, _, err := procPostMessageW.Call(hwnd, WM_CLOSE, 0, 0)
	if ret == 0 {
		w.log.Debug("PostMessage WM_CLOSE failed",
			slog.String("title", title),
			slog.Uint64("hwnd", uint64(hwnd)),
			slog.Any("error", err))
	}

	time.Sleep(timeouts.WindowMessageDelay)
}

// SetForeground brings a window to the foreground using AttachThreadInput technique
func (w *windowManager) SetForeground(hwnd uintptr) bool {
	// Restore window if minimized
	ret, _, _ := procShowWindow.Call(hwnd, uintptr(SW_RESTORE))
	w.log.Debug("ShowWindow(SW_RESTORE)", slog.Uint64("ret", uint64(ret)))

	// Try standard SetForegroundWindow first
	ret, _, _ = procSetForegroundWindow.Call(hwnd)
	if ret != 0 {
		w.log.Debug("SetForegroundWindow succeeded (standard)")
		return w.verifyForeground(hwnd)
	}

	w.log.Debug("Standard SetForegroundWindow failed, trying AttachThreadInput technique")

	// Get current foreground window and its thread
	fgHwnd, _, _ := procGetForegroundWindow.Call()
	if fgHwnd == 0 || fgHwnd == hwnd {
		w.log.Debug("No foreground window or already focused")
		return true
	}

	// Get thread IDs
	fgThreadID, _, _ := procGetWindowThreadProcessId.Call(fgHwnd, 0)
	targetThreadID, _, _ := procGetWindowThreadProcessId.Call(hwnd, 0)

	if fgThreadID == 0 || targetThreadID == 0 {
		w.log.Warn("Could not get thread IDs",
			slog.Uint64("fgThreadID", uint64(fgThreadID)),
			slog.Uint64("targetThreadID", uint64(targetThreadID)))
		return false
	}

	w.log.Debug("Attaching threads",
		slog.Uint64("fgThreadID", uint64(fgThreadID)),
		slog.Uint64("targetThreadID", uint64(targetThreadID)))

	// Attach our thread to the foreground window's thread
	ret, _, _ = procAttachThreadInput.Call(targetThreadID, fgThreadID, 1)
	if ret == 0 {
		w.log.Warn("AttachThreadInput failed")
		return false
	}

	// Now SetForegroundWindow should work
	ret, _, _ = procSetForegroundWindow.Call(hwnd)
	success := ret != 0

	// Detach threads
	ret, _, _ = procAttachThreadInput.Call(targetThreadID, fgThreadID, 0)
	if ret == 0 {
		w.log.Warn("Failed to detach threads")
	}

	if success {
		w.log.Debug("SetForegroundWindow succeeded (with AttachThreadInput)")
		return w.verifyForeground(hwnd)
	}

	w.log.Warn("SetForegroundWindow still failed after AttachThreadInput")
	return false
}

// verifyForeground checks if the window is now in foreground
func (w *windowManager) verifyForeground(hwnd uintptr) bool {
	time.Sleep(timeouts.WindowMessageDelay)

	fgHwnd, _, _ := procGetForegroundWindow.Call()
	if fgHwnd == hwnd {
		w.log.Debug("Window confirmed in foreground")
		return true
	}

	w.log.Warn("Different window in foreground",
		slog.Uint64("expected", uint64(hwnd)),
		slog.Uint64("got", uint64(fgHwnd)))

	return false
}

// VerifyForegroundWindow checks if the specified window is currently in the foreground
// and optionally verifies it belongs to the expected PID
func (w *windowManager) VerifyForegroundWindow(expectedHwnd uintptr, expectedPid uint32) bool {
	fgHwnd, _, _ := procGetForegroundWindow.Call()

	if fgHwnd != expectedHwnd {
		w.log.Warn("Wrong window in foreground",
			slog.Uint64("expected_hwnd", uint64(expectedHwnd)),
			slog.Uint64("actual_hwnd", uint64(fgHwnd)),
		)
		return false
	}

	// If PID verification requested, check it
	if expectedPid != 0 {
		var actualPid uint32
		ret, _, err := procGetWindowThreadProcessId.Call(fgHwnd, uintptr(unsafe.Pointer(&actualPid)))
		if ret == 0 {
			w.log.Debug("GetWindowThreadProcessId failed", slog.Any("error", err))
		}

		if actualPid != expectedPid {
			w.log.Warn("Foreground window has wrong PID",
				slog.Uint64("hwnd", uint64(fgHwnd)),
				slog.Uint64("expected_pid", uint64(expectedPid)),
				slog.Uint64("actual_pid", uint64(actualPid)),
			)
			return false
		}

		w.log.Debug("Foreground window verified",
			slog.Uint64("hwnd", uint64(fgHwnd)),
			slog.Uint64("pid", uint64(actualPid)),
		)
	}

	return true
}

// IsElevated returns whether the current process is running with administrator privileges
func (w *windowManager) IsElevated() bool {
	return IsElevated()
}

// CollectChildInfos collects information about all child windows
func (w *windowManager) CollectChildInfos(hwnd uintptr) []ChildInfo {
	return CollectChildInfos(hwnd)
}

// WaitOnMonitor waits for a window event matching any of the provided predicates
func (w *windowManager) WaitOnMonitor(timeout time.Duration, matchers ...func(WindowEvent) bool) (WindowEvent, bool) {
	if MonitorCh == nil {
		return WindowEvent{}, false
	}

	// First, check recent cache to avoid missing already-seen dialogs
	recentMu.Lock()
	for i := len(recentEvents) - 1; i >= 0; i-- {
		ev := recentEvents[i]

		for _, m := range matchers {
			if m(ev) {
				recentMu.Unlock()
				return ev, true
			}
		}
	}

	recentMu.Unlock()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case ev := <-MonitorCh:
			for _, m := range matchers {
				if m(ev) {
					return ev, true
				}
			}
		case <-timer.C:
			return WindowEvent{}, false
		}
	}
}

// FindAndClickButton finds a button child control with the specified text and clicks it
func (w *windowManager) FindAndClickButton(parentHwnd uintptr, buttonText string) bool {
	childInfos := CollectChildInfos(parentHwnd)

	for _, ci := range childInfos {
		if ci.ClassName == "Button" && strings.EqualFold(ci.Text, buttonText) {
			w.log.Debug("Found button, sending click",
				slog.String("text", buttonText),
				slog.Uint64("hwnd", uint64(ci.Hwnd)),
			)

			// Send BN_CLICKED notification to parent
			// WM_COMMAND: wParam = MAKEWPARAM(controlID, BN_CLICKED), lParam = hwnd
			ret, _, err := procSendMessageW.Call(parentHwnd, WM_COMMAND, uintptr(BN_CLICKED), ci.Hwnd)
			if ret == 0 {
				w.log.Debug("SendMessage BN_CLICKED failed",
					slog.String("text", ci.Text),
					slog.Any("error", err))
			}

			return true
		}
	}

	w.log.Debug("Button not found", slog.String("text", buttonText))
	return false
}
