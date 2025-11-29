package simpl

import "time"

// RealProcessManager implements interfaces.ProcessManager
type RealProcessManager struct{}

func NewRealProcessManager() *RealProcessManager {
	return &RealProcessManager{}
}

func (r *RealProcessManager) GetPid() uint32 {
	return GetPid()
}

func (r *RealProcessManager) FindWindow(processName string, debug bool) (uintptr, string) {
	return FindWindow(processName, debug)
}

func (r *RealProcessManager) WaitForReady(hwnd uintptr, timeout time.Duration) bool {
	return WaitForReady(hwnd, timeout)
}
