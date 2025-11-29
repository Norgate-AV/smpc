package simpl

import (
	"log/slog"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/Norgate-AV/smpc/internal/windows"
)

func FindWindow(processName string, debug bool) (uintptr, string) {
	// First get the process ID of smpwin.exe
	var targetPID uint32

	snapshot, _, _ := windows.ProcCreateToolhelp32Snapshot.Call(windows.TH32CS_SNAPPROCESS, 0)
	if snapshot == 0 {
		return 0, ""
	}

	defer func() { _, _, _ = windows.ProcCloseHandle.Call(snapshot) }()

	var pe windows.PROCESSENTRY32
	pe.DwSize = uint32(unsafe.Sizeof(pe))

	ret, _, _ := windows.ProcProcess32First.Call(snapshot, uintptr(unsafe.Pointer(&pe)))
	if ret != 0 {
		processName = strings.ToLower(processName)
		
		for {
			exeName := syscall.UTF16ToString(pe.SzExeFile[:])
			if strings.ToLower(exeName) == processName {
				targetPID = pe.Th32ProcessID
				break
			}

			ret, _, _ := windows.ProcProcess32Next.Call(snapshot, uintptr(unsafe.Pointer(&pe)))
			if ret == 0 {
				break
			}
		}
	}

	if targetPID == 0 {
		if debug {
			slog.Debug("smpwin.exe process not found")
		}

		return 0, ""
	}

	// Now enumerate all windows
	// Enumerate windows (thread-safe)
	windowsList := windows.EnumerateWindows()

	if debug {
		slog.Debug("Found visible windows from smpwin.exe", "count", len(windowsList), "pid", targetPID)
	}

	// Look for windows belonging to our process
	var mainWindow windows.WindowInfo
	var splashWindow windows.WindowInfo

	for _, w := range windowsList {
		if w.Pid == targetPID {
			if debug {
				slog.Debug("Window found", "title", w.Title)
			}

			// Skip splash screens and loading dialogs
			title := strings.ToLower(w.Title)

			// If window title contains .smw, it's definitely the main window with file loaded
			if strings.Contains(w.Title, ".smw") {
				mainWindow = w
				break
			}

			// Generic "SIMPL Windows" is likely the splash screen - remember it but keep looking
			if w.Title == "SIMPL Windows" {
				splashWindow = w
				continue
			}

			// Look for other SIMPL-related windows that aren't splash/about
			if !strings.Contains(title, "splash") &&
				!strings.Contains(title, "loading") &&
				!strings.Contains(title, "about") &&
				len(w.Title) > 5 {
				if strings.Contains(title, "simpl") {
					mainWindow = w
					break
				}
			}
		}
	}

	// If we found a main window with a more specific title, use it
	if mainWindow.Hwnd != 0 {
		if debug {
			slog.Debug("Found main window", "title", mainWindow.Title)
		}

		return mainWindow.Hwnd, mainWindow.Title
	}

	// If we only found the generic splash screen, return false to keep waiting
	if splashWindow.Hwnd != 0 {
		if debug {
			slog.Debug("Only found splash screen, continuing to wait")
		}

		return 0, ""
	}

	return 0, ""
}

// getSimplPid retrieves the PID of smpwin.exe, returns 0 if not found
func GetPid() uint32 {
	var targetPID uint32

	snapshot, _, _ := windows.ProcCreateToolhelp32Snapshot.Call(windows.TH32CS_SNAPPROCESS, 0)
	if snapshot == 0 {
		return 0
	}

	defer func() { _, _, _ = windows.ProcCloseHandle.Call(snapshot) }()

	var pe windows.PROCESSENTRY32
	pe.DwSize = uint32(unsafe.Sizeof(pe))

	ret, _, _ := windows.ProcProcess32First.Call(snapshot, uintptr(unsafe.Pointer(&pe)))
	if ret != 0 {
		for {
			exeName := syscall.UTF16ToString(pe.SzExeFile[:])
			if strings.ToLower(exeName) == "smpwin.exe" {
				targetPID = pe.Th32ProcessID
				break
			}

			ret, _, _ := windows.ProcProcess32Next.Call(snapshot, uintptr(unsafe.Pointer(&pe)))
			if ret == 0 {
				break
			}
		}
	}

	return targetPID
}

func isWindowResponsive(hwnd uintptr, debug bool) bool {
	var result uintptr

	// Send WM_NULL message with 1 second timeout
	ret, _, _ := windows.ProcSendMessageTimeoutW.Call(
		hwnd,
		windows.WM_NULL,
		0,
		0,
		windows.SMTO_ABORTIFHUNG,
		1000, // 1 second timeout in milliseconds
		uintptr(unsafe.Pointer(&result)),
	)

	responsive := ret != 0
	if debug {
		if responsive {
			slog.Debug("Window is responsive")
		} else {
			slog.Debug("Window is not responding")
		}
	}

	return responsive
}

func WaitForReady(hwnd uintptr, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	elapsed := 0

	slog.Info("Waiting for window to be fully ready...")

	for time.Now().Before(deadline) {
		debug := elapsed%30 == 0 // Debug every 3 seconds

		if isWindowResponsive(hwnd, debug) {
			// Window is responsive, wait a bit more to ensure stability
			consecutiveResponses := 0
			for range 3 {
				time.Sleep(500 * time.Millisecond)
				if isWindowResponsive(hwnd, false) {
					consecutiveResponses++
				}
			}

			if consecutiveResponses >= 2 {
				slog.Debug("Window is stable and ready")
				return true
			}
		}

		time.Sleep(100 * time.Millisecond)
		elapsed++
	}

	slog.Debug("Timeout waiting for window to be ready")
	return false
}

func WaitForAppear(timeout time.Duration) (uintptr, bool) {
	deadline := time.Now().Add(timeout)
	elapsed := 0

	for time.Now().Before(deadline) {
		// Show debug output every 5 seconds
		debug := elapsed%50 == 0

		// Check for the main SIMPL Windows window
		hwnd, title := FindWindow("smpwin.exe", debug)
		if hwnd != 0 {
			slog.Debug("Found main SIMPL Windows window", "title", title)
			return hwnd, true
		}

		time.Sleep(100 * time.Millisecond)
		elapsed++
	}

	slog.Debug("Timeout reached, performing final detailed check")
	hwnd, title := FindWindow("smpwin.exe", true)
	if hwnd != 0 {
		slog.Debug("Found window at timeout", "title", title)
		return hwnd, true
	}

	return 0, false
}

// Cleanup ensures SIMPL Windows is properly closed, with fallback to force termination
func Cleanup(hwnd uintptr) {
	slog.Info("Cleaning up...")
	if hwnd == 0 {
		return
	}

	// Try to close gracefully
	windows.CloseWindow(hwnd, "SIMPL Windows")
	time.Sleep(1 * time.Second)

	// Verify the window is actually closed
	testHwnd, _ := FindWindow("SIMPL Windows", false)
	if testHwnd != 0 {
		slog.Warn("SIMPL Windows did not close properly")
		// If we have the PID, attempt to terminate the process
		pid := GetPid()
		if pid != 0 {
			slog.Debug("Attempting to force terminate process", "pid", pid)
			_ = windows.TerminateProcess(pid)
		}
	}
}
