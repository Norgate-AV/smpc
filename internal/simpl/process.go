package simpl

import (
	"strings"
	"syscall"
	"unsafe"

	"github.com/Norgate-AV/smpc/internal/windows"
)

// findProcessByName searches for a process by executable name (case-insensitive)
// Returns the process ID if found, 0 otherwise
func findProcessByName(processName string) uint32 {
	snapshot, _, _ := windows.ProcCreateToolhelp32Snapshot.Call(windows.TH32CS_SNAPPROCESS, 0)
	if snapshot == 0 {
		return 0
	}

	defer func() { _, _, _ = windows.ProcCloseHandle.Call(snapshot) }()

	var pe windows.PROCESSENTRY32
	pe.DwSize = uint32(unsafe.Sizeof(pe))

	ret, _, _ := windows.ProcProcess32First.Call(snapshot, uintptr(unsafe.Pointer(&pe)))
	if ret == 0 {
		return 0
	}

	processName = strings.ToLower(processName)

	for {
		exeName := syscall.UTF16ToString(pe.SzExeFile[:])
		if strings.ToLower(exeName) == processName {
			return pe.Th32ProcessID
		}

		ret, _, _ := windows.ProcProcess32Next.Call(snapshot, uintptr(unsafe.Pointer(&pe)))
		if ret == 0 {
			break
		}
	}

	return 0
}
