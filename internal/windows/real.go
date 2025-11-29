package windows

import (
	"time"
)

// RealWindowManager implements interfaces.WindowManager using real Windows APIs
type RealWindowManager struct{}

func NewRealWindowManager() *RealWindowManager {
	return &RealWindowManager{}
}

func (r *RealWindowManager) CloseWindow(hwnd uintptr, title string) {
	CloseWindow(hwnd, title)
}

func (r *RealWindowManager) SetForeground(hwnd uintptr) bool {
	return SetForeground(hwnd)
}

func (r *RealWindowManager) IsElevated() bool {
	return IsElevated()
}

func (r *RealWindowManager) CollectChildInfos(hwnd uintptr) []ChildInfo {
	return CollectChildInfos(hwnd)
}

func (r *RealWindowManager) WaitOnMonitor(timeout time.Duration, matchers ...func(WindowEvent) bool) (WindowEvent, bool) {
	return WaitOnMonitor(timeout, matchers...)
}

// RealKeyboardInjector implements interfaces.KeyboardInjector
type RealKeyboardInjector struct{}

func NewRealKeyboardInjector() *RealKeyboardInjector {
	return &RealKeyboardInjector{}
}

func (r *RealKeyboardInjector) SendF12() bool {
	return SendF12()
}

func (r *RealKeyboardInjector) SendAltF12() bool {
	return SendAltF12()
}

func (r *RealKeyboardInjector) SendEnter() bool {
	return SendEnter()
}

// RealControlReader implements interfaces.ControlReader
type RealControlReader struct{}

func NewRealControlReader() *RealControlReader {
	return &RealControlReader{}
}

func (r *RealControlReader) GetListBoxItems(hwnd uintptr) []string {
	return GetListBoxItems(hwnd)
}

func (r *RealControlReader) GetEditText(hwnd uintptr) string {
	return GetEditText(hwnd)
}

func (r *RealControlReader) FindAndClickButton(parentHwnd uintptr, buttonText string) bool {
	return FindAndClickButton(parentHwnd, buttonText)
}
