package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

const SIMPL_WINDOWS_PATH = "C:\\Program Files (x86)\\Crestron\\Simpl\\smpwin.exe"

var (
	shell32          = syscall.NewLazyDLL("shell32.dll")
	procShellExecute = shell32.NewProc("ShellExecuteW")
)

func shellExecute(hwnd uintptr, verb, file, args, cwd string, showCmd int) error {
	var verbPtr, filePtr, argsPtr, cwdPtr *uint16
	var err error

	if verb != "" {
		verbPtr, err = syscall.UTF16PtrFromString(verb)
		if err != nil {
			return err
		}
	}

	filePtr, err = syscall.UTF16PtrFromString(file)
	if err != nil {
		return err
	}

	if args != "" {
		argsPtr, err = syscall.UTF16PtrFromString(args)
		if err != nil {
			return err
		}
	}

	if cwd != "" {
		cwdPtr, err = syscall.UTF16PtrFromString(cwd)
		if err != nil {
			return err
		}
	}

	ret, _, _ := procShellExecute.Call(
		hwnd,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(filePtr)),
		uintptr(unsafe.Pointer(argsPtr)),
		uintptr(unsafe.Pointer(cwdPtr)),
		uintptr(showCmd),
	)

	// ShellExecute returns a value > 32 on success
	if ret <= 32 {
		return fmt.Errorf("ShellExecute failed with error code: %d", ret)
	}

	return nil
}

func main() {
	// Check if a file path argument was provided
	if len(os.Args) < 2 {
		fmt.Println("Usage: smpc <file-path>")
		os.Exit(1)
	}

	// Get the file path from the first command line argument
	filePath := os.Args[1]

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("Error: File does not exist: %s\n", filePath)
		os.Exit(1)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		fmt.Printf("Error resolving file path: %v\n", err)
		os.Exit(1)
	}

	// Open the file with SIMPL Windows application using elevated privileges
	// SW_SHOWNORMAL = 1
	if err := shellExecute(0, "runas", SIMPL_WINDOWS_PATH, absPath, "", 1); err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Opening file with elevation: %s\n", absPath)
}
