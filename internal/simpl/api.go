package simpl

import (
	"time"

	"github.com/Norgate-AV/smpc/internal/logger"
)

// SimplProcessAPI is a concrete implementation of the SIMPL process management interface
// It wraps the Client for backward compatibility with the interface
type SimplProcessAPI struct {
	client *Client
}

func NewSimplProcessAPI(log logger.LoggerInterface) *SimplProcessAPI {
	return &SimplProcessAPI{
		client: NewClient(log),
	}
}

func (s SimplProcessAPI) GetPid() uint32 {
	return s.client.GetPid()
}

func (s SimplProcessAPI) FindWindow(targetPid uint32, debug bool) (uintptr, string) {
	return s.client.FindWindow(targetPid, debug)
}

func (s SimplProcessAPI) WaitForReady(hwnd uintptr, timeout time.Duration) bool {
	return s.client.WaitForReady(hwnd, timeout)
}
