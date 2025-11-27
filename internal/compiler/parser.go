package compiler

import (
	"fmt"
	"regexp"
)

// parseStatLine parses a line like "Program Warnings: 1" and returns (1, true) if matched, else (0, false)
func ParseStatLine(line, prefix string) (int, bool) {
	pattern := "^" + regexp.QuoteMeta(prefix) + `\s*:\s*(\d+)`
	fmt.Printf("[DEBUG] parseStatLine: line=%q, prefix=%q, pattern=%q\n", line, prefix, pattern)

	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(line)

	if len(matches) == 2 {
		fmt.Printf("[DEBUG] parseStatLine: match found, value=%q\n", matches[1])

		var n int

		if _, err := fmt.Sscanf(matches[1], "%d", &n); err == nil {
			fmt.Printf("[DEBUG] parseStatLine: parsed int=%d\n", n)
			return n, true
		} else {
			fmt.Printf("[DEBUG] parseStatLine: failed to parse int: %v\n", err)
		}
	} else {
		fmt.Printf("[DEBUG] parseStatLine: no match\n")
	}

	return 0, false
}

// parseCompileTimeLine parses a line like "Compile Time: 0.23 seconds" and returns (0.23, true) if matched, else (0, false)
func ParseCompileTimeLine(line string) (float64, bool) {
	pattern := `^Compile Time\s*:\s*([0-9.]+)\s*(s|seconds)?`
	fmt.Printf("[DEBUG] parseCompileTimeLine: line=%q, pattern=%q\n", line, pattern)

	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(line)

	if len(matches) >= 2 {
		fmt.Printf("[DEBUG] parseCompileTimeLine: match found, value=%q\n", matches[1])

		var secs float64

		if _, err := fmt.Sscanf(matches[1], "%f", &secs); err == nil {
			fmt.Printf("[DEBUG] parseCompileTimeLine: parsed float=%f\n", secs)
			return secs, true
		} else {
			fmt.Printf("[DEBUG] parseCompileTimeLine: failed to parse float: %v\n", err)
		}
	} else {
		fmt.Printf("[DEBUG] parseCompileTimeLine: no match\n")
	}

	return 0, false
}
