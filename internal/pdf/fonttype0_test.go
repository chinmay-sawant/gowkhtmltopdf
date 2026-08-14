//nolint:testpackage,exhaustruct // tests reach into unexported state
package pdf

import (
	"bytes"
	"errors"
	"os"
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
