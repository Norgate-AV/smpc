// Package interfaces defines core interfaces for dependency injection and testing.
package interfaces

import (
	"time"

	"github.com/Norgate-AV/smpc/internal/windows"
)

// WindowManager handles window operations
type WindowManager interface {
	CloseWindow(hwnd uintptr, title string)
	SetForeground(hwnd uintptr) bool
	VerifyForegroundWindow(expectedHwnd uintptr, expectedPid uint32) bool
	IsElevated() bool
	CollectChildInfos(hwnd uintptr) []windows.ChildInfo
	WaitOnMonitor(timeout time.Duration, matchers ...func(windows.WindowEvent) bool) (windows.WindowEvent, bool)
}

// KeyboardInjector handles keyboard input
type KeyboardInjector interface {
	SendF12()
	SendAltF12()
	SendEnter()
	SendF12ToWindow(hwnd uintptr) bool
	SendAltF12ToWindow(hwnd uintptr) bool
	SendF12WithSendInput() bool
	SendAltF12WithSendInput() bool
}

// ProcessManager handles SIMPL process operations
type ProcessManager interface {
	FindWindow(targetPid uint32, debug bool) (uintptr, string)
	WaitForReady(hwnd uintptr, timeout time.Duration) bool
}

// ControlReader reads window controls
type ControlReader interface {
	GetListBoxItems(hwnd uintptr) []string
	GetEditText(hwnd uintptr) string
	FindAndClickButton(parentHwnd uintptr, buttonText string) bool
}
