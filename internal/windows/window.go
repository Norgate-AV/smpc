package windows

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

func CloseWindow(hwnd uintptr, title string) {
	fmt.Printf("Closing window: %s\n", title)
	procPostMessageW.Call(hwnd, WM_CLOSE, 0, 0)
	time.Sleep(500 * time.Millisecond)
}

func SetForeground(hwnd uintptr) bool {
	// Restore window if minimized, then bring to foreground
	r1, r2, lastErr := procShowWindow.Call(hwnd, uintptr(SW_RESTORE))
	fmt.Printf("[DEBUG] ShowWindow(SW_RESTORE) r1=%d r2=%d err=%v\n", r1, r2, lastErr)

	ret, _, err := procSetForegroundWindow.Call(hwnd)
	if ret == 0 {
		fmt.Printf("[DEBUG] SetForegroundWindow failed: %v\n", err)
		return false
	}

	fmt.Println("[DEBUG] SetForegroundWindow succeeded")

	// Give it a moment and verify
	time.Sleep(500 * time.Millisecond)
	fgHwnd, _, _ := procGetForegroundWindow.Call()
	if fgHwnd == hwnd {
		fmt.Println("[DEBUG] Window confirmed in foreground")
	} else {
		fmt.Printf("[DEBUG] WARNING - Different window in foreground (expected %d, got %d)\n", hwnd, fgHwnd)
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
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

func GetClassName(hwnd uintptr) string {
	buf := make([]uint16, 256)
	procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

func IsWindowVisible(hwnd uintptr) bool {
	ret, _, _ := procIsWindowVisible.Call(hwnd)
	return ret != 0
}

func GetWindowProcessId(hwnd uintptr) uint32 {
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	return pid
}
