package windows

import (
	"sync"
	"syscall"
)

var (
	foundWindows []WindowInfo
	windowsMu    sync.Mutex
)

// Channel to broadcast window events from the monitor
var MonitorCh chan WindowEvent

var (
	recentEvents []WindowEvent
	recentMu     sync.Mutex
)

func enumWindowsCallback(hwnd uintptr, lparam uintptr) uintptr {
	if IsWindowVisible(hwnd) {
		title := GetWindowText(hwnd)
		pid := GetWindowPid(hwnd)

		// Include even if title is empty; we may match by child text later
		foundWindows = append(foundWindows, WindowInfo{Hwnd: hwnd, Title: title, Pid: pid})
	}

	return 1 // Continue enumeration
}

// EnumerateWindows performs a thread-safe enumeration of visible top-level windows
func EnumerateWindows() []WindowInfo {
	windowsMu.Lock()
	defer windowsMu.Unlock()

	foundWindows = nil
	callback := syscall.NewCallback(enumWindowsCallback)
	ret, _, _ := procEnumWindows.Call(callback, 0)
	if ret == 0 {
		return nil
	}

	// Make a copy to avoid races with subsequent enumerations
	windows := make([]WindowInfo, len(foundWindows))
	copy(windows, foundWindows)

	return windows
}
