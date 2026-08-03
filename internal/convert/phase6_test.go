package convert

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/settings"
)

// longBody builds an HTML body long enough to span at least two pages.
func longBody(marker string) string {
	var b bytes.Buffer
	b.WriteString("<html><body>")
	for i := 0; i < 60; i++ {
		b.WriteString("<p>paragraph ")
		b.WriteString(string(rune('a' + i%26)))
		b.WriteString(" with some words to wrap across the page width</p>")
	}
	b.WriteString(marker)
	b.WriteString("</body></html>")
	return b.String()
}

func TestTextHeaderFooter(t *testing.T) {
	cmd, _ := newCommand(t, longBody(""), filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Header.Left = "Page [page]/[topage]"
	cmd.Global.Footer.Right = "doc [title]"
	cmd.Global.Title = "T"
	cmd.Global.UseCompression = false
	data := runPDF(t, cmd)

	pages := pageCount(data)
	if pages < 2 {
		t.Fatalf("pages = %d, want >= 2", pages)
	}
	want1 := []byte("Page 1/" + itoa(pages))
	want2 := []byte("Page 2/" + itoa(pages))
	if !bytes.Contains(data, want1) {
		t.Errorf("page 1 header missing %q", want1)
	}
	if !bytes.Contains(data, want2) {
		t.Errorf("page 2 header missing %q", want2)
	}
	if !bytes.Contains(data, []byte("doc T")) {
		t.Error("[title] placeholder not substituted in footer")
	}
}

func TestPlaceholderReplace(t *testing.T) {
	cmd, _ := newCommand(t, `<html><body><p>x</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Footer.Center = "[who] inc. — [unknown]"
	cmd.Global.Header.Replace = map[string]string{"[who]": "Acme"}
	cmd.Global.UseCompression = false
	data := runPDF(t, cmd)
	if !bytes.Contains(data, []byte("Acme inc.")) {
		t.Error("--replace key not substituted in footer text")
	}
	if !bytes.Contains(data, []byte("[unknown]")) {
		t.Error("unknown placeholder must pass through literally")
	}
}

func TestSectionSubsectionPlaceholder(t *testing.T) {
	body := `<html><body><h1>Chap A</h1><h2>Sec B</h2><p>` +
		`text text text text text text text text text text text text ` +
		`text text text text text text text text text text text text</p></body></html>`
	cmd, _ := newCommand(t, body, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Header.Right = "[section]/[subsection]"
	cmd.Global.UseCompression = false
	data := runPDF(t, cmd)
	if !bytes.Contains(data, []byte("Chap A/Sec B")) {
		t.Error("[section]/[subsection] placeholders not resolved from the outline")
	}
}

func TestOutlineWiring(t *testing.T) {
	cmd, _ := newCommand(t, `<html><body><h1>Book One</h1><p>text</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	data := runPDF(t, cmd)
	for _, want := range []string{
		"/Outlines", "/PageMode /UseOutlines",
		"/Type /Outlines", "/Count 1",
		"/Dest [", "/XYZ",
		"(Book One)",
	} {
		if !bytes.Contains(data, []byte(want)) {
			t.Errorf("outline output missing %q", want)
		}
	}
}

func TestOutlineDisabled(t *testing.T) {
	cmd, _ := newCommand(t, `<html><body><h1>Book</h1></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Outline = false
	data := runPDF(t, cmd)
	if bytes.Contains(data, []byte("/Outlines")) {
		t.Error("outline present although --outline is disabled")
	}
}

// tocCommand builds a command with a TOC object followed by two body objects,
// each a chapter on its own page.
func tocCommand(t *testing.T, out string) *cli.Command {
	t.Helper()
	cmd := newCommandMulti(t,
		[]string{
			`<html><body><h1>Chapter One</h1><p>text of the first chapter</p></body></html>`,
			`<html><body><h1>Chapter Two</h1><p>text of the second chapter</p></body></html>`,
		},
		out)
	toc := settings.PdfObject{IsTableOfContent: true, Load: settings.DefaultLoadPage()}
	cmd.Objects = append([]settings.PdfObject{toc}, cmd.Objects...)
	cmd.Global.TOC.ForwardLinks = true
	cmd.Global.UseCompression = false
	return cmd
}

func TestTOC(t *testing.T) {
	cmd := tocCommand(t, filepath.Join(t.TempDir(), "out.pdf"))
	data := runPDF(t, cmd)

	if n := pageCount(data); n < 2 {
		t.Fatalf("pages = %d, want >= 2 (TOC + body)", n)
	}
	// Layout splits text into per-word content ops, so assertions match the
	// word runs; the TOC entry text is drawn as a single inline run whose
	// dotted leader ("....") proves the entry (not just the caption) painted.
	for _, want := range []string{
		"(Table )", "(Contents)", // TOC caption
		"(Chapter )", "(One)", // TOC entry + body heading
		"(Two)",
		".... ) Tj", // dotted leader in the TOC entry
		"/Annots [", // forward-link annotations
		"/Dest [",
	} {
		if !bytes.Contains(data, []byte(want)) {
			t.Errorf("TOC output missing %q", want)
		}
	}
}

var destRe = regexp.MustCompile(`/Dest \[(\d+) 0 R /XYZ`)

func TestInternalLinkDest(t *testing.T) {
	cmd := tocCommand(t, filepath.Join(t.TempDir(), "out.pdf"))
	data := runPDF(t, cmd)
	// 2 outline items + 2 TOC forward links = 4 GoTo destinations.
	if got := len(destRe.FindAll(data, -1)); got != 4 {
		t.Errorf("/Dest /XYZ destinations = %d, want 4 (2 outline + 2 forward links)", got)
	}
}

func TestHTMLHeader(t *testing.T) {
	dir := t.TempDir()
	headerPath := filepath.Join(dir, "header.html")
	if err := os.WriteFile(headerPath, []byte(`<html><body><b>HEADERMARK</b> from file</body></html>`), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd, _ := newCommand(t, `<html><body><p>body</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Header.HTMLURL = headerPath
	cmd.Global.UseCompression = false
	data := runPDF(t, cmd)
	if !bytes.Contains(data, []byte("HEADERMARK")) {
		t.Error("HTML header text not present in output")
	}
}

func TestHTMLHeaderPlaceholderPerPage(t *testing.T) {
	dir := t.TempDir()
	headerPath := filepath.Join(dir, "header.html")
	if err := os.WriteFile(headerPath, []byte(`<html><body><p>page [page] of [topage]</p></body></html>`), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd, _ := newCommand(t, longBody(""), filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Header.HTMLURL = headerPath
	cmd.Global.UseCompression = false
	data := runPDF(t, cmd)
	pages := pageCount(data)
	if pages < 2 {
		t.Fatalf("pages = %d, want >= 2", pages)
	}
	// "page N of M" is word-split in the content stream: the literal words
	// only appear in the per-page-substituted header.
	if !bytes.Contains(data, []byte("(page )")) || !bytes.Contains(data, []byte("(of )")) {
		t.Error("per-page HTML header substitution [page]/[topage] missing")
	}
}

func TestHTMLHeaderRawMarkupRejected(t *testing.T) {
	cmd, _ := newCommand(t, `<html><body><p>x</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Header.HTMLURL = "<html><body>bad</body></html>"
	var log bytes.Buffer
	if err := RunPDF(cmd, &log); err != nil {
		t.Fatalf("RunPDF: %v", err)
	}
	if !bytes.Contains(log.Bytes(), []byte("warning")) {
		t.Errorf("expected a warning for markup-as-URL, log: %q", log.String())
	}
}

func TestAutoMargin(t *testing.T) {
	cmd, _ := newCommand(t, `<html><body><p>BODYTEXT</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Margin.Top = -1
	cmd.Global.Margin.Bottom = 10
	cmd.Global.Header.Left = "TOPHEADER"
	cmd.Global.UseCompression = false
	data := runPDF(t, cmd)
	if !bytes.Contains(data, []byte("TOPHEADER")) {
		t.Error("auto-margin header text missing")
	}
	if !bytes.Contains(data, []byte("BODYTEXT")) {
		t.Error("auto-margin body text missing")
	}
}

func TestExternalLinksDefaultOn(t *testing.T) {
	// External links are ON by default (DefaultPdfObject; the CLI's zero-value
	// objects read as on — see applyObjectDefaults). The annotation is
	// painted by the layout engine and serialized as /URI.
	body := `<html><body><p>see <a href="http://example.com/x">link</a></p></body></html>`
	cmd, _ := newCommand(t, body, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.UseCompression = false
	data := runPDF(t, cmd)
	if !bytes.Contains(data, []byte("/Annots [")) {
		t.Fatal("expected a link annotation")
	}
	if !bytes.Contains(data, []byte("/URI")) || !bytes.Contains(data, []byte("http://example.com/x")) {
		t.Error("expected an external URI annotation")
	}
}

func TestCoverNoHeaderFooter(t *testing.T) {
	// Cover pages must not carry headers/footers (wkhtmltopdf parity).
	cover := `<html><body><h1>COVER</h1></body></html>`
	body := `<html><body><p>BODY</p></body></html>`
	cmd := newCommandMulti(t, []string{cover, body}, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Objects[0].IsCover = true
	cmd.Global.Header.Left = "HDR"
	cmd.Global.Footer.Right = "FTR"
	cmd.Global.UseCompression = false
	data := runPDF(t, cmd)
	if n := pageCount(data); n != 2 {
		t.Fatalf("pages = %d, want 2", n)
	}
	// The header text must appear only once (the body page), not twice.
	if got := bytes.Count(data, []byte("(HDR) Tj")); got != 1 {
		t.Errorf("header drawn %d times, want 1 (cover page must be clean)", got)
	}
}

func TestFromPagePlaceholder(t *testing.T) {
	cmd, _ := newCommand(t, longBody(""), filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Header.Left = "f[frompage]"
	cmd.Global.UseCompression = false
	data := runPDF(t, cmd)
	if !bytes.Contains(data, []byte("(f1) Tj")) {
		t.Error("[frompage] placeholder not substituted on the first page")
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
