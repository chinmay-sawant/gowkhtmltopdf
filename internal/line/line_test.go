package line

import "testing"

func TestSeverityOf(t *testing.T) {
	cases := []struct {
		in   string
		want Severity
	}{
		{"warning: skipping <style>: parse error", Warn},
		{"Warning: object 1: large stylesheet volume", Warn},
		{"warn: short alias", Warn},
		{"error: failed to load", Error},
		{"err: short alias", Error},
		{"info: scanned 3 font path(s)", Info},
		{"Loading pages (1/1)", Info},
		{"Done", Info},
		// The bug this grammar fixes: an info line whose *message* mentions
		// "error" is not an error line.
		{"info: load error policy is skip, omitting", Info},
		{"  warning: leading whitespace is trimmed", Warn},
		{"", Info},
	}
	for _, tc := range cases {
		if got := SeverityOf(tc.in); got != tc.want {
			t.Errorf("SeverityOf(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestEmit(t *testing.T) {
	var buf w
	Emit(&buf, Warn, "object %d: %s", 1, "boom")
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
