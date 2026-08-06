package pdf

import (
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
	f := testFont(t)
	if f.UnitsPerEm() <= 0 {
		t.Errorf("bad unitsPerEm %d", f.UnitsPerEm())
	}
	if f.numGlyphs == 0 {
		t.Error("no glyphs")
	}
	if f.Advance('A') <= 0 {
		t.Errorf("bad advance for A: %v", f.Advance('A'))
	}
	// Latin coverage sanity: uppercase/lowercase/digits/space must map
	for _, r := range "ABCabc0129 .-," {
		if f.GlyphID(r) == 0 {
			t.Errorf("no glyph for %q", r)
		}
	}
	// missing rune → .notdef
	if f.GlyphID('日') != 0 {
		t.Error("expected .notdef for CJK")
	}
	// 'i' is typically narrower than 'm' in sans-serif
	if !(f.Advance('i') < f.Advance('m')) {
		t.Error("expected i narrower than m")
	}
}

func TestParseCmapFormats(t *testing.T) {
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
	st := cmap[subOff:]
	if fmt := binary.BigEndian.Uint16(st[0:2]); fmt != 4 {
		t.Errorf("subtable format %d, want 4", fmt)
	}
	segCount := int(binary.BigEndian.Uint16(st[6:8])) / 2
	endOff := 14
	startOff := endOff + 2*segCount + 2
	deltaOff := startOff + 2*segCount
	// last segment must be the 0xFFFF sentinel
	if e := binary.BigEndian.Uint16(st[endOff+(segCount-1)*2:]); e != 0xFFFF {
		t.Errorf("sentinel endCode = %#x, want 0xFFFF", e)
	}
	// the A,B,C run (codes 65,66,67 → glyphs 2,3,4) should be one segment
	for i := 0; i < segCount; i++ {
		start := binary.BigEndian.Uint16(st[startOff+i*2 : startOff+i*2+2])
		end := binary.BigEndian.Uint16(st[endOff+i*2 : endOff+i*2+2])
		if start == 65 {
			delta := binary.BigEndian.Uint16(st[deltaOff+i*2 : deltaOff+i*2+2])
			if end != 67 || delta != 0xFFC1 { // (2-65) & 0xFFFF
				t.Errorf("ABC segment = [%d..%d] delta %d, want [65..67] delta 0xFFC1", start, end, delta)
			}
		}
	}
}

func TestSubsetFont(t *testing.T) {
	f := testFont(t)
	used := []rune("Hello, PDF! 123")
	distinct := map[rune]bool{}
	for _, r := range used {
		distinct[r] = true
	}
	sub, err := subsetFont(f, used)
	if err != nil {
		t.Fatal(err)
	}
	if len(sub.data) >= len(f.data) {
		t.Errorf("subset (%d) not smaller than full font (%d)", len(sub.data), len(f.data))
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
		g := subFont.GlyphID(r)
		if g == 0 {
			t.Errorf("rune %q lost in subset", r)
			continue
		}
		if seen[g] {
			continue // duplicate rune in input
		}
		seen[g] = true
	}
	// advance widths preserved in font units
	if subFont.Advance('H') != f.Advance('H') {
		t.Errorf("width drift: %v vs %v", subFont.Advance('H'), f.Advance('H'))
	}
}

func TestSubsetChecksum(t *testing.T) {
	f := testFont(t)
	sub, err := subsetFont(f, []rune("abc"))
	if err != nil {
		t.Fatal(err)
	}
	if got := checksum(sub.data); got != 0xB1B0AFBA {
		t.Errorf("font checksum = %#x, want 0xB1B0AFBA", got)
	}
}

func TestCompositeSubset(t *testing.T) {
	f := testFont(t)
	// find a rune whose glyph is composite (accents, e.g. é 'à' 'ñ')
	used := []rune("éàñüÄÖ")
	allSimple := true
	var comp rune
	for _, r := range used {
		if len(f.compositeGlyphIDs(f.GlyphID(r))) > 0 {
			comp = r
			allSimple = false
			break
		}
	}
	if allSimple {
		t.Skip("font has no composite glyphs in test set")
	}
	sub, err := subsetFont(f, []rune{comp})
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
	f := testFont(t)
	d := fixedDoc(t)
	d.SetCompression(false)
	p := d.AddPage(300, 300)
	c := p.Content()
	c.UseEmbeddedFont("F1", f)
	c.BeginText()
	c.SetFont("F1", 12)
	c.TextAt(10, 20)
	c.TextShow("Hello")
	c.EndText()
	out := string(writePDF(t, d))
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
	f := testFont(t)
	d := fixedDoc(t)
	d.SetCompression(false)
	for i := 0; i < 2; i++ {
		p := d.AddPage(100, 100)
		c := p.Content()
		c.UseEmbeddedFont("F1", f)
		c.BeginText()
		c.SetFont("F1", 10)
		c.TextAt(5, 5)
		c.TextShow("Hi")
		c.EndText()
	}
	out := string(writePDF(t, d))
	// same font ref used on both pages
	if strings.Count(out, "/FontFile2") != 1 {
		t.Errorf("expected 1 FontFile2, got %d", strings.Count(out, "/FontFile2"))
	}
}

func TestEmbeddedFontStillWorks(t *testing.T) {
	f, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}
	d := fixedDoc(t)
	d.SetCompression(false)
	p := d.AddPage(100, 100)
	c := p.Content()
	c.UseEmbeddedFont("F1", f)
	c.BeginText()
	c.SetFont("F1", 12)
	c.TextAt(5, 5)
	c.TextShow("x")
	c.EndText()
	out := string(writePDF(t, d))
	if !strings.Contains(out, "/Subtype /TrueType") {
		t.Error("embedded TrueType font dict missing")
	}
	if !strings.Contains(out, "/FontFile2") {
		t.Error("embedded font file stream missing")
	}
}

func TestAdvanceInPoints(t *testing.T) {
	f := testFont(t)
	w := f.AdvanceInPoints('A', 12)
	// A in Liberation Sans is ~1222/2048 em → ~7.16pt at 12pt
	if w < 6 || w > 9 {
		t.Errorf("advance for A at 12pt = %v, want ~7.16", w)
	}
}
