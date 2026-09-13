//nolint:wsl // white-box PDF resource test
package pdf

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
)

func TestContentBlendModeUsesPDFExtGState(t *testing.T) {
	t.Parallel()

	content := NewContent()
	content.Save()
	content.SetBlendMode("multiply")
	content.Restore()

	if !strings.Contains(string(content.Bytes()), "/bmMultiply gs\n") {
		t.Fatal("content stream did not select multiply blend mode")
	}
	if got := content.extGState(); got != "/ExtGState << /bmMultiply << /BM /Multiply >> >>" {
		t.Fatalf("ExtGState = %q", got)
	}
}

func TestContentBlendModeIgnoresUnknownMode(t *testing.T) {
	t.Parallel()

	content := NewContent()
	content.SetBlendMode("not-a-pdf-mode")

	if content.Bytes() != nil {
		t.Fatalf("unknown blend mode emitted content: %q", content.Bytes())
	}
	if got := content.extGState(); got != "" {
		t.Fatalf("ExtGState = %q, want empty", got)
	}
}

// TestContentTransparencyGroupFormXObject pins the transparency-group
// plumbing: the buffered operations become one Form XObject with /Group
// /S /Transparency /I true, painted through /BM /Multiply.
func TestContentTransparencyGroupFormXObject(t *testing.T) {
	t.Parallel()

	doc := NewDocument()
	doc.SetCompression(false)
	content := doc.AddPage(200, 200).Content()

	group := content.BeginTransparencyGroup()
	group.SetFillColor(1, 0, 0)
	group.Rect(10, 10, 20, 20)
	group.Fill()

	if err := content.EndTransparencyGroup(group, "multiply", [4]float64{0, 0, 200, 200}); err != nil {
		t.Fatal(err)
	}

	if stream := string(content.Bytes()); !strings.Contains(stream, "/Fm0 Do\n") {
		t.Fatalf("content stream did not invoke the group form: %q", stream)
	}

	gs := content.extGState()
	if !strings.Contains(gs, "/bmMultiply << /BM /Multiply >>") {
		t.Fatalf("ExtGState = %q, want the multiply blend entry", gs)
	}

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatal(err)
	}

	raw := buf.String()
	for _, want := range []string{
		"/Subtype /Form",
		"/Group << /S /Transparency /I true /CS /DeviceRGB >>",
		"/BM /Multiply",
		"/Fm0 Do",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("PDF lacks %q", want)
		}
	}
}

// TestContentTransparencyGroupNests pins one form per nested group: the
// inner form is emitted into the outer group's buffer, which pulls it in as
// an /XObject resource.
func TestContentTransparencyGroupNests(t *testing.T) {
	t.Parallel()

	doc := NewDocument()
	doc.SetCompression(false)
	content := doc.AddPage(200, 200).Content()

	outer := content.BeginTransparencyGroup()
	inner := outer.BeginTransparencyGroup()
	inner.SetFillColor(0, 0, 1)
	inner.Rect(0, 0, 5, 5)
	inner.Fill()

	if err := outer.EndTransparencyGroup(inner, "screen", [4]float64{0, 0, 200, 200}); err != nil {
		t.Fatal(err)
	}

	if err := content.EndTransparencyGroup(outer, "multiply", [4]float64{0, 0, 200, 200}); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatal(err)
	}

	raw := buf.String()

	if got := strings.Count(raw, "/Subtype /Form"); got != 2 {
		t.Fatalf("Form XObject count = %d, want 2 (one per nested group)", got)
	}

	if got := strings.Count(raw, " Do\n"); got != 2 {
		t.Fatalf("Do count = %d, want 2 (page invokes outer, outer invokes inner)", got)
	}

	if !strings.Contains(raw, "/bmScreen") || !strings.Contains(raw, "/bmMultiply") {
		t.Fatal("nested forms did not register both blend modes")
	}
}

// TestSemanticReadsTextInsideTransparencyGroups proves the semantic oracle
// follows Form XObject invocations: text painted inside a blend group must
// still be visible to callers that assert authored content.
func TestSemanticReadsTextInsideTransparencyGroups(t *testing.T) {
	t.Parallel()

	doc := NewDocument()
	doc.SetCompression(false)
	content := doc.AddPage(200, 200).Content()

	group := content.BeginTransparencyGroup()
	group.BeginText()
	group.TextAt(20, 40)
	group.TextShowLanguage("grouped", "")

	group.EndText()

	if err := content.EndTransparencyGroup(group, "multiply", [4]float64{0, 0, 200, 200}); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatal(err)
	}

	semantic, err := ParseSemantic(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(semantic.DocumentText(), "grouped") {
		t.Fatalf("semantic text = %q, want text from inside the form", semantic.DocumentText())
	}
}

// TestTransparencyGroupFontMatchesPageFont pins the deferred group
// materialization: a form's /Resources must reference the same font object the
// page uses, and that object's subset must carry the glyphs painted inside the
// form. Materializing the form during paint subsetted from an empty rune set,
// so the form embedded a space-only font and every glyph rendered as .notdef.
//
//nolint:cyclop,funlen // setup, resource lookups, and assertions in one test
func TestTransparencyGroupFontMatchesPageFont(t *testing.T) {
	t.Parallel()

	doc := NewDocument()
	doc.SetCompression(false)

	page := doc.AddPage(200, 200)
	content := page.Content()

	face, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	content.UseEmbeddedFont("F1", face)
	content.SetFont("F1", 9.5)
	content.BeginText()
	content.TextAt(20, 40)
	content.TextShow("page")
	content.EndText()

	group := content.BeginTransparencyGroup()
	group.UseEmbeddedFont("F1", face)
	group.SetFont("F1", 9.5)
	group.BeginText()
	group.TextAt(20, 80)
	group.TextShow("group")
	group.EndText()

	if err := content.EndTransparencyGroup(group, "multiply", [4]float64{0, 0, 200, 200}); err != nil {
		t.Fatal(err)
	}

	formRef := 0

	for name, ref := range content.imageUses {
		if name != "Fm0" {
			continue
		}

		formRef, err = strconv.Atoi(strings.TrimSuffix(ref, " 0 R"))
		if err != nil {
			t.Fatal(err)
		}
	}

	if formRef == 0 {
		t.Fatal("form resource was not registered on the page content")
	}

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatal(err)
	}

	pageFonts := resourceFontRefs(doc.objects[page.ref-1].dict)
	formFonts := resourceFontRefs(doc.objects[formRef-1].dict)

	pageRef, pageOK := pageFonts["F1"]
	formRefFont, formOK := formFonts["F1"]

	if !pageOK || !formOK {
		t.Fatalf("font resources missing: page=%v form=%v", pageFonts, formFonts)
	}

	if pageRef != formRefFont {
		t.Fatalf("form font ref = %d, page font ref = %d; the form must share the page subset", formRefFont, pageRef)
	}

	fontDict := doc.objects[pageRef-1].dict
	if strings.Contains(fontDict, "/FirstChar 32 /LastChar 32") {
		t.Fatalf("embedded subset is space-only: %s", fontDict)
	}
}

// resourceFontRefs returns the /Font name to object-ref map from an object
// dict's /Resources.
func resourceFontRefs(dict string) map[string]int {
	resources, err := requiredDictionary(dict, "/Resources")
	if err != nil {
		return map[string]int{}
	}

	fonts, err := resourceRefs(resources, "/Font")
	if err != nil {
		return map[string]int{}
	}

	return fonts
}
