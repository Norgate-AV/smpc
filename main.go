package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const SIMPL_WINDOWS_PATH = "C:\\Program Files (x86)\\Crestron\\Simpl\\smpwin.exe"

var (
	shell32                      = syscall.NewLazyDLL("shell32.dll")
	procShellExecute             = shell32.NewProc("ShellExecuteW")
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First           = kernel32.NewProc("Process32FirstW")
	procProcess32Next            = kernel32.NewProc("Process32NextW")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
	user32                       = syscall.NewLazyDLL("user32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procSendMessageTimeoutW      = user32.NewProc("SendMessageTimeoutW")
)

const (
	WM_NULL          = 0x0000
	SMTO_ABORTIFHUNG = 0x0002
	SMTO_BLOCK       = 0x0003
)

const (
	TH32CS_SNAPPROCESS = 0x00000002
	MAX_PATH           = 260
)

type PROCESSENTRY32 struct {
	dwSize              uint32
	cntUsage            uint32
	th32ProcessID       uint32
	th32DefaultHeapID   uintptr
	th32ModuleID        uint32
	cntThreads          uint32
	th32ParentProcessID uint32
	pcPriClassBase      int32
	dwFlags             uint32
	szExeFile           [MAX_PATH]uint16
}

func shellExecute(hwnd uintptr, verb, file, args, cwd string, showCmd int) error {
	var verbPtr, filePtr, argsPtr, cwdPtr *uint16
	var err error

	if verb != "" {
		verbPtr, err = syscall.UTF16PtrFromString(verb)
		if err != nil {
			return err
		}
	}

	filePtr, err = syscall.UTF16PtrFromString(file)
	if err != nil {
		return err
	}

	if args != "" {
		argsPtr, err = syscall.UTF16PtrFromString(args)
		if err != nil {
			return err
		}
	}

	if cwd != "" {
		cwdPtr, err = syscall.UTF16PtrFromString(cwd)
		if err != nil {
			return err
		}
	}

	ret, _, _ := procShellExecute.Call(
		hwnd,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(filePtr)),
		uintptr(unsafe.Pointer(argsPtr)),
		uintptr(unsafe.Pointer(cwdPtr)),
		uintptr(showCmd),
	)

	// ShellExecute returns a value > 32 on success
	if ret <= 32 {
		return fmt.Errorf("ShellExecute failed with error code: %d", ret)
	}

	return nil
}

func getWindowText(hwnd uintptr) string {
	buf := make([]uint16, 256)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

func isWindowVisible(hwnd uintptr) bool {
	ret, _, _ := procIsWindowVisible.Call(hwnd)
	return ret != 0
}

func getWindowProcessId(hwnd uintptr) uint32 {
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	return pid
}

type windowInfo struct {
	hwnd  uintptr
	title string
	pid   uint32
}

var foundWindows []windowInfo

func enumWindowsCallback(hwnd uintptr, lparam uintptr) uintptr {
	if isWindowVisible(hwnd) {
		title := getWindowText(hwnd)
		pid := getWindowProcessId(hwnd)

		if title != "" {
			foundWindows = append(foundWindows, windowInfo{hwnd: hwnd, title: title, pid: pid})
		}
	}

	return 1 // Continue enumeration
}

func findSIMPLWindow(processName string, debug bool) (uintptr, string) {
	// First get the process ID of smpwin.exe
	var targetPID uint32

	snapshot, _, _ := procCreateToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snapshot == 0 {
		return 0, ""
	}

	defer procCloseHandle.Call(snapshot)

	var pe PROCESSENTRY32
	pe.dwSize = uint32(unsafe.Sizeof(pe))

	ret, _, _ := procProcess32First.Call(snapshot, uintptr(unsafe.Pointer(&pe)))
	if ret != 0 {
		processName = strings.ToLower(processName)
		for {
			exeName := syscall.UTF16ToString(pe.szExeFile[:])
			if strings.ToLower(exeName) == processName {
				targetPID = pe.th32ProcessID
				break
			}

			ret, _, _ := procProcess32Next.Call(snapshot, uintptr(unsafe.Pointer(&pe)))
			if ret == 0 {
				break
			}
		}
	}

	if targetPID == 0 {
		if debug {
			fmt.Println("Debug: smpwin.exe process not found")
		}

		return 0, ""
	}

	// Now enumerate all windows
	foundWindows = nil
	callback := syscall.NewCallback(enumWindowsCallback)
	procEnumWindows.Call(callback, 0)

	if debug {
		fmt.Printf("Debug: Found %d visible windows from smpwin.exe (PID: %d):\n", len(foundWindows), targetPID)
	}

	// Look for windows belonging to our process
	var mainWindow windowInfo
	var splashWindow windowInfo

	for _, w := range foundWindows {
		if w.pid == targetPID {
			if debug {
				fmt.Printf("  - %s\n", w.title)
			}

			// Skip splash screens and loading dialogs
			title := strings.ToLower(w.title)

			// If window title contains .smw, it's definitely the main window with file loaded
			if strings.Contains(w.title, ".smw") {
				mainWindow = w
				break
			}

			// Generic "SIMPL Windows" is likely the splash screen - remember it but keep looking
			if w.title == "SIMPL Windows" {
				splashWindow = w
				continue
			}

			// Look for other SIMPL-related windows that aren't splash/about
			if !strings.Contains(title, "splash") &&
				!strings.Contains(title, "loading") &&
				!strings.Contains(title, "about") &&
				len(w.title) > 5 {
				if strings.Contains(title, "simpl") {
					mainWindow = w
					break
				}
			}
		}
	}

	// If we found a main window with a more specific title, use it
	if mainWindow.hwnd != 0 {
		if debug {
			fmt.Printf("Debug: Found main window: %s\n", mainWindow.title)
		}

		return mainWindow.hwnd, mainWindow.title
	}

	// If we only found the generic splash screen, return false to keep waiting
	if splashWindow.hwnd != 0 {
		if debug {
			fmt.Printf("Debug: Only found splash screen, continuing to wait...\n")
		}

		return 0, ""
	}

	return 0, ""
}

// Note: direct FindWindowW-based search removed in favor of
// enumerating windows and matching by owning process and title.

func isWindowResponsive(hwnd uintptr, debug bool) bool {
	var result uintptr

	// Send WM_NULL message with 1 second timeout
	ret, _, _ := procSendMessageTimeoutW.Call(
		hwnd,
		WM_NULL,
		0,
		0,
		SMTO_ABORTIFHUNG,
		1000, // 1 second timeout in milliseconds
		uintptr(unsafe.Pointer(&result)),
	)

	responsive := ret != 0
	if debug {
		if responsive {
			fmt.Println("Debug: Window is responsive")
		} else {
			fmt.Println("Debug: Window is not responding")
		}
	}

	return responsive
}

func waitForWindowToBeReady(hwnd uintptr, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	elapsed := 0

	fmt.Println("Waiting for window to be fully ready...")

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
				fmt.Println("Debug: Window is stable and ready")
				return true
			}
		}

		time.Sleep(100 * time.Millisecond)
		elapsed++
	}

	fmt.Println("Debug: Timeout waiting for window to be ready")
	return false
}

func waitForWindowToAppear(timeout time.Duration) (uintptr, bool) {
	deadline := time.Now().Add(timeout)
	elapsed := 0

	for time.Now().Before(deadline) {
		// Show debug output every 5 seconds
		debug := elapsed%50 == 0

		// Check for the main SIMPL Windows window
		hwnd, title := findSIMPLWindow("smpwin.exe", debug)
		if hwnd != 0 {
			fmt.Printf("Debug: Found main SIMPL Windows window: %s\n", title)
			return hwnd, true
		}

		time.Sleep(100 * time.Millisecond)
		elapsed++
	}

	fmt.Println("Debug: Timeout reached, performing final detailed check...")
	hwnd, title := findSIMPLWindow("smpwin.exe", true)
	if hwnd != 0 {
		fmt.Printf("Debug: Found window at timeout: %s\n", title)
		return hwnd, true
	}

	return 0, false
}

// Deprecated: we now wait for the actual main window and its responsiveness
// rather than only the process presence.

func main() {
	// Check if a file path argument was provided
	if len(os.Args) < 2 {
		fmt.Println("Usage: smpc <file-path>")
		os.Exit(1)
	}

	// Get the file path from the first command line argument
	filePath := os.Args[1]

	// Check if the file has .smw extension
	if filepath.Ext(filePath) != ".smw" {
		fmt.Printf("Error: File must have .smw extension\n")
		os.Exit(1)
	}

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("Error: File does not exist: %s\n", filePath)
		os.Exit(1)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		fmt.Printf("Error resolving file path: %v\n", err)
		os.Exit(1)
	}

	// Open the file with SIMPL Windows application using elevated privileges
	// SW_SHOWNORMAL = 1
	if err := shellExecute(0, "runas", SIMPL_WINDOWS_PATH, absPath, "", 1); err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}

	// Wait for the main window to appear (with a 1 minute timeout)
	fmt.Printf("Waiting for SIMPL Windows to fully launch...\n")
	hwnd, found := waitForWindowToAppear(60 * time.Second)
	if !found {
		fmt.Printf("Warning: Timed out waiting for SIMPL Windows window to appear\n")
		os.Exit(1)
	}

	// Wait for the window to be fully ready and responsive (with a 30 second timeout)
	if !waitForWindowToBeReady(hwnd, 30*time.Second) {
		fmt.Printf("Warning: Window appeared but is not responding properly\n")
		os.Exit(1)
	}

	// Small extra delay to allow UI to finish settling
	fmt.Println("Waiting a few extra seconds for UI to settle...")
	time.Sleep(5 * time.Second)

	fmt.Printf("Successfully opened file: %s\n", absPath)
}
