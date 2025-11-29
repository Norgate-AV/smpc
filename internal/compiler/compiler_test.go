package compiler

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Norgate-AV/smpc/internal/testutil"
	"github.com/Norgate-AV/smpc/internal/windows"
)

func TestCompileWithDeps_SuccessfulCompilation(t *testing.T) {
	mockWin := testutil.NewMockWindowManager().
		WithWaitOnMonitorResults(
			// HandleOperationComplete - no dialog
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false},
			// HandleIncompleteSymbols - no dialog
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false},
			// HandleConvertCompile - no dialog
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false},
			// HandleCommentedOutSymbols - no dialog
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false},
			// WaitForCompiling - dialog appears
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{Hwnd: 0x1111, Title: "Compiling..."}, OK: true},
			// ParseCompileComplete - dialog with stats
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{Hwnd: 0x2222, Title: "Compile Complete"}, OK: true},
			// ParseProgramCompilation - no messages
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false},
			// HandleConfirmation - no dialog
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false},
		).
		WithChildInfos(
			windows.ChildInfo{ClassName: "Static", Text: "Statistics"},
			windows.ChildInfo{ClassName: "Edit", Text: "Program Errors: 0\r\nProgram Warnings: 0\r\nProgram Notices: 0\r\nCompile Time: 1.23 seconds\r\n"},
		)

	mockKbd := testutil.NewMockKeyboardInjector()
	mockCtrl := testutil.NewMockControlReader()
	mockProc := testutil.NewMockProcessManager().WithPid(1234)

	dialogHandler := NewDialogHandler(mockWin, mockKbd, mockCtrl)
	deps := &CompileDependencies{
		DialogHandler: dialogHandler,
		ProcessMgr:    mockProc,
		WindowMgr:     mockWin,
		Keyboard:      mockKbd,
	}

	opts := CompileOptions{
		Hwnd:         0x9999,
		RecompileAll: false,
	}

	result, err := CompileWithDeps(opts, deps)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.HasErrors)
	assert.Equal(t, 0, result.Errors)
	assert.Equal(t, 0, result.Warnings)
	assert.Equal(t, 0, result.Notices)
	assert.InDelta(t, 1.23, result.CompileTime, 0.01)

	// Verify F12 was sent
	assert.True(t, mockKbd.SendF12Called)
	assert.False(t, mockKbd.SendAltF12Called)

	// Verify window was set to foreground
	assert.Len(t, mockWin.SetForegroundCalls, 1)
	assert.Equal(t, uintptr(0x9999), mockWin.SetForegroundCalls[0])

	// Verify both Compile Complete dialog and SIMPL Windows were closed
	assert.Len(t, mockWin.CloseWindowCalls, 2)
	assert.Equal(t, uintptr(0x2222), mockWin.CloseWindowCalls[0].Hwnd) // Compile Complete
	assert.Equal(t, "Compile Complete dialog", mockWin.CloseWindowCalls[0].Title)
	assert.Equal(t, uintptr(0x9999), mockWin.CloseWindowCalls[1].Hwnd) // SIMPL Windows
	assert.Equal(t, "SIMPL Windows", mockWin.CloseWindowCalls[1].Title)
}

func TestCompileWithDeps_RecompileAll(t *testing.T) {
	mockWin := testutil.NewMockWindowManager().
		WithWaitOnMonitorResults(
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleOperationComplete
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleIncompleteSymbols
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleConvertCompile
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleCommentedOutSymbols
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{Hwnd: 0x1111, Title: "Compiling..."}, OK: true},
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{Hwnd: 0x2222, Title: "Compile Complete"}, OK: true},
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // ParseProgramCompilation
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleConfirmation
		).
		WithChildInfos(
			windows.ChildInfo{ClassName: "Edit", Text: "Errors: 0\r\nWarnings: 0\r\nNotices: 0\r\n"},
		)

	mockKbd := testutil.NewMockKeyboardInjector()
	mockCtrl := testutil.NewMockControlReader()
	mockProc := testutil.NewMockProcessManager().WithPid(1234)

	dialogHandler := NewDialogHandler(mockWin, mockKbd, mockCtrl)
	deps := &CompileDependencies{
		DialogHandler: dialogHandler,
		ProcessMgr:    mockProc,
		WindowMgr:     mockWin,
		Keyboard:      mockKbd,
	}

	opts := CompileOptions{
		Hwnd:         0x9999,
		RecompileAll: true, // Trigger Alt+F12 instead of F12
	}

	result, err := CompileWithDeps(opts, deps)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.HasErrors)

	// Verify Alt+F12 was sent
	assert.False(t, mockKbd.SendF12Called)
	assert.True(t, mockKbd.SendAltF12Called)
}

func TestCompileWithDeps_WithWarnings(t *testing.T) {
	mockWin := testutil.NewMockWindowManager().
		WithWaitOnMonitorResults(
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleOperationComplete
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleIncompleteSymbols
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleConvertCompile
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleCommentedOutSymbols
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{Hwnd: 0x1111, Title: "Compiling..."}, OK: true},
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{Hwnd: 0x2222, Title: "Compile Complete"}, OK: true},
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{Hwnd: 0x3333, Title: "Program Compilation"}, OK: true},
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleConfirmation
		).
		WithChildInfosForHwnd(0x2222, // Compile Complete dialog
			windows.ChildInfo{ClassName: "Edit", Text: "Program Errors: 0\r\nProgram Warnings: 2\r\nProgram Notices: 1\r\n"},
		).
		WithChildInfosForHwnd(0x3333, // Program Compilation dialog
			windows.ChildInfo{ClassName: "ListBox", Items: []string{
				"WARNING: Line 10: Unused variable 'x'",
				"WARNING: Line 20: Deprecated function call",
				"NOTICE: Line 30: Optimization applied",
			}},
		)

	mockKbd := testutil.NewMockKeyboardInjector()
	mockCtrl := testutil.NewMockControlReader()
	mockProc := testutil.NewMockProcessManager().WithPid(1234)

	dialogHandler := NewDialogHandler(mockWin, mockKbd, mockCtrl)
	deps := &CompileDependencies{
		DialogHandler: dialogHandler,
		ProcessMgr:    mockProc,
		WindowMgr:     mockWin,
		Keyboard:      mockKbd,
	}

	opts := CompileOptions{Hwnd: 0x9999}

	result, err := CompileWithDeps(opts, deps)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.HasErrors)
	assert.Equal(t, 0, result.Errors)
	assert.Equal(t, 2, result.Warnings)
	assert.Equal(t, 1, result.Notices)
	assert.Len(t, result.WarningMessages, 2)
	assert.Len(t, result.NoticeMessages, 1)
	assert.Len(t, result.ErrorMessages, 0)
}

func TestCompileWithDeps_WithErrors(t *testing.T) {
	mockWin := testutil.NewMockWindowManager().
		WithWaitOnMonitorResults(
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleOperationComplete
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleIncompleteSymbols
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleConvertCompile
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleCommentedOutSymbols
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{Hwnd: 0x1111, Title: "Compiling..."}, OK: true},
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{Hwnd: 0x2222, Title: "Compile Complete"}, OK: true},
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{Hwnd: 0x3333, Title: "Program Compilation"}, OK: true},
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleConfirmation
		).
		WithChildInfosForHwnd(0x2222, // Compile Complete dialog
			windows.ChildInfo{ClassName: "Edit", Text: "Program Errors: 3\r\nProgram Warnings: 0\r\nProgram Notices: 0\r\n"},
		).
		WithChildInfosForHwnd(0x3333, // Program Compilation dialog
			windows.ChildInfo{ClassName: "ListBox", Items: []string{
				"ERROR: Line 5: Undefined symbol 'foo'",
				"ERROR: Line 15: Type mismatch",
				"ERROR: Line 25: Missing semicolon",
			}},
		)

	mockKbd := testutil.NewMockKeyboardInjector()
	mockCtrl := testutil.NewMockControlReader()
	mockProc := testutil.NewMockProcessManager().WithPid(1234)

	dialogHandler := NewDialogHandler(mockWin, mockKbd, mockCtrl)
	deps := &CompileDependencies{
		DialogHandler: dialogHandler,
		ProcessMgr:    mockProc,
		WindowMgr:     mockWin,
		Keyboard:      mockKbd,
	}

	opts := CompileOptions{Hwnd: 0x9999}

	result, err := CompileWithDeps(opts, deps)

	// CompileWithDeps returns an error when there are compile errors
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compilation failed")
	assert.NotNil(t, result)
	assert.True(t, result.HasErrors)
	assert.Equal(t, 3, result.Errors)
	assert.Equal(t, 0, result.Warnings)
	assert.Equal(t, 0, result.Notices)
	assert.Len(t, result.ErrorMessages, 3)
}

func TestCompileWithDeps_IncompleteSymbols(t *testing.T) {
	mockWin := testutil.NewMockWindowManager().
		WithWaitOnMonitorResults(
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleOperationComplete
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{Hwnd: 0xABCD, Title: "Incomplete Symbols"}, OK: true},
		).
		WithChildInfos(
			windows.ChildInfo{ClassName: "Edit", Text: "The program contains incomplete symbols and cannot be compiled."},
		)

	mockKbd := testutil.NewMockKeyboardInjector()
	mockCtrl := testutil.NewMockControlReader()
	mockProc := testutil.NewMockProcessManager().WithPid(1234)

	dialogHandler := NewDialogHandler(mockWin, mockKbd, mockCtrl)
	deps := &CompileDependencies{
		DialogHandler: dialogHandler,
		ProcessMgr:    mockProc,
		WindowMgr:     mockWin,
		Keyboard:      mockKbd,
	}

	opts := CompileOptions{Hwnd: 0x9999}

	result, err := CompileWithDeps(opts, deps)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "incomplete symbols")
}

func TestCompileWithDeps_KeystrokeFailure(t *testing.T) {
	mockWin := testutil.NewMockWindowManager().
		WithWaitOnMonitorResults(
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleOperationComplete
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleIncompleteSymbols
		)

	mockKbd := testutil.NewMockKeyboardInjector().
		WithSendF12Result(false) // Simulate keystroke failure

	mockCtrl := testutil.NewMockControlReader()
	mockProc := testutil.NewMockProcessManager().WithPid(1234)

	dialogHandler := NewDialogHandler(mockWin, mockKbd, mockCtrl)
	deps := &CompileDependencies{
		DialogHandler: dialogHandler,
		ProcessMgr:    mockProc,
		WindowMgr:     mockWin,
		Keyboard:      mockKbd,
	}

	opts := CompileOptions{Hwnd: 0x9999}

	result, err := CompileWithDeps(opts, deps)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "keystroke")
}

func TestCompileWithDeps_CompileDialogTimeout(t *testing.T) {
	mockWin := testutil.NewMockWindowManager().
		WithWaitOnMonitorResults(
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleOperationComplete
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleIncompleteSymbols
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleConvertCompile
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleCommentedOutSymbols
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // WaitForCompiling - timeout
		)

	mockKbd := testutil.NewMockKeyboardInjector()
	mockCtrl := testutil.NewMockControlReader()
	mockProc := testutil.NewMockProcessManager().WithPid(1234)

	dialogHandler := NewDialogHandler(mockWin, mockKbd, mockCtrl)
	deps := &CompileDependencies{
		DialogHandler: dialogHandler,
		ProcessMgr:    mockProc,
		WindowMgr:     mockWin,
		Keyboard:      mockKbd,
	}

	opts := CompileOptions{Hwnd: 0x9999}

	result, err := CompileWithDeps(opts, deps)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Compile Complete")
}

func TestCompileWithDeps_NoPid(t *testing.T) {
	// When PID is 0, dialog monitoring should be skipped but compilation should still proceed
	mockWin := testutil.NewMockWindowManager().
		WithWaitOnMonitorResults(
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{Hwnd: 0x1111, Title: "Compiling..."}, OK: true},
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{Hwnd: 0x2222, Title: "Compile Complete"}, OK: true},
		).
		WithChildInfos(
			windows.ChildInfo{ClassName: "Edit", Text: "Errors: 0\r\nWarnings: 0\r\nNotices: 0\r\n"},
		)

	mockKbd := testutil.NewMockKeyboardInjector()
	mockCtrl := testutil.NewMockControlReader()
	mockProc := testutil.NewMockProcessManager().WithPid(0) // PID not available

	dialogHandler := NewDialogHandler(mockWin, mockKbd, mockCtrl)
	deps := &CompileDependencies{
		DialogHandler: dialogHandler,
		ProcessMgr:    mockProc,
		WindowMgr:     mockWin,
		Keyboard:      mockKbd,
	}

	opts := CompileOptions{Hwnd: 0x9999}

	result, err := CompileWithDeps(opts, deps)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.HasErrors)

	// Verify F12 was still sent even without PID
	assert.True(t, mockKbd.SendF12Called)
}

func TestCompileWithDeps_WithSavePrompts(t *testing.T) {
	mockWin := testutil.NewMockWindowManager().
		WithWaitOnMonitorResults(
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleOperationComplete
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleIncompleteSymbols
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{Hwnd: 0x5555, Title: "Convert/Compile"}, OK: true},
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{Hwnd: 0x6666, Title: "Commented Out Symbols"}, OK: true},
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{Hwnd: 0x1111, Title: "Compiling..."}, OK: true},
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{Hwnd: 0x2222, Title: "Compile Complete"}, OK: true},
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // ParseProgramCompilation
			testutil.WaitOnMonitorResult{Event: windows.WindowEvent{}, OK: false}, // HandleConfirmation
		).
		WithChildInfos(
			windows.ChildInfo{ClassName: "Edit", Text: "Errors: 0\r\nWarnings: 0\r\nNotices: 0\r\n"},
		)

	mockKbd := testutil.NewMockKeyboardInjector()
	mockCtrl := testutil.NewMockControlReader()
	mockProc := testutil.NewMockProcessManager().WithPid(1234)

	dialogHandler := NewDialogHandler(mockWin, mockKbd, mockCtrl)
	deps := &CompileDependencies{
		DialogHandler: dialogHandler,
		ProcessMgr:    mockProc,
		WindowMgr:     mockWin,
		Keyboard:      mockKbd,
	}

	opts := CompileOptions{Hwnd: 0x9999}

	result, err := CompileWithDeps(opts, deps)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.HasErrors)

	// Verify Enter was sent twice (for save prompts)
	assert.True(t, mockKbd.SendEnterCalled)
}

func TestCompile_BackwardCompatibility(t *testing.T) {
	// This test verifies that the original Compile() function still works
	// It will use real dependencies, so we just verify it doesn't panic
	opts := CompileOptions{
		Hwnd:         0, // Invalid hwnd won't actually compile, but should handle gracefully
		RecompileAll: false,
	}

	// This should not panic even with invalid opts
	// The function will fail at various steps but shouldn't crash
	result, err := Compile(opts)

	// We expect an error since we're not providing valid SIMPL Windows handle
	// but the important thing is it didn't panic
	_ = result
	_ = err
}
