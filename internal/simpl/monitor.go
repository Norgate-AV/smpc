package simpl

import (
	"context"
	"log/slog"
	"time"

	"github.com/Norgate-AV/smpc/internal/windows"
)

// StartMonitoring starts a background goroutine that monitors SIMPL Windows dialogs
func StartMonitoring(ctx context.Context) {
	// Try to obtain PID repeatedly until found, then monitor that PID
	var pid uint32

	for i := 0; i < 50 && pid == 0; i++ { // up to ~5s
		select {
		case <-ctx.Done():
			return
		default:
			pid = GetPid()
			if pid == 0 {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}

	// Init channel
	windows.MonitorCh = make(chan windows.WindowEvent, 64)
	if pid == 0 {
		slog.Debug("Window monitor falling back to all processes (SIMPL PID not found yet)")
		windows.StartWindowMonitor(0, 500*time.Millisecond)
	} else {
		slog.Debug("Window monitor targeting SIMPL PID", "pid", pid)
		windows.StartWindowMonitor(pid, 500*time.Millisecond)
	}

	// Wait for context cancellation
	<-ctx.Done()
}
