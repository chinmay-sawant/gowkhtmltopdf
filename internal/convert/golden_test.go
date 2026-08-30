//nolint:all
package convert //nolint:testpackage // white-box tests need unexported access

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
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
func attachHFCompanions(req *Request, dir, file string) {
	prefix := fixtureIDPrefix(file)
	if prefix == "" || isHFCompanionHTML(file) {
		return
	}

	header := filepath.Join(dir, prefix+"-header.html")
	if _, err := os.Stat(header); err == nil {
		req.Global.Header.HTMLURL = header
		req.Global.Margin.Top = -1
	}

	footer := filepath.Join(dir, prefix+"-footer.html")
	if _, err := os.Stat(footer); err == nil {
		req.Global.Footer.HTMLURL = footer
		req.Global.Margin.Bottom = -1
	}
}

// requestForFixture builds a convert.Request that converts a golden fixture:
// A4 page, 10 mm margins, backgrounds on, local file access enabled so the
// fixture's relative links and images resolve.
func requestForFixture(t *testing.T, file string) *Request {
	t.Helper()
	dir := t.TempDir()

	if err := copyGoldenTree(goldenDir(), dir); err != nil {
		t.Fatalf("copy golden directory: %v", err)
	}

	obj := settings.DefaultPdfObject()
	obj.Page = filepath.Join(dir, file)
	obj.Load.BlockLocalFileAccess = false
	global := settings.DefaultPdfGlobal()
	global.Load.EnableLocalFileAccess = true
	global.PageSize = "A4"
	global.Margin = settings.DefaultMargins()
	global.Background = true

	fontDirs := []string{}
	if _, err := os.Stat("/usr/share/fonts/truetype/droid"); err == nil {
		fontDirs = append(fontDirs, "/usr/share/fonts/truetype/droid")
	}

	testFonts := filepath.Join("..", "..", "testdata", "fonts")
	if _, err := os.Stat(testFonts); err == nil {
		fontDirs = append(fontDirs, testFonts)
	}

	global.FontPaths = fontDirs
	req := NewPDFRequest(global, []settings.PdfObject{obj}, nil, nil)
	attachHFCompanions(req, dir, file)

	return req
}

// copyGoldenTree copies the fixture corpus into the isolated conversion
// directory while preserving nested stylesheet, font, and image assets.
func copyGoldenTree(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read golden dir %s: %w", src, err)
	}

	for _, entry := range entries {
		sourcePath := filepath.Join(src, entry.Name())
		destinationPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := os.MkdirAll(destinationPath, 0o700); err != nil {
				return fmt.Errorf("mkdir %s: %w", destinationPath, err)
			}

			if err := copyGoldenTree(sourcePath, destinationPath); err != nil {
				return err
			}

			continue
		}

		content, err := os.ReadFile(sourcePath)
		if err != nil {
			return fmt.Errorf("read %s: %w", sourcePath, err)
		}

		if err := os.WriteFile(destinationPath, content, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", destinationPath, err)
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
			data := runPDF(t, requestForFixture(t, testCase.file))

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
	images   bool     // expect >= 1 embedded /Subtype /Image xobject
	uris     bool     // expect >= 1 URI link annotation (/S /URI)
	needles  []string // ordered extracted-text needles
}

// pagination behaviour across releases: a change to wrapping, table layout.
var fixturePageBounds = map[string]fixtureBounds{ //nolint:gochecknoglobals // immutable test corpus
	"fixture-01-simple-invoice.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1, needles: []string{"Invoice", "234.40"},
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
		minPages: 1, maxPages: 1, uris: true, needles: []string{"Partner Handbook"},
	},
	"fixture-07-image-logo.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 1, maxPages: 1, images: true, needles: []string{"Nordwind"},
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
		minPages: 2, maxPages: 2, needles: []string{"Internal link report", "Appendix"},
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
	"fixture-54-ember-harbor-storybook.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 4, maxPages: 4, images: true, needles: []string{"Ember Harbor"},
	},
	"fixture-55-lantern-cooperative-report.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 3, maxPages: 3, needles: []string{"NORTHLINE"},
	},
	"font-examples.html": { //nolint:exhaustruct // fallback Liberation; page count varies with wrap
		minPages: 1, maxPages: 30,
	},
	"complex-css.html": { //nolint:exhaustruct // catalog stress fixture
		minPages: 1, maxPages: 40, needles: []string{"Alexandria"},
	},
	"architecture-diagram.html": { //nolint:exhaustruct // corpus fixture; not written by testdata/golden/api
		minPages: 1, maxPages: 12, needles: []string{"Architecture"},
	},
	"fixture-56-architecture-diagram.html": { //nolint:exhaustruct // intentional zero-value fields
		// gap and logical margin/padding now apply, but dom-foot orphans are
		// excluded so the footer stays with its section; page count is 20.
		minPages: 20, maxPages: 20,
	},
	"fixture-57-vanguard-telemetry-audit.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 9, maxPages: 9,
		needles: []string{"VANGUARD", "TELEMETRY", "VANGUARD-CSS-356-IMPLEMENTED"},
	},
	"fixture-58-unsupported-worklist-audit.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 9, maxPages: 9,
		needles: []string{"UNSUPPORTED-WORKLIST-AUDIT", "VANGUARD-CSS-UNSUPPORTED-SAFE"},
	},
	"fixture-59-apex-digital-landing.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 7, maxPages: 9, images: true,
		needles: []string{"Core Solutions", "Recent Work", "Transparent Pricing"},
	},
	"fixture-60-implemented-props-a.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 7, maxPages: 9, images: true,
		needles: []string{"IMPLEMENTED-PROPS-A"},
	},
	"fixture-61-implemented-props-b.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 7, maxPages: 9, images: true,
		needles: []string{"IMPLEMENTED-PROPS-B"},
	},
	"fixture-62-implemented-props-c.html": { //nolint:exhaustruct // intentional zero-value fields
		minPages: 7, maxPages: 9, images: true,
		needles: []string{"IMPLEMENTED-PROPS-C"},
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

			data := runPDF(t, requestForFixture(t, file))
			pages := pageCount(data)
			buf, ok := fixturePageBounds[file]

			if !ok {
				t.Fatalf("missing fixturePageBounds for %s", file)
			}

			if pages < buf.minPages || (buf.maxPages > 0 && pages > buf.maxPages) {
				t.Errorf("pages = %d, want [%d, %d]", pages, buf.minPages, buf.maxPages)
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

			if len(buf.needles) > 0 {
				doc, err := pdf.ParseSemantic(data)
				if err != nil {
					t.Fatalf("ParseSemantic: %v", err)
				}

				text := doc.DocumentText()
				pos := 0

				for _, needle := range buf.needles {
					idx := strings.Index(text[pos:], needle)
					if idx < 0 {
						t.Errorf("extracted text missing %q after offset %d; text=%q", needle, pos, text)

						continue
					}

					pos += idx + len(needle)
				}
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

	cmd := requestForFixture(t, "fixture-03-multi-page-invoice.html")

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

//nolint:cyclop,funlen // golden needle assertions for PDF 1.7 vs 1.4 default
func TestConvertPDF17GoldenNeedles(t *testing.T) {
	t.Parallel()

	htmlContent := `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>Invoice — July 2026 🚀</title>
</head>
<body>
<h1>Invoice — July 2026</h1>
<p>Total amount: $1,250.00</p>
</body>
</html>`

	// 1. Convert with --pdf-version 1.7
	cmd17, _ := newCommand(t, htmlContent, "")
	cmd17.Global.PdfVersion = pdfVersion17
	cmd17.Global.Title = "Invoice — July 2026 🚀"
	data17 := runPDF(t, cmd17)
	str17 := string(data17)

	// Needle assertions on 1.7 output:
	// Starts with %PDF-1.7
	if !bytes.HasPrefix(data17, []byte("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n")) {
		t.Errorf("PDF 1.7 output missing expected header prefix, got %q", data17[:min(25, len(data17))])
	}

	// Contains trailer /ID [ <HEX> <HEX> ]
	idRe := regexp.MustCompile(`/ID\s*\[\s*<([0-9A-Fa-f]{32})>\s*<([0-9A-Fa-f]{32})>\s*\]`)
	if !idRe.MatchString(str17) {
		t.Errorf("PDF 1.7 output missing trailer /ID [ <hex> <hex> ]:\n%s", str17)
	}

	// Contains /Type /Metadata /Subtype /XML
	if !strings.Contains(str17, "/Type /Metadata /Subtype /XML") {
		t.Error("PDF 1.7 output missing metadata stream object /Type /Metadata /Subtype /XML")
	}

	// Producer contains 1.7
	if !strings.Contains(str17, "/Producer (gowkhtmltopdf 1.7)") {
		t.Errorf("PDF 1.7 Info dict missing /Producer (gowkhtmltopdf 1.7)")
	}

	if !strings.Contains(str17, "<pdf:Producer>gowkhtmltopdf 1.7</pdf:Producer>") {
		t.Errorf("PDF 1.7 XMP metadata missing <pdf:Producer>gowkhtmltopdf 1.7</pdf:Producer>")
	}

	// Does NOT contain pdfaid, pdfuaid, or pdfaExtension
	for _, forbidden := range []string{"pdfaid", "pdfuaid", "pdfaExtension"} {
		if strings.Contains(str17, forbidden) {
			t.Errorf("PDF 1.7 output contains forbidden claim token %q", forbidden)
		}
	}

	// Title with Unicode is encoded with UTF-16BE (<FEFF...>), NOT raw UTF-8
	if !strings.Contains(str17, "/Title <FEFF") {
		t.Errorf("PDF 1.7 Unicode Title should be encoded as UTF-16BE hex string <FEFF...>")
	}

	if strings.Contains(str17, "/Title (Invoice — July 2026 🚀)") {
		t.Errorf("PDF 1.7 Title was emitted as raw UTF-8 text string")
	}

	// 2. Convert the same HTML without version setting (default 1.4)
	cmd14, _ := newCommand(t, htmlContent, "")
	cmd14.Global.Title = "Invoice — July 2026 🚀"
	data14 := runPDF(t, cmd14)
	str14 := string(data14)

	// Starts with %PDF-1.4
	if !bytes.HasPrefix(data14, []byte("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")) {
		t.Errorf("PDF 1.4 output missing expected header prefix, got %q", data14[:min(25, len(data14))])
	}

	// No /Metadata in catalog
	catRe := regexp.MustCompile(`<<\s*/Type\s*/Catalog[^>]*>>`)
	catMatch14 := catRe.FindString(str14)

	if strings.Contains(catMatch14, "/Metadata") {
		t.Errorf("Default PDF 1.4 catalog contains /Metadata: %s", catMatch14)
	}

	// No trailer /ID
	trailerIdx14 := strings.Index(str14, "trailer\n")
	if trailerIdx14 >= 0 && strings.Contains(str14[trailerIdx14:], "/ID") {
		t.Errorf("Default PDF 1.4 trailer contains /ID: %s", str14[trailerIdx14:])
	}

	// Producer contains 1.4
	if !strings.Contains(str14, "/Producer (gowkhtmltopdf 1.4)") {
		t.Errorf("Default PDF 1.4 Info dict missing /Producer (gowkhtmltopdf 1.4)")
	}
}

func assertMultiPageTOC(t *testing.T, version string) {
	t.Helper()

	htmlBody := `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>Multi-Page Doc</title></head>
<body>
<h1>Chapter 1: Overview</h1>
<p>` + strings.Repeat("This is section one content providing details about the system. ", 50) + `</p>
<div style="page-break-before: always;"></div>
<h1>Chapter 2: Architecture</h1>
<p>` + strings.Repeat("This is section two describing architecture and pipelines in detail. ", 50) + `</p>
<div style="page-break-before: always;"></div>
<h1>Chapter 3: Verification</h1>
<p>` + strings.Repeat("This is section three covering verification and quality assurance. ", 50) + `</p>
</body>
</html>`

	cmd, _ := newCommand(t, htmlBody, "")
	cmd.Global.PdfVersion = version
	cmd.Global.Header.Left = "Document Header [page]"
	cmd.Global.Footer.Right = "Page [page] of [topage]"
	cmd.Global.UseCompression = false

	tocObj := settings.DefaultPdfObject()
	tocObj.IsTableOfContent = true
	cmd.Objects = append([]settings.PdfObject{tocObj}, cmd.Objects...)

	data := runPDF(t, cmd)

	if !bytes.HasPrefix(data, []byte("%PDF-"+version+"\n")) {
		t.Errorf("expected %%PDF-%s header, got %q", version, data[:min(15, len(data))])
	}

	pages := pageCount(data)
	if pages < 4 {
		t.Errorf("page count = %d, want >= 4", pages)
	}

	if !bytes.Contains(data, []byte("/Type /Outlines")) || !bytes.Contains(data, []byte("/PageMode /UseOutlines")) {
		t.Error("expected outline bookmarks in PDF " + version + " output with TOC")
	}

	if !bytes.Contains(data, []byte("Document Header")) {
		t.Error("header text missing in PDF " + version + " output")
	}

	if !bytes.Contains(data, []byte("Page 1 of")) {
		t.Error("footer text missing in PDF " + version + " output")
	}

	sem, err := pdf.ParseSemantic(data)
	if err != nil {
		t.Fatalf("ParseSemantic: %v", err)
	}

	if sem.Version != version {
		t.Errorf("sem.Version = %q, want %s", sem.Version, version)
	}

	if sem.PageCount() != pages {
		t.Errorf("sem.PageCount = %d, want %d", sem.PageCount(), pages)
	}
}

func TestConvertPDF17MultiPageTOCHF(t *testing.T) {
	t.Parallel()
	assertMultiPageTOC(t, pdfVersion17)
}

func TestConvertPDF20GoldenNeedles(t *testing.T) {
	t.Parallel()

	// 1. Convert a small committed fixture with version 2.0.
	cmd20 := requestForFixture(t, "fixture-01-simple-invoice.html")
	cmd20.Global.PdfVersion = pdfVersion20
	data20 := runPDF(t, cmd20)
	str20 := string(data20)

	if !bytes.HasPrefix(data20, []byte("%PDF-2.0\n%\xe2\xe3\xcf\xd3\n")) {
		t.Errorf("PDF 2.0 output missing expected header prefix, got %q", data20[:min(25, len(data20))])
	}

	// Trailer /ID [ <hex> <hex> ].
	idRe := regexp.MustCompile(`/ID\s*\[\s*<([0-9A-Fa-f]{32})>\s*<([0-9A-Fa-f]{32})>\s*\]`)
	if !idRe.MatchString(str20) {
		t.Errorf("PDF 2.0 output missing trailer /ID [ <hex> <hex> ]:\n%s", str20)
	}

	// Catalog references /Metadata.
	if !strings.Contains(str20, "/Type /Metadata /Subtype /XML") {
		t.Error("PDF 2.0 output missing metadata stream object /Type /Metadata /Subtype /XML")
	}

	// Producer claims the 2.0 version.
	if !strings.Contains(str20, "/Producer (gowkhtmltopdf 2.0)") {
		t.Errorf("PDF 2.0 Info dict missing /Producer (gowkhtmltopdf 2.0)")
	}

	// No conformance claims (#33 boundary).
	for _, forbidden := range []string{"pdfaid", "pdfuaid"} {
		if strings.Contains(str20, forbidden) {
			t.Errorf("PDF 2.0 output contains forbidden claim token %q", forbidden)
		}
	}

	// 2. The same fixture without the setting keeps the 1.4 envelope.
	cmd14 := requestForFixture(t, "fixture-01-simple-invoice.html")
	data14 := runPDF(t, cmd14)

	if !bytes.HasPrefix(data14, []byte("%PDF-1.4\n")) {
		t.Errorf("default output missing %%PDF-1.4 header, got %q", data14[:min(10, len(data14))])
	}

	if bytes.Contains(data14, []byte("%PDF-2.0")) {
		t.Error("default 1.4 output must not claim %PDF-2.0")
	}
}

func TestConvertPDF20MultiPageTOCHF(t *testing.T) {
	t.Parallel()
	assertMultiPageTOC(t, pdfVersion20)
}

// TestOptionalPDFValidation opens converted files with an independent parser
// (qpdf --check and/or mutool info) when one is installed. This is the #32
// optional external gate: it skips when neither binary exists and never fails
// CI for a missing tool. veraPDF profile checks are deliberately absent here
// (compliance profiles are #33).
func TestOptionalPDFValidation(t *testing.T) {
	t.Parallel()

	// Check if qpdf or mutool is available
	qpdfPath, errQpdf := exec.LookPath("qpdf")
	mutoolPath, errMutool := exec.LookPath("mutool")

	if errQpdf != nil && errMutool != nil {
		t.Skip("optional validator qpdf/mutool not installed")
	}

	htmlContent := `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>Validation — 2026</title></head>
<body>
<h1>Validation Document</h1>
<p>Testing PDF compliance with external validator.</p>
</body>
</html>`

	validate := func(name, version string) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd, _ := newCommand(t, htmlContent, "")
			cmd.Global.PdfVersion = version
			cmd.Global.Title = "Validation — 2026"
			data := runPDF(t, cmd)

			pdfFile := filepath.Join(t.TempDir(), name+".pdf")
			if err := os.WriteFile(pdfFile, data, 0o600); err != nil {
				t.Fatalf("write PDF file: %v", err)
			}

			if errQpdf == nil {
				out, err := exec.CommandContext(t.Context(), qpdfPath, "--check", pdfFile).CombinedOutput()
				if err != nil {
					t.Errorf("qpdf --check failed: %v\nOutput: %s", err, string(out))
				}
			}

			if errMutool == nil {
				out, err := exec.CommandContext(t.Context(), mutoolPath, "info", pdfFile).CombinedOutput()
				if err != nil {
					t.Errorf("mutool info failed: %v\nOutput: %s", err, string(out))
				}

				cleanOut, errClean := exec.CommandContext(
					t.Context(), mutoolPath, "clean", "-s", pdfFile, filepath.Join(t.TempDir(), "clean.pdf"),
				).CombinedOutput()
				if errClean != nil {
					t.Errorf("mutool clean failed: %v\nOutput: %s", errClean, string(cleanOut))
				}
			}
		})
	}

	validate("test_17", pdfVersion17)
	validate("test_20", pdfVersion20)
}
