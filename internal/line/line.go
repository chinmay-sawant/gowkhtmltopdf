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
	// Unknown is the zero value of Severity. The engine never emits it; a
	// zero Severity must not silently print as "info".
	Unknown Severity = iota
	// Info is a plain log line (phases, progress, diagnostics).
	Info
	// Warn is a warning line: non-fatal, conversion continues.
	Warn
	// Error is an error line: the failure was reported.
	Error
)

// String returns the canonical name of the severity level.
func (s Severity) String() string {
	switch s {
	case Info:
		return "info"
	case Warn:
		return "warning"
	case Error:
		return "error"
	default:
		return "unknown"
	}
}

// Prefix returns the formatted prefix marker for the severity level.
func (s Severity) Prefix() string {
	return s.String() + ": "
}

// Emit writes one newline-terminated log line to writer, prefixed with the
// severity marker the engine's consumers understand ("info: ",
// "warning: " or "error: ").
func Emit(writer io.Writer, sev Severity, format string, args ...any) {
	if writer == nil {
		return
	}

	prefix := sev.Prefix()
	if len(args) == 0 {
		_, _ = io.WriteString(writer, prefix)
		_, _ = io.WriteString(writer, format)
		_, _ = io.WriteString(writer, "\n")

		return
	}

	// Avoid prefix+format allocation: write prefix, then formatted message.
	_, _ = io.WriteString(writer, prefix)
	_, _ = fmt.Fprintf(writer, format+"\n", args...)
}

// SeverityOf classifies one engine log line by its leading marker token.
// Lines without a marker (or with an unknown one) are Info by design: the
// engine prints bare progress lines ("Loading pages (1/1)", "Done") with no
// prefix, and those must classify as the least alarming level. Unknown (the
// zero value) is never returned here; it exists so a zero Severity used as
// an Emit argument is visible in output as "unknown" instead of masquerading
// as info.
func SeverityOf(s string) Severity {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) == 0 {
		return Info
	}

	if len(trimmed) >= 8 && strings.EqualFold(trimmed[:8], "warning:") {
		return Warn
	}

	if len(trimmed) >= 5 && strings.EqualFold(trimmed[:5], "warn:") {
		return Warn
	}

	if len(trimmed) >= 6 && strings.EqualFold(trimmed[:6], "error:") {
		return Error
	}

	if len(trimmed) >= 4 && strings.EqualFold(trimmed[:4], "err:") {
		return Error
	}

	return Info
}
