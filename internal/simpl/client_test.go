package simpl

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Norgate-AV/smpc/internal/logger"
	"github.com/Norgate-AV/smpc/internal/testutil"
	"github.com/Norgate-AV/smpc/internal/windows"
)

type buttonClick struct {
	hwnd uintptr
	text string
}

func newTestClient(clicks *[]buttonClick) *Client {
	log := logger.NewNoOpLogger()
	c := &Client{log: log}
	c.clickButton = func(hwnd uintptr, text string) bool {
		*clicks = append(*clicks, buttonClick{hwnd, text})
		return true
	}
	return c
}

func TestClient_DismissStartupBlockers_ContinueReplace(t *testing.T) {
	testutil.SetupMonitorChannel()
	defer testutil.CleanupMonitorChannel()

	var clicks []buttonClick
	c := newTestClient(&clicks)

	testutil.SendEventsToMonitor(
		windows.WindowEvent{Hwnd: 0x1111, Title: "Continue Replace?"},
	)

	c.dismissStartupBlockers()

	assert.Len(t, clicks, 1)
	assert.Equal(t, uintptr(0x1111), clicks[0].hwnd)
	assert.Equal(t, "&Yes", clicks[0].text)
}

func TestClient_DismissStartupBlockers_ReplaceControlSystem(t *testing.T) {
	testutil.SetupMonitorChannel()
	defer testutil.CleanupMonitorChannel()

	var clicks []buttonClick
	c := newTestClient(&clicks)

	testutil.SendEventsToMonitor(
		windows.WindowEvent{Hwnd: 0x2222, Title: "Replace Control System: keep revised program?"},
	)

	c.dismissStartupBlockers()

	assert.Len(t, clicks, 1)
	assert.Equal(t, uintptr(0x2222), clicks[0].hwnd)
	assert.Equal(t, "&Yes", clicks[0].text)
}

func TestClient_DismissStartupBlockers_DeferesNonStartupEvents(t *testing.T) {
	testutil.SetupMonitorChannel()
	defer testutil.CleanupMonitorChannel()

	var clicks []buttonClick
	c := newTestClient(&clicks)

	testutil.SendEventsToMonitor(
		windows.WindowEvent{Hwnd: 0x3333, Title: "Operation Complete"},
		windows.WindowEvent{Hwnd: 0x1111, Title: "Continue Replace?"},
	)

	c.dismissStartupBlockers()

	// "Continue Replace?" must have been clicked
	assert.Len(t, clicks, 1)
	assert.Equal(t, "&Yes", clicks[0].text)

	// "Operation Complete" must have been re-queued for later handlers
	select {
	case ev := <-windows.MonitorCh:
		assert.Equal(t, "Operation Complete", ev.Title)
		assert.Equal(t, uintptr(0x3333), ev.Hwnd)
	default:
		t.Fatal("expected Operation Complete to be re-queued but channel was empty")
	}
}

func TestClient_DismissStartupBlockers_NilChannel(t *testing.T) {
	// Ensure a nil MonitorCh does not panic
	windows.MonitorCh = nil

	var clicks []buttonClick
	c := newTestClient(&clicks)

	assert.NotPanics(t, func() { c.dismissStartupBlockers() })
	assert.Empty(t, clicks)
}
