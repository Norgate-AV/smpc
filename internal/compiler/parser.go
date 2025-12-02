package compiler

import (
	"fmt"
	"regexp"
)

// parseStatLine parses a line like "Program Warnings: 1" and returns (1, true) if matched, else (0, false)
func ParseStatLine(line, prefix string) (int, bool) {
	pattern := "^" + regexp.QuoteMeta(prefix) + `\s*:\s*(\d+)`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(line)

	if len(matches) != 2 {
		return 0, false
	}

	var n int
	if _, err := fmt.Sscanf(matches[1], "%d", &n); err != nil {
		return 0, false
	}

	return n, true
}

// parseCompileTimeLine parses a line like "Compile Time: 0.23 seconds" and returns (0.23, true) if matched, else (0, false)
func ParseCompileTimeLine(line string) (float64, bool) {
	pattern := `^Compile Time\s*:\s*([0-9.]+)\s*(s|seconds)?`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(line)

	if len(matches) < 2 {
		return 0, false
	}

	var secs float64
	if _, err := fmt.Sscanf(matches[1], "%f", &secs); err != nil {
		return 0, false
	}

	return secs, true
}
