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

// paintUnionPage adds a page that registers face as F1 and paints text.
func paintUnionPage(doc *Document, face *Font, text string) {
	page := doc.AddPage(200, 200)
	content := page.Content()
	content.UseEmbeddedFont("F1", face)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(10, 50)
	content.TextShow(text)
	content.EndText()
}
