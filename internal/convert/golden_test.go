package convert

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/layout"
	"gowkhtmltopdf/internal/load"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/settings"
)

// goldenFixtures is the golden corpus: every fixture converts through
// RunPDF and must satisfy the structural PDF assertions in TestGoldenCorpus.
var goldenFixtures = []struct {
	name     string
	file     string
	minPages int
}{
	{"simple invoice", "fixture-01-simple-invoice.html", 1},
	{"table-heavy invoice", "fixture-02-table-heavy-invoice.html", 1},
	{"multi-page invoice", "fixture-03-multi-page-invoice.html", 2},
}

// goldenPath resolves a fixture (or fixture asset) inside the golden corpus.
func goldenPath(file string) string {
	return filepath.Join("..", "..", "testdata", "golden", file)
}

// goldenDir is the corpus directory itself.
func goldenDir() string {
	return filepath.Join("..", "..", "testdata", "golden")
}

// isHFCompanionHTML reports whether name is a nested HTML header/footer
// companion (fixture-NN-header.html / fixture-NN-footer.html), not a body
// golden fixture. Companions are copied into the temp dir but skipped by
// TestGoldenCorpusAllFixtures.
func isHFCompanionHTML(name string) bool {
	return strings.HasSuffix(name, "-header.html") || strings.HasSuffix(name, "-footer.html")
}

// fixtureIDPrefix returns "fixture-NN" from a body fixture file name
// (e.g. fixture-36-hf-nested-flex.html → fixture-36), or "" if the name
// does not match the corpus convention.
func fixtureIDPrefix(file string) string {
	base := strings.TrimSuffix(filepath.Base(file), ".html")
	parts := strings.SplitN(base, "-", 3)
	if len(parts) < 2 || parts[0] != "fixture" {
		return ""
	}
	for _, c := range parts[1] {
		if c < '0' || c > '9' {
			return ""
		}
	}
	return parts[0] + "-" + parts[1]
}

// attachHFCompanions sets Header.HTMLURL / Footer.HTMLURL when
// fixture-NN-header.html and/or fixture-NN-footer.html exist beside the
// body fixture in dir. Auto (-1) margins reserve the measured HF bands.
func attachHFCompanions(cmd *cli.Command, dir, file string) {
	prefix := fixtureIDPrefix(file)
	if prefix == "" || isHFCompanionHTML(file) {
		return
	}
	header := filepath.Join(dir, prefix+"-header.html")
	if _, err := os.Stat(header); err == nil {
		cmd.Global.Header.HTMLURL = header
		cmd.Global.Margin.Top = -1
	}
	footer := filepath.Join(dir, prefix+"-footer.html")
	if _, err := os.Stat(footer); err == nil {
		cmd.Global.Footer.HTMLURL = footer
		cmd.Global.Margin.Bottom = -1
	}
}

// commandForFixture builds a cli.Command that converts a golden fixture:
// A4 page, 10 mm margins, backgrounds on, local file access enabled so the
// fixture's relative links and images resolve (same ACL shape as newCommand).
// The whole corpus directory is copied next to the fixture (html, css, png),
// so relative references in the fixture keep working after the copy.
// If fixture-NN-header.html / fixture-NN-footer.html exist beside the body
// fixture, they are wired as nested HTML HF URLs (auto top/bottom margins).
func commandForFixture(t *testing.T, file string) *cli.Command {
	t.Helper()
	dir := t.TempDir()
	entries, err := os.ReadDir(goldenDir())
	if err != nil {
		t.Fatalf("read golden dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		content, err := os.ReadFile(goldenPath(e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(dir, e.Name()), content, 0o644); err != nil {
			t.Fatalf("write %s: %v", e.Name(), err)
		}
	}
	cmd := &cli.Command{
		Global: settings.DefaultPdfGlobal(),
		Objects: []settings.PdfObject{
			{Page: filepath.Join(dir, file), Load: settings.DefaultLoadPage()},
		},
		Output: filepath.Join(t.TempDir(), "out.pdf"),
	}
	// --enable-local-file-access: global flag on, object-level block off.
	cmd.Global.EnableLocalFileAccess = true
	cmd.Objects[0].Load.BlockLocalFileAccess = false
	cmd.Global.Size = settings.Size{PageSize: cmd.Global.PageSize}
	// A4, 10 mm margins, backgrounds on (already the defaults; set explicitly).
	cmd.Global.PageSize = "A4"
	cmd.Global.Margin = settings.DefaultMargins()
	cmd.Global.Background = true
	attachHFCompanions(cmd, dir, file)
	// Opt-in CJK/Hangul faces for fixture-27 (and any CSS that names them).
	fontDirs := []string{}
	if _, err := os.Stat("/usr/share/fonts/truetype/droid"); err == nil {
		fontDirs = append(fontDirs, "/usr/share/fonts/truetype/droid")
	}
	testFonts := filepath.Join("..", "..", "testdata", "fonts")
	if _, err := os.Stat(testFonts); err == nil {
		fontDirs = append(fontDirs, testFonts)
	}
	cmd.Global.FontPaths = fontDirs
	return cmd
}

// assertPDFStructure runs the pdf.ReadAll-style sanity checks on the raw
// output: header magic, %%EOF trailer, and an xref section reachable via the
// byte offset printed just before %%EOF.
func assertPDFStructure(t *testing.T, data []byte) {
	t.Helper()
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("output does not start with %%PDF-")
	}
	trimmed := bytes.TrimRight(data, "\r\n")
	if !bytes.HasSuffix(trimmed, []byte("%%EOF")) {
		t.Errorf("output does not end with %%EOF")
		return
	}
	lines := bytes.Split(trimmed, []byte("\n"))
	if len(lines) < 2 {
		t.Fatalf("trailer has %d lines, want at least 2", len(lines))
	}
	// The line before %%EOF is the decimal xref offset.
	offsetLine := strings.TrimSpace(string(lines[len(lines)-2]))
	off, err := strconv.ParseInt(offsetLine, 10, 64)
	if err != nil {
		t.Errorf("line before %%EOF is %q, want a decimal xref offset", offsetLine)
		return
	}
	if off < 0 || off >= int64(len(data)) {
		t.Fatalf("xref offset %d out of range (output length %d)", off, len(data))
	}
	if !bytes.HasPrefix(data[off:], []byte("xref")) {
		t.Errorf("bytes at xref offset %d start with %q, want \"xref\"", off, data[off:off+4])
	}
}

func TestGoldenCorpus(t *testing.T) {
	for _, tc := range goldenFixtures {
		t.Run(tc.name, func(t *testing.T) {
			cmd := commandForFixture(t, tc.file)
			data := runPDF(t, cmd)

			if n := pageCount(data); n < tc.minPages {
				t.Errorf("pages = %d, want >= %d", n, tc.minPages)
			}
			if !bytes.Contains(data, []byte("/FontFile2")) {
				t.Error("expected embedded subset font (/FontFile2)")
			}
			// No fixture references an image today (checked fixture-01/02/03:
			// zero <img> elements), so there is no /Subtype /Image assertion.
			assertPDFStructure(t, data)
		})
	}
}

// fixtureBounds is the per-fixture page envelope and feature expectation
// for TestGoldenCorpusAllFixtures. maxPages 0 means "no upper bound".
type fixtureBounds struct {
	minPages int
	maxPages int
	images   bool // expect >= 1 embedded /Subtype /Image xobject
	uris     bool // expect >= 1 URI link annotation (/S /URI)
}

// fixturePageBounds maps a fixture file name to its expected page envelope.
// Bounds are recorded from the phase-9.1 measurement pass and pin the
// pagination behaviour across releases: a change to wrapping, table layout
// or page-break handling that moves a fixture out of its envelope fails.
var fixturePageBounds = map[string]fixtureBounds{
	"fixture-01-simple-invoice.html":        {minPages: 1, maxPages: 1},
	"fixture-02-table-heavy-invoice.html":   {minPages: 1, maxPages: 2},
	"fixture-03-multi-page-invoice.html":    {minPages: 2, maxPages: 0},
	"fixture-04-two-column-layout.html":     {minPages: 1, maxPages: 1},
	"fixture-05-linked-stylesheet.html":     {minPages: 1, maxPages: 1},
	"fixture-06-external-link.html":         {minPages: 1, maxPages: 1, uris: true},
	"fixture-07-image-logo.html":            {minPages: 1, maxPages: 1, images: true},
	"fixture-08-forced-page-breaks.html":    {minPages: 5, maxPages: 5},
	"fixture-09-multi-section-doc.html":     {minPages: 2, maxPages: 0},
	"fixture-10-table-colspan.html":         {minPages: 1, maxPages: 1},
	"fixture-11-long-text-wrap.html":        {minPages: 3, maxPages: 0},
	"fixture-12-lists.html":                 {minPages: 1, maxPages: 1},
	"fixture-13-pre-code-block.html":        {minPages: 1, maxPages: 1},
	"fixture-14-colorful-report.html":       {minPages: 1, maxPages: 1},
	"fixture-15-bulleted-requirements.html": {minPages: 1, maxPages: 2},
	"fixture-16-invoice-with-css.html":      {minPages: 1, maxPages: 2},
	"fixture-17-cover-and-content.html":     {minPages: 2, maxPages: 2},
	"fixture-18-typography.html":            {minPages: 1, maxPages: 1},
	"fixture-19-margin-and-sizing.html":     {minPages: 1, maxPages: 1},
	"fixture-20-image-grid.html":            {minPages: 1, maxPages: 1, images: true},
	"fixture-21-detailed-report.html":       {minPages: 3, maxPages: 0},
	"fixture-22-float-invoice-chrome.html":  {minPages: 1, maxPages: 1},
	"fixture-23-thead-repeat.html":          {minPages: 2, maxPages: 0},
	"fixture-24-internal-anchors.html":      {minPages: 2, maxPages: 2},
	"fixture-25-flex-row.html":              {minPages: 1, maxPages: 1},
	"fixture-26-position-lite.html":         {minPages: 1, maxPages: 1},
	"fixture-27-cjk-fontpath.html":          {minPages: 1, maxPages: 1},
	"fixture-28-flex-wrap-grid-fixed.html":  {minPages: 2, maxPages: 2},
	"fixture-29-float-beside-table.html":    {minPages: 1, maxPages: 1},
	"fixture-30-orphans-heuristic.html":     {minPages: 2, maxPages: 0},
	"fixture-31-sticky-top.html":            {minPages: 2, maxPages: 0},
	"fixture-32-flex-grid-full.html":        {minPages: 1, maxPages: 1},
	"fixture-33-flex-cyclic-basis.html":     {minPages: 1, maxPages: 1},
	"fixture-34-grid-areas-dense.html":      {minPages: 1, maxPages: 1},
	"fixture-35-grid-minmax-intrinsic.html": {minPages: 1, maxPages: 1},
	"fixture-36-hf-nested-flex.html":        {minPages: 1, maxPages: 1, images: true},
	"fixture-37-orphans-css.html":           {minPages: 2, maxPages: 0},
	"fixture-38-float-inside-td.html":       {minPages: 1, maxPages: 1},
	"fixture-39-multicol-article.html":      {minPages: 2, maxPages: 0},
	"fixture-40-transform-badge.html":       {minPages: 1, maxPages: 1},
	"fixture-41-has-selector.html":          {minPages: 1, maxPages: 1},
	"fixture-42-container-inline-size.html": {minPages: 1, maxPages: 1},
}

// fixtureHeaderOK enforces the corpus hygiene rule: every fixture starts
// with a DOCTYPE and its opening comment header names the fixture.
func fixtureHeaderOK(t *testing.T, file string, data []byte) {
	t.Helper()
	name := strings.TrimSuffix(file, ".html")
	lines := strings.Split(string(data), "\n")
	if len(lines) > 6 {
		lines = lines[:6]
	}
	head := strings.Join(lines, "\n")
	if !strings.HasPrefix(head, "<!DOCTYPE html>") {
		t.Errorf("fixture %s must start with a DOCTYPE", file)
	}
	if !strings.Contains(head, "<!--") || !strings.Contains(head, name) {
		t.Errorf("fixture %s: opening comment header must name the fixture (found %q)", file, head)
	}
}

// TestGoldenCorpusAllFixtures walks every *.html fixture in
// testdata/golden, converts each through the same pipeline as
// TestGoldenCorpus, and asserts the structural PDF invariants (%PDF-,
// /FontFile2, xref offset, %%EOF), the per-fixture page envelope from
// fixturePageBounds, and the feature expectations (embedded images, URI
// annotations). This is the test the `make golden` target runs.
func TestGoldenCorpusAllFixtures(t *testing.T) {
	entries, err := os.ReadDir(goldenDir())
	if err != nil {
		t.Fatalf("read golden dir: %v", err)
	}
	fixtureCount := 0
	for _, e := range entries {
		file := e.Name()
		if e.IsDir() || !strings.HasSuffix(file, ".html") || isHFCompanionHTML(file) {
			continue
		}
		fixtureCount++
		t.Run(file, func(t *testing.T) {
			content, err := os.ReadFile(goldenPath(file))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			fixtureHeaderOK(t, file, content)

			cmd := commandForFixture(t, file)
			data := runPDF(t, cmd)

			n := pageCount(data)
			b := fixturePageBounds[file]
			if n < b.minPages || (b.maxPages > 0 && n > b.maxPages) {
				t.Errorf("pages = %d, want [%d, %d]", n, b.minPages, b.maxPages)
			}
			if !bytes.Contains(data, []byte("/FontFile2")) {
				t.Error("expected embedded subset font (/FontFile2)")
			}
			if b.images && !bytes.Contains(data, []byte("/Subtype /Image")) {
				t.Error("expected an embedded image xobject (/Subtype /Image)")
			}
			if b.uris && !bytes.Contains(data, []byte("/S /URI")) {
				t.Error("expected a URI link annotation (/S /URI)")
			}
			if file == "fixture-27-cjk-fontpath.html" && bytes.Contains(data, []byte("NotoSansKR")) {
				// Hangul subset on testdata/fonts — prove Type0 path for KR glyphs.
				if !bytes.Contains(data, []byte("/Subtype /Type0")) {
					t.Error("fixture-27: expected Type0 font when NotoSansKR is embedded")
				}
			}
			assertPDFStructure(t, data)
		})
	}
	if fixtureCount < 20 {
		t.Errorf("golden corpus has %d html fixtures, want >= 20", fixtureCount)
	}
}

// TestGoldenFixture03Performance records the layout+paint cost of the
// largest golden fixture (plan item 4.8). It reuses the same load → parse →
// style → layout → paint path as RunPDF but times only Layout and Paint.
func TestGoldenFixture03Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("perf budget test skipped in -short mode")
	}
	cmd := commandForFixture(t, "fixture-03-multi-page-invoice.html")

	ctx := context.Background()
	loader := load.NewLoader(cmd.Global.Load)
	loader.EnableLocalFileAccess = cmd.Global.EnableLocalFileAccess
	res, err := loader.Load(ctx, cmd.Objects[0].Page, cmd.Objects[0].Load)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	root, err := html.Parse(string(res.Body))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	sheets := collectStyleSheets(root)
	font, err := pdf.DefaultFont()
	if err != nil {
		t.Fatalf("default font: %v", err)
	}

	// Page geometry: A4 minus 10 mm margins on every side (same as convert).
	pageW, pageH, err := settings.ParsePageSize("A4")
	if err != nil {
		t.Fatalf("page size: %v", err)
	}
	const mm = 72.0 / 25.4
	m := settings.DefaultMargins()
	contentW := pageW - (m.Left+m.Right)*mm
	contentH := pageH - (m.Top+m.Bottom)*mm

	layoutStart := time.Now()
	lres, err := layout.Layout(root, layout.Options{
		Width:      contentW,
		Height:     contentH,
		Font:       font,
		Sheets:     sheets,
		Media:      "print",
		Background: true,
	})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	layoutDur := time.Since(layoutStart)

	paintStart := time.Now()
	doc := pdf.NewDocument()
	if err := layout.Paint(doc, lres, layout.PaintOptions{
		PageWidth:    pageW,
		PageHeight:   pageH,
		MarginTop:    m.Top * mm,
		MarginBottom: m.Bottom * mm,
		MarginLeft:   m.Left * mm,
		MarginRight:  m.Right * mm,
	}); err != nil {
		t.Fatalf("paint: %v", err)
	}
	paintDur := time.Since(paintStart)

	total := layoutDur + paintDur
	t.Logf("fixture-03 layout+paint: layout=%v paint=%v total=%v", layoutDur, paintDur, total)
	if total >= 2*time.Second {
		t.Errorf("layout+paint took %v, want < 2s", total)
	}
}

// collectStyleSheets extracts inline <style> blocks from the fixture's head.
func collectStyleSheets(root *html.Node) []*css.Stylesheet {
	var sheets []*css.Stylesheet
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		if n.Name == "style" {
			var sb strings.Builder
			for _, c := range n.Children {
				if c.Type == html.TextNode {
					sb.WriteString(c.Text)
				}
			}
			if s, err := css.Parse(sb.String()); err == nil && s != nil {
				sheets = append(sheets, s)
			}
			return
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	return sheets
}
