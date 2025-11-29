package testutil

import (
	"time"

	"github.com/Norgate-AV/smpc/internal/windows"
)

// MockWindowManager records all calls for verification
type MockWindowManager struct {
	CloseWindowCalls     []CloseWindowCall
	SetForegroundCalls   []uintptr
	SetForegroundResult  bool
	IsElevatedResult     bool
	ChildInfos           []windows.ChildInfo
	ChildInfosMap        map[uintptr][]windows.ChildInfo
	WaitOnMonitorResults []WaitOnMonitorResult
	currentWaitIndex     int
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
		CloseWindowCalls:     []CloseWindowCall{},
		SetForegroundCalls:   []uintptr{},
		SetForegroundResult:  true,
		IsElevatedResult:     true,
		WaitOnMonitorResults: []WaitOnMonitorResult{},
		ChildInfos:           []windows.ChildInfo{},
		ChildInfosMap:        make(map[uintptr][]windows.ChildInfo),
	}
}

func (m *MockWindowManager) CloseWindow(hwnd uintptr, title string) {
	m.CloseWindowCalls = append(m.CloseWindowCalls, CloseWindowCall{hwnd, title})
}

func (m *MockWindowManager) SetForeground(hwnd uintptr) bool {
	m.SetForegroundCalls = append(m.SetForegroundCalls, hwnd)
	return m.SetForegroundResult
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
	SendF12Called    bool
	SendAltF12Called bool
	SendEnterCalled  bool
	SendF12Result    bool
	SendAltF12Result bool
	SendEnterResult  bool
}

func NewMockKeyboardInjector() *MockKeyboardInjector {
	return &MockKeyboardInjector{
		SendF12Result:    true,
		SendAltF12Result: true,
		SendEnterResult:  true,
	}
}

func (m *MockKeyboardInjector) SendF12() bool {
	m.SendF12Called = true
	return m.SendF12Result
}

func (m *MockKeyboardInjector) SendAltF12() bool {
	m.SendAltF12Called = true
	return m.SendAltF12Result
}

func (m *MockKeyboardInjector) SendEnter() bool {
	m.SendEnterCalled = true
	return m.SendEnterResult
}

func (m *MockKeyboardInjector) WithSendF12Result(result bool) *MockKeyboardInjector {
	m.SendF12Result = result
	return m
}

func (m *MockKeyboardInjector) WithSendAltF12Result(result bool) *MockKeyboardInjector {
	m.SendAltF12Result = result
	return m
}

func (m *MockKeyboardInjector) WithSendEnterResult(result bool) *MockKeyboardInjector {
	m.SendEnterResult = result
	return m
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
