package windows

import (
	"log"
	"syscall"
)

var (
	kernel32DLL           = syscall.NewLazyDLL("kernel32.dll")
	setConsoleCtrlHandler = kernel32DLL.NewProc("SetConsoleCtrlHandler")
)

// ConsoleCtrlHandler is a callback function for console control events
type ConsoleCtrlHandler func(ctrlType uint32) uintptr

var globalHandler ConsoleCtrlHandler

// SetConsoleCtrlHandler sets up a Windows console control handler
// This catches Ctrl+C, window close, logoff, and shutdown events
func SetConsoleCtrlHandler(handler ConsoleCtrlHandler) error {
	globalHandler = handler

	// Register the handler
	ret, _, err := setConsoleCtrlHandler.Call(
		syscall.NewCallback(consoleCtrlHandlerCallback),
		1, // TRUE - add handler
	)

	if ret == 0 {
		log.Printf("SetConsoleCtrlHandler failed: %v", err)
		return err
	}

	log.Println("Windows console control handler registered")
	return nil
}

// consoleCtrlHandlerCallback is the actual callback that Windows calls
func consoleCtrlHandlerCallback(ctrlType uint32) uintptr {
	if globalHandler != nil {
		return globalHandler(ctrlType)
	}
	return 0 // FALSE - let default handler process it
}

// Console control event types
const (
	CTRL_C_EVENT        = 0
	CTRL_BREAK_EVENT    = 1
	CTRL_CLOSE_EVENT    = 2
	CTRL_LOGOFF_EVENT   = 5
	CTRL_SHUTDOWN_EVENT = 6
)

// GetCtrlTypeName returns a human-readable name for a control event type
func GetCtrlTypeName(ctrlType uint32) string {
	switch ctrlType {
	case CTRL_C_EVENT:
		return "CTRL_C"
	case CTRL_BREAK_EVENT:
		return "CTRL_BREAK"
	case CTRL_CLOSE_EVENT:
		return "CTRL_CLOSE"
	case CTRL_LOGOFF_EVENT:
		return "CTRL_LOGOFF"
	case CTRL_SHUTDOWN_EVENT:
		return "CTRL_SHUTDOWN"
	default:
		return "UNKNOWN"
	}
}
