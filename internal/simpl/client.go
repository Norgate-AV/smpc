package simpl

import (
	"context"
	"log/slog"
	"strings"
	"time"
	"unsafe"

	"github.com/Norgate-AV/smpc/internal/logger"
	"github.com/Norgate-AV/smpc/internal/timeouts"
	"github.com/Norgate-AV/smpc/internal/windows"
)

// Client provides methods for interacting with SIMPL Windows processes
type Client struct {
	log logger.LoggerInterface
	win *windows.Client
}

// NewClient creates a new SIMPL Windows client
func NewClient(log logger.LoggerInterface) *Client {
	return &Client{
		log: log,
		win: windows.NewClient(log),
	}
}

// GetPid retrieves the PID of smpwin.exe, returns 0 if not found
func (c *Client) GetPid() uint32 {
	return findProcessByName("smpwin.exe")
}

// FindWindow searches for the SIMPL Windows main window belonging to a specific process
// If targetPid is 0, it will search for any smpwin.exe process (legacy behavior)
// The seenWindows map tracks windows that have already been logged to avoid repetitive output
func (c *Client) FindWindow(targetPid uint32, debug bool) (uintptr, string) {
	return c.findWindowWithTracking(targetPid, debug, nil)
}

// findWindowWithTracking is the internal implementation that supports window tracking
func (c *Client) findWindowWithTracking(targetPid uint32, debug bool, seenWindows map[uintptr]bool) (uintptr, string) {
	// If no target PID specified, search for any smpwin.exe process
	if targetPid == 0 {
		targetPid = findProcessByName("smpwin.exe")

		if targetPid == 0 {
			if debug {
				c.log.Debug("smpwin.exe process not found")
			}

			return 0, ""
		}
	}

	// Now enumerate all windows
	// Enumerate windows (thread-safe)
	windowsList := windows.EnumerateWindows()

	// Look for windows belonging to our process
	var mainWindow windows.WindowInfo
	var splashWindow windows.WindowInfo

	for _, w := range windowsList {
		if w.Pid == targetPid {
			// Only log if debug is enabled AND we haven't seen this window before
			shouldLog := debug && (seenWindows == nil || !seenWindows[w.Hwnd])
			if shouldLog {
				c.log.Debug("Window found",
					slog.String("title", w.Title),
					slog.Uint64("hwnd", uint64(w.Hwnd)),
				)
				if seenWindows != nil {
					seenWindows[w.Hwnd] = true
				}
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
			c.log.Debug("Found main window", slog.String("title", mainWindow.Title))
		}

		return mainWindow.Hwnd, mainWindow.Title
	}

	// If we only found the generic splash screen, return 0 to keep waiting
	if splashWindow.Hwnd != 0 {
		return 0, ""
	}

	return 0, ""
}

// WaitForReady waits for a window to become fully responsive
func (c *Client) WaitForReady(hwnd uintptr, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	elapsed := 0

	c.log.Debug("Waiting for window ready state",
		slog.Uint64("hwnd", uint64(hwnd)),
		slog.String("timeout", timeout.String()),
	)

	for time.Now().Before(deadline) {
		debug := elapsed%30 == 0 // Debug every 3 seconds

		if c.isWindowResponsive(hwnd, debug) {
			// Window is responsive, wait a bit more to ensure stability
			consecutiveResponses := 0
			for range 3 {
				time.Sleep(timeouts.StabilityCheckInterval)
				if c.isWindowResponsive(hwnd, false) {
					consecutiveResponses++
				}
			}

			if consecutiveResponses >= 2 {
				c.log.Debug("Window is stable and ready")
				return true
			}
		}

		time.Sleep(timeouts.StatePollingInterval)
		elapsed++
	}

	c.log.Debug("Timeout waiting for window to be ready")
	return false
}

// WaitForAppear waits for the SIMPL Windows main window to appear for a specific process
// If targetPid is 0, it will search for any smpwin.exe process
func (c *Client) WaitForAppear(targetPid uint32, timeout time.Duration) (uintptr, bool) {
	deadline := time.Now().Add(timeout)
	seenWindows := make(map[uintptr]bool) // Track windows we've already logged
	loggedSplashOnly := false             // Track if we've logged "only splash screen" message

	c.log.Debug("Searching for window", slog.Uint64("pid", uint64(targetPid)))

	for time.Now().Before(deadline) {
		// Check for the main SIMPL Windows window, passing seenWindows for tracking
		hwnd, title := c.findWindowWithTracking(targetPid, true, seenWindows)

		if hwnd != 0 {
			c.log.Debug("Found main SIMPL Windows window", slog.String("title", title))
			return hwnd, true
		}

		// If we haven't found the main window yet and haven't logged it, log once
		// TODO: Is this needed?
		if !loggedSplashOnly {
			c.log.Debug("Only found splash screen, continuing to wait")
			loggedSplashOnly = true
		}

		time.Sleep(timeouts.StatePollingInterval)
	}

	c.log.Debug("Timeout reached, performing final detailed check")
	hwnd, title := c.findWindowWithTracking(targetPid, true, seenWindows)
	if hwnd != 0 {
		c.log.Debug("Found window at timeout", slog.String("title", title))
		return hwnd, true
	}

	return 0, false
}

// Cleanup ensures SIMPL Windows is properly closed, with fallback to force termination
func (c *Client) Cleanup(hwnd uintptr) {
	c.log.Debug("Cleaning up...")
	if hwnd == 0 {
		return
	}

	// Try to close gracefully
	c.win.Window.CloseWindow(hwnd, "SIMPL Windows")
	time.Sleep(timeouts.CleanupDelay)

	// Verify the window is actually closed - check any smpwin.exe process
	testHwnd, _ := c.FindWindow(0, false)
	if testHwnd != 0 {
		c.log.Warn("SIMPL Windows did not close properly")
		// If we have the PID, attempt to terminate the process
		pid := c.GetPid()
		if pid != 0 {
			c.log.Debug("Attempting to force terminate process", slog.Uint64("pid", uint64(pid)))
			_ = windows.TerminateProcess(pid)
		}
	}
}

// ForceCleanup attempts to forcefully close SIMPL Windows using multiple strategies.
// It tries three approaches in order:
// 1. Use hwnd if available (graceful close)
// 2. Use known PID (forced termination)
// 3. Search for process and terminate (last resort)
func (c *Client) ForceCleanup(hwnd uintptr, knownPid uint32) {
	// Strategy 1: Use hwnd if available for graceful close
	if hwnd != 0 {
		c.Cleanup(hwnd)
		return
	}

	// Strategy 2: Use known PID for forced termination
	if knownPid != 0 {
		c.log.Debug("Force terminating with known PID", slog.Uint64("pid", uint64(knownPid)))
		_ = windows.TerminateProcess(knownPid)
		return
	}

	// Strategy 3: Last resort - search for process and terminate
	pid := c.GetPid()
	if pid != 0 {
		c.log.Debug("Force terminating found process", slog.Uint64("pid", uint64(pid)))
		_ = windows.TerminateProcess(pid)
	} else {
		c.log.Warn("Unable to find SIMPL Windows process for cleanup")
	}
}

// StartMonitoring starts a background goroutine that monitors SIMPL Windows dialogs
// Returns a function to stop the monitoring
func (c *Client) StartMonitoring() func() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		// Try to obtain PID repeatedly until found, then monitor that PID
		var pid uint32

		for i := 0; i < 50 && pid == 0; i++ { // up to ~5s
			select {
			case <-ctx.Done():
				return
			default:
				pid = c.GetPid()
				if pid == 0 {
					time.Sleep(timeouts.StatePollingInterval)
				}
			}
		}

		// Init channel
		windows.MonitorCh = make(chan windows.WindowEvent, 64)
		if pid == 0 {
			c.log.Debug("Window monitor falling back to all processes (SIMPL PID not found yet)")
			c.win.Monitor.StartWindowMonitor(ctx, 0, timeouts.MonitorPollingInterval)
		} else {
			c.log.Debug("Window monitor targeting SIMPL PID", slog.Uint64("pid", uint64(pid)))
			c.win.Monitor.StartWindowMonitor(ctx, pid, timeouts.MonitorPollingInterval)
		}

		// Wait for cancellation
		<-ctx.Done()
	}()

	return func() {
		cancel()
	}
}

// isWindowResponsive checks if a window is responding to messages
func (c *Client) isWindowResponsive(hwnd uintptr, debug bool) bool {
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
			c.log.Debug("Window is responsive")
		} else {
			c.log.Debug("Window is not responding")
		}
	}

	return responsive
}
