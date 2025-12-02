package testutil

import (
	"time"

	"github.com/Norgate-AV/smpc/internal/windows"
)

// MockWindowManager records all calls for verification
type MockWindowManager struct {
	CloseWindowCalls             []CloseWindowCall
	SetForegroundCalls           []uintptr
	SetForegroundResult          bool
	VerifyForegroundWindowResult bool
	IsElevatedResult             bool
	ChildInfos                   []windows.ChildInfo
	ChildInfosMap                map[uintptr][]windows.ChildInfo
	WaitOnMonitorResults         []WaitOnMonitorResult
	currentWaitIndex             int
}

type CloseWindowCall struct {
	Hwnd  uintptr
	Title string
}

type WaitOnMonitorResult struct {
	Event windows.WindowEvent
	OK    bool
}

func NewMockWindowManager() *MockWindowManager {
	return &MockWindowManager{
		CloseWindowCalls:             []CloseWindowCall{},
		SetForegroundCalls:           []uintptr{},
		SetForegroundResult:          true,
		VerifyForegroundWindowResult: true,
		IsElevatedResult:             true,
		WaitOnMonitorResults:         []WaitOnMonitorResult{},
		ChildInfos:                   []windows.ChildInfo{},
		ChildInfosMap:                make(map[uintptr][]windows.ChildInfo),
	}
}

func (m *MockWindowManager) CloseWindow(hwnd uintptr, title string) {
	m.CloseWindowCalls = append(m.CloseWindowCalls, CloseWindowCall{hwnd, title})
}

func (m *MockWindowManager) SetForeground(hwnd uintptr) bool {
	m.SetForegroundCalls = append(m.SetForegroundCalls, hwnd)
	return m.SetForegroundResult
}

func (m *MockWindowManager) VerifyForegroundWindow(expectedHwnd uintptr, expectedPid uint32) bool {
	return m.VerifyForegroundWindowResult
}

func (m *MockWindowManager) IsElevated() bool {
	return m.IsElevatedResult
}

func (m *MockWindowManager) CollectChildInfos(hwnd uintptr) []windows.ChildInfo {
	// Check if we have hwnd-specific child infos
	if infos, ok := m.ChildInfosMap[hwnd]; ok {
		return infos
	}

	// Fall back to default ChildInfos
	return m.ChildInfos
}

func (m *MockWindowManager) WaitOnMonitor(timeout time.Duration, matchers ...func(windows.WindowEvent) bool) (windows.WindowEvent, bool) {
	if m.currentWaitIndex >= len(m.WaitOnMonitorResults) {
		return windows.WindowEvent{}, false
	}

	result := m.WaitOnMonitorResults[m.currentWaitIndex]
	m.currentWaitIndex++
	return result.Event, result.OK
}

// Helper methods for fluent configuration
func (m *MockWindowManager) WithWaitResult(title string, hwnd uintptr, ok bool) *MockWindowManager {
	m.WaitOnMonitorResults = append(m.WaitOnMonitorResults, WaitOnMonitorResult{
		Event: windows.WindowEvent{Title: title, Hwnd: hwnd},
		OK:    ok,
	})

	return m
}

func (m *MockWindowManager) WithChildInfo(className, text string) *MockWindowManager {
	m.ChildInfos = append(m.ChildInfos, windows.ChildInfo{
		ClassName: className,
		Text:      text,
	})

	return m
}

func (m *MockWindowManager) WithChildInfoItems(className string, items []string) *MockWindowManager {
	m.ChildInfos = append(m.ChildInfos, windows.ChildInfo{
		ClassName: className,
		Items:     items,
	})

	return m
}

func (m *MockWindowManager) WithElevated(elevated bool) *MockWindowManager {
	m.IsElevatedResult = elevated
	return m
}

func (m *MockWindowManager) WithSetForegroundResult(result bool) *MockWindowManager {
	m.SetForegroundResult = result
	return m
}

func (m *MockWindowManager) WithWaitOnMonitorResults(results ...WaitOnMonitorResult) *MockWindowManager {
	m.WaitOnMonitorResults = results
	m.currentWaitIndex = 0
	return m
}

func (m *MockWindowManager) WithChildInfos(infos ...windows.ChildInfo) *MockWindowManager {
	m.ChildInfos = infos
	return m
}

func (m *MockWindowManager) WithChildInfosForHwnd(hwnd uintptr, infos ...windows.ChildInfo) *MockWindowManager {
	m.ChildInfosMap[hwnd] = infos
	return m
}

// MockKeyboardInjector
type MockKeyboardInjector struct {
	SendF12Called                 bool
	SendAltF12Called              bool
	SendEnterCalled               bool
	SendF12ToWindowCalled         bool
	SendAltF12ToWindowCalled      bool
	SendF12WithSendInputCalled    bool
	SendAltF12WithSendInputCalled bool
	SendToWindowResult            bool
	SendInputResult               bool
}

func NewMockKeyboardInjector() *MockKeyboardInjector {
	return &MockKeyboardInjector{
		SendToWindowResult: true, // Default to success
		SendInputResult:    true, // Default to success
	}
}

func (m *MockKeyboardInjector) SendF12() {
	m.SendF12Called = true
}

func (m *MockKeyboardInjector) SendAltF12() {
	m.SendAltF12Called = true
}

func (m *MockKeyboardInjector) SendEnter() {
	m.SendEnterCalled = true
}

func (m *MockKeyboardInjector) SendF12ToWindow(hwnd uintptr) bool {
	m.SendF12ToWindowCalled = true
	return m.SendToWindowResult
}

func (m *MockKeyboardInjector) SendAltF12ToWindow(hwnd uintptr) bool {
	m.SendAltF12ToWindowCalled = true
	return m.SendToWindowResult
}

func (m *MockKeyboardInjector) SendF12WithSendInput() bool {
	m.SendF12WithSendInputCalled = true
	return m.SendInputResult
}

func (m *MockKeyboardInjector) SendAltF12WithSendInput() bool {
	m.SendAltF12WithSendInputCalled = true
	return m.SendInputResult
}

// MockControlReader
type MockControlReader struct {
	ListBoxItems            []string
	EditText                string
	FindButtonResult        bool
	FindButtonCalls         []string
	FindAndClickButtonCalls []FindAndClickButtonCall
}

type FindAndClickButtonCall struct {
	ParentHwnd uintptr
	ButtonText string
}

func NewMockControlReader() *MockControlReader {
	return &MockControlReader{
		FindButtonResult: true,
		FindButtonCalls:  []string{},
	}
}

func (m *MockControlReader) GetListBoxItems(hwnd uintptr) []string {
	return m.ListBoxItems
}

func (m *MockControlReader) GetEditText(hwnd uintptr) string {
	return m.EditText
}

func (m *MockControlReader) FindAndClickButton(parentHwnd uintptr, buttonText string) bool {
	m.FindButtonCalls = append(m.FindButtonCalls, buttonText)
	m.FindAndClickButtonCalls = append(m.FindAndClickButtonCalls, FindAndClickButtonCall{
		ParentHwnd: parentHwnd,
		ButtonText: buttonText,
	})

	return m.FindButtonResult
}

func (m *MockControlReader) WithListBoxItems(items []string) *MockControlReader {
	m.ListBoxItems = items
	return m
}

func (m *MockControlReader) WithEditText(text string) *MockControlReader {
	m.EditText = text
	return m
}

func (m *MockControlReader) WithFindButtonResult(result bool) *MockControlReader {
	m.FindButtonResult = result
	return m
}

func (m *MockControlReader) WithFindAndClickButtonResult(result bool) *MockControlReader {
	m.FindButtonResult = result
	return m
}
