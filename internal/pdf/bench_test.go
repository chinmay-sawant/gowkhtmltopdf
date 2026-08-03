package pdf

import (
	"bytes"
	"fmt"
	"testing"
)

// BenchmarkWrite50Pages measures full PDF serialization cost for 50 pages
// of text (embedded subset font, compressed streams).
func BenchmarkWrite50Pages(b *testing.B) {
	f, err := DefaultFont()
	if err != nil {
		b.Fatal(err)
	}
	build := func() *Document {
		d := NewDocument()
		d.SetInfo("Title", "Bench")
		for i := 0; i < 50; i++ {
			p := d.AddPage(595, 842)
			c := p.Content()
			c.UseEmbeddedFont("F1", f)
			c.BeginText()
			c.SetFont("F1", 11)
			c.TextAt(50, 800)
			for l := 0; l < 40; l++ {
				c.TextShow(fmt.Sprintf("Page %d line %d - the quick brown fox jumps over the lazy dog", i, l))
				c.TextNextLine()
			}
			c.EndText()
		}
		return d
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d := build()
		var buf bytes.Buffer
		if err := d.Write(&buf); err != nil {
			b.Fatal(err)
		}
		b.SetBytes(int64(buf.Len()))
	}
}
