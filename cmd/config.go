// Package cmd implements the command-line interface for smpc.
package cmd

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/Norgate-AV/smpc/internal/timeouts"
)

// Config holds all application configuration
type Config struct {
	Verbose            bool
	RecompileAll       bool
	ShowLogs           bool
	CompilationTimeout time.Duration
}

// NewConfigFromFlags creates a Config from parsed command flags
func NewConfigFromFlags(cmd *cobra.Command) *Config {
	// Try to get from local flags first, fall back to persistent flags
	verbose := getBoolFlag(cmd, "verbose")
	recompileAll := getBoolFlag(cmd, "recompile-all")
	showLogs := getBoolFlag(cmd, "logs")

	return &Config{
		Verbose:            verbose,
		RecompileAll:       recompileAll,
		ShowLogs:           showLogs,
		CompilationTimeout: getDurationFlag(cmd, "compile-timeout"),
	}
}

func getDurationFlag(cmd *cobra.Command, name string) time.Duration {
	value, err := cmd.Flags().GetDuration(name)
	if err != nil {
		value, err = cmd.PersistentFlags().GetDuration(name)
	}
	if err != nil {
		return timeouts.CompilationCompleteTimeout
	}
	return value
}

// getBoolFlag retrieves a boolean flag, checking both local and persistent flags
func getBoolFlag(cmd *cobra.Command, name string) bool {
	val, err := cmd.Flags().GetBool(name)
	if err != nil {
		// Try persistent flags if not found in local flags
		val, _ = cmd.PersistentFlags().GetBool(name)
	}

	return val
}
