package main

import (
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"

	"github.com/Norgate-AV/smpc/cmd"
	"github.com/Norgate-AV/smpc/internal/logging"
)

func main() {
	// Initialize logging system with file rotation and dual output
	if err := logging.Setup(false); err != nil { // Set to true for verbose mode
		fmt.Fprintf(os.Stderr, "Failed to setup logging: %v\n", err)
		os.Exit(1)
	}

	defer logging.Close()

	// Recover from panics and log them
	defer func() {
		if r := recover(); r != nil {
			slog.Error("PANIC RECOVERED", "panic", r, "stack", string(debug.Stack()))
			fmt.Fprintf(os.Stderr, "\n*** PANIC: %v ***\n", r)
			fmt.Fprintf(os.Stderr, "Check log file for details\n")
			fmt.Println("\nPress Enter to close...")
			fmt.Scanln()
			os.Exit(2)
		}
	}()

	slog.Debug("Executing RootCmd...")
	if err := cmd.RootCmd.Execute(); err != nil {
		slog.Error("RootCmd.Execute failed", "error", err)
		fmt.Fprintln(os.Stderr, err)
		fmt.Println("\nPress Enter to close...")
		fmt.Scanln()
		os.Exit(1)
	}

	slog.Debug("RootCmd completed successfully")
}
