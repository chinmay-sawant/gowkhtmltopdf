package line_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/line"
)

func TestSeverityOf(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want line.Severity
	}{
		{"warning: skipping <style>: parse error", line.Warn},
		{"Warning: object 1: large stylesheet volume", line.Warn},
		{"warn: short alias", line.Warn},
		{"error: failed to load", line.Error},
		{"err: short alias", line.Error},
		{"info: scanned 3 font path(s)", line.Info},
		{"Loading pages (1/1)", line.Info},
		{"Done", line.Info},
		// The bug this grammar fixes: an info line whose *message* mentions
		// "error" is not an error line.
		{"info: load error policy is skip, omitting", line.Info},
		{"  warning: leading whitespace is trimmed", line.Warn},
		{"", line.Info},
	}
	for _, tc := range cases {
		if got := line.SeverityOf(tc.in); got != tc.want {
			t.Errorf("SeverityOf(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestEmit(t *testing.T) {
	t.Parallel()

	var buf w

	line.Emit(&buf, line.Warn, "object %d: %s", 1, "boom")

	if got := buf.String(); got != "warning: object 1: boom\n" {
		t.Errorf("Emit(Warn) = %q", got)
	}
}

type w struct{ s string }

func (w *w) Write(p []byte) (int, error) {
	w.s += string(p)

	return len(p), nil
}

func (w *w) String() string { return w.s }
