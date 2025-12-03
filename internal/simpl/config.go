package simpl

import (
	"fmt"
	"os"
)

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

// ValidateSimplWindowsInstallation checks if the SIMPL Windows executable exists.
// Returns an error with helpful guidance if the file is not found.
func ValidateSimplWindowsInstallation() error {
	path := GetSimplWindowsPath()

	var err error
	if _, err = os.Stat(path); os.IsNotExist(err) {
		if os.Getenv("SIMPL_WINDOWS_PATH") != "" {
			return fmt.Errorf("SIMPL Windows not found at custom path: %s\n"+
				"Please verify the SIMPL_WINDOWS_PATH environment variable is correct", path)
		}

		return fmt.Errorf("SIMPL Windows not found at default path: %s\n"+
			"Please install SIMPL Windows or set SIMPL_WINDOWS_PATH environment variable", path)
	}

	if err != nil {
		return fmt.Errorf("error checking SIMPL Windows installation at %s: %w", path, err)
	}

	return nil
}
