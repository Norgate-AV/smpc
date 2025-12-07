package compiler

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Norgate-AV/smpc/internal/logger"
	"github.com/Norgate-AV/smpc/internal/testutil"
	"github.com/Norgate-AV/smpc/internal/windows"
)

func TestCompiler_SuccessfulCompilation(t *testing.T) {
	// Setup monitor channel for event-driven testing
	testutil.SetupMonitorChannel()
	defer testutil.CleanupMonitorChannel()

	mockWin := testutil.NewMockWindowManager().
		WithChildInfosForHwnd(0x2222, // Compile Complete dialog
			windows.ChildInfo{ClassName: "Static", Text: "Statistics"},
			windows.ChildInfo{ClassName: "Edit", Text: "Program Errors: 0\r\nProgram Warnings: 0\r\nProgram Notices: 0\r\nCompile Time: 1.23 seconds\r\n"},
		)

	mockKbd := testutil.NewMockKeyboardInjector()
	mockCtrl := testutil.NewMockControlReader()
	mockProc := testutil.NewMockProcessManager().WithPid(1234)

	log := logger.NewNoOpLogger()
	deps := &CompileDependencies{
		ProcessMgr:    mockProc,
		WindowMgr:     mockWin,
		Keyboard:      mockKbd,
		ControlReader: mockCtrl,
	}

	compiler := NewCompilerWithDeps(log, deps)
	opts := CompileOptions{
		Hwnd:         0x9999,
		RecompileAll: false,
	}

	// Send dialog events that will appear during compilation
	testutil.SendEventsToMonitor(
		windows.WindowEvent{Hwnd: 0x1111, Title: "Compiling..."},
		windows.WindowEvent{Hwnd: 0x2222, Title: "Compile Complete"},
	)

	result, err := compiler.Compile(opts)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.HasErrors)
	assert.Equal(t, 0, result.Errors)
	assert.Equal(t, 0, result.Warnings)
	assert.Equal(t, 0, result.Notices)
	assert.InDelta(t, 1.23, result.CompileTime, 0.01)

	// Verify F12 was sent (new SendInput method should be called)
	assert.True(t, mockKbd.SendF12WithSendInputCalled)
	assert.False(t, mockKbd.SendAltF12WithSendInputCalled)
	assert.False(t, mockKbd.SendF12Called) // Old method should not be called when SendInput succeeds

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

func TestCompiler_RecompileAll(t *testing.T) {
	testutil.SetupMonitorChannel()
	defer testutil.CleanupMonitorChannel()

	mockWin := testutil.NewMockWindowManager().
		WithChildInfosForHwnd(0x2222,
			windows.ChildInfo{ClassName: "Edit", Text: "Errors: 0\r\nWarnings: 0\r\nNotices: 0\r\n"},
		)

	mockKbd := testutil.NewMockKeyboardInjector()
	mockCtrl := testutil.NewMockControlReader()
	mockProc := testutil.NewMockProcessManager().WithPid(1234)

	log := logger.NewNoOpLogger()
	deps := &CompileDependencies{
		ProcessMgr:    mockProc,
		WindowMgr:     mockWin,
		Keyboard:      mockKbd,
		ControlReader: mockCtrl,
	}

	compiler := NewCompilerWithDeps(log, deps)

	opts := CompileOptions{
		Hwnd:         0x9999,
		RecompileAll: true, // Trigger Alt+F12 instead of F12
	}

	testutil.SendEventsToMonitor(
		windows.WindowEvent{Hwnd: 0x1111, Title: "Compiling..."},
		windows.WindowEvent{Hwnd: 0x2222, Title: "Compile Complete"},
	)

	result, err := compiler.Compile(opts)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.HasErrors)

	// Verify Alt+F12 was sent (new SendInput method should be called)
	assert.False(t, mockKbd.SendF12WithSendInputCalled)
	assert.True(t, mockKbd.SendAltF12WithSendInputCalled)
	assert.False(t, mockKbd.SendAltF12Called) // Old method should not be called when SendInput succeeds
}

func TestCompiler_WithWarnings(t *testing.T) {
	testutil.SetupMonitorChannel()
	defer testutil.CleanupMonitorChannel()

	mockWin := testutil.NewMockWindowManager().
		WithChildInfosForHwnd(0x2222, // Compile Complete dialog
			windows.ChildInfo{ClassName: "Edit", Text: "Program Errors: 0\r\nProgram Warnings: 2\r\nProgram Notices: 1\r\n"},
		).
		WithChildInfosForHwnd(0x3333, // Program Compilation dialog
			windows.ChildInfo{ClassName: "ListBox", Items: []string{
				"WARNING    (LGCMCVT102) ** Signal foo has no driving source",
				"WARNING    (LGCMCVT102) ** Signal bar has no driving source",
				"NOTICE     (LGCMCVT103) ** Signal baz has no destination",
			}},
		)

	mockKbd := testutil.NewMockKeyboardInjector()
	mockCtrl := testutil.NewMockControlReader()
	mockProc := testutil.NewMockProcessManager().WithPid(1234)

	log := logger.NewNoOpLogger()
	deps := &CompileDependencies{
		ProcessMgr:    mockProc,
		WindowMgr:     mockWin,
		Keyboard:      mockKbd,
		ControlReader: mockCtrl,
	}

	compiler := NewCompilerWithDeps(log, deps)

	opts := CompileOptions{Hwnd: 0x9999}

	testutil.SendEventsToMonitor(
		windows.WindowEvent{Hwnd: 0x1111, Title: "Compiling..."},
		windows.WindowEvent{Hwnd: 0x2222, Title: "Compile Complete"},
		windows.WindowEvent{Hwnd: 0x3333, Title: "Program Compilation"},
	)

	result, err := compiler.Compile(opts)

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

func TestCompiler_WithErrors(t *testing.T) {
	testutil.SetupMonitorChannel()
	defer testutil.CleanupMonitorChannel()

	mockWin := testutil.NewMockWindowManager().
		WithChildInfosForHwnd(0x2222, // Compile Complete dialog
			windows.ChildInfo{ClassName: "Edit", Text: "Program Errors: 3\r\nProgram Warnings: 0\r\nProgram Notices: 0\r\n"},
		).
		WithChildInfosForHwnd(0x3333, // Program Compilation dialog
			windows.ChildInfo{ClassName: "ListBox", Items: []string{
				"ERROR      (LGSPLS1700) Line 5: Undefined symbol 'foo'",
				"ERROR      (LGCMCVT247) Line 15: Type mismatch",
				"ERROR      (LGCMCVT101) Line 25: Missing semicolon",
			}},
		)

	mockKbd := testutil.NewMockKeyboardInjector()
	mockCtrl := testutil.NewMockControlReader()
	mockProc := testutil.NewMockProcessManager().WithPid(1234)

	log := logger.NewNoOpLogger()
	deps := &CompileDependencies{
		ProcessMgr:    mockProc,
		WindowMgr:     mockWin,
		Keyboard:      mockKbd,
		ControlReader: mockCtrl,
	}

	compiler := NewCompilerWithDeps(log, deps)

	opts := CompileOptions{Hwnd: 0x9999}

	testutil.SendEventsToMonitor(
		windows.WindowEvent{Hwnd: 0x1111, Title: "Compiling..."},
		windows.WindowEvent{Hwnd: 0x2222, Title: "Compile Complete"},
		windows.WindowEvent{Hwnd: 0x3333, Title: "Program Compilation"},
	)

	result, err := compiler.Compile(opts)

	// Compile returns an error when there are compile errors
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compilation failed")
	assert.NotNil(t, result)
	assert.True(t, result.HasErrors)
	assert.Equal(t, 3, result.Errors)
	assert.Equal(t, 0, result.Warnings)
	assert.Equal(t, 0, result.Notices)
	assert.Len(t, result.ErrorMessages, 3)
}

func TestCompiler_IncompleteSymbols(t *testing.T) {
	testutil.SetupMonitorChannel()
	defer testutil.CleanupMonitorChannel()

	mockWin := testutil.NewMockWindowManager().
		WithChildInfos(
			windows.ChildInfo{ClassName: "Edit", Text: "The program contains incomplete symbols and cannot be compiled."},
		)

	mockKbd := testutil.NewMockKeyboardInjector()
	mockCtrl := testutil.NewMockControlReader()
	mockProc := testutil.NewMockProcessManager().WithPid(1234)

	log := logger.NewNoOpLogger()
	deps := &CompileDependencies{
		ProcessMgr:    mockProc,
		WindowMgr:     mockWin,
		Keyboard:      mockKbd,
		ControlReader: mockCtrl,
	}

	compiler := NewCompilerWithDeps(log, deps)

	opts := CompileOptions{Hwnd: 0x9999}

	testutil.SendEventsToMonitor(
		windows.WindowEvent{Hwnd: 0xABCD, Title: "Incomplete Symbols"},
	)

	result, err := compiler.Compile(opts)

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, err.Error(), "incomplete symbols")
	assert.True(t, result.HasErrors)
	assert.Equal(t, 1, result.Errors)
	assert.Len(t, result.ErrorMessages, 1)
}

func TestCompiler_CompileDialogTimeout(t *testing.T) {
	testutil.SetupMonitorChannel()
	defer testutil.CleanupMonitorChannel()

	mockWin := testutil.NewMockWindowManager()

	mockKbd := testutil.NewMockKeyboardInjector()
	mockCtrl := testutil.NewMockControlReader()
	mockProc := testutil.NewMockProcessManager().WithPid(1234)

	log := logger.NewNoOpLogger()
	deps := &CompileDependencies{
		ProcessMgr:    mockProc,
		WindowMgr:     mockWin,
		Keyboard:      mockKbd,
		ControlReader: mockCtrl,
	}

	compiler := NewCompilerWithDeps(log, deps)

	opts := CompileOptions{Hwnd: 0x9999}

	// Don't send any events to trigger timeout

	result, err := compiler.Compile(opts)

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, err.Error(), "Compile Complete")
	assert.True(t, result.HasErrors)
	assert.Equal(t, 1, result.Errors)
	assert.Len(t, result.ErrorMessages, 1)
}

func TestCompiler_NoPid(t *testing.T) {
	testutil.SetupMonitorChannel()
	defer testutil.CleanupMonitorChannel()

	// When PID is 0, dialog monitoring should be skipped but compilation should still proceed
	mockWin := testutil.NewMockWindowManager().
		WithChildInfos(
			windows.ChildInfo{ClassName: "Edit", Text: "Errors: 0\r\nWarnings: 0\r\nNotices: 0\r\n"},
		)

	mockKbd := testutil.NewMockKeyboardInjector()
	mockCtrl := testutil.NewMockControlReader()
	mockProc := testutil.NewMockProcessManager().WithPid(0) // PID not available

	log := logger.NewNoOpLogger()
	deps := &CompileDependencies{
		ProcessMgr:    mockProc,
		WindowMgr:     mockWin,
		Keyboard:      mockKbd,
		ControlReader: mockCtrl,
	}

	compiler := NewCompilerWithDeps(log, deps)

	opts := CompileOptions{Hwnd: 0x9999}

	// PID=0 means no monitoring, so don't send events
	testutil.SendEventsToMonitor(
		windows.WindowEvent{Hwnd: 0x1111, Title: "Compiling..."},
		windows.WindowEvent{Hwnd: 0x2222, Title: "Compile Complete"},
	)

	result, err := compiler.Compile(opts)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.HasErrors)

	// Verify F12 was still sent even without PID (new SendInput method should be called)
	assert.True(t, mockKbd.SendF12WithSendInputCalled)
}

func TestCompiler_WithSavePrompts(t *testing.T) {
	testutil.SetupMonitorChannel()
	defer testutil.CleanupMonitorChannel()

	mockWin := testutil.NewMockWindowManager().
		WithChildInfos(
			windows.ChildInfo{ClassName: "Edit", Text: "Errors: 0\r\nWarnings: 0\r\nNotices: 0\r\n"},
		)

	mockKbd := testutil.NewMockKeyboardInjector()
	mockCtrl := testutil.NewMockControlReader()
	mockProc := testutil.NewMockProcessManager().WithPid(1234)

	log := logger.NewNoOpLogger()
	deps := &CompileDependencies{
		ProcessMgr:    mockProc,
		WindowMgr:     mockWin,
		Keyboard:      mockKbd,
		ControlReader: mockCtrl,
	}

	compiler := NewCompilerWithDeps(log, deps)

	opts := CompileOptions{Hwnd: 0x9999}

	testutil.SendEventsToMonitor(
		windows.WindowEvent{Hwnd: 0x5555, Title: "Convert/Compile"},
		windows.WindowEvent{Hwnd: 0x6666, Title: "Commented Out Symbols"},
		windows.WindowEvent{Hwnd: 0x1111, Title: "Compiling..."},
		windows.WindowEvent{Hwnd: 0x2222, Title: "Compile Complete"},
	)

	result, err := compiler.Compile(opts)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.HasErrors)

	// Verify Enter was sent twice (for save prompts)
	assert.True(t, mockKbd.SendEnterCalled)
}
