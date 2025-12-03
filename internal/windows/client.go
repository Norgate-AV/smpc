//go:build windows

package windows

import (
	"github.com/Norgate-AV/smpc/internal/logger"
)

// Client provides methods for interacting with Windows APIs
// It composes specialized managers for different categories of functionality
type Client struct {
	log      logger.LoggerInterface
	Window   *windowManager
	Keyboard *keyboardInjector
	Monitor  *monitorManager
}

// NewClient creates a new Windows API client
func NewClient(log logger.LoggerInterface) *Client {
	return &Client{
		log:      log,
		Window:   newWindowManager(log),
		Keyboard: newKeyboardInjector(log),
		Monitor:  newMonitorManager(log),
	}
}
