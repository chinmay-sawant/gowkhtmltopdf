//nolint:testpackage,exhaustruct // tests reach into unexported state
package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const (
	droidSansFallbackPath   = "/usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf"
	droidSansFallbackPSName = "DroidSansFallback"
)

func TestType0CJKEmbedding(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(droidSansFallbackPath)
	if err != nil {
		t.Skip("system CJK font not available:", err)
	}

	fVal, err := ParseTTF(data)
	if err != nil {
		t.Fatal(err)
	}

	fVal.PostScriptName = droidSansFallbackPSName
	dVal := fixedDoc(t)
	dVal.SetCompression(false)
	p := dVal.AddPage(400, 200)
	cur := p.Content()
	cur.UseEmbeddedFont("F1", fVal)
	cur.BeginText()
	cur.SetFont("F1", 14)
	cur.TextAt(20, 100)
	cur.TextShow("你好世界")
	cur.EndText()

	out := string(writePDF(t, dVal))
	for _, want := range []string{
		"/Subtype /Type0",
		"/CIDFontType2",
		"/Encoding /Identity-H",
		"<4F60597D4E16754C> Tj",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}

// TestType0FontDescriptorNameMatchesBaseFont locks the Arlington rule that
// CIDFontType2 FontDescriptor /FontName equals the CIDFont /BaseFont
// (and the parent Type0 /BaseFont).
func TestType0FontDescriptorNameMatchesBaseFont(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(droidSansFallbackPath)
	if err != nil {
		t.Skip("system CJK font not available:", err)
	}

	fVal, err := ParseTTF(data)
	if err != nil {
		t.Fatal(err)
	}

	fVal.PostScriptName = droidSansFallbackPSName
	dVal := fixedDoc(t)
	dVal.SetCompression(false)
	p := dVal.AddPage(400, 200)
	cur := p.Content()
	cur.UseEmbeddedFont("F1", fVal)
	cur.BeginText()
	cur.SetFont("F1", 14)
	cur.TextAt(20, 100)
	cur.TextShow("你好世界")
	cur.EndText()

	out := writePDF(t, dVal)
	if err := assertCIDFontDescriptorNamesMatch(out); err != nil {
		t.Fatal(err)
	}
}

// assertCIDFontDescriptorNamesMatch verifies every CIDFontType2 FontDescriptor
// /FontName equals that CIDFont's /BaseFont (Arlington FontDescriptorCIDType2.FontName).
func assertCIDFontDescriptorNamesMatch(pdf []byte) error {
	objs := map[int]string{}

	reObj := regexp.MustCompile(`(?s)(\d+) 0 obj\s*(.*?)\s*endobj`)
	for _, m := range reObj.FindAllSubmatch(pdf, -1) {
		n, _ := strconv.Atoi(string(m[1]))
		objs[n] = string(m[2])
	}

	checked := 0

	for _, body := range objs {
		if !strings.Contains(body, "/Subtype /CIDFontType2") {
			continue
		}

		base := regexp.MustCompile(`/BaseFont\s*/([^\s/]+)`).FindStringSubmatch(body)

		desc := regexp.MustCompile(`/FontDescriptor\s+(\d+)\s+0\s+R`).FindStringSubmatch(body)
		if base == nil || desc == nil {
			return fmt.Errorf("%w: %s", errCIDFontType2MissingRefs, body[:min(120, len(body))])
		}

		descBody := objs[mustAtoi(desc[1])]

		fname := regexp.MustCompile(`/FontName\s*/([^\s/]+)`).FindStringSubmatch(descBody)
		if fname == nil {
			return fmt.Errorf("%w: %s", errFontDescriptorNoFontName, desc[1])
		}

		if fname[1] != base[1] {
			return fmt.Errorf("%w: %s != %s", errCIDFontDescriptorMismatch, base[1], fname[1])
		}

		checked++
	}

	if checked == 0 {
		return fmt.Errorf("%w", errNoCIDFontType2Objects)
	}

	return nil
}

var (
	errCIDFontType2MissingRefs   = errors.New("CIDFontType2 missing BaseFont or FontDescriptor")
	errFontDescriptorNoFontName  = errors.New("FontDescriptor missing FontName")
	errCIDFontDescriptorMismatch = errors.New("CIDFontType2 BaseFont != FontDescriptor FontName")
	errNoCIDFontType2Objects     = errors.New("no CIDFontType2 objects found")
)

func mustAtoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}

	return n
}

func TestRegistryScanDroid(t *testing.T) {
	t.Parallel()

	dir := "/usr/share/fonts/truetype/droid"
	if _, err := os.Stat(dir); err != nil {
		t.Skip(err)
	}

	reg := ScanFontDirs([]string{dir})

	fnt := reg.Lookup([]string{"Droid Sans Fallback"}, 400, false)
	if fnt == nil {
		// try any family from the file
		names := []string{}
		for fam := range reg.byFamily {
			names = append(names, fam)
		}

		if len(names) == 0 {
			t.Fatal("no fonts indexed")
		}

		fnt = reg.Lookup(names[:1], 400, false)
	}

	if fnt == nil {
		t.Fatal("lookup failed")
	}

	if fnt.GlyphID('你') == 0 {
		t.Error("expected CJK glyph in DroidSansFallback")
	}
}

func TestFamilyNamesLiberation(t *testing.T) {
	t.Parallel()
	f := testFont(t)

	names := f.FamilyNames()
	if len(names) == 0 {
		t.Fatal("expected family names")
	}

	found := false

	for _, n := range names {
		if strings.Contains(strings.ToLower(n), "liberation") {
			found = true
		}
	}

	if !found {
		t.Errorf("family names = %v, want Liberation*", names)
	}
}

func TestFontEmbedErrorPropagates(t *testing.T) {
	t.Parallel()
	// error, not silently drop the resource (which renders text invisible).
	data := fixedDoc(t)
	p := data.AddPage(100, 100)
	p.Content().UseEmbeddedFont("F1", &Font{}) //nolint:exhaustruct // intentional zero-value fields

	var buf bytes.Buffer

	err := data.Write(&buf)
	if err == nil {
		t.Fatal("expected Write error for unembeddable font")
	}

	if !strings.Contains(err.Error(), "embed font F1") {
		t.Errorf("Write error = %q, want it to wrap %q", err, "embed font F1")
	}
}

func TestType0MixedLatinFallback(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(droidSansFallbackPath)
	if err != nil {
		t.Skip("system CJK font not available:", err)
	}

	fVal, err := ParseTTF(data)
	if err != nil {
		t.Fatal(err)
	}

	fVal.PostScriptName = droidSansFallbackPSName
	dVal := fixedDoc(t)
	dVal.SetCompression(false)
	p := dVal.AddPage(500, 200)
	cur := p.Content()
	cur.UseEmbeddedFont("F0", fVal)
	cur.BeginText()
	cur.SetFont("F0", 14)
	cur.TextAt(20, 100)
	cur.TextShow("Hello 中文 world")
	cur.EndText()

	out := string(writePDF(t, dVal))
	if strings.Contains(out, "/FL_u") {
		t.Fatal("Latin fallback must not grow a Type0 sibling FL_u")
	}

	if !strings.Contains(out, "/F0_u") {
		t.Fatal("expected Type0 sibling F0_u for CJK run")
	}

	if !strings.Contains(out, "/FL ") {
		t.Fatal("expected Liberation Latin fallback resource FL")
	}

	if !strings.Contains(out, "(Hello )") && !strings.Contains(out, "(Hello)") {
		t.Fatal("expected simple-show Latin run")
	}

	if !strings.Contains(out, "<4E2D6587>") {
		t.Fatal("expected Identity-H CIDs for 中文")
	}
}

//nolint:dupl // PDF17 Type0 twin of the PDF20 test; identical apart from version tokens
func TestType0CJKEmbeddingPDF17(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(droidSansFallbackPath)
	if err != nil {
		t.Skip("system CJK font not available:", err)
	}

	fVal, err := ParseTTF(data)
	if err != nil {
		t.Fatal(err)
	}

	fVal.PostScriptName = droidSansFallbackPSName

	dVal, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF17): %v", err)
	}

	dVal.SetCompression(false)
	p := dVal.AddPage(400, 200)
	cur := p.Content()
	cur.UseEmbeddedFont("F1", fVal)
	cur.BeginText()
	cur.SetFont("F1", 14)
	cur.TextAt(20, 100)
	cur.TextShow("你好世界")
	cur.EndText()

	out := string(writePDF(t, dVal))
	for _, want := range []string{
		"%PDF-1.7",
		"/Subtype /Type0",
		"/CIDFontType2",
		"/Encoding /Identity-H",
		"/ToUnicode",
		"begincmap",
		"beginbfchar",
		"<4F60597D4E16754C> Tj",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in PDF 1.7 Type0 output", want)
		}
	}
}

//nolint:cyclop // assertions across multiple mixed font runs
func TestType0MixedLatinFallbackPDF17(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(droidSansFallbackPath)
	if err != nil {
		t.Skip("system CJK font not available:", err)
	}

	fVal, err := ParseTTF(data)
	if err != nil {
		t.Fatal(err)
	}

	fVal.PostScriptName = droidSansFallbackPSName

	dVal, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF17): %v", err)
	}

	dVal.SetCompression(false)
	p := dVal.AddPage(500, 200)
	cur := p.Content()
	cur.UseEmbeddedFont("F0", fVal)
	cur.BeginText()
	cur.SetFont("F0", 14)
	cur.TextAt(20, 100)
	cur.TextShow("Hello 中文 world")
	cur.EndText()

	out := string(writePDF(t, dVal))
	if !strings.HasPrefix(out, "%PDF-1.7") {
		t.Errorf("expected %%PDF-1.7 header")
	}

	if strings.Contains(out, "/FL_u") {
		t.Fatal("Latin fallback must not grow a Type0 sibling FL_u")
	}

	if !strings.Contains(out, "/F0_u") {
		t.Fatal("expected Type0 sibling F0_u for CJK run")
	}

	if !strings.Contains(out, "/FL ") {
		t.Fatal("expected Liberation Latin fallback resource FL")
	}

	if !strings.Contains(out, "(Hello )") && !strings.Contains(out, "(Hello)") {
		t.Fatal("expected simple-show Latin run")
	}

	if !strings.Contains(out, "<4E2D6587>") {
		t.Fatal("expected Identity-H CIDs for 中文")
	}
}

func TestCFFOTTORejection(t *testing.T) {
	t.Parallel()

	// An OpenType font with OTTO header (CFF outlines) must be rejected with errFontCFFNotSupported.
	ottoData := make([]byte, 100)
	copy(ottoData, "OTTO")

	_, err := ParseTTF(ottoData)
	if !errors.Is(err, errFontCFFNotSupported) {
		t.Fatalf("ParseTTF(OTTO) err = %v, want %v", err, errFontCFFNotSupported)
	}
}

// TestType0MissingGlyphsDoNotEmptyCmap: non-Latin runes the face lacks must
// not create an F0_u subset with an empty cmap (Write used to fail that way
// for fixture-61 katakana on Liberation).
func TestType0MissingGlyphsDoNotEmptyCmap(t *testing.T) {
	t.Parallel()

	face, err := DefaultFont()
	if err != nil || face == nil {
		t.Fatal(err)
	}

	dVal := fixedDoc(t)
	dVal.SetCompression(false)
	p := dVal.AddPage(400, 200)
	cur := p.Content()
	cur.UseEmbeddedFont("F0", face)
	cur.BeginText()
	cur.SetFont("F0", 12)
	cur.TextAt(20, 100)
	cur.TextShow("スーパー / super")
	cur.EndText()

	out := writePDF(t, dVal)
	s := string(out)
	if !bytes.Contains(out, []byte("%PDF-")) {
		t.Fatal("expected a PDF")
	}
	// No Type0 sibling when Liberation has no katakana glyphs.
	if strings.Contains(s, "/F0_u") {
		t.Fatal("Liberation must not grow F0_u for missing katakana glyphs")
	}
	if !strings.Contains(s, "(super)") && !strings.Contains(s, "super") {
		t.Fatal("expected Latin run to survive")
	}
}
