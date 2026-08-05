package convert

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// webFixturePath resolves a static HTML file under testdata/web/.
func webFixturePath(file string) string {
	return filepath.Join("..", "..", "testdata", "web", file)
}

// loadWebFixture reads a vendored web fixture for acceptance tests.
func loadWebFixture(t *testing.T, file string) string {
	t.Helper()
	html, err := os.ReadFile(webFixturePath(file))
	if err != nil {
		t.Fatalf("read web fixture %s: %v", file, err)
	}
	return string(html)
}

// TestWebWikiFixtureAcceptance is Phase 21 §21.6: vendored wiki-like HTML
// must produce a non-empty, structurally valid PDF that contains the article
// title and stays within a small page-count band (no live network).
func TestWebWikiFixtureAcceptance(t *testing.T) {
	html := loadWebFixture(t, "wiki-like-article.html")
	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "wiki.pdf"))
	cmd.Global.UseCompression = false
	data := runPDF(t, cmd)

	if len(data) == 0 {
		t.Fatal("PDF is empty")
	}
	assertPDFStructure(t, data)

	if !bytes.Contains(data, []byte("Ana de Armas")) {
		t.Error("PDF missing article title string \"Ana de Armas\"")
	}

	n := pageCount(data)
	const minPages, maxPages = 1, 3
	if n < minPages || n > maxPages {
		t.Errorf("pages = %d, want %d..%d", n, minPages, maxPages)
	}

	// Chrome present in HTML but display:none — must not dominate print.
	if bytes.Contains(data, []byte("Random article")) {
		t.Error("nav chrome text should be display:none and absent from PDF")
	}
}

// TestWebMarketingFixtureAcceptance is Phase 21 §21.6: marketing landing
// fixture must surface hero + primary CTA text in the PDF (CI-safe, offline).
func TestWebMarketingFixtureAcceptance(t *testing.T) {
	html := loadWebFixture(t, "marketing-landing.html")
	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "marketing.pdf"))
	cmd.Global.UseCompression = false
	data := runPDF(t, cmd)

	if len(data) == 0 {
		t.Fatal("PDF is empty")
	}
	assertPDFStructure(t, data)

	if !bytes.Contains(data, []byte("Ship readable PDFs from any HTML")) {
		t.Error("PDF missing hero headline")
	}
	// CTA may wrap across Tj runs ("Start free" + "trial"); accept contiguous or split.
	hasCTA := bytes.Contains(data, []byte("Start free trial")) ||
		(bytes.Contains(data, []byte("Start free")) && bytes.Contains(data, []byte("trial")))
	if !hasCTA {
		t.Error("PDF missing primary CTA text")
	}

	n := pageCount(data)
	if n < 1 {
		t.Errorf("pages = %d, want >= 1", n)
	}
}
