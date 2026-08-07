// Package line owns the engine's log-line severity protocol. Emitters
// prefix lines with a severity marker (via Emit) and consumers classify
// them with SeverityOf, so the grammar lives in exactly one place instead
// of being re-derived with substring guesses.
package line

import (
	"fmt"
	"io"
	"strings"
)

// Severity is the classification of one engine log line.
type Severity int

const (
	// Info is a plain log line (phases, progress, diagnostics).
	Info Severity = iota
	// Warn is a "warning: …" line: non-fatal, conversion continues.
	Warn
	// Error is an "error: …" line: the failure was reported.
	Error
)

// Emit writes one newline-terminated log line to w, prefixed with the
// severity marker the engine's consumers understand ("info: ",
// "warning: " or "error: ").
func Emit(w io.Writer, sev Severity, format string, args ...any) {
	prefix := "info: "
	switch sev {
	case Warn:
		prefix = "warning: "
	case Error:
		prefix = "error: "
	}
	fmt.Fprintf(w, prefix+format+"\n", args...)
}

// SeverityOf classifies one engine log line by its leading marker token;
// lines without a marker (or with an unknown one) are Info.
func SeverityOf(s string) Severity {
	lower := strings.ToLower(strings.TrimSpace(s))
	switch {
	case strings.HasPrefix(lower, "warning:"), strings.HasPrefix(lower, "warn:"):
		return Warn
	case strings.HasPrefix(lower, "error:"), strings.HasPrefix(lower, "err:"):
		return Error
	default:
		return Info
	}
}
