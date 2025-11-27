package windows

import (
	"fmt"
	"sync"
	"syscall"
	"time"
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

// waitOnMonitor waits for a window event whose title matches any of the
// provided predicates within the given timeout. Returns the matching event
// and true on success, or a zero-value event and false on timeout.
func WaitOnMonitor(timeout time.Duration, matchers ...func(WindowEvent) bool) (WindowEvent, bool) {
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

func enumWindowsCallback(hwnd uintptr, lparam uintptr) uintptr {
	if IsWindowVisible(hwnd) {
		title := GetWindowText(hwnd)
		pid := GetWindowProcessId(hwnd)

		// Include even if title is empty; we may match by child text later
		foundWindows = append(foundWindows, WindowInfo{Hwnd: hwnd, Title: title, Pid: pid})
	}

	return 1 // Continue enumeration
}

// enumerateWindows performs a thread-safe enumeration of visible top-level windows
func EnumerateWindows() []WindowInfo {
	windowsMu.Lock()
	defer windowsMu.Unlock()

	foundWindows = nil
	callback := syscall.NewCallback(enumWindowsCallback)
	procEnumWindows.Call(callback, 0)

	// Make a copy to avoid races with subsequent enumerations
	windows := make([]WindowInfo, len(foundWindows))
	copy(windows, foundWindows)

	return windows
}

// startWindowMonitor launches a background goroutine that periodically
// enumerates windows and logs any newly seen windows/dialogs and their child texts.
// If pid==0, it will log windows from all processes; otherwise it filters to that PID.
func StartWindowMonitor(pid uint32, interval time.Duration) {
	seen := make(map[uintptr]bool)

	go func() {
		fmt.Println("[DEBUG] Window monitor started")
		for {
			windows := EnumerateWindows()

			for _, w := range windows {
				if pid != 0 && w.Pid != pid {
					continue
				}
				if !seen[w.Hwnd] {
					seen[w.Hwnd] = true
					// Log top-level window info
					fmt.Printf("[MON] hwnd=%d pid=%d class=%s title=%q\n", w.Hwnd, w.Pid, GetClassName(w.Hwnd), w.Title)

					// Enumerate child controls and log their text
					childTexts := CollectChildTexts(w.Hwnd)
					if len(childTexts) > 0 {
						for _, ct := range childTexts {
							if ct != "" {
								fmt.Printf("[MON]   child=%q\n", ct)
							}
						}
					}

					// Broadcast event (non-blocking) and store in recent cache
					if MonitorCh != nil {
						ev := WindowEvent{Hwnd: w.Hwnd, Title: w.Title, Pid: w.Pid, Class: GetClassName(w.Hwnd)}

						recentMu.Lock()
						recentEvents = append(recentEvents, ev)

						if len(recentEvents) > 256 {
							recentEvents = recentEvents[len(recentEvents)-256:]
						}

						recentMu.Unlock()

						select {
						case MonitorCh <- ev:
						default:
							// drop if buffer full
						}
					}
				}
			}

			time.Sleep(interval)
		}
	}()
}
