package pdf

import (
	"os"
	"strings"
	"testing"
)

func TestType0CJKEmbedding(t *testing.T) {
	path := "/usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("system CJK font not available:", err)
	}
	f, err := ParseTTF(data)
	if err != nil {
		t.Fatal(err)
	}
	f.PostScriptName = "DroidSansFallback"
	d := fixedDoc(t)
	d.SetCompression(false)
	p := d.AddPage(400, 200)
	c := p.Content()
	c.UseEmbeddedFont("F1", f)
	c.BeginText()
	c.SetFont("F1", 14)
	c.TextAt(20, 100)
	c.TextShow("你好世界")
	c.EndText()
	out := string(writePDF(t, d))
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
	dir := "/usr/share/fonts/truetype/droid"
	if _, err := os.Stat(dir); err != nil {
		t.Skip(err)
	}
	reg, err := ScanFontDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	f := reg.Lookup([]string{"Droid Sans Fallback"}, 400, false)
	if f == nil {
		// try any family from the file
		names := []string{}
		for fam := range reg.byFamily {
			names = append(names, fam)
		}
		if len(names) == 0 {
			t.Fatal("no fonts indexed")
		}
		f = reg.Lookup(names[:1], 400, false)
	}
	if f == nil {
		t.Fatal("lookup failed")
	}
	if f.GlyphID('你') == 0 {
		t.Error("expected CJK glyph in DroidSansFallback")
	}
}

func TestFamilyNamesLiberation(t *testing.T) {
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

func TestType0MixedLatinFallback(t *testing.T) {
	path := "/usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("system CJK font not available:", err)
	}
	f, err := ParseTTF(data)
	if err != nil {
		t.Fatal(err)
	}
	f.PostScriptName = "DroidSansFallback"
	d := fixedDoc(t)
	d.SetCompression(false)
	p := d.AddPage(500, 200)
	c := p.Content()
	c.UseEmbeddedFont("F0", f)
	c.BeginText()
	c.SetFont("F0", 14)
	c.TextAt(20, 100)
	c.TextShow("Hello 中文 world")
	c.EndText()
	out := string(writePDF(t, d))
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
