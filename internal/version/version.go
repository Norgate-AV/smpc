package version

var (
	// Version is the semantic version (injected at build time via -ldflags)
	Version = "dev"
	// Commit is the git commit hash (injected at build time via -ldflags)
	Commit = "none"
	// Date is the build date (injected at build time via -ldflags)
	Date = "unknown"
)

// GetVersion returns the full version string
func GetVersion() string {
	return Version
}

// GetFullVersion returns version with commit and date info
func GetFullVersion() string {
	return Version + " (commit: " + Commit + ", built: " + Date + ")"
}
