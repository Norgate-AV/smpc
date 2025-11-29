package compiler

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Norgate-AV/smpc/internal/testutil"
	"github.com/Norgate-AV/smpc/internal/windows"
)

func TestDialogHandler_HandleOperationComplete(t *testing.T) {
	tests := []struct {
		name                 string
		pid                  uint32
		waitOnMonitorResults []testutil.WaitOnMonitorResult
		expectClose          bool
	}{
		{
			name: "dialog detected and dismissed",
			pid:  1234,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{
				{Event: windows.WindowEvent{Hwnd: 0x5678, Title: "Operation Complete"}, OK: true},
			},
			expectClose: true,
		},
		{
			name: "no dialog detected",
			pid:  1234,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{
				{Event: windows.WindowEvent{}, OK: false},
			},
			expectClose: false,
		},
		{
			name:                 "pid is zero - no action",
			pid:                  0,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{},
			expectClose:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWin := testutil.NewMockWindowManager().
				WithWaitOnMonitorResults(tt.waitOnMonitorResults...)
			mockKbd := testutil.NewMockKeyboardInjector()
			mockCtrl := testutil.NewMockControlReader()

			handler := NewDialogHandler(mockWin, mockKbd, mockCtrl)
			err := handler.HandleOperationComplete(tt.pid)

			assert.NoError(t, err)
			if tt.expectClose {
				assert.Len(t, mockWin.CloseWindowCalls, 1)
				assert.Equal(t, uintptr(0x5678), mockWin.CloseWindowCalls[0].Hwnd)
				assert.Equal(t, "Operation Complete", mockWin.CloseWindowCalls[0].Title)
			} else {
				assert.Len(t, mockWin.CloseWindowCalls, 0)
			}
		})
	}
}

func TestDialogHandler_HandleIncompleteSymbols(t *testing.T) {
	tests := []struct {
		name                 string
		pid                  uint32
		waitOnMonitorResults []testutil.WaitOnMonitorResult
		childInfos           []windows.ChildInfo
		expectError          bool
	}{
		{
			name: "incomplete symbols dialog detected",
			pid:  1234,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{
				{Event: windows.WindowEvent{Hwnd: 0x9ABC, Title: "Incomplete Symbols"}, OK: true},
			},
			childInfos: []windows.ChildInfo{
				{ClassName: "Edit", Text: strings.Repeat("Error details about incomplete symbols ", 5)},
			},
			expectError: true,
		},
		{
			name: "no incomplete symbols dialog",
			pid:  1234,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{
				{Event: windows.WindowEvent{}, OK: false},
			},
			expectError: false,
		},
		{
			name:                 "pid is zero - no action",
			pid:                  0,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{},
			expectError:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWin := testutil.NewMockWindowManager().
				WithWaitOnMonitorResults(tt.waitOnMonitorResults...).
				WithChildInfos(tt.childInfos...)
			mockKbd := testutil.NewMockKeyboardInjector()
			mockCtrl := testutil.NewMockControlReader()

			handler := NewDialogHandler(mockWin, mockKbd, mockCtrl)
			err := handler.HandleIncompleteSymbols(tt.pid)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "incomplete symbols")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDialogHandler_HandleConvertCompile(t *testing.T) {
	tests := []struct {
		name                 string
		pid                  uint32
		waitOnMonitorResults []testutil.WaitOnMonitorResult
		expectEnter          bool
	}{
		{
			name: "convert/compile dialog detected",
			pid:  1234,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{
				{Event: windows.WindowEvent{Hwnd: 0xDEF0, Title: "Convert/Compile"}, OK: true},
			},
			expectEnter: true,
		},
		{
			name: "no convert/compile dialog",
			pid:  1234,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{
				{Event: windows.WindowEvent{}, OK: false},
			},
			expectEnter: false,
		},
		{
			name:                 "pid is zero - no action",
			pid:                  0,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{},
			expectEnter:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWin := testutil.NewMockWindowManager().
				WithWaitOnMonitorResults(tt.waitOnMonitorResults...)
			mockKbd := testutil.NewMockKeyboardInjector()
			mockCtrl := testutil.NewMockControlReader()

			handler := NewDialogHandler(mockWin, mockKbd, mockCtrl)
			err := handler.HandleConvertCompile(tt.pid)

			assert.NoError(t, err)
			if tt.expectEnter {
				assert.True(t, mockKbd.SendEnterCalled)
			} else {
				assert.False(t, mockKbd.SendEnterCalled)
			}
		})
	}
}

func TestDialogHandler_HandleCommentedOutSymbols(t *testing.T) {
	tests := []struct {
		name                 string
		pid                  uint32
		waitOnMonitorResults []testutil.WaitOnMonitorResult
		expectEnter          bool
	}{
		{
			name: "commented out symbols dialog detected",
			pid:  1234,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{
				{Event: windows.WindowEvent{Hwnd: 0x1234, Title: "Commented Out Symbols"}, OK: true},
			},
			expectEnter: true,
		},
		{
			name: "no commented out symbols dialog",
			pid:  1234,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{
				{Event: windows.WindowEvent{}, OK: false},
			},
			expectEnter: false,
		},
		{
			name:                 "pid is zero - no action",
			pid:                  0,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{},
			expectEnter:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWin := testutil.NewMockWindowManager().
				WithWaitOnMonitorResults(tt.waitOnMonitorResults...)
			mockKbd := testutil.NewMockKeyboardInjector()
			mockCtrl := testutil.NewMockControlReader()

			handler := NewDialogHandler(mockWin, mockKbd, mockCtrl)
			err := handler.HandleCommentedOutSymbols(tt.pid)

			assert.NoError(t, err)
			if tt.expectEnter {
				assert.True(t, mockKbd.SendEnterCalled)
			} else {
				assert.False(t, mockKbd.SendEnterCalled)
			}
		})
	}
}

func TestDialogHandler_WaitForCompiling(t *testing.T) {
	tests := []struct {
		name                 string
		pid                  uint32
		waitOnMonitorResults []testutil.WaitOnMonitorResult
		expectError          bool
	}{
		{
			name: "compiling dialog detected",
			pid:  1234,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{
				{Event: windows.WindowEvent{Hwnd: 0x5678, Title: "Compiling..."}, OK: true},
			},
			expectError: false,
		},
		{
			name: "compiling dialog timeout",
			pid:  1234,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{
				{Event: windows.WindowEvent{}, OK: false},
			},
			expectError: false, // WaitForCompiling logs warning but doesn't error
		},
		{
			name:                 "pid is zero - no action",
			pid:                  0,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{},
			expectError:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWin := testutil.NewMockWindowManager().
				WithWaitOnMonitorResults(tt.waitOnMonitorResults...)
			mockKbd := testutil.NewMockKeyboardInjector()
			mockCtrl := testutil.NewMockControlReader()

			handler := NewDialogHandler(mockWin, mockKbd, mockCtrl)
			err := handler.WaitForCompiling(tt.pid)

			if tt.expectError {
				assert.Error(t, err)
				if err != nil {
					assert.Contains(t, err.Error(), "Compiling")
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDialogHandler_ParseCompileComplete(t *testing.T) {
	tests := []struct {
		name                 string
		pid                  uint32
		waitOnMonitorResults []testutil.WaitOnMonitorResult
		childInfos           []windows.ChildInfo
		expectError          bool
		expectStats          bool
		expectedErrors       int
		expectedWarnings     int
		expectedNotices      int
	}{
		{
			name: "compile complete with statistics",
			pid:  1234,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{
				{Event: windows.WindowEvent{Hwnd: 0xABCD, Title: "Compile Complete"}, OK: true},
			},
			childInfos: []windows.ChildInfo{
				{ClassName: "Static", Text: "Statistics"},
				{ClassName: "Edit", Text: "Program Errors: 2\r\nProgram Warnings: 5\r\nProgram Notices: 3\r\nCompile Time: 1.23 seconds\r\n"},
			},
			expectError:      false,
			expectStats:      true,
			expectedErrors:   2,
			expectedWarnings: 5,
			expectedNotices:  3,
		},
		{
			name: "compile complete timeout",
			pid:  1234,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{
				{Event: windows.WindowEvent{}, OK: false},
			},
			expectError: true,
		},
		{
			name:                 "pid is zero - no action",
			pid:                  0,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{},
			expectError:          false,
			expectStats:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWin := testutil.NewMockWindowManager().
				WithWaitOnMonitorResults(tt.waitOnMonitorResults...).
				WithChildInfos(tt.childInfos...)
			mockKbd := testutil.NewMockKeyboardInjector()
			mockCtrl := testutil.NewMockControlReader()

			handler := NewDialogHandler(mockWin, mockKbd, mockCtrl)
			_, warnings, notices, errors, compileTime, err := handler.ParseCompileComplete(tt.pid)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expectStats {
					assert.Equal(t, tt.expectedErrors, errors)
					assert.Equal(t, tt.expectedWarnings, warnings)
					assert.Equal(t, tt.expectedNotices, notices)
					assert.InDelta(t, 1.23, compileTime, 0.01)
				}
			}
		})
	}
}

func TestDialogHandler_ParseProgramCompilation(t *testing.T) {
	tests := []struct {
		name                 string
		pid                  uint32
		waitOnMonitorResults []testutil.WaitOnMonitorResult
		childInfos           []windows.ChildInfo
		expectError          bool
		expectedWarnings     int
		expectedNotices      int
		expectedErrors       int
	}{
		{
			name: "program compilation with messages",
			pid:  1234,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{
				{Event: windows.WindowEvent{Hwnd: 0x7890, Title: "Program Compilation"}, OK: true},
			},
			childInfos: []windows.ChildInfo{
				{ClassName: "ListBox", Items: []string{
					"ERROR: Line 10: Undefined symbol 'foo'",
					"WARNING: Line 20: Unused variable 'bar'",
					"NOTICE: Line 30: Optimization applied",
				}},
			},
			expectError:      false,
			expectedErrors:   1,
			expectedWarnings: 1,
			expectedNotices:  1,
		},
		{
			name: "no program compilation dialog",
			pid:  1234,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{
				{Event: windows.WindowEvent{}, OK: false},
			},
			expectError:      false,
			expectedErrors:   0,
			expectedWarnings: 0,
			expectedNotices:  0,
		},
		{
			name:                 "pid is zero - no action",
			pid:                  0,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{},
			expectError:          false,
			expectedErrors:       0,
			expectedWarnings:     0,
			expectedNotices:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWin := testutil.NewMockWindowManager().
				WithWaitOnMonitorResults(tt.waitOnMonitorResults...).
				WithChildInfos(tt.childInfos...)
			mockKbd := testutil.NewMockKeyboardInjector()
			mockCtrl := testutil.NewMockControlReader()

			handler := NewDialogHandler(mockWin, mockKbd, mockCtrl)
			warnings, notices, errors, err := handler.ParseProgramCompilation(tt.pid)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, warnings, tt.expectedWarnings)
				assert.Len(t, notices, tt.expectedNotices)
				assert.Len(t, errors, tt.expectedErrors)
			}
		})
	}
}

func TestDialogHandler_HandleConfirmation(t *testing.T) {
	tests := []struct {
		name                 string
		pid                  uint32
		waitOnMonitorResults []testutil.WaitOnMonitorResult
		findAndClickResult   bool
		expectClick          bool
	}{
		{
			name: "confirmation dialog with yes button",
			pid:  1234,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{
				{Event: windows.WindowEvent{Hwnd: 0xBEEF, Title: "Confirm"}, OK: true},
			},
			findAndClickResult: true,
			expectClick:        true,
		},
		{
			name: "no confirmation dialog",
			pid:  1234,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{
				{Event: windows.WindowEvent{}, OK: false},
			},
			findAndClickResult: false,
			expectClick:        false,
		},
		{
			name:                 "pid is zero - no action",
			pid:                  0,
			waitOnMonitorResults: []testutil.WaitOnMonitorResult{},
			findAndClickResult:   false,
			expectClick:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWin := testutil.NewMockWindowManager().
				WithWaitOnMonitorResults(tt.waitOnMonitorResults...)
			mockKbd := testutil.NewMockKeyboardInjector()
			mockCtrl := testutil.NewMockControlReader().
				WithFindAndClickButtonResult(tt.findAndClickResult)

			handler := NewDialogHandler(mockWin, mockKbd, mockCtrl)
			err := handler.HandleConfirmation(tt.pid)

			assert.NoError(t, err)
			if tt.expectClick {
				assert.Len(t, mockCtrl.FindAndClickButtonCalls, 1)
			} else {
				assert.Len(t, mockCtrl.FindAndClickButtonCalls, 0)
			}
		})
	}
}
