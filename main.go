// Package main is the entry point for the smpc command-line tool.
package main

import (
	"os"

	"github.com/Norgate-AV/smpc/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
