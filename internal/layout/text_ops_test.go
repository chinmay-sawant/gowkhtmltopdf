package layout

import (
	"testing"
)

// TestTableTextOpsPreserved pins the text ops for a small table. The per-glyph
// advance cache this once covered is gone; the test keeps the metric result
// (one text op per cell string, in document order) under the direct
// face.AdvanceInPoints path.
func TestTableTextOpsPreserved(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body>
		<table><tr><th>SKU</th><th>SKU</th></tr>
		<tr><td>SKU-001</td><td>SKU-002</td></tr></table>
	</body></html>`)

	got := make([]string, 0, 4)

	for _, op := range res.Ops {
		if op.Kind == OpText {
			got = append(got, op.Text)
		}
	}

	want := []string{"SKU", "SKU", "SKU-001", "SKU-002"}
	if len(got) < len(want) {
		t.Fatalf("text ops = %v, want at least %v", got, want)
	}

	for i, needle := range want {
		if got[i] != needle {
			t.Fatalf("text[%d] = %q, want %q (ops=%v)", i, got[i], needle, got)
		}
	}
}
