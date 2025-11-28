package compiler

import (
	"fmt"
	"log/slog"
	"regexp"
)

// parseStatLine parses a line like "Program Warnings: 1" and returns (1, true) if matched, else (0, false)
func ParseStatLine(line, prefix string) (int, bool) {
	pattern := "^" + regexp.QuoteMeta(prefix) + `\s*:\s*(\d+)`
	slog.Debug("parseStatLine", "line", line, "prefix", prefix, "pattern", pattern)

	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(line)

	if len(matches) == 2 {
		slog.Debug("parseStatLine match found", "value", matches[1])

		var n int

		if _, err := fmt.Sscanf(matches[1], "%d", &n); err == nil {
			slog.Debug("parseStatLine parsed int", "value", n)
			return n, true
		} else {
			slog.Debug("parseStatLine failed to parse int", "error", err)
		}
	} else {
		slog.Debug("parseStatLine no match")
	}

	return 0, false
}

// parseCompileTimeLine parses a line like "Compile Time: 0.23 seconds" and returns (0.23, true) if matched, else (0, false)
func ParseCompileTimeLine(line string) (float64, bool) {
	pattern := `^Compile Time\s*:\s*([0-9.]+)\s*(s|seconds)?`
	slog.Debug("parseCompileTimeLine", "line", line, "pattern", pattern)

	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(line)

	if len(matches) >= 2 {
		slog.Debug("parseCompileTimeLine match found", "value", matches[1])

		var secs float64

		if _, err := fmt.Sscanf(matches[1], "%f", &secs); err == nil {
			slog.Debug("parseCompileTimeLine parsed float", "value", secs)
			return secs, true
		} else {
			slog.Debug("parseCompileTimeLine failed to parse float", "error", err)
		}
	} else {
		slog.Debug("parseCompileTimeLine no match")
	}

	return 0, false
}
