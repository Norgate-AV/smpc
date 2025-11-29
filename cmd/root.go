package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/Norgate-AV/smpc/internal/compiler"
	"github.com/Norgate-AV/smpc/internal/logging"
	"github.com/Norgate-AV/smpc/internal/simpl"
	"github.com/Norgate-AV/smpc/internal/version"
	"github.com/Norgate-AV/smpc/internal/windows"
)

var (
	verbose      bool
	recompileAll bool
	showLogs     bool

	// Track SIMPL Windows for cleanup on interrupt
	simplHwnd uintptr
	simplPid  uint32

	// osExit allows mocking os.Exit for testing
	osExit = os.Exit
)

var RootCmd = &cobra.Command{
	Use:     "smpc <file-path>",
	Short:   "smpc - Automate compilation of .smw files",
	Version: version.GetVersion(),
	Args:    validateArgs,
	RunE:    Execute,
}

func init() {
	// Set custom version template to show full version info
	RootCmd.SetVersionTemplate(`{{printf "%s\n" .Version}}`)

	// Add flags
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "V", false, "enable verbose output")
	RootCmd.PersistentFlags().BoolVarP(&recompileAll, "recompile-all", "r", false, "trigger Recompile All (Alt+F12) instead of Compile (F12)")
	RootCmd.PersistentFlags().BoolVarP(&showLogs, "logs", "l", false, "print the current log file to stdout and exit")
}

// validateArgs validates arguments or handles --logs flag
func validateArgs(cmd *cobra.Command, args []string) error {
	// If --logs flag is set, print log file and exit
	if showLogs {
		logPath := logging.GetLogPath()
		if logPath == "" {
			fmt.Fprintln(os.Stderr, "ERROR: Log file path not initialized")
			osExit(1)
		}

		file, err := os.Open(logPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Log file does not exist: %s\n", logPath)
				osExit(1)
			}
			fmt.Fprintf(os.Stderr, "ERROR: Failed to open log file: %v\n", err)
			osExit(1)
		}

		defer file.Close()

		if _, err := io.Copy(os.Stdout, file); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to read log file: %v\n", err)
			osExit(1)
		}

		osExit(0)
		return nil
	}

	// Otherwise, validate .smw file argument
	if err := cobra.ExactArgs(1)(cmd, args); err != nil {
		return err
	}

	if filepath.Ext(args[0]) != ".smw" {
		return fmt.Errorf("file must have .smw extension")
	}

	return nil
}

func Execute(cmd *cobra.Command, args []string) error {
	slog.Debug("Execute() called", "args", args)
	slog.Debug("Flags set", "verbose", verbose, "recompileAll", recompileAll)

	// Check if running as admin
	slog.Debug("Checking elevation status")
	if !windows.IsElevated() {
		slog.Info("This program requires administrator privileges")
		slog.Info("Relaunching as administrator")
		slog.Debug("Not elevated, relaunching as admin")

		if err := windows.RelaunchAsAdmin(); err != nil {
			slog.Error("RelaunchAsAdmin failed", "error", err)
			return fmt.Errorf("error relaunching as admin: %w", err)
		}

		// Exit this instance, the elevated one will continue
		slog.Debug("Relaunched successfully, exiting non-elevated instance")
		return nil
	}

	slog.Info("Running with administrator privileges ✓")
	slog.Debug("Running with administrator privileges")

	// Get the file path from the command arguments
	filePath := args[0]
	slog.Debug("Processing file", "path", filePath)

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("error resolving file path: %w", err)
	}

	// Start background window monitor to observe dialogs and window changes in real time
	slog.Debug("Creating context and starting monitor goroutine")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure the goroutine is cleaned up
	go simpl.StartMonitoring(ctx)
	slog.Debug("Monitor goroutine started")

	// Open the file with SIMPL Windows application using elevated privileges
	// SW_SHOWNORMAL = 1
	slog.Debug("Launching SIMPL Windows with file", "path", absPath)
	if err := windows.ShellExecute(0, "runas", simpl.SIMPL_WINDOWS_PATH, absPath, "", 1); err != nil {
		slog.Error("ShellExecute failed", "error", err)
		return fmt.Errorf("error opening file: %w", err)
	}

	slog.Debug("SIMPL Windows launched successfully")

	// At this point, the SIMPL Windows process should have started
	// Meaning, if we fail from any point onward, we need to make sure we
	// clean up by closing SIMPL Windows

	// Try to get the PID early so signal handlers can use it for cleanup
	// Give it a moment for the process to actually start
	time.Sleep(500 * time.Millisecond)
	earlyPid := simpl.GetPid()
	if earlyPid != 0 {
		simplPid = earlyPid
		slog.Debug("Early PID detection", "pid", earlyPid)
	}

	// Set up Windows console control handler to catch window close events
	// This is more reliable than signal handling on Windows
	_ = windows.SetConsoleCtrlHandler(func(ctrlType uint32) uintptr {
		slog.Debug("Received console control event", "type", windows.GetCtrlTypeName(ctrlType), "code", ctrlType)
		slog.Info("Received console control event, cleaning up", "type", windows.GetCtrlTypeName(ctrlType))

		// Try to cleanup using hwnd if we have it
		if simplHwnd != 0 {
			slog.Debug("Cleaning up SIMPL Windows", "hwnd", simplHwnd)
			simpl.Cleanup(simplHwnd)
		} else if simplPid != 0 {
			// If we don't have hwnd yet but have PID, force terminate
			slog.Debug("Force terminating SIMPL Windows", "pid", simplPid)
			_ = windows.TerminateProcess(simplPid)
		} else {
			// Last resort - try to find and kill any smpwin.exe process we may have started
			slog.Debug("Attempting to find and terminate SIMPL Windows process")
			pid := simpl.GetPid()

			if pid != 0 {
				slog.Debug("Found SIMPL Windows PID, terminating", "pid", pid)
				_ = windows.TerminateProcess(pid)
			} else {
				slog.Debug("Could not find SIMPL Windows process to terminate")
			}
		}

		slog.Debug("Cleanup completed, exiting")

		// Must call os.Exit to actually terminate the process
		// Otherwise the handler returns and execution continues
		os.Exit(130)

		// Return TRUE to indicate we handled the event
		// (this won't actually be reached due to os.Exit, but required for the function signature)
		return 1
	})

	// Set up signal handler immediately to catch Ctrl+C during window wait
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		slog.Debug("Received signal", "signal", sig)
		slog.Info("Received interrupt signal, cleaning up")
		slog.Debug("Running cleanup due to interrupt")

		// Try to cleanup using hwnd if we have it
		if simplHwnd != 0 {
			slog.Debug("Cleaning up SIMPL Windows", "hwnd", simplHwnd)
			simpl.Cleanup(simplHwnd)
		} else if simplPid != 0 {
			// If we don't have hwnd yet but have PID, force terminate
			slog.Debug("Force terminating SIMPL Windows", "pid", simplPid)
			_ = windows.TerminateProcess(simplPid)
		} else {
			// Last resort - try to find and kill any smpwin.exe process we may have started
			slog.Debug("Attempting to find and terminate SIMPL Windows process")
			pid := simpl.GetPid()

			if pid != 0 {
				slog.Debug("Found SIMPL Windows PID, terminating", "pid", pid)
				_ = windows.TerminateProcess(pid)
			}
		}

		slog.Debug("Cleanup completed, exiting")
		os.Exit(130) // Standard exit code for Ctrl+C
	}()

	slog.Debug("Signal handler registered (early)")

	// Wait for the main window to appear (with a 1 minute timeout)
	slog.Info("Waiting for SIMPL Windows to fully launch...")
	slog.Debug("Waiting for SIMPL Windows window to appear")

	hwnd, found := simpl.WaitForAppear(60 * time.Second)
	if !found {
		slog.Error("Timeout waiting for window to appear")
		return fmt.Errorf("timed out waiting for SIMPL Windows window to appear")
	}

	slog.Debug("Window appeared", "hwnd", hwnd)

	// Store hwnd for signal handler cleanup
	simplHwnd = hwnd
	slog.Debug("Stored hwnd for signal handler", "hwnd", simplHwnd)

	// Set up deferred cleanup to ensure SIMPL Windows is closed on exit
	defer simpl.Cleanup(hwnd)
	slog.Debug("Cleanup deferred")

	// Wait for the window to be fully ready and responsive (with a 30 second timeout)
	slog.Debug("Waiting for window to be ready")
	if !simpl.WaitForReady(hwnd, 30*time.Second) {
		slog.Error("Window not responding properly")
		return fmt.Errorf("window appeared but is not responding properly")
	}

	slog.Debug("Window is ready")

	// Small extra delay to allow UI to finish settling
	slog.Info("Waiting a few extra seconds for UI to settle...")
	time.Sleep(5 * time.Second)

	slog.Info("Successfully opened file", "path", absPath)

	// Run the compilation
	result, err := compiler.Compile(compiler.CompileOptions{
		FilePath:     absPath,
		RecompileAll: recompileAll,
		Hwnd:         hwnd,
		Ctx:          ctx,
		SimplPidPtr:  &simplPid,
	})
	if err != nil {
		slog.Error("Compilation failed", "error", err)
		slog.Info("Press Enter to exit...")
		_, _ = fmt.Scanln()
		return err
	}

	// Show compilation summary
	slog.Info("=== Compile Summary ===")
	if result.Errors > 0 {
		slog.Info("Errors", "count", result.Errors)
	}
	slog.Info("Warnings", "count", result.Warnings)
	slog.Info("Notices", "count", result.Notices)
	slog.Info("Compile Time", "seconds", result.CompileTime)
	slog.Info("=======================")

	// Exit with error if compilation failed
	if result.HasErrors {
		slog.Error("Compilation failed with errors")
		slog.Info("Press Enter to exit...")
		_, _ = fmt.Scanln()
		return fmt.Errorf("compilation failed with %d error(s)", result.Errors)
	}

	slog.Debug("Compilation completed successfully")
	slog.Info("Press Enter to exit...")
	_, _ = fmt.Scanln()

	slog.Debug("Execute() completed successfully")
	return nil
}
