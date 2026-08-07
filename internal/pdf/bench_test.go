package pdf

import (
	"bytes"
	"fmt"
	"testing"
)

// BenchmarkWrite50Pages measures full PDF serialization cost for 50 pages
// of text (embedded subset font, compressed streams).
func BenchmarkWrite50Pages(b *testing.B) {
	fnt, err := DefaultFont()
	if err != nil {
		b.Fatal(err)
	}

	build := func() *Document {
		doc := NewDocument()
		doc.SetInfo("Title", "Bench")

		for idx := range 50 {
			p := doc.AddPage(595, 842)
			cur := p.Content()
			cur.UseEmbeddedFont("F1", fnt)
			cur.BeginText()
			cur.SetFont("F1", 11)
			cur.TextAt(50, 800)

			for l := range 40 {
				cur.TextShow(fmt.Sprintf("Page %d line %d - the quick brown fox jumps over the lazy dog", idx, l))
				cur.TextNextLine()
			}

			cur.EndText()
		}

		return doc
	}

	b.ResetTimer()

	for range b.N {
		d := build()

		var buf bytes.Buffer

		if err := d.Write(&buf); err != nil {
			b.Fatal(err)
		}

		b.SetBytes(int64(buf.Len()))
	}
}

// BenchmarkShapeRun measures the shared shaping + advance path consumed by
// both PDF text emission and image rasterization.
func BenchmarkShapeRun(b *testing.B) {
	fnt, err := DefaultFont()
	if err != nil {
		b.Fatal(err)
	}

	const text = "The quick brown fox jumps over the lazy dog"

	b.ResetTimer()

	for range b.N {
		run := ShapeRun(text, fnt, 11)
		if len(run.Runes) != len(run.Advances) {
			b.Fatal("shaped run has unpaired advances")
		}
	}
}
