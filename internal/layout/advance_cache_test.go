package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

func TestGlyphAdvanceCacheHits(t *testing.T) {
	t.Parallel()

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	eng := &engine{
		font:  faces.Regular,
		faces: faces,
		scale: 1,
	}
	sty := &ResolvedStyle{FontSize: 9}

	first, found := eng.primaryFaceRun("SKU", sty)
	if !found {
		t.Fatal("primaryFaceRun SKU failed")
	}

	if len(eng.advanceCache) == 0 {
		t.Fatal("first SKU measure did not populate the glyph cache")
	}

	hitsBefore := eng.advanceHits
	second, found := eng.primaryFaceRun("SKU", sty)

	if !found {
		t.Fatal("second primaryFaceRun SKU failed")
	}

	if first.w != second.w {
		t.Fatalf("SKU width changed: %v vs %v", first.w, second.w)
	}

	if eng.advanceHits-hitsBefore != len(eng.advanceCache) {
		t.Fatalf("second SKU measure hits=%d, cache entries=%d",
			eng.advanceHits-hitsBefore, len(eng.advanceCache))
	}
}

func TestGlyphAdvanceCachePreservesText(t *testing.T) {
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
