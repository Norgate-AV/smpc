package version

var (
	// Version is the semantic version (injected at build time via -ldflags)
	version = "dev"
	// Commit is the git commit hash (injected at build time via -ldflags)
	commit = "none"
	// Date is the build date (injected at build time via -ldflags)
	date = "unknown"
)

// GetVersion returns the full version string
func GetVersion() string {
	return version
}

// Commit returns the git commit hash
func GetCommit() string {
	return commit
}

// Date returns the build date
func GetDate() string {
	return date
}

// GetFullVersion returns version with commit and date info
func GetFullVersion() string {
	return version + " (commit: " + commit + ", built: " + date + ")"
}
