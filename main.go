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

type TOKEN_ELEVATION struct {
	TokenIsElevated uint32
}

func isElevated() bool {
	var token uintptr

	currentProcess, _, _ := procGetCurrentProcess.Call()
	ret, _, _ := procOpenProcessToken.Call(
		currentProcess,
		uintptr(TOKEN_QUERY),
		uintptr(unsafe.Pointer(&token)),
	)

	if ret == 0 {
		return false
	}

	defer procCloseHandle.Call(token)

	var elevation TOKEN_ELEVATION
	var returnLength uint32

	ret, _, _ = procGetTokenInformation.Call(
		token,
		uintptr(TokenElevation),
		uintptr(unsafe.Pointer(&elevation)),
		uintptr(unsafe.Sizeof(elevation)),
		uintptr(unsafe.Pointer(&returnLength)),
	)

	if ret == 0 {
		return false
	}

	return elevation.TokenIsElevated != 0
}

func relaunchAsAdmin() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	// Check if running via 'go run' (exe will be in temp dir)
	if strings.Contains(exe, "go-build") {
		fmt.Println("Detected 'go run' - please build the executable first with: go build -o smpc.exe")
		fmt.Println("Then run: .\\smpc.exe <file-path>")
		return fmt.Errorf("cannot relaunch when run via 'go run', please build first")
	}

	// Build args string (excluding the exe name)
	args := strings.Join(os.Args[1:], " ")

	return shellExecute(0, "runas", exe, args, "", 1)
}

var (
	shell32                      = syscall.NewLazyDLL("shell32.dll")
	procShellExecute             = shell32.NewProc("ShellExecuteW")
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First           = kernel32.NewProc("Process32FirstW")
	procProcess32Next            = kernel32.NewProc("Process32NextW")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
	procGetCurrentProcess        = kernel32.NewProc("GetCurrentProcess")
	procOpenProcessToken         = kernel32.NewProc("OpenProcessToken")
	advapi32                     = syscall.NewLazyDLL("advapi32.dll")
	procGetTokenInformation      = advapi32.NewProc("GetTokenInformation")
	user32                       = syscall.NewLazyDLL("user32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procSendMessageTimeoutW      = user32.NewProc("SendMessageTimeoutW")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procKeybd_event              = user32.NewProc("keybd_event")
	procShowWindow               = user32.NewProc("ShowWindow")
	procEnumChildWindows         = user32.NewProc("EnumChildWindows")
)

const (
	WM_NULL          = 0x0000
	WM_KEYDOWN       = 0x0100
	WM_KEYUP         = 0x0101
	SMTO_ABORTIFHUNG = 0x0002
	SMTO_BLOCK       = 0x0003

	INPUT_KEYBOARD        = 1
	KEYEVENTF_SCANCODE    = 0x0008
	KEYEVENTF_KEYUP       = 0x0002
	KEYEVENTF_EXTENDEDKEY = 0x0001

	SC_F12     = 0x58
	SW_RESTORE = 9
	GW_CHILD   = 5

	TOKEN_QUERY    = 0x0008
	TokenElevation = 20
)

// Structures for SendInput
type KEYBDINPUT struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type MOUSEINPUT struct {
	Dx, Dy      int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type HARDWAREINPUT struct {
	UMsg    uint32
	WParamL uint16
	WParamH uint16
}

type INPUT struct {
	Type uint32
	_    [4]byte  // Padding to align to 8 bytes
	Data [32]byte // Union data (largest is MOUSEINPUT at 24 bytes, padded to 32)
}

func setForeground(hwnd uintptr) bool {
	// Restore window if minimized, then bring to foreground
	r1, r2, lastErr := procShowWindow.Call(hwnd, uintptr(SW_RESTORE))
	fmt.Printf("Debug: ShowWindow(SW_RESTORE) r1=%d r2=%d err=%v\n", r1, r2, lastErr)

	ret, _, err := procSetForegroundWindow.Call(hwnd)
	if ret == 0 {
		fmt.Printf("Debug: SetForegroundWindow failed: %v\n", err)
		return false
	}

	fmt.Println("Debug: SetForegroundWindow succeeded")

	// Give it a moment and verify
	time.Sleep(500 * time.Millisecond)
	fgHwnd, _, _ := procGetForegroundWindow.Call()
	if fgHwnd == hwnd {
		fmt.Println("Debug: Window confirmed in foreground")
	} else {
		fmt.Printf("Debug: WARNING - Different window in foreground (expected %d, got %d)\n", hwnd, fgHwnd)
	}

	return true
}

func sendF12ViaKeybdEvent() bool {
	fmt.Println("Debug: Trying keybd_event approach...")

	// VK_F12 = 0x7B
	vkCode := uintptr(0x7B)

	// keybd_event(vk, scan, flags, extraInfo)
	// Key down
	fmt.Println("Debug: Sending keybd_event KEYDOWN")
	procKeybd_event.Call(vkCode, 0, 0x1, 0) // KEYEVENTF_EXTENDEDKEY

	time.Sleep(50 * time.Millisecond)

	// Key up
	fmt.Println("Debug: Sending keybd_event KEYUP")
	procKeybd_event.Call(vkCode, 0, 0x1|0x2, 0) // KEYEVENTF_EXTENDEDKEY | KEYEVENTF_KEYUP

	fmt.Println("Debug: keybd_event succeeded")
	return true
}

func sendEnterViaKeybdEvent() bool {
	// VK_RETURN = 0x0D
	vkCode := uintptr(0x0D)
	fmt.Println("Debug: Sending Enter via keybd_event")
	procKeybd_event.Call(vkCode, 0, 0x1, 0)
	time.Sleep(50 * time.Millisecond)
	procKeybd_event.Call(vkCode, 0, 0x1|0x2, 0)
	return true
}

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

		// Include even if title is empty; we may match by child text later
		foundWindows = append(foundWindows, windowInfo{hwnd: hwnd, title: title, pid: pid})
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

// waitForDialog looks for a visible window belonging to the SIMPL process
// whose title contains the provided substring. It polls until timeout.
func waitForDialog(pid uint32, titleSubstring string, timeout time.Duration, debug bool) (uintptr, string, bool) {
	deadline := time.Now().Add(timeout)
	normalized := strings.ToLower(titleSubstring)

	for time.Now().Before(deadline) {
		foundWindows = nil
		callback := syscall.NewCallback(enumWindowsCallback)
		procEnumWindows.Call(callback, 0)

		// First pass: match within specified PID (if provided)
		for _, w := range foundWindows {
			if pid != 0 && w.pid != pid {
				continue
			}
			if windowOrChildrenContain(w.hwnd, normalized) {
				if debug {
					fmt.Printf("Debug: Detected dialog '%s' (matched '%s')\n", w.title, titleSubstring)
				}
				return w.hwnd, w.title, true
			}
		}

		// Second pass: relaxed (any process) to catch helper-process dialogs
		if pid != 0 {
			for _, w := range foundWindows {
				if windowOrChildrenContain(w.hwnd, normalized) {
					if debug {
						fmt.Printf("Debug: Detected dialog (any PID) '%s' (matched '%s')\n", w.title, titleSubstring)
					}
					return w.hwnd, w.title, true
				}
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	if debug {
		fmt.Printf("Debug: Timeout waiting for dialog containing '%s'\n", titleSubstring)
	}
	return 0, "", false
}

var (
	childMatchSubstr string
	childFound       bool
)

func enumChildCallback(hwnd uintptr, lparam uintptr) uintptr {
	t := strings.ToLower(getWindowText(hwnd))
	if t != "" && strings.Contains(t, childMatchSubstr) {
		childFound = true
		return 0 // stop enumeration
	}
	return 1
}

func windowOrChildrenContain(hwnd uintptr, substrLower string) bool {
	// Check window title
	if strings.Contains(strings.ToLower(getWindowText(hwnd)), substrLower) {
		return true
	}
	// Check child controls' text
	childMatchSubstr = substrLower
	childFound = false
	cb := syscall.NewCallback(enumChildCallback)
	procEnumChildWindows.Call(hwnd, cb, 0)
	return childFound
}

// getSimplPid retrieves the PID of smpwin.exe, returns 0 if not found
func getSimplPid() uint32 {
	var targetPID uint32
	snapshot, _, _ := procCreateToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snapshot == 0 {
		return 0
	}
	defer procCloseHandle.Call(snapshot)

	var pe PROCESSENTRY32
	pe.dwSize = uint32(unsafe.Sizeof(pe))

	ret, _, _ := procProcess32First.Call(snapshot, uintptr(unsafe.Pointer(&pe)))
	if ret != 0 {
		for {
			exeName := syscall.UTF16ToString(pe.szExeFile[:])
			if strings.ToLower(exeName) == "smpwin.exe" {
				targetPID = pe.th32ProcessID
				break
			}
			ret, _, _ := procProcess32Next.Call(snapshot, uintptr(unsafe.Pointer(&pe)))
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

func main() {
	// Check if running as admin
	if !isElevated() {
		fmt.Println("This program requires administrator privileges.")
		fmt.Println("Relaunching as administrator...")

		if err := relaunchAsAdmin(); err != nil {
			fmt.Printf("Error relaunching as admin: %v\n", err)
			os.Exit(1)
		}

		// Exit this instance, the elevated one will continue
		os.Exit(0)
	}

	fmt.Println("Running with administrator privileges ✓")

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

	// Confirm elevation before sending keystrokes
	if isElevated() {
		fmt.Println("Debug: Process is elevated, proceeding with keystroke injection")
	} else {
		fmt.Println("Debug: WARNING - Process is NOT elevated, keystroke injection may fail")
	}

	// Bring window to foreground and send F12 (compile)
	_ = setForeground(hwnd)

	fmt.Println("Waiting for window to receive focus...")
	time.Sleep(1 * time.Second)

	// Use keybd_event (older API that works with SIMPL Windows)
	fmt.Println("Sending F12 keystroke to trigger compile...")
	if sendF12ViaKeybdEvent() {
		fmt.Println("Successfully sent F12 keystroke")

		// Detect SIMPL Windows process PID
		pid := getSimplPid()
		if pid == 0 {
			fmt.Println("Warning: Could not determine SIMPL Windows process PID; dialog detection may be limited")
		}

		// Detect save prompt ("Convert/Compile") and auto-confirm "Yes"
		if pid != 0 {
			if hwndDlg, title, ok := waitForDialog(pid, "Convert/Compile", 5*time.Second, true); ok {
				fmt.Printf("Detected save prompt: %s\n", title)
				// Bring dialog to foreground and press Enter (default is "Yes")
				_ = setForeground(hwndDlg)
				time.Sleep(300 * time.Millisecond)
				_ = sendEnterViaKeybdEvent()
				fmt.Println("Auto-confirmed save prompt with 'Yes'")
			}
		}

		// Detect compile progress start ("Compiling...")
		if pid != 0 {
			fmt.Println("Waiting for 'Compiling...' dialog...")
			if _, title, ok := waitForDialog(pid, "Compiling...", 30*time.Second, true); ok {
				fmt.Printf("Compile started: %s\n", title)
			} else {
				fmt.Println("Warning: Did not detect 'Compiling...' dialog within timeout")
			}
		}

		// Detect compile completion ("Compile Complete")
		if pid != 0 {
			fmt.Println("Waiting for 'Compile Complete' dialog...")
			if _, title, ok := waitForDialog(pid, "Compile Complete", 120*time.Second, true); ok {
				fmt.Printf("Compile completed: %s\n", title)
			} else {
				fmt.Println("Warning: Did not detect 'Compile Complete' dialog within timeout")
			}
		}
	}

	fmt.Println("\nPress Enter to exit...")
	fmt.Scanln()
}
