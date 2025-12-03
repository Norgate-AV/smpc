package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/Norgate-AV/smpc/internal/compiler"
	"github.com/Norgate-AV/smpc/internal/logger"
	"github.com/Norgate-AV/smpc/internal/simpl"
	"github.com/Norgate-AV/smpc/internal/timeouts"
	"github.com/Norgate-AV/smpc/internal/version"
	"github.com/Norgate-AV/smpc/internal/windows"
)

// ExecutionContext holds state needed throughout the compilation process
// and for cleanup in signal handlers.
type ExecutionContext struct {
	simplHwnd   uintptr
	simplPid    uint32
	log         logger.LoggerInterface
	simplClient *simpl.Client
	exitFunc    func(int) // Injectable for testing; defaults to os.Exit
}

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
	RootCmd.PersistentFlags().BoolP("verbose", "V", false, "enable verbose output")
	RootCmd.PersistentFlags().BoolP("recompile-all", "r", false, "trigger Recompile All (Alt+F12) instead of Compile (F12)")
	RootCmd.PersistentFlags().BoolP("logs", "l", false, "print the current log file to stdout and exit")
}

// validateArgs validates that a .smw file argument is provided (if any args given)
func validateArgs(cmd *cobra.Command, args []string) error {
	// Allow 0 args for --logs flag, which is handled in Execute
	if len(args) == 0 {
		return nil
	}

	// Validate .smw file argument
	if err := cobra.ExactArgs(1)(cmd, args); err != nil {
		return err
	}

	if filepath.Ext(args[0]) != ".smw" {
		return fmt.Errorf("file must have .smw extension")
	}

	return nil
}

// handleLogsFlag processes the --logs flag and exits if needed
func handleLogsFlag(cfg *Config, exitFunc func(int)) error {
	if !cfg.ShowLogs {
		return nil
	}

	if err := logger.PrintLogFile(nil, logger.LoggerOptions{}); err != nil {
		if os.IsNotExist(err) {
			logPath := logger.GetLogPath(logger.LoggerOptions{})
			fmt.Fprintf(os.Stderr, "Log file does not exist: %s\n", logPath)
			exitFunc(1)
		}

		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		exitFunc(1)
	}

	exitFunc(0)
	return nil // Won't actually reach here due to exitFunc
}

// initializeLogger creates a logger and logs startup information
func initializeLogger(cfg *Config, args []string) (logger.LoggerInterface, error) {
	log, err := logger.NewLogger(logger.LoggerOptions{
		Verbose:  cfg.Verbose,
		Compress: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return log, nil
}

// ensureElevated checks for admin privileges and relaunches if needed
func ensureElevated(log logger.LoggerInterface) error {
	log.Debug("Checking elevation status")
	if !windows.IsElevated() {
		log.Info("This program requires administrator privileges")
		log.Info("Relaunching as administrator")

		if err := windows.RelaunchAsAdmin(); err != nil {
			log.Error("RelaunchAsAdmin failed", slog.Any("error", err))
			return fmt.Errorf("error relaunching as admin: %w", err)
		}

		// Exit this instance, the elevated one will continue
		log.Debug("Relaunched successfully, exiting non-elevated instance")
		return nil
	}

	log.Debug("Running with administrator privileges")
	return nil
}

// validateAndResolvePath validates the file exists and returns its absolute path
func validateAndResolvePath(filePath string, log logger.LoggerInterface) (string, error) {
	log.Debug("Processing file", slog.String("path", filePath))

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", filePath)
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("error resolving file path: %w", err)
	}

	return absPath, nil
}

// launchSIMPLWindows starts monitoring, launches SIMPL, and returns cleanup function
func launchSIMPLWindows(simplClient *simpl.Client, absPath string, log logger.LoggerInterface) (hwnd uintptr, pid uint32, cleanup func(), err error) {
	// Start background window monitor
	stopMonitor := simplClient.StartMonitoring()
	log.Debug("Background window monitor started")

	// Open the file with SIMPL Windows application using elevated privileges
	// SW_SHOWNORMAL = 1
	log.Debug("Launching SIMPL Windows with file", slog.String("path", absPath))
	pid, err = windows.ShellExecuteEx(0, "open", simpl.GetSimplWindowsPath(), absPath, "", 1)
	if err != nil {
		stopMonitor()
		log.Error("ShellExecuteEx failed", slog.Any("error", err))
		return 0, 0, nil, fmt.Errorf("error opening file: %w", err)
	}

	log.Info("SIMPL Windows process started", slog.Uint64("pid", uint64(pid)))

	// Return cleanup function that stops monitor
	cleanup = func() {
		stopMonitor()
	}

	return 0, pid, cleanup, nil
}

// setupSignalHandlers configures console control and interrupt signal handlers
// It captures the ExecutionContext in closures to access state for cleanup
func setupSignalHandlers(ctx *ExecutionContext) {
	// Set up Windows console control handler to catch window close events
	_ = windows.SetConsoleCtrlHandler(func(ctrlType uint32) uintptr {
		ctx.log.Debug("Received console control event",
			slog.String("type", windows.GetCtrlTypeName(ctrlType)),
			slog.Uint64("code", uint64(ctrlType)),
		)

		ctx.log.Info("Cleaning up after console control event")
		ctx.simplClient.ForceCleanup(ctx.simplHwnd, ctx.simplPid)
		ctx.log.Debug("Cleanup completed, exiting")

		ctx.exitFunc(130)
		return 1
	})

	// Set up signal handler for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		ctx.log.Debug("Received signal", slog.Any("signal", sig))
		ctx.log.Info("Interrupt signal received, starting cleanup")

		ctx.simplClient.ForceCleanup(ctx.simplHwnd, ctx.simplPid)

		ctx.log.Debug("Cleanup completed, exiting")
		ctx.exitFunc(130)
	}()
}

// waitForWindowReady waits for SIMPL window to appear and become responsive
func waitForWindowReady(simplClient *simpl.Client, pid uint32, log logger.LoggerInterface) (uintptr, error) {
	log.Info("Waiting for SIMPL Windows to fully launch...")

	hwnd, found := simplClient.WaitForAppear(pid, timeouts.WindowAppearTimeout)
	if !found {
		log.Error("Timeout waiting for window to appear after 3 minutes")
		log.Info("Forcing SIMPL Windows to terminate due to timeout")
		simplClient.ForceCleanup(0, pid)
		return 0, fmt.Errorf("timed out waiting for SIMPL Windows window to appear after 3 minutes")
	}

	log.Debug("Window appeared", slog.Uint64("hwnd", uint64(hwnd)))

	// Wait for the window to be fully ready and responsive
	if !simplClient.WaitForReady(hwnd, timeouts.WindowReadyTimeout) {
		log.Error("Window not responding properly")
		return 0, fmt.Errorf("window appeared but is not responding properly")
	}

	// Small extra delay to allow UI to finish settling
	log.Debug("Waiting a few extra seconds for UI to settle...")
	time.Sleep(timeouts.UISettlingDelay)

	return hwnd, nil
}

// runCompilation creates a compiler and executes the compilation
func runCompilation(absPath string, hwnd uintptr, pidPtr *uint32, cfg *Config, log logger.LoggerInterface) (*compiler.CompileResult, error) {
	comp := compiler.NewCompiler(log)

	result, err := comp.Compile(compiler.CompileOptions{
		FilePath:     absPath,
		RecompileAll: cfg.RecompileAll,
		Hwnd:         hwnd,
		SimplPidPtr:  pidPtr,
	})
	if err != nil {
		log.Error("Compilation failed", slog.Any("error", err))
		return nil, err
	}

	return result, nil
}

// displayCompilationResults shows the compilation summary to the user
func displayCompilationResults(result *compiler.CompileResult, log logger.LoggerInterface) {
	log.Info("=== Compile Summary ===")
	if result.Errors > 0 {
		log.Info(fmt.Sprintf("Errors: %d", result.Errors))
	}

	log.Info(fmt.Sprintf("Warnings: %d", result.Warnings))
	log.Info(fmt.Sprintf("Notices: %d", result.Notices))
	log.Info(fmt.Sprintf("Compile Time: %.2f seconds", result.CompileTime))
	log.Info("=======================")

	// Also log structured data to file
	log.Info("Compilation complete",
		slog.Int("errors", result.Errors),
		slog.Int("warnings", result.Warnings),
		slog.Int("notices", result.Notices),
		slog.Float64("compileTime", result.CompileTime),
	)
}

func Execute(cmd *cobra.Command, args []string) error {
	cfg := NewConfigFromFlags(cmd)

	if err := handleLogsFlag(cfg, os.Exit); err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("file path required")
	}

	log, err := initializeLogger(cfg, args)
	if err != nil {
		return err
	}

	defer log.Close()

	log.Debug("Starting smpc", slog.Any("args", args))
	log.Debug("Flags set",
		slog.Bool("verbose", cfg.Verbose),
		slog.Bool("recompileAll", cfg.RecompileAll),
	)

	// Recover from panics and log them
	defer func() {
		if r := recover(); r != nil {
			log.Error("PANIC RECOVERED",
				slog.Any("panic", r),
				slog.String("stack", string(debug.Stack())),
			)

			fmt.Fprintf(os.Stderr, "\n*** PANIC: %v ***\n", r)
			fmt.Fprintf(os.Stderr, "Check log file for details\n")
		}
	}()

	// Validate SIMPL Windows installation before checking elevation
	if err := simpl.ValidateSimplWindowsInstallation(); err != nil {
		log.Error("SIMPL Windows installation check failed", slog.Any("error", err))
		return err
	}

	log.Debug("SIMPL Windows installation validated", slog.String("path", simpl.GetSimplWindowsPath()))

	// Validate file path before requesting elevation
	absPath, err := validateAndResolvePath(args[0], log)
	if err != nil {
		return err
	}

	if err := ensureElevated(log); err != nil {
		return err
	}

	simplClient := simpl.NewClient(log)
	_, pid, cleanup, err := launchSIMPLWindows(simplClient, absPath, log)
	if err != nil {
		return err
	}

	defer cleanup()

	// Create execution context to hold state for signal handlers
	ctx := &ExecutionContext{
		simplPid:    pid,
		log:         log,
		simplClient: simplClient,
		exitFunc:    os.Exit,
	}

	setupSignalHandlers(ctx)

	hwnd, err := waitForWindowReady(simplClient, pid, log)
	if err != nil {
		return err
	}

	// Store hwnd in context for signal handlers and cleanup
	ctx.simplHwnd = hwnd
	log.Debug("Stored hwnd in execution context", slog.Uint64("hwnd", uint64(hwnd)))

	defer simplClient.Cleanup(hwnd)

	result, err := runCompilation(absPath, hwnd, &ctx.simplPid, cfg, log)
	if err != nil {
		return err
	}

	displayCompilationResults(result, log)

	if result.HasErrors {
		log.Error("Compilation failed with errors")
		return fmt.Errorf("compilation failed with %d error(s)", result.Errors)
	}

	return nil
}
