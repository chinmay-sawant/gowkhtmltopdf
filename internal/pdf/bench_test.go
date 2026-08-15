//nolint:testpackage,gochecknoglobals,exhaustruct,cyclop,varnamelen,wsl,lll // tests reach into unexported state for benchmarks
package pdf

import (
	"bytes"
	"fmt"
	"testing"
)

type benchTarget struct {
	name   string
	policy WriterPolicy
	isUA   bool
	isPDFA bool
}

var benchProfiles = []benchTarget{
	{name: "default-1.4", policy: WriterPolicy{Version: PDF14}, isUA: false, isPDFA: false},
	{name: "pdf-1.7", policy: WriterPolicy{Version: PDF17}, isUA: false, isPDFA: false},
	{name: "pdf-2.0", policy: WriterPolicy{Version: PDF20}, isUA: false, isPDFA: false},
	{name: "pdfa-3a", policy: WriterPolicy{Version: PDF17, ConformanceProfile: "PDF/A-3a"}, isUA: false, isPDFA: true},
	{name: "pdfua-1", policy: WriterPolicy{Version: PDF17, ConformanceProfile: "PDF/UA-1"}, isUA: true, isPDFA: false},
	{
		name:   "a3a-ua1",
		policy: WriterPolicy{Version: PDF17, ConformanceProfile: "PDF/A-3a+PDF/UA-1"},
		isUA:   true,
		isPDFA: true,
	},
	{name: "pdfa-4", policy: WriterPolicy{Version: PDF20, ConformanceProfile: "PDF/A-4"}, isUA: false, isPDFA: true},
	{name: "pdfua-2", policy: WriterPolicy{Version: PDF20, ConformanceProfile: "PDF/UA-2"}, isUA: true, isPDFA: false},
	{
		name:   "a4-ua2",
		policy: WriterPolicy{Version: PDF20, ConformanceProfile: "PDF/A-4+PDF/UA-2"},
		isUA:   true,
		isPDFA: true,
	},
}

func runBenchPages(b *testing.B, numPages int, target benchTarget, fnt *Font) {
	b.Helper()

	build := func() *Document {
		doc, err := NewDocumentWithPolicy(target.policy)
		if err != nil {
			b.Fatal(err)
		}
		doc.SetInfo("Title", "Bench Document")

		var docElem *StructElem
		if target.isUA {
			root := doc.CreateStructTreeRoot()
			docElem = root.NewChild(StructTypeDocument)
		}

		for idx := range numPages {
			p := doc.AddPage(595, 842)
			cur := p.Content()
			cur.UseEmbeddedFont("F1", fnt)

			var pElem *StructElem
			var mcid int
			if target.isUA && docElem != nil {
				pElem = docElem.NewChild(StructTypeP)
				mcid = p.AllocMCID(pElem)
				cur.BeginMarkedContent("P", mcid)
			}

			cur.BeginText()
			cur.SetFont("F1", 11)
			cur.TextAt(50, 800)

			for l := range 40 {
				cur.TextShow(fmt.Sprintf("Page %d line %d - the quick brown fox jumps over the lazy dog", idx, l))
				cur.TextNextLine()
			}

			cur.EndText()

			if target.isUA && docElem != nil {
				cur.EndMarkedContent()
			}
		}

		return doc
	}

	for b.Loop() {
		d := build()

		var buf bytes.Buffer

		if err := d.Write(&buf); err != nil {
			b.Fatal(err)
		}

		b.SetBytes(int64(buf.Len()))
	}
}

// BenchmarkWrite50Pages measures full PDF serialization cost for 50 pages across profiles.
func BenchmarkWrite50Pages(b *testing.B) {
	fnt, err := DefaultFont()
	if err != nil {
		b.Fatal(err)
	}

	for _, target := range benchProfiles {
		b.Run(target.name, func(sub *testing.B) {
			runBenchPages(sub, 50, target, fnt)
		})
	}
}

// BenchmarkWrite500Pages measures full PDF serialization cost for 500 pages across profiles.
func BenchmarkWrite500Pages(b *testing.B) {
	fnt, err := DefaultFont()
	if err != nil {
		b.Fatal(err)
	}

	for _, target := range benchProfiles {
		b.Run(target.name, func(sub *testing.B) {
			runBenchPages(sub, 500, target, fnt)
		})
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

	for b.Loop() {
		run := ShapeRun(text, fnt, 11)
		if len(run.Runes) != len(run.Advances) {
			b.Fatal("shaped run has unpaired advances")
		}
	}
}
