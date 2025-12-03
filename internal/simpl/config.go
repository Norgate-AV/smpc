package simpl

import "os"

const DefaultSimplWindowsPath = "C:\\Program Files (x86)\\Crestron\\Simpl\\smpwin.exe"

// GetSimplWindowsPath returns the path to the SIMPL Windows executable.
// It checks the SIMPL_WINDOWS_PATH environment variable first,
// falling back to the default installation path if not set.
func GetSimplWindowsPath() string {
	if envPath := os.Getenv("SIMPL_WINDOWS_PATH"); envPath != "" {
		return envPath
	}

	return DefaultSimplWindowsPath
}
