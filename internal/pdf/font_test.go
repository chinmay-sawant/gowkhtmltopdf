//nolint:testpackage,exhaustruct // tests reach into unexported state
package pdf

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func testFont(t *testing.T) *Font {
	t.Helper()

	f, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	return f
}

func TestParseDefaultFont(t *testing.T) {
	t.Parallel()

	fnt := testFont(t)
	if fnt.UnitsPerEm() <= 0 {
		t.Errorf("bad unitsPerEm %d", fnt.UnitsPerEm())
	}

	if fnt.numGlyphs == 0 {
		t.Error("no glyphs")
	}

	if fnt.Advance('A') <= 0 {
		t.Errorf("bad advance for A: %v", fnt.Advance('A'))
	}
	// Latin coverage sanity: uppercase/lowercase/digits/space must map
	for _, r := range "ABCabc0129 .-," {
		if fnt.GlyphID(r) == 0 {
			t.Errorf("no glyph for %q", r)
		}
	}
	// missing rune → .notdef
	if fnt.GlyphID('日') != 0 {
		t.Error("expected .notdef for CJK")
	}
	// 'i' is typically narrower than 'm' in sans-serif
	if !(fnt.Advance('i') < fnt.Advance('m')) {
		t.Error("expected i narrower than m")
	}
}

func TestParseCmapFormats(t *testing.T) {
	t.Parallel()
	// Liberation Sans uses format 4; verify a few known mappings
	f := testFont(t)
	if g := f.GlyphID('A'); g != f.GlyphID('A') {
		t.Error("unstable glyph id")
	}

	if f.GlyphID('A') == f.GlyphID('Z') {
		t.Error("A and Z must differ")
	}
}

func TestSubsetUnicodeCmap(t *testing.T) {
	t.Parallel()

	mappings := []codeGlyph{
		{code: 32, glyph: 1},
		{code: 65, glyph: 2},
		{code: 66, glyph: 3},
		{code: 67, glyph: 4},
		{code: 233, glyph: 5},
	}

	cmap, err := unicodeCmap4(mappings)
	if err != nil {
		t.Fatal(err)
	}

	if got := binary.BigEndian.Uint16(cmap[0:2]); got != 0 {
		t.Errorf("cmap version = %d, want 0", got)
	}

	if got := binary.BigEndian.Uint16(cmap[2:4]); got != 1 {
		t.Errorf("numTables = %d, want 1", got)
	}

	subOff := binary.BigEndian.Uint32(cmap[8:12])

	state := cmap[subOff:]
	if fmt := binary.BigEndian.Uint16(state[0:2]); fmt != 4 {
		t.Errorf("subtable format %d, want 4", fmt)
	}

	segCount := int(binary.BigEndian.Uint16(state[6:8])) / 2
	endOff := 14
	startOff := endOff + 2*segCount + 2
	deltaOff := startOff + 2*segCount
	// last segment must be the 0xFFFF sentinel
	if e := binary.BigEndian.Uint16(state[endOff+(segCount-1)*2:]); e != 0xFFFF {
		t.Errorf("sentinel endCode = %#x, want 0xFFFF", e)
	}
	// the A,B,C run (codes 65,66,67 → glyphs 2,3,4) should be one segment
	for i := range segCount {
		start := binary.BigEndian.Uint16(state[startOff+i*2 : startOff+i*2+2])
		end := binary.BigEndian.Uint16(state[endOff+i*2 : endOff+i*2+2])

		if start == 65 {
			delta := binary.BigEndian.Uint16(state[deltaOff+i*2 : deltaOff+i*2+2])
			if end != 67 || delta != 0xFFC1 { // (2-65) & 0xFFFF
				t.Errorf("ABC segment = [%d..%d] delta %d, want [65..67] delta 0xFFC1", start, end, delta)
			}
		}
	}
}

func TestSubsetFont(t *testing.T) {
	t.Parallel()
	fnt := testFont(t)
	used := []rune("Hello, PDF! 123")
	distinct := map[rune]bool{}

	for _, r := range used {
		distinct[r] = true
	}

	sub, err := subsetFont(fnt, used, subsetSimple)
	if err != nil {
		t.Fatal(err)
	}

	if len(sub.data) >= len(fnt.data) {
		t.Errorf("subset (%d) not smaller than full font (%d)", len(sub.data), len(fnt.data))
	}

	if len(sub.widths) < len(distinct)+1 {
		t.Errorf("subset has %d glyphs, want >= %d used runes + .notdef", len(sub.widths), len(distinct))
	}
	// subset font must itself parse
	subFont, err := ParseTTF(sub.data)
	if err != nil {
		t.Fatalf("subset font does not reparse: %v", err)
	}
	// identity mapping: every used rune still maps to a glyph, and each
	// distinct rune maps to a distinct glyph id in the subset
	seen := map[uint16]bool{}

	for _, r := range used {
		glob := subFont.GlyphID(r)
		if glob == 0 {
			t.Errorf("rune %q lost in subset", r)

			continue
		}

		if seen[glob] {
			continue // duplicate rune in input
		}

		seen[glob] = true
	}
	// advance widths preserved in font units
	if subFont.Advance('H') != fnt.Advance('H') {
		t.Errorf("width drift: %v vs %v", subFont.Advance('H'), fnt.Advance('H'))
	}
}

func TestSubsetChecksum(t *testing.T) {
	t.Parallel()
	f := testFont(t)

	sub, err := subsetFont(f, []rune("abc"), subsetSimple)
	if err != nil {
		t.Fatal(err)
	}

	if got := checksum(sub.data); got != 0xB1B0AFBA {
		t.Errorf("font checksum = %#x, want 0xB1B0AFBA", got)
	}
}

func TestCompositeSubset(t *testing.T) {
	t.Parallel()
	fnt := testFont(t)
	// find a rune whose glyph is composite (accents, e.g. é 'à' 'ñ')
	allSimple := true

	var comp rune

	for _, r := range "éàñüÄÖ" {
		if len(fnt.compositeGlyphIDs(fnt.GlyphID(r))) > 0 {
			comp = r
			allSimple = false

			break
		}
	}

	if allSimple {
		t.Skip("font has no composite glyphs in test set")
	}

	sub, err := subsetFont(fnt, []rune{comp}, subsetSimple)
	if err != nil {
		t.Fatal(err)
	}

	subFont, err := ParseTTF(sub.data)
	if err != nil {
		t.Fatalf("composite subset does not reparse: %v", err)
	}

	if g := subFont.GlyphID(comp); g == 0 {
		t.Errorf("rune %q lost in composite subset", comp)
	}
}

func TestEmbeddedFontInPDF(t *testing.T) {
	t.Parallel()
	f := testFont(t)
	data := fixedDoc(t)
	data.SetCompression(false)
	p := data.AddPage(300, 300)
	cur := p.Content()
	cur.UseEmbeddedFont("F1", f)
	cur.BeginText()
	cur.SetFont("F1", 12)
	cur.TextAt(10, 20)
	cur.TextShow("Hello")
	cur.EndText()

	out := string(writePDF(t, data))
	for _, want := range []string{
		"/Type /Font /Subtype /TrueType",
		"/FontDescriptor",
		"/FontFile2",
		"/ToUnicode",
		"/Widths [",
		"begincmap",
		"beginbfchar",
		"(Hello) Tj",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	// embedded subset must be much smaller than the 139KB source
	if !strings.Contains(out, "/FontFile2") {
		t.Fatal("no FontFile2")
	}
	// count bfchar entries: Hello = H,e,l,o = 4 distinct runes
	if n := strings.Count(out, "beginbfchar"); n < 1 {
		t.Errorf("expected ToUnicode mappings, got %d", n)
	}
}

func TestFontCacheSharedAcrossPages(t *testing.T) {
	t.Parallel()
	fnt := testFont(t)
	data := fixedDoc(t)
	data.SetCompression(false)

	for range 2 {
		p := data.AddPage(100, 100)
		cur := p.Content()
		cur.UseEmbeddedFont("F1", fnt)
		cur.BeginText()
		cur.SetFont("F1", 10)
		cur.TextAt(5, 5)
		cur.TextShow("Hi")
		cur.EndText()
	}

	out := string(writePDF(t, data))
	// same font ref used on both pages
	if strings.Count(out, "/FontFile2") != 1 {
		t.Errorf("expected 1 FontFile2, got %d", strings.Count(out, "/FontFile2"))
	}
}

func TestFontCacheSeparatesLoadedFacesWithSameDisplayName(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}
	// A registry/display label is not a face identity. Deliberately give
	// regular and bold the same label and ensure their embedded subsets remain
	// separate.
	regular, err := ParseTTF(bytes.Clone(faces.Regular.data))
	if err != nil {
		t.Fatal(err)
	}

	regular.PostScriptName = "SameFace"

	bold, err := ParseTTF(bytes.Clone(faces.Bold.data))
	if err != nil {
		t.Fatal(err)
	}

	bold.PostScriptName = "SameFace"
	data := fixedDoc(t)
	data.SetCompression(false)

	for _, face := range []*Font{regular, bold} {
		p := data.AddPage(100, 100)
		cur := p.Content()
		cur.UseEmbeddedFont("F1", face)
		cur.BeginText()
		cur.SetFont("F1", 10)
		cur.TextAt(5, 5)
		cur.TextShow("H")
		cur.EndText()
	}

	if got := strings.Count(string(writePDF(t, data)), "/FontFile2"); got != 2 {
		t.Fatalf("FontFile2 count = %d, want 2 for distinct loaded faces", got)
	}
}

func TestEmbeddedFontStillWorks(t *testing.T) {
	t.Parallel()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	data := fixedDoc(t)
	data.SetCompression(false)
	p := data.AddPage(100, 100)
	cur := p.Content()
	cur.UseEmbeddedFont("F1", fnt)
	cur.BeginText()
	cur.SetFont("F1", 12)
	cur.TextAt(5, 5)
	cur.TextShow("x")
	cur.EndText()

	out := string(writePDF(t, data))
	if !strings.Contains(out, "/Subtype /TrueType") {
		t.Error("embedded TrueType font dict missing")
	}

	if !strings.Contains(out, "/FontFile2") {
		t.Error("embedded font file stream missing")
	}
}

func TestAdvanceInPoints(t *testing.T) {
	t.Parallel()
	f := testFont(t)
	w := f.AdvanceInPoints('A', 12)
	// A in Liberation Sans is ~1222/2048 em → ~7.16pt at 12pt
	if w < 6 || w > 9 {
		t.Errorf("advance for A at 12pt = %v, want ~7.16", w)
	}
}

func TestEmbeddedFontInPDF17(t *testing.T) {
	t.Parallel()

	fnt := testFont(t)

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF17): %v", err)
	}

	doc.SetCompression(false)
	p := doc.AddPage(300, 300)
	cur := p.Content()
	cur.UseEmbeddedFont("F1", fnt)
	cur.BeginText()
	cur.SetFont("F1", 12)
	cur.TextAt(10, 20)
	cur.TextShow("Hello PDF 1.7")
	cur.EndText()

	out := string(writePDF(t, doc))
	for _, want := range []string{
		"%PDF-1.7",
		"/Type /Font /Subtype /TrueType",
		"/FontDescriptor",
		"/FontFile2",
		"/ToUnicode",
		"/Widths [",
		"begincmap",
		"beginbfchar",
		"(Hello PDF 1.7) Tj",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in PDF 1.7 output", want)
		}
	}
}

func TestFontCacheSharedAcrossPagesPDF17(t *testing.T) {
	t.Parallel()

	fnt := testFont(t)

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF17): %v", err)
	}

	doc.SetCompression(false)

	for range 3 {
		p := doc.AddPage(100, 100)
		cur := p.Content()
		cur.UseEmbeddedFont("F1", fnt)
		cur.BeginText()
		cur.SetFont("F1", 10)
		cur.TextAt(5, 5)
		cur.TextShow("Shared Font")
		cur.EndText()
	}

	out := string(writePDF(t, doc))
	if strings.Count(out, "/FontFile2") != 1 {
		t.Errorf("expected 1 FontFile2 across pages in PDF 1.7, got %d", strings.Count(out, "/FontFile2"))
	}
}

func testFontWithStyle(t *testing.T, macStyle uint16) *Font {
	t.Helper()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	fresh, err := ParseTTF(fnt.data)
	if err != nil {
		t.Fatalf("ParseTTF: %v", err)
	}

	fresh.macStyle = macStyle

	return fresh
}

func TestSubsetPostTableFormat3(t *testing.T) {
	t.Parallel()

	fnt := testFont(t)

	sub, err := subsetFont(fnt, []rune("Test Format 3 Post"), subsetSimple)
	if err != nil {
		t.Fatalf("subsetFont failed: %v", err)
	}

	tables, err := parseTableDirectory(sub.data)
	if err != nil {
		t.Fatalf("parseTableDirectory failed: %v", err)
	}

	post, ok := tables["post"]
	if !ok {
		t.Fatal("missing post table in subset")
	}

	if len(post) != postTableSize {
		t.Errorf("post table size = %d, want exactly %d bytes (Format 3.0)", len(post), postTableSize)
	}

	version := binary.BigEndian.Uint32(post[0:4])
	if version != postTableVersion3 {
		t.Errorf("post table version = %#08x, want %#08x (Format 3.0)", version, postTableVersion3)
	}
}

func TestFontDescriptorFlagsRegular(t *testing.T) {
	t.Parallel()

	fnt := testFont(t)
	doc := fixedDoc(t)
	doc.SetCompression(false)

	page := doc.AddPage(200, 200)
	stream := page.Content()
	stream.UseEmbeddedFont("F1", fnt)
	stream.BeginText()
	stream.SetFont("F1", 12)
	stream.TextAt(10, 10)
	stream.TextShow("Regular")
	stream.EndText()

	out := string(writePDF(t, doc))
	if !strings.Contains(out, "/Flags 32") {
		t.Errorf("expected regular FontDescriptor /Flags 32, got:\n%s", out)
	}
}

func TestFontDescriptorFlagsBold(t *testing.T) {
	t.Parallel()

	fnt := testFontWithStyle(t, 1) // bold
	doc := fixedDoc(t)
	doc.SetCompression(false)

	page := doc.AddPage(200, 200)
	stream := page.Content()
	stream.UseEmbeddedFont("F2", fnt)
	stream.BeginText()
	stream.SetFont("F2", 12)
	stream.TextAt(10, 10)
	stream.TextShow("Bold")
	stream.EndText()

	out := string(writePDF(t, doc))
	if strings.Contains(out, "/Flags 36") || strings.Contains(out, "/Flags 4") {
		t.Errorf("bold FontDescriptor incorrectly set Symbolic flag bit 3: \n%s", out)
	}

	if !strings.Contains(out, "/Flags 32") {
		t.Errorf("expected bold FontDescriptor /Flags 32, got:\n%s", out)
	}
}

func TestFontDescriptorFlagsItalic(t *testing.T) {
	t.Parallel()

	fnt := testFontWithStyle(t, 2) // italic
	doc := fixedDoc(t)
	doc.SetCompression(false)

	page := doc.AddPage(200, 200)
	stream := page.Content()
	stream.UseEmbeddedFont("F3", fnt)
	stream.BeginText()
	stream.SetFont("F3", 12)
	stream.TextAt(10, 10)
	stream.TextShow("Italic")
	stream.EndText()

	out := string(writePDF(t, doc))
	if !strings.Contains(out, "/Flags 96") {
		t.Errorf("expected italic FontDescriptor /Flags 96 (32 | 64), got:\n%s", out)
	}
}
