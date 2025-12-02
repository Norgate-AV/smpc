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

	defer func() { _, _, _ = ProcCloseHandle.Call(token) }()

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
