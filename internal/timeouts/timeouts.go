// Package timeouts defines timing constants used throughout the application.
// These values have been empirically determined for reliable interaction with
// SIMPL Windows and the Windows API.
package timeouts

import "time"

const (
	// SIMPL Windows Lifecycle Timeouts

	// WindowAppearTimeout is the maximum time to wait for SIMPL Windows to appear
	// after launching the process. SIMPL Windows typically loads within 2 minutes,
	// but we allow 3 minutes to account for slower systems.
	WindowAppearTimeout = 3 * time.Minute

	// WindowReadyTimeout is the maximum time to wait for the SIMPL Windows UI
	// to stabilize and become responsive after the window appears.
	WindowReadyTimeout = 30 * time.Second

	// UISettlingDelay allows time for window animations, focus events, and
	// UI state to stabilize before interacting with the application.
	UISettlingDelay = 5 * time.Second

	// FocusVerificationDelay allows time to verify that window focus has
	// successfully changed after a focus operation.
	FocusVerificationDelay = 1 * time.Second

	// Windows API Interaction Delays

	// WindowMessageDelay is the delay after sending window messages (WM_CLOSE,
	// WM_SETFOCUS, etc.) to allow the target application to process the message.
	WindowMessageDelay = 500 * time.Millisecond

	// KeystrokeDelay is the delay between keyboard events (key down/up) to ensure
	// the target application reliably receives and processes the input.
	KeystrokeDelay = 50 * time.Millisecond

	// Compiler Dialog Timeouts

	// CompilationCompleteTimeout is the maximum time to wait for the entire
	// compilation process to complete, from initiating compile to receiving
	// the "Compile Complete" dialog. This accounts for large programs that
	// may take several minutes to compile.
	CompilationCompleteTimeout = 5 * time.Minute

	// DialogResponseDelay is the delay after sending input to dialog boxes to
	// allow the dialog to process the input and respond.
	DialogResponseDelay = 300 * time.Millisecond

	// DialogOperationCompleteTimeout is the maximum time to wait for the
	// "Operation Complete" dialog to appear after initiating a compilation.
	DialogOperationCompleteTimeout = 3 * time.Second

	// DialogIncompleteSymbolsTimeout is the maximum time to wait for the
	// "Incomplete Symbols" dialog to appear during compilation checks.
	DialogIncompleteSymbolsTimeout = 2 * time.Second

	// DialogConvertCompileTimeout is the maximum time to wait for the
	// "Convert/Compile" dialog to appear.
	DialogConvertCompileTimeout = 5 * time.Second

	// DialogCommentedSymbolsTimeout is the maximum time to wait for the
	// "Commented out Symbols and/or Devices" dialog to appear.
	DialogCommentedSymbolsTimeout = 5 * time.Second

	// DialogCompilingTimeout is the maximum time to wait for the "Compiling..."
	// progress dialog to appear and complete.
	DialogCompilingTimeout = 30 * time.Second

	// DialogProgramCompilationTimeout is the maximum time to wait for the
	// "Program Compilation" results dialog to appear.
	DialogProgramCompilationTimeout = 10 * time.Second

	// DialogConfirmationTimeout is the maximum time to wait for a
	// confirmation dialog to appear.
	DialogConfirmationTimeout = 2 * time.Second

	// Polling and Verification Intervals

	// StatePollingInterval is the delay between checks in tight polling loops
	// when actively waiting for state changes (window appearance, readiness,
	// process discovery, etc.).
	StatePollingInterval = 100 * time.Millisecond

	// StabilityCheckInterval is the delay between consecutive responsiveness
	// checks to ensure a window is stable and ready for interaction.
	StabilityCheckInterval = 500 * time.Millisecond

	// MonitorPollingInterval is the interval at which the background window
	// monitor checks for new windows and dialog events.
	MonitorPollingInterval = 500 * time.Millisecond

	// CleanupDelay allows time for windows and processes to close gracefully
	// before performing verification checks or additional cleanup operations.
	CleanupDelay = 1 * time.Second
)
