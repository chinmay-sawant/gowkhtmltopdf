package pdf

import (
	"bytes"
	"strings"
	"testing"
)

// TestCatalogLangEmittedForUntaggedDocument makes /Lang a writer-level
// guarantee for every version, not only the PDF/UA catalog.
func TestCatalogLangEmittedForUntaggedDocument(t *testing.T) {
	t.Parallel()

	doc := NewDocument()
	doc.SetLanguage("en-GB")
	doc.AddPage(595, 842)

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if !strings.Contains(buf.String(), "/Lang (en-GB)") {
		t.Fatalf("catalog missing /Lang (en-GB):\n%s", buf.String())
	}
}

// TestCatalogHasNoLangWhenUnset keeps untagged documents free of a fabricated
// language while the PDF/UA default (en-US) stays in place.
func TestCatalogHasNoLangWhenUnset(t *testing.T) {
	t.Parallel()

	doc := NewDocument()
	doc.AddPage(595, 842)

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if strings.Contains(buf.String(), "/Lang") {
		t.Fatalf("unset-language catalog contains /Lang:\n%s", buf.String())
	}
}
