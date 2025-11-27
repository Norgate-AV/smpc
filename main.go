package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/Norgate-AV/smpc/cmd"
)

func main() {
	// Set up log file in %LOCALAPPDATA%\smpc\smpc.log
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
	}

	logDir := filepath.Join(localAppData, "smpc")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not create log directory: %v\n", err)
	}

	logPath := filepath.Join(logDir, "smpc.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err == nil {
		defer logFile.Close()

		// Use unbuffered writer to ensure immediate writes
		// This prevents log loss if the program crashes
		log.SetOutput(io.Writer(logFile))
		log.SetFlags(log.LstdFlags | log.Lmicroseconds) // Add timestamps with microseconds

		log.Printf("=== SMPC started at %s ===", time.Now().Format(time.RFC3339))
		log.Printf("Log file: %s", logPath)
		fmt.Printf("Debug log: %s\n", logPath)
	}

	// Recover from panics and log them
	defer func() {
		if r := recover(); r != nil {
			log.Printf("\n\n*** PANIC RECOVERED ***")
			log.Printf("Panic: %v", r)
			log.Printf("Stack trace:\n%s", debug.Stack())
			fmt.Fprintf(os.Stderr, "\n*** PANIC: %v ***\n", r)
			fmt.Fprintf(os.Stderr, "Check log file: %s\n", logPath)
			fmt.Println("\nPress Enter to close...")
			fmt.Scanln()
			os.Exit(2)
		}
	}()

	log.Println("Executing RootCmd...")
	if err := cmd.RootCmd.Execute(); err != nil {
		log.Printf("Error from RootCmd.Execute: %v", err)
		fmt.Fprintln(os.Stderr, err)
		fmt.Println("\nPress Enter to close...")
		fmt.Scanln()
		os.Exit(1)
	}

	log.Println("RootCmd completed successfully")
}
