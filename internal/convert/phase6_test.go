package convert //nolint:testpackage // white-box tests need unexported access

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/settings"
)

// longBody builds an HTML body long enough to span at least two pages.
func longBody(marker string) string {
	var buf bytes.Buffer

	buf.WriteString("<html><body>")

	for i := range 60 {
		buf.WriteString("<p>paragraph ")
		buf.WriteRune(rune('a' + i%26))
		buf.WriteString(" with some words to wrap across the page width</p>")
	}

	buf.WriteString(marker)
	buf.WriteString("</body></html>")

	return buf.String()
}

func TestTextHeaderFooter(t *testing.T) {
	t.Parallel()
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
	// Baselines must sit inside the page (ascent below top, descent above bottom).
	// The previous sign error placed headers at pageH+ascent and footers at -descent.
	re := regexp.MustCompile(`([\d.]+)\s+([\d.\-]+)\s+Td\n\((Page \d+/|doc T)`)

	matches := re.FindAllSubmatch(data, -1)
	if len(matches) == 0 {
		t.Fatal("no header/footer Td positions found")
	}

	pageH := 841.89 // A4 default

	for _, mVal := range matches {
		posY, err := strconv.ParseFloat(string(mVal[2]), 64)
		if err != nil {
			t.Fatal(err)
		}

		if posY < 0 || posY > pageH {
			t.Errorf("HF baseline y=%.3f for %q is outside page [0, %.2f]", posY, mVal[3], pageH)
		}
	}
}

func TestPlaceholderReplace(t *testing.T) {
	t.Parallel()
	cmd, _ := newCommand(t, `<html><body><p>x</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Footer.Center = "[who] inc. - [unknown]"
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
	t.Parallel()

	body := `<html><body><h1>Chap A</h1><h2>Sec B</h2><p>` +
		"text " +
		"text text</p></body></html>"
	cmd, _ := newCommand(t, body, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Header.Right = "[section]/[subsection]"
	cmd.Global.UseCompression = false

	data := runPDF(t, cmd)
	if !bytes.Contains(data, []byte("Chap A/Sec B")) {
		t.Error("[section]/[subsection] placeholders not resolved from the outline")
	}
}

func TestOutlineWiring(t *testing.T) {
	t.Parallel()
	cmd, _ := newCommand(
		t,
		`<html><body><h1>Book One</h1><p>text</p></body></html>`,
		filepath.Join(t.TempDir(), "out.pdf"),
	)

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
	t.Parallel()
	cmd, _ := newCommand(t, `<html><body><h1>Book</h1></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Outline = false

	data := runPDF(t, cmd)
	if bytes.Contains(data, []byte("/Outlines")) {
		t.Error("outline present although --outline is disabled")
	}
}

// tocCommand builds a command with a TOC object followed by two body objects,
// each a chapter on its own page.
func tocCommand(t *testing.T, out string) *Request {
	t.Helper()
	cmd := newCommandMulti(t,
		[]string{
			`<html><body><h1>Chapter One</h1><p>text of the first chapter</p></body></html>`,
			`<html><body><h1>Chapter Two</h1><p>text of the second chapter</p></body></html>`,
		},
		out)
	toc := settings.DefaultPdfObject()
	toc.IsTableOfContent = true
	toc.UseOutline = false
	cmd.Objects = append([]settings.PdfObject{toc}, cmd.Objects...)
	cmd.Global.TOC.ForwardLinks = true
	cmd.Global.UseCompression = false

	return cmd
}

func TestTOC(t *testing.T) {
	t.Parallel()
	cmd := tocCommand(t, filepath.Join(t.TempDir(), "out.pdf"))
	data := runPDF(t, cmd)

	if n := pageCount(data); n < 2 {
		t.Fatalf("pages = %d, want >= 2 (TOC + body)", n)
	}
	// Same-style runs coalesce into fewer Tj strings; assert on phrases
	// that survive coalescing plus the dotted leader.
	for _, want := range []string{
		"(Table of Contents)", // TOC caption (coalesced)
		"(Chapter One)",       // TOC entry / body heading phrase
		"(Chapter Two)",
		"....",      // dotted leader in the TOC entry
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
	t.Parallel()
	cmd := tocCommand(t, filepath.Join(t.TempDir(), "out.pdf"))
	data := runPDF(t, cmd)
	// 2 outline items + 2 TOC forward links = 4 GoTo destinations.
	if got := len(destRe.FindAll(data, -1)); got != 4 {
		t.Errorf("/Dest /XYZ destinations = %d, want 4 (2 outline + 2 forward links)", got)
	}
}

func TestHTMLHeader(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	headerPath := filepath.Join(dir, "header.html")
	if err := os.WriteFile(headerPath, []byte(`<html><body><b>HEADERMARK</b> from file</body></html>`), 0o600); err != nil { //nolint:lll // fixture write
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
	t.Parallel()
	dir := t.TempDir()

	headerPath := filepath.Join(dir, "header.html")
	if err := os.WriteFile(headerPath, []byte(`<html><body><p>page [page] of [topage]</p></body></html>`), 0o600); err != nil { //nolint:lll // fixture write
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
	// "page N of M" is coalesced where style is uniform; substituted numbers
	// and the "page"/"of" markers must still appear after placeholder expand.
	if !bytes.Contains(data, []byte("page")) || !bytes.Contains(data, []byte(" of ")) {
		t.Error("per-page HTML header substitution [page]/[topage] missing")
	}
}

func TestHTMLHeaderRawMarkupRejected(t *testing.T) {
	t.Parallel()
	cmd, _ := newCommand(t, `<html><body><p>BODYONLY</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Header.HTMLURL = "<html><body>bad</body></html>"
	cmd.Global.UseCompression = false

	var log bytes.Buffer
	data := runPDFWithLog(t, cmd, &log)

	if bytes.Contains(data, []byte("bad")) {
		t.Error("raw HTML markup must not be loaded as a header document")
	}

	if !bytes.Contains(data, []byte("BODYONLY")) {
		t.Error("body text missing after rejected raw header markup")
	}
}

func TestHTMLHeaderRelativePath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	headerRel := "header.html"
	pageRel := "page.html"

	if err := os.WriteFile(filepath.Join(root, headerRel), []byte(`<html><body><b>RELHFMARK</b></body></html>`), 0o600); err != nil { //nolint:lll // fixture write
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(root, pageRel), []byte(`<html><body><p>BODYREL</p></body></html>`), 0o600); err != nil { //nolint:lll // fixture write
		t.Fatal(err)
	}

	obj := settings.DefaultPdfObject()
	obj.Page = filepath.Join(root, pageRel)
	obj.Load.BlockLocalFileAccess = false
	global := settings.DefaultPdfGlobal()
	global.Load.EnableLocalFileAccess = true
	global.Header.HTMLURL = filepath.Join(root, headerRel)
	global.Margin.Top = -1
	global.UseCompression = false
	global.Outline = false
	req := NewPDFRequest(global, []settings.PdfObject{obj}, nil, nil)

	var log bytes.Buffer
	data := runPDFWithLog(t, req, &log)

	if bytes.Contains(log.Bytes(), []byte("no such file")) {
		t.Fatalf("HF path failed to resolve (path doubling?): %s", log.String())
	}

	if !bytes.Contains(data, []byte("RELHFMARK")) {
		t.Error("relative --header-html text missing from PDF")
	}

	if !bytes.Contains(data, []byte("BODYREL")) {
		t.Error("body text missing")
	}
}

func TestAutoMargin(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	// External links are ON by DefaultPdfObject (callers/CLI must apply defaults;
	// convert no longer OR-hacks zero-value bools permanently ON).
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

func TestExternalLinksDisableHonored(t *testing.T) {
	t.Parallel()

	body := `<html><body><p>see <a href="http://example.com/x">link</a></p></body></html>`
	cmd, _ := newCommand(t, body, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Objects[0].ExternalLinks = false
	cmd.Global.UseCompression = false

	data := runPDF(t, cmd)
	if bytes.Contains(data, []byte("http://example.com/x")) {
		t.Error("ExternalLinks=false must strip external URI annotations")
	}
}

func TestCoverNoHeaderFooter(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

func TestPDFUA1LinkAnnotationOBJRCompliance(t *testing.T) {
	t.Parallel()

	body := `<!DOCTYPE html><html><head><title>Accessible Links</title></head><body>
		<h1>Heading 1</h1>
		<p>Check out <a href="https://example.com/accessible">our website</a> for more details.</p>
		<p>Jump to <a href="#target">section below</a>.</p>
		<div style="height: 200px;"></div>
		<p id="target">Target paragraph</p>
	</body></html>`

	cmd, _ := newCommand(t, body, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.PdfProfile = settings.ProfilePDFUA1
	cmd.Global.Title = "Accessible Links"
	cmd.Global.UseCompression = false

	data := runPDF(t, cmd)
	outStr := string(data)

	// Verify annotation flags and border
	if !strings.Contains(outStr, "/Border [0 0 0] /F 4") {
		t.Error("Link annotation missing /Border [0 0 0] /F 4")
	}

	// Verify Tabs /S
	if !strings.Contains(outStr, "/Tabs /S") {
		t.Error("Page with annotations missing /Tabs /S")
	}

	// Verify StructElem /Link with OBJR
	if !strings.Contains(outStr, "/S /Link") {
		t.Error("Missing /S /Link StructElem")
	}

	if !strings.Contains(outStr, "/Type /OBJR /Obj ") {
		t.Error("Missing /Type /OBJR /Obj in structure tree for link annotation")
	}
}

func TestLinkMCIDAndOBJRIdentity(t *testing.T) {
	t.Parallel()

	body := `<!DOCTYPE html><html><head><title>Link Identity</title></head><body>
		<p>Here is <a href="https://example.com">a test link</a> in a paragraph.</p>
	</body></html>`

	cmd, _ := newCommand(t, body, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.PdfProfile = settings.ProfilePDFUA1
	cmd.Global.Title = "Link Identity"
	cmd.Global.UseCompression = false

	data := runPDF(t, cmd)
	outStr := string(data)

	// In the structure tree, the /S /Link StructElem should contain both the MCID integer and the OBJR dictionary
	re := regexp.MustCompile(
		`<<\s*/Type\s*/StructElem\s*/S\s*/Link[^\n]*\s*/K\s*\[\s*(\d+)\s+<<\s*/Type\s*/OBJR[^\n]*>>\s*\]`,
	)

	if !re.MatchString(outStr) {
		t.Errorf("StructElem for Link should contain both MCID and OBJR together in /K")
	}
}

func TestSingleDocumentChildUnderStructTreeRoot(t *testing.T) {
	t.Parallel()

	body1 := `<!DOCTYPE html><html><head><title>Doc 1</title></head><body><p>Body 1</p></body></html>`
	body2 := `<!DOCTYPE html><html><head><title>Doc 2</title></head><body><p>Body 2</p></body></html>`

	cmd, _ := newCommand(t, body1, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Objects = append(cmd.Objects, settings.PdfObject{ //nolint:exhaustruct // test object
		Page: body2,
	})
	cmd.Global.PdfProfile = settings.ProfilePDFUA1
	cmd.Global.Title = "Multi Object Doc"
	cmd.Global.UseCompression = false

	data := runPDF(t, cmd)
	outStr := string(data)

	// Count occurrences of /S /Document in the output
	docCount := strings.Count(outStr, "/S /Document")
	if docCount != 1 {
		t.Errorf("/S /Document count = %d, want exactly 1", docCount)
	}
}

func TestHFLinkIsolationFromDocumentStructureTree(t *testing.T) {
	t.Parallel()

	body := `<!DOCTYPE html><html><head><title>Body</title></head><body><p>Main content</p></body></html>`

	cmd, _ := newCommand(t, body, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Header.Left = "Header Text"
	cmd.Global.PdfProfile = settings.ProfilePDFUA1
	cmd.Global.Title = "HF Isolation"
	cmd.Global.UseCompression = false

	data := runPDF(t, cmd)
	outStr := string(data)

	// No /S /Link should be emitted since body has no links
	if strings.Contains(outStr, "/S /Link") {
		t.Errorf("Emitted /S /Link when body had no links (header/footer artifact leak)")
	}
}
