package cmd

import "github.com/spf13/cobra"

// Config holds all application configuration
type Config struct {
	Verbose      bool
	RecompileAll bool
	ShowLogs     bool
}

// NewConfigFromFlags creates a Config from parsed command flags
func NewConfigFromFlags(cmd *cobra.Command) *Config {
	// Try to get from local flags first, fall back to persistent flags
	verbose := getBoolFlag(cmd, "verbose")
	recompileAll := getBoolFlag(cmd, "recompile-all")
	showLogs := getBoolFlag(cmd, "logs")

	return &Config{
		Verbose:      verbose,
		RecompileAll: recompileAll,
		ShowLogs:     showLogs,
	}
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
