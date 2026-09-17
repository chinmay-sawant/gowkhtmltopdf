package pdf

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// TestFontRuneUnionKeyedByFace is the two-page probe: page 1 registers face A
// as F1, page 2 registers face B as F1. Each subset must contain only the
// runes its own face painted. Before the union key carried the face, both
// subsets inherited the other page's rune.
func TestFontRuneUnionKeyedByFace(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	doc := NewDocument()
	doc.SetCompression(false)

	paintUnionPage(doc, faces.Regular, "A")
	paintUnionPage(doc, faces.Bold, "B")

	// Page 3 reuses the regular face as F1: the union is per face, so its
	// runes merge with page 1's instead of forming a second subset.
	paintUnionPage(doc, faces.Regular, "C")

	out := string(writePDF(t, doc))

	regularKey := fontUnionKey{name: "F1", face: faces.Regular}
	boldKey := fontUnionKey{name: "F1", face: faces.Bold}

	if got := doc.fontRunes[regularKey]; !slices.Equal(got, []rune{'A', 'C'}) {
		t.Errorf("regular union = %q, want [A C]", string(got))
	}

	if got := doc.fontRunes[boldKey]; !slices.Equal(got, []rune{'B'}) {
		t.Errorf("bold union = %q, want [B]", string(got))
	}

	if doc.fontKeys[regularKey] == "" || doc.fontKeys[boldKey] == "" {
		t.Fatal("subset cache keys not populated")
	}

	if doc.fontKeys[regularKey] == doc.fontKeys[boldKey] {
		t.Error("regular and bold faces share one subset cache key")
	}

	if !strings.Contains(doc.fontKeys[regularKey], fmt.Sprintf("%x", faces.Regular.fingerprint)) {
		t.Error("regular subset key is not keyed by the regular fingerprint")
	}

	if !strings.Contains(doc.fontKeys[boldKey], fmt.Sprintf("%x", faces.Bold.fingerprint)) {
		t.Error("bold subset key is not keyed by the bold fingerprint")
	}

	// Each subset's /ToUnicode maps only the runes its own face painted. A
	// name-keyed union would emit A, B and C in both subsets.
	assertToUnicodeOnce(t, out, "41", "0041")
	assertToUnicodeOnce(t, out, "42", "0042")
	assertToUnicodeOnce(t, out, "43", "0043")
}

// assertToUnicodeOnce fails unless the /ToUnicode pair appears exactly once.
func assertToUnicodeOnce(t *testing.T, out, byteHex, runeHex string) {
	t.Helper()

	pair := "<" + byteHex + "> <" + runeHex + ">"
	if got := strings.Count(out, pair); got != 1 {
		t.Errorf("/ToUnicode pair %s appears %d times, want 1", pair, got)
	}
}

// assertFontRuneUnion fails unless doc.fontRunes[key] equals want.
func assertFontRuneUnion(t *testing.T, doc *Document, key fontUnionKey, want []rune, label string) {
	t.Helper()

	if got := doc.fontRunes[key]; !slices.Equal(got, want) {
		t.Errorf("%s union = %q, want %q", label, string(got), string(want))
	}
}

// assertDistinctSubsetKeys fails unless both keys have cache entries and the
// entries differ (one subset per differing rune union).
func assertDistinctSubsetKeys(t *testing.T, doc *Document, first, second fontUnionKey) {
	t.Helper()

	if doc.fontKeys[first] == "" || doc.fontKeys[second] == "" {
		t.Fatal("subset cache keys not populated")
	}

	if doc.fontKeys[first] == doc.fontKeys[second] {
		t.Error("differing rune unions share one subset cache key")
	}
}

// assertCountIn fails unless needle appears want times in out.
func assertCountIn(t *testing.T, out, needle string, want int, msg string) {
	t.Helper()

	if got := strings.Count(out, needle); got != want {
		t.Errorf("%s: count = %d, want %d", msg, got, want)
	}
}

// assertDistinctSubsetTags fails unless out carries two distinct BaseFont
// subset tags for face (different 6-letter tags, same PostScript suffix).
func assertDistinctSubsetTags(t *testing.T, out string, face *Font) {
	t.Helper()

	face.ensureParsed()

	suffix := "+" + face.PostScriptName

	var names []string

	for _, name := range baseFontNames([]byte(out)) {
		if strings.HasSuffix(name, suffix) {
			names = append(names, name)
		}
	}

	if len(names) != 2 {
		t.Fatalf("BaseFont names = %v, want 2 distinct subset names", names)
	}

	if names[0] == names[1] {
		t.Errorf("both subsets carry the same tag %q", names[0])
	}
}

// TestFontSubsetSplitByResourceName pins the learncpp-5 shape: one face
// painted under two page-local resource names gets one subset per name when
// each name's rune union differs. In plans/0.2.7/real-sites/learncpp/
// evidence/learncpp.pdf, LiberationSans-Bold is F2 on 12 pages (52 codes) and
// F1 on page 10 only (22 codes, a strict subset of the F2 union). The two
// subsets carry different tags because the subset programs differ; this is
// the documented one-subset-per-(name, face) rule, not a cache miss.
func TestFontSubsetSplitByResourceName(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	doc := NewDocument()
	doc.SetCompression(false)

	// Mirror the artifact: the name with the wider union first, the name
	// whose union is a strict subset second.
	paintUnionPageNamed(doc, faces.Bold, "F2", "AB")
	paintUnionPageNamed(doc, faces.Bold, "F1", "A")

	out := string(writePDF(t, doc))

	wideKey := fontUnionKey{name: "F2", face: faces.Bold}
	narrowKey := fontUnionKey{name: "F1", face: faces.Bold}

	assertFontRuneUnion(t, doc, wideKey, []rune{'A', 'B'}, "F2")
	assertFontRuneUnion(t, doc, narrowKey, []rune{'A'}, "F1")
	assertDistinctSubsetKeys(t, doc, wideKey, narrowKey)

	// One FontFile2 per subset, so two independent embedded programs.
	assertCountIn(t, out, "/FontFile2", 2, "one subset per resource name")

	// Both subsets map A because both unions contain it; B lives only in the
	// wider F2 subset.
	assertCountIn(t, out, "<41> <0041>", 2, "/ToUnicode pair <41> <0041>")
	assertToUnicodeOnce(t, out, "42", "0042")
	assertDistinctSubsetTags(t, out, faces.Bold)
}

// TestFontSubsetSharedAcrossResourceNames is the other half of the rule: when
// two page-local names collect the same rune union, the name-independent
// cache key shares one subset. The learncpp artifact shows this in practice
// with OpenSans: xref 399 is /F1 on 12 pages and /F2 on page 10.
func TestFontSubsetSharedAcrossResourceNames(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	doc := NewDocument()
	doc.SetCompression(false)

	paintUnionPageNamed(doc, faces.Bold, "F1", "A")
	paintUnionPageNamed(doc, faces.Bold, "F2", "A")

	out := string(writePDF(t, doc))

	key1 := fontUnionKey{name: "F1", face: faces.Bold}
	key2 := fontUnionKey{name: "F2", face: faces.Bold}

	if doc.fontKeys[key1] == "" {
		t.Fatal("subset cache key not populated")
	}

	if doc.fontKeys[key1] != doc.fontKeys[key2] {
		t.Errorf("equal rune unions use different cache keys: %q vs %q", doc.fontKeys[key1], doc.fontKeys[key2])
	}

	if got := strings.Count(out, "/FontFile2"); got != 1 {
		t.Errorf("/FontFile2 count = %d, want 1 (shared subset)", got)
	}

	assertToUnicodeOnce(t, out, "41", "0041")
}

// paintUnionPage adds a page that registers face as F1 and paints text.
func paintUnionPage(doc *Document, face *Font, text string) {
	paintUnionPageNamed(doc, face, "F1", text)
}

// paintUnionPageNamed adds a page that registers face under the page-local
// resource name and paints text with it.
func paintUnionPageNamed(doc *Document, face *Font, name, text string) {
	page := doc.AddPage(200, 200)
	content := page.Content()
	content.UseEmbeddedFont(name, face)
	content.BeginText()
	content.SetFont(name, 12)
	content.TextAt(10, 50)
	content.TextShow(text)
	content.EndText()
}
