package testutil

import "time"

// MockProcessManager implements interfaces.ProcessManager for testing
type MockProcessManager struct {
	pid                uint32 // Internal PID for WithPid() method
	FindWindowResult   uintptr
	FindWindowTitle    string
	WaitForReadyResult bool
	FindWindowCalls    []FindWindowCall
}

type FindWindowCall struct {
	TargetPid uint32
	Debug     bool
}

func NewMockProcessManager() *MockProcessManager {
	return &MockProcessManager{
		pid:                0,
		FindWindowResult:   0,
		FindWindowTitle:    "",
		WaitForReadyResult: true,
		FindWindowCalls:    []FindWindowCall{},
	}
}

func (m *MockProcessManager) WithPid(pid uint32) *MockProcessManager {
	m.pid = pid
	return m
}

func (m *MockProcessManager) FindWindow(targetPid uint32, debug bool) (uintptr, string) {
	m.FindWindowCalls = append(m.FindWindowCalls, FindWindowCall{targetPid, debug})
	return m.FindWindowResult, m.FindWindowTitle
}

func (m *MockProcessManager) WaitForReady(hwnd uintptr, timeout time.Duration) bool {
	return m.WaitForReadyResult
}

// Helper methods for fluent configuration
func (m *MockProcessManager) WithFindWindowResult(hwnd uintptr, title string) *MockProcessManager {
	m.FindWindowResult = hwnd
	m.FindWindowTitle = title
	return m
}

func (m *MockProcessManager) WithWaitForReadyResult(result bool) *MockProcessManager {
	m.WaitForReadyResult = result
	return m
}
