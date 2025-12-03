//go:build windows

package windows

import (
	"log/slog"
	"time"

	"github.com/Norgate-AV/smpc/internal/logger"
)

// monitorManager handles window monitoring functionality
type monitorManager struct {
	log logger.LoggerInterface
}

// newMonitorManager creates a new monitor manager
func newMonitorManager(log logger.LoggerInterface) *monitorManager {
	return &monitorManager{log: log}
}

// StartWindowMonitor launches a background goroutine that monitors windows
func (m *monitorManager) StartWindowMonitor(pid uint32, interval time.Duration) {
	seen := make(map[uintptr]bool)

	go func() {
		m.log.Debug("Window monitor started")

		for {
			windows := EnumerateWindows()

			for _, w := range windows {
				if pid != 0 && w.Pid != pid {
					continue
				}
				if !seen[w.Hwnd] {
					seen[w.Hwnd] = true
					// Log top-level window info
					m.log.Debug("Window detected",
						slog.Uint64("hwnd", uint64(w.Hwnd)),
						slog.Uint64("pid", uint64(w.Pid)),
						slog.String("class", GetClassName(w.Hwnd)),
						slog.String("title", w.Title),
					)

					// Enumerate child controls and log their text
					childTexts := CollectChildTexts(w.Hwnd)
					if len(childTexts) > 0 {
						for _, ct := range childTexts {
							if ct != "" {
								m.log.Debug("Child control", slog.String("text", ct))
							}
						}
					}

					// Broadcast event (non-blocking) and store in recent cache
					if MonitorCh != nil {
						ev := WindowEvent{Hwnd: w.Hwnd, Title: w.Title, Pid: w.Pid, Class: GetClassName(w.Hwnd)}

						recentMu.Lock()
						recentEvents = append(recentEvents, ev)

						if len(recentEvents) > 256 {
							recentEvents = recentEvents[len(recentEvents)-256:]
						}

						recentMu.Unlock()

						select {
						case MonitorCh <- ev:
						default:
							// drop if buffer full
						}
					}
				}
			}

			time.Sleep(interval)
		}
	}()
}
