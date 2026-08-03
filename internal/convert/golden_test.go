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

func goldenPath(file string) string {
	return filepath.Join("..", "..", "testdata", "golden", file)
}

// commandForFixture builds a cli.Command that converts a golden fixture:
// A4 page, 10 mm margins, backgrounds on, local file access enabled so the
// fixture's relative links and images resolve (same ACL shape as newCommand).
func commandForFixture(t *testing.T, file string) *cli.Command {
	t.Helper()
	dir := t.TempDir()
	dest := filepath.Join(dir, filepath.Base(file))
	content, err := os.ReadFile(goldenPath(file))
	if err != nil {
		t.Fatalf("read fixture %s: %v", file, err)
	}
	if err := os.WriteFile(dest, content, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cmd := &cli.Command{
		Global: settings.DefaultPdfGlobal(),
		Objects: []settings.PdfObject{
			{Page: dest, Load: settings.DefaultLoadPage()},
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
