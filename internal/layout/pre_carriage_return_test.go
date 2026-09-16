package layout

import (
	"strings"
	"testing"
)

// HTML input preprocessing turns CR and CRLF into LF. The engine kept the CR
// in white-space:pre lines, so the PDF painter emitted U+FFFD tofu boxes
// (programiz-cpp finding 3: 16 boxes in a CRLF code block).
func TestPreCarriageReturnIsLineBreak(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, "<html><body><pre>line1\r\nline2\rline3</pre></body></html>",
		sheet(t, `body { margin: 0 }`))

	lines := make([]string, 0, len(res.Ops))

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		if strings.ContainsAny(paintOp.Text, "\r\n") || strings.ContainsRune(paintOp.Text, '\uFFFD') {
			t.Fatalf("text op %q still carries a carriage return or replacement glyph", paintOp.Text)
		}

		lines = append(lines, strings.TrimSpace(paintOp.Text))
	}

	if len(lines) != 3 || lines[0] != "line1" || lines[1] != "line2" || lines[2] != "line3" {
		t.Fatalf("pre lines = %q, want line1/line2/line3 split on CR and CRLF", lines)
	}
}
