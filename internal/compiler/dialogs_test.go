package compiler

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Norgate-AV/smpc/internal/logger"
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
			log := logger.NewNoOpLogger()

			handler := NewDialogHandler(log, mockWin, mockKbd, mockCtrl)
			err := handler.HandleOperationComplete()

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
			log := logger.NewNoOpLogger()

			handler := NewDialogHandler(log, mockWin, mockKbd, mockCtrl)
			err := handler.HandleConfirmation()

			assert.NoError(t, err)
			if tt.expectClick {
				assert.Len(t, mockCtrl.FindAndClickButtonCalls, 1)
			} else {
				assert.Len(t, mockCtrl.FindAndClickButtonCalls, 0)
			}
		})
	}
}
