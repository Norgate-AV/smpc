package windows

import (
	"fmt"
	"log/slog"
	"syscall"
	"time"
	"unsafe"
)

func CloseWindow(hwnd uintptr, title string) {
	slog.Info("Closing window", "title", title)
	_, _, _ = procPostMessageW.Call(hwnd, WM_CLOSE, 0, 0)
	time.Sleep(500 * time.Millisecond)
}

func SetForeground(hwnd uintptr) bool {
	// Restore window if minimized, then bring to foreground
	r1, r2, lastErr := procShowWindow.Call(hwnd, uintptr(SW_RESTORE))
	slog.Debug("ShowWindow(SW_RESTORE)", "r1", r1, "r2", r2, "err", lastErr)

	ret, _, err := procSetForegroundWindow.Call(hwnd)
	if ret == 0 {
		slog.Debug("SetForegroundWindow failed", "error", err)
		return false
	}

	slog.Debug("SetForegroundWindow succeeded")

	// Give it a moment and verify
	time.Sleep(500 * time.Millisecond)
	fgHwnd, _, _ := procGetForegroundWindow.Call()
	if fgHwnd == hwnd {
		slog.Debug("Window confirmed in foreground")
	} else {
		slog.Warn("Different window in foreground", "expected", hwnd, "got", fgHwnd)
	}

	return true
}

func ShellExecute(hwnd uintptr, verb, file, args, cwd string, showCmd int) error {
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

func GetWindowText(hwnd uintptr) string {
	buf := make([]uint16, 256)
	_, _, _ = procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

func GetClassName(hwnd uintptr) string {
	buf := make([]uint16, 256)
	_, _, _ = procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

func IsWindowVisible(hwnd uintptr) bool {
	ret, _, _ := procIsWindowVisible.Call(hwnd)
	return ret != 0
}

func GetWindowPid(hwnd uintptr) uint32 {
	var pid uint32
	_, _, _ = procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	return pid
}

// TerminateProcess forcefully terminates a process by its PID
func TerminateProcess(pid uint32) error {
	const PROCESS_TERMINATE = 0x0001

	// Open the process with terminate rights
	hProcess, _, err := procOpenProcess.Call(
		uintptr(PROCESS_TERMINATE),
		uintptr(0),
		uintptr(pid),
	)

	if hProcess == 0 {
		return fmt.Errorf("failed to open process: %v", err)
	}
	defer func() { _, _, _ = ProcCloseHandle.Call(hProcess) }()

	// Terminate the process
	ret, _, err := procTerminateProcess.Call(hProcess, uintptr(1))
	if ret == 0 {
		return fmt.Errorf("failed to terminate process: %v", err)
	}

	return nil
}
