package convert

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWikiLikeArticleFixture(t *testing.T) {
	t.Parallel()

	src := filepath.Join("..", "..", "testdata", "web", "wiki-like-article.html")

	html, err := os.ReadFile(src)
	if err != nil {
		t.Skip(err)
	}

	cmd, _ := newCommand(t, string(html), filepath.Join(t.TempDir(), "wiki.pdf"))
	cmd.Global.UseCompression = false
	data := runPDF(t, cmd)

	if !bytes.Contains(data, []byte("Ana de Armas")) && !bytes.Contains(data, []byte("(Ana de Armas)")) {
		// coalesced or split - require Ana at minimum
		if !bytes.Contains(data, []byte("Ana")) {
			t.Error("PDF missing article title evidence")
		}
	}

	if !bytes.Contains(data, []byte("Cuban")) && !bytes.Contains(data, []byte("actress")) {
		t.Error("PDF missing article body evidence")
	}
	// nav chrome should be display:none
	if bytes.Contains(data, []byte("Random article")) {
		t.Error("nav chrome text should be display:none and absent from PDF")
	}

	if pageCount(data) < 1 {
		t.Fatal("empty PDF")
	}
}

func TestBoldFaceInInvoicePDF(t *testing.T) {
	t.Parallel()

	src := filepath.Join("..", "..", "testdata", "golden", "fixture-01-simple-invoice.html")

	html, err := os.ReadFile(src)
	if err != nil {
		t.Skip(err)
	}

	cmd, _ := newCommand(t, string(html), filepath.Join(t.TempDir(), "inv.pdf"))
	cmd.Global.UseCompression = false

	data := runPDF(t, cmd)
	if !bytes.Contains(data, []byte("LiberationSans-Bold")) {
		t.Error("invoice PDF should embed bold face for headings/strong")
	}

	if n := bytes.Count(data, []byte("/FontFile2")); n < 2 {
		t.Errorf("FontFile2 count = %d, want >= 2 (regular+bold)", n)
	}
}
