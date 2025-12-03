//go:build windows

package windows

import (
	"fmt"
	"os"
	"strings"
	"unsafe"
)

func IsElevated() bool {
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

	defer func() {
		if ret, _, err := ProcCloseHandle.Call(token); ret == 0 {
			// Handle leak detected but can't log - no logger available in utility function
			// This is acceptable as IsElevated is called once per process lifetime
			_ = err
		}
	}()

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

func RelaunchAsAdmin() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	// Check if running via 'go run' (exe will be in temp dir)
	if strings.Contains(exe, "go-build") {
		return fmt.Errorf("cannot relaunch when run via 'go run', please build the executable first with: go build -o smpc.exe")
	}

	// Build args string (excluding the exe name)
	args := strings.Join(os.Args[1:], " ")

	return ShellExecute(0, "runas", exe, args, "", 1)
}
