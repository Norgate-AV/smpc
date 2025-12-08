// Package simpl provides SIMPL Windows process management and interaction.
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

// FindWindow searches for the SIMPL Windows main window belonging to a specific process
// targetPid must be a valid process ID - passing 0 will return no results
func (c *Client) FindWindow(targetPid uint32, debug bool) (uintptr, string) {
	result := c.findWindowWithTracking(targetPid, debug, nil)
	return result.mainHwnd, result.mainTitle
}

// windowSearchResult contains the results of a window search
type windowSearchResult struct {
	mainHwnd    uintptr
	mainTitle   string
	foundSplash bool
}

// findWindowWithTracking is the internal implementation that supports window tracking
// Returns the main window handle and title if found, or indicates if only splash screen was detected
func (c *Client) findWindowWithTracking(targetPid uint32, debug bool, seenWindows map[uintptr]bool) windowSearchResult {
	result := windowSearchResult{}

	// Must have a valid PID to search for windows
	if targetPid == 0 {
		if debug {
			c.log.Debug("No PID provided for window search")
		}
		return result
	}

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

		result.mainHwnd = mainWindow.Hwnd
		result.mainTitle = mainWindow.Title
		return result
	}

	// If we only found the generic splash screen, indicate it but return no handle
	if splashWindow.Hwnd != 0 {
		result.foundSplash = true
	}

	return result
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
// targetPid must be a valid process ID - passing 0 will immediately return failure
func (c *Client) WaitForAppear(targetPid uint32, timeout time.Duration) (uintptr, bool) {
	deadline := time.Now().Add(timeout)
	seenWindows := make(map[uintptr]bool) // Track windows we've already logged
	loggedSplashOnly := false             // Track if we've logged "splash screen detected" message

	c.log.Debug("Searching for window", slog.Uint64("pid", uint64(targetPid)))

	for time.Now().Before(deadline) {
		// Check for the main SIMPL Windows window, passing seenWindows for tracking
		result := c.findWindowWithTracking(targetPid, true, seenWindows)

		if result.mainHwnd != 0 {
			return result.mainHwnd, true
		}

		// If we detected a splash screen but no main window yet, log it once
		if result.foundSplash && !loggedSplashOnly {
			c.log.Debug("Found splash screen, continuing to wait for main window")
			loggedSplashOnly = true
		}

		time.Sleep(timeouts.StatePollingInterval)
	}

	c.log.Debug("Timeout reached, performing final detailed check")
	result := c.findWindowWithTracking(targetPid, true, seenWindows)
	if result.mainHwnd != 0 {
		c.log.Debug("Found window at timeout", slog.String("title", result.mainTitle))
		return result.mainHwnd, true
	}

	return 0, false
}

// Cleanup ensures SIMPL Windows is properly closed, with fallback to force termination
func (c *Client) Cleanup(hwnd uintptr, pid uint32) {
	if hwnd == 0 {
		return
	}

	// Check if the window still exists before attempting cleanup
	if !windows.IsWindow(hwnd) {
		return
	}

	c.log.Debug("Cleaning up...")

	// Try to close gracefully
	c.win.Window.CloseWindow(hwnd, "SIMPL Windows")

	// Poll for up to 3 seconds to see if window closes
	maxWait := 3 * time.Second
	pollInterval := 200 * time.Millisecond
	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		if !windows.IsWindow(hwnd) {
			c.log.Debug("Window closed successfully")
			return
		}

		time.Sleep(pollInterval)
	}

	// Window still exists after waiting - force terminate
	c.log.Warn("SIMPL Windows did not close properly after waiting")
	if pid != 0 {
		c.log.Debug("Attempting to force terminate process", slog.Uint64("pid", uint64(pid)))
		_ = windows.TerminateProcess(pid)
	}
}

// ForceCleanup attempts to forcefully close SIMPL Windows using the known PID.
// It tries two approaches in order:
// 1. Use hwnd if available (graceful close with PID for force termination)
// 2. Use known PID (forced termination)
func (c *Client) ForceCleanup(hwnd uintptr, knownPid uint32) {
	// Strategy 1: Use hwnd if available for graceful close
	if hwnd != 0 {
		c.Cleanup(hwnd, knownPid)
		return
	}

	// Strategy 2: Use known PID for forced termination
	if knownPid != 0 {
		c.log.Debug("Force terminating with known PID", slog.Uint64("pid", uint64(knownPid)))
		_ = windows.TerminateProcess(knownPid)
		return
	}

	c.log.Warn("Unable to cleanup SIMPL Windows - no hwnd or PID provided")
}

// StartMonitoring starts a background goroutine that monitors SIMPL Windows dialogs for a specific PID
// Returns a function to stop the monitoring
func (c *Client) StartMonitoring(pid uint32) func() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		// Init channel
		windows.MonitorCh = make(chan windows.WindowEvent, 64)

		if pid == 0 {
			c.log.Warn("Window monitor started with PID=0, monitoring all processes (not recommended)")
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
