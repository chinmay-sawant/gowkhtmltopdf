package convert //nolint:testpackage // white-box tests need unexported access

import (
	"bytes"
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
var goldenFixtures = []struct { //nolint:gochecknoglobals // immutable test corpus
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

	if err := copyGoldenTree(goldenDir(), dir); err != nil {
		t.Fatalf("copy golden directory: %v", err)
	}

	obj := settings.DefaultPdfObject()
	obj.Page = filepath.Join(dir, file)
	obj.Load.BlockLocalFileAccess = false
	cmd := &cli.Command{ //nolint:exhaustruct // intentional zero-value fields
		Global:  settings.DefaultPdfGlobal(),
		Objects: []settings.PdfObject{obj},
		Output:  filepath.Join(t.TempDir(), "out.pdf"),
	}
	// --enable-local-file-access: global flag on, object-level block off.
	cmd.Global.Load.EnableLocalFileAccess = true
	cmd.Global.Size = settings.Size{PageSize: cmd.Global.PageSize} //nolint:exhaustruct // intentional zero-value fields
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

// copyGoldenTree copies the fixture corpus into the isolated conversion
// directory while preserving nested stylesheet, font, and image assets.
func copyGoldenTree(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		sourcePath := filepath.Join(src, entry.Name())
		destinationPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(destinationPath, 0o700); err != nil {
				return err
			}
			if err := copyGoldenTree(sourcePath, destinationPath); err != nil {
				return err
			}
			continue
		}

		content, err := os.ReadFile(sourcePath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(destinationPath, content, 0o600); err != nil {
			return err
		}
	}

	return nil
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
	t.Parallel()

	for _, testCase := range goldenFixtures {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cmd := commandForFixture(t, testCase.file)
			data := runPDF(t, cmd)

			if n := pageCount(data); n < testCase.minPages {
				t.Errorf("pages = %d, want >= %d", n, testCase.minPages)
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

// pagination behaviour across releases: a change to wrapping, table layout.
var fixturePageBounds = map[string]fixtureBounds{ //nolint:gochecknoglobals // immutable test corpus
	"fixture-01-simple-invoice.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-02-table-heavy-invoice.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 2,
	},
	"fixture-03-multi-page-invoice.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 2, maxPages: 0,
	},
	"fixture-04-two-column-layout.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-05-linked-stylesheet.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-06-external-link.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1, uris: true,
	},
	"fixture-07-image-logo.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1, images: true,
	},
	"fixture-08-forced-page-breaks.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 5, maxPages: 5,
	},
	"fixture-09-multi-section-doc.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 2, maxPages: 0,
	},
	"fixture-10-table-colspan.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-11-long-text-wrap.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 3, maxPages: 0,
	},
	"fixture-12-lists.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-13-pre-code-block.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-14-colorful-report.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-15-bulleted-requirements.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 2,
	},
	"fixture-16-invoice-with-css.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 2,
	},
	"fixture-17-cover-and-content.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 2, maxPages: 2,
	},
	"fixture-18-typography.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-19-margin-and-sizing.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-20-image-grid.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1, images: true,
	},
	"fixture-21-detailed-report.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 3, maxPages: 0,
	},
	"fixture-22-float-invoice-chrome.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-23-thead-repeat.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 2, maxPages: 0,
	},
	"fixture-24-internal-anchors.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 2, maxPages: 2,
	},
	"fixture-25-flex-row.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-26-position-lite.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-27-cjk-fontpath.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-28-flex-wrap-grid-fixed.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 2, maxPages: 2,
	},
	"fixture-29-float-beside-table.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-30-orphans-heuristic.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 2, maxPages: 0,
	},
	"fixture-31-sticky-top.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 2, maxPages: 0,
	},
	"fixture-32-flex-grid-full.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-33-flex-cyclic-basis.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-34-grid-areas-dense.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-35-grid-minmax-intrinsic.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-36-hf-nested-flex.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1, images: true,
	},
	"fixture-37-orphans-css.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 2, maxPages: 0,
	},
	"fixture-38-float-inside-td.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-39-multicol-article.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 2, maxPages: 0,
	},
	"fixture-40-transform-badge.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-41-has-selector.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-42-container-inline-size.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-43-complex-dossier.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 5, maxPages: 5, images: true,
	},
	"fixture-44-receipt.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-45-purchase-order.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-46-contract.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-47-certificate.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1, images: true,
	},
	"fixture-48-shipping-document.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-49-night-train-poster.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1, images: true,
	},
	"fixture-50-letter-template.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1, images: true,
	},
	"fixture-51-asteria-storybook.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 4, maxPages: 4, images: true,
	},
	"fixture-52-airline-boarding-pass.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1,
	},
	"fixture-53-asteria-observatory-poster.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1, images: true,
	},
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
func TestGoldenCorpusAllFixtures(t *testing.T) { //nolint:gocognit,cyclop,funlen // per-fixture structural assertions
	t.Parallel()

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
			t.Parallel()

			content, err := os.ReadFile(goldenPath(file))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}

			fixtureHeaderOK(t, file, content)

			cmd := commandForFixture(t, file)
			data := runPDF(t, cmd)
			n := pageCount(data)
			buf := fixturePageBounds[file]

			if n < buf.minPages || (buf.maxPages > 0 && n > buf.maxPages) {
				t.Errorf("pages = %d, want [%d, %d]", n, buf.minPages, buf.maxPages)
			}

			if !bytes.Contains(data, []byte("/FontFile2")) {
				t.Error("expected embedded subset font (/FontFile2)")
			}

			if buf.images && !bytes.Contains(data, []byte("/Subtype /Image")) {
				t.Error("expected an embedded image xobject (/Subtype /Image)")
			}

			if buf.uris && !bytes.Contains(data, []byte("/S /URI")) {
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
func TestGoldenFixture03Performance(t *testing.T) { //nolint:funlen // perf harness with many setup steps
	t.Parallel()

	if testing.Short() {
		t.Skip("perf budget test skipped in -short mode")
	}

	cmd := commandForFixture(t, "fixture-03-multi-page-invoice.html")

	ctx := t.Context()
	loader := load.NewLoader(cmd.Global.Load)

	res, err := loader.Load(ctx, cmd.Objects[0].Page, cmd.Objects[0].Load)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}

	root, err := html.ParseDocument(res.Body)
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

	const mmPt = 72.0 / 25.4

	mVal := settings.DefaultMargins()
	contentW := pageW - (mVal.Left+mVal.Right)*mmPt
	contentH := pageH - (mVal.Top+mVal.Bottom)*mmPt

	layoutStart := time.Now()

	lres, err := layout.Layout(root, layout.Options{ //nolint:exhaustruct // intentional zero-value fields
		Width:      contentW,
		Height:     contentH,
		Font:       font,
		Sheets:     sheets,
		Media:      mediaPrint,
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
		MarginTop:    mVal.Top * mmPt,
		MarginBottom: mVal.Bottom * mmPt,
		MarginLeft:   mVal.Left * mmPt,
		MarginRight:  mVal.Right * mmPt,
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
	walk = func(num *html.Node) {
		if num.Type != html.ElementNode {
			return
		}

		if num.Name == "style" {
			var strB strings.Builder

			for _, c := range num.Children {
				if c.Type == html.TextNode {
					strB.WriteString(c.Text)
				}
			}

			if s, err := css.Parse(strB.String()); err == nil && s != nil {
				sheets = append(sheets, s)
			}

			return
		}

		for _, c := range num.Children {
			walk(c)
		}
	}
	walk(root)

	return sheets
}
