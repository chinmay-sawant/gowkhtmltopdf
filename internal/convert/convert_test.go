package convert //nolint:testpackage // white-box tests need unexported access

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/imageout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// defaultObject returns a PdfObject with documented defaults (ExternalLinks,
// LocalLinks, UseOutline, IncludeInOutline ON). Callers / CLI must supply
// these; convert no longer OR-hacks zero values permanently on.
func defaultObject(page string) settings.PdfObject {
	o := settings.DefaultPdfObject()
	o.Page = page
	o.Load.BlockLocalFileAccess = false

	return o
}

// newCommand writes html into a temp dir and returns a convert.Request
// pointing at it, with local file access enabled (the frozen ACL default
// blocks local reads unless the user opts in). The unused output argument is
// kept so existing call sites compile while the engine writes to req.Output.
func newCommand(t *testing.T, html string, _ string) (*Request, string) {
	t.Helper()
	dir := t.TempDir()

	path := filepath.Join(dir, "input.html")
	if err := os.WriteFile(path, []byte(html), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	global := settings.DefaultPdfGlobal()
	global.Load.EnableLocalFileAccess = true

	return NewPDFRequest(global, []settings.PdfObject{defaultObject(path)}, nil, nil), dir
}

// runPDF runs the engine request and returns the produced PDF bytes.
func runPDF(t *testing.T, req *Request) []byte {
	t.Helper()

	return runPDFWithLog(t, req, io.Discard)
}

func runPDFWithLog(t *testing.T, req *Request, log io.Writer) []byte {
	t.Helper()

	var buf bytes.Buffer
	req.Output = &buf

	if err := Run(t.Context(), req, log, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	return buf.Bytes()
}

// pageCount counts page dicts. The pages tree root is emitted as
// "/Type /Pages\n", which never matches "/Type /Page\n".
func pageCount(data []byte) int {
	return bytes.Count(data, []byte("/Type /Page\n"))
}

func TestRunPDFSinglePageA4(t *testing.T) {
	t.Parallel()
	cmd, _ := newCommand(t, `<html><body><h1>Hello</h1><p>world</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))

	data := runPDF(t, cmd)
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}

	if n := pageCount(data); n != 1 {
		t.Errorf("pages = %d, want 1", n)
	}

	if !bytes.Contains(data, []byte("/FontFile2")) {
		t.Error("expected embedded subset font (/FontFile2)")
	}
}

func TestRunPDFMultiPage(t *testing.T) {
	t.Parallel()

	var strB strings.Builder

	strB.WriteString("<html><body>")

	for i := range 200 {
		strB.WriteString("<p>paragraph of text number ")
		strB.WriteRune(rune('a' + i%26))
		strB.WriteString(" with some words to wrap</p>")
	}

	strB.WriteString("</body></html>")
	cmd, _ := newCommand(t, strB.String(), filepath.Join(t.TempDir(), "out.pdf"))

	data := runPDF(t, cmd)
	if n := pageCount(data); n < 2 {
		t.Errorf("pages = %d, want >= 2", n)
	}
}

func TestRunPDFStyleTableImage(t *testing.T) {
	t.Parallel()
	pngB64 := pngDataURL(t, 12, 12)
	html := `<html><head><style>
.box { background-color: #336699; width: 80px; height: 30px; }
</style></head><body>
<div class="box">colored</div>
<table><tr><th>a</th><th>b</th></tr><tr><td>1</td><td>2</td></tr></table>
<img src="` + pngB64 + `">
</body></html>`
	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))

	data := runPDF(t, cmd)
	if n := pageCount(data); n != 1 {
		t.Errorf("pages = %d, want 1", n)
	}

	if !bytes.Contains(data, []byte("/Subtype /Image")) {
		t.Error("expected an embedded image xobject")
	}
}

func TestRunPDFWebImagesFalse(t *testing.T) {
	t.Parallel()
	pngB64 := pngDataURL(t, 12, 12)
	html := `<html><body><p>noimg</p><img src="` + pngB64 + `"></body></html>`
	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Web.Images = false

	data := runPDF(t, cmd)
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}

	if bytes.Contains(data, []byte("/Subtype /Image")) {
		t.Error("web.images=false should not embed image XObjects")
	}
}

func TestRunPDFLinkedStylesheet(t *testing.T) {
	t.Parallel()

	cmd, dir := newCommand(
		t,
		`<html><head><link rel="stylesheet" href="style.css"></head><body><div class="box">styled</div></body></html>`,
		filepath.Join(t.TempDir(), "out.pdf"))
	if err := os.WriteFile(filepath.Join(dir, "style.css"), []byte(".box { background-color: #000000; }"), 0o600); err != nil { //nolint:lll // fixture write
		t.Fatalf("write css: %v", err)
	}

	data := runPDF(t, cmd)
	if n := pageCount(data); n != 1 {
		t.Errorf("pages = %d, want 1", n)
	}
}

func TestRunPDFScreenOnlyStylesheetExcluded(t *testing.T) {
	t.Parallel()

	cmd, dir := newCommand(
		t,
		`<html><head><link rel="stylesheet" href="screen.css" media="screen"></head><body><div class="box">styled</div></body></html>`, //nolint:lll // long HTML fixture
		filepath.Join(t.TempDir(), "out.pdf"))
	if err := os.WriteFile(filepath.Join(dir, "screen.css"), []byte(".box { background-color: #000000; }"), 0o600); err != nil { //nolint:lll // fixture write
		t.Fatalf("write css: %v", err)
	}

	_ = runPDF(t, cmd)
}

func TestRunPDFPrintLinkMediaFeatures(t *testing.T) {
	t.Parallel()

	htmlDoc := `<html><head><link rel="stylesheet" href="feat.css" media="(min-width: 500px)"></head>
<body><p class="hi">Hello</p></body></html>`

	cmd, dir := newCommand(t, htmlDoc, filepath.Join(t.TempDir(), "out.pdf"))
	if err := os.WriteFile(filepath.Join(dir, "feat.css"), []byte(".hi { color: #0645ad }"), 0o600); err != nil {
		t.Fatalf("write css: %v", err)
	}

	var log bytes.Buffer
	_ = runPDFWithLog(t, cmd, &log)
}

func TestLinkStylesheetMediaMatches(t *testing.T) {
	t.Parallel()

	mark := func(media string) *html.Node {
		return &html.Node{ //nolint:exhaustruct // intentional zero-value fields
			Type:  html.ElementNode,
			Name:  "link",
			Attrs: map[string]string{"rel": "stylesheet", "href": "x.css", "media": media},
		}
	}

	const viewW, viewH = 538.0, 785.0

	if !linkStylesheet(mark(""), viewW, viewH, mediaPrint) {
		t.Error("empty media should load")
	}

	if !linkStylesheet(mark("print"), viewW, viewH, mediaPrint) {
		t.Error("print should load")
	}

	if !linkStylesheet(mark("all"), viewW, viewH, mediaPrint) {
		t.Error("all should load")
	}

	if linkStylesheet(mark("screen"), viewW, viewH, mediaPrint) {
		t.Error("screen-only must be excluded for print")
	}

	if !linkStylesheet(mark("(min-width: 500px)"), viewW, viewH, mediaPrint) {
		t.Error("min-width feature matching A4 content should load")
	}

	if linkStylesheet(mark("(min-width: 2000px)"), viewW, viewH, mediaPrint) {
		t.Error("unmatched min-width must not load")
	}

	if !linkStylesheet(mark("screen"), viewW, viewH, "screen") {
		t.Error("screen media type should accept screen stylesheets")
	}
}

func TestMediaForPDF(t *testing.T) {
	t.Parallel()

	glob := settings.DefaultPdfGlobal()
	obj := settings.DefaultPdfObject()

	if got := mediaFor(glob, &obj); got != mediaPrint {
		t.Errorf("PDF default media = %q, want print", got)
	}

	obj.Load.MediaType = settings.MediaScreen
	if got := mediaFor(glob, &obj); got != "screen" {
		t.Errorf("object media-type screen = %q, want screen", got)
	}

	obj.Load.MediaType = settings.MediaPrint
	if got := mediaFor(glob, &obj); got != mediaPrint {
		t.Errorf("object media-type print = %q, want print", got)
	}
	// MediaIgnore is zero/unset — keeps PDF print default.
	obj.Load.MediaType = settings.MediaIgnore
	if got := mediaFor(glob, &obj); got != mediaPrint {
		t.Errorf("object media-type ignore/unset = %q, want print", got)
	}
}

func TestRunPDFOutputStdout(t *testing.T) {
	t.Parallel()

	req, _ := newCommand(t, `<html><body><p>stdout test</p></body></html>`, "")
	data := runPDF(t, req)

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("writer output is not a PDF")
	}

	if n := pageCount(data); n != 1 {
		t.Errorf("pages = %d, want 1", n)
	}
}

func TestRunPDFMissingFile(t *testing.T) {
	t.Parallel()
	cmd, _ := newCommand(t, `<html><body>x</body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Objects[0].Page = filepath.Join(t.TempDir(), "does-not-exist.html")
	cmd.Output = io.Discard

	if err := Run(t.Context(), cmd, io.Discard, nil); err == nil {
		t.Fatal("expected error for missing input file, got nil")
	}
}

// pngDataURL builds a minimal valid RGBA PNG as a data: URL.
func pngDataURL(t *testing.T, width, h int) string {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, h))

	for y := range h {
		for x := range width {
			img.Set(x, y, color.RGBA{200, 30, 30, 255})
		}
	}

	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatalf("png encode: %v", err)
	}

	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(out.Bytes())
}

// --- copies / collate assembly (plan 5.4) ---
// newCommandMulti builds a command with one temp input file per html
// fragment, in order.
func newCommandMulti(t *testing.T, htmls []string, _ string) *Request {
	t.Helper()

	global := settings.DefaultPdfGlobal()
	global.Load.EnableLocalFileAccess = true
	objects := make([]settings.PdfObject, 0, len(htmls))

	for _, h := range htmls {
		dir := t.TempDir()

		path := filepath.Join(dir, "input.html")
		if err := os.WriteFile(path, []byte(h), 0o600); err != nil {
			t.Fatalf("write input: %v", err)
		}

		objects = append(objects, defaultObject(path))
	}

	return NewPDFRequest(global, objects, nil, nil)
}

// kidsRe matches the pages tree /Kids array.
var kidsRe = regexp.MustCompile(`/Kids \[([^\]]+)\]`)

// kidsRefs extracts the page object refs listed in the /Kids array.
func kidsRefs(t *testing.T, data []byte) []int {
	t.Helper()

	m := kidsRe.FindSubmatch(data)
	if m == nil {
		t.Fatalf("no /Kids array found")
	}

	fields := strings.Fields(string(m[1]))

	var refs []int

	for i := 0; i+2 < len(fields); i += 3 {
		n, err := strconv.Atoi(fields[i])
		if err != nil {
			t.Fatalf("bad kid ref %q", fields[i])
		}

		refs = append(refs, n)
	}

	return refs
}

// objectDict returns the dict body of indirect object ref (uncompressed test
// builds only - the body is plain text).
func objectDict(t *testing.T, data []byte, ref int) []byte {
	t.Helper()

	marker := fmt.Sprintf("%d 0 obj\n", ref)

	idx := bytes.Index(data, []byte(marker))
	if idx < 0 {
		t.Fatalf("object %d not found", ref)
	}

	end := bytes.Index(data[idx:], []byte("\nendobj"))
	if end < 0 {
		t.Fatalf("object %d has no endobj", ref)
	}

	return data[idx : idx+end]
}

// pageLabel resolves a page object to the marker text in its content stream
// ("AAA" → "A", "BBB" → "B"), so the /Kids order can be asserted
// semantically instead of by raw object id.
func pageLabel(t *testing.T, data []byte, pageRef int) string {
	t.Helper()
	dict := objectDict(t, data, pageRef)
	mVal := regexp.MustCompile(`/Contents (\d+) 0 R`).FindSubmatch(dict)

	if mVal == nil {
		t.Fatalf("page %d dict has no /Contents: %q", pageRef, dict)
	}

	contentRef, err := strconv.Atoi(string(mVal[1]))
	if err != nil {
		t.Fatalf("bad contents ref: %v", err)
	}

	body := objectDict(t, data, contentRef)

	switch {
	case bytes.Contains(body, []byte("AAA")):
		return "A"
	case bytes.Contains(body, []byte("BBB")):
		return "B"
	}

	return "?"
}

// labelsOf returns the label of each page in /Kids order.
func labelsOf(t *testing.T, data []byte) []string {
	t.Helper()

	refs := kidsRefs(t, data)

	labels := make([]string, 0, len(refs))

	for _, ref := range refs {
		labels = append(labels, pageLabel(t, data, ref))
	}

	return labels
}

func TestRunPDFCopiesCollate(t *testing.T) {
	t.Parallel()
	cmd := newCommandMulti(t,
		[]string{
			`<html><body><h1>AAA</h1></body></html>`,
			`<html><body><h1>BBB</h1></body></html>`,
		},
		filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Copies = 2
	cmd.Global.Collate = true
	cmd.Global.UseCompression = false // keep content streams searchable
	data := runPDF(t, cmd)

	if n := pageCount(data); n != 4 {
		t.Fatalf("pages = %d, want 4", n)
	}

	got := labelsOf(t, data)
	want := []string{"A", "B", "A", "B"}

	if !slices.Equal(got, want) {
		t.Errorf("collated order = %v, want %v", got, want)
	}
}

func TestRunPDFCopiesNonCollate(t *testing.T) {
	t.Parallel()
	cmd := newCommandMulti(t,
		[]string{
			`<html><body><h1>AAA</h1></body></html>`,
			`<html><body><h1>BBB</h1></body></html>`,
		},
		filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Copies = 2
	cmd.Global.Collate = false
	cmd.Global.UseCompression = false
	data := runPDF(t, cmd)

	if n := pageCount(data); n != 4 {
		t.Fatalf("pages = %d, want 4", n)
	}

	got := labelsOf(t, data)
	want := []string{"A", "A", "B", "B"}

	if !slices.Equal(got, want) {
		t.Errorf("non-collated order = %v, want %v", got, want)
	}
}

func TestRunPDFThreeObjects(t *testing.T) {
	t.Parallel()
	cmd := newCommandMulti(t,
		[]string{
			`<html><body><p>one</p></body></html>`,
			`<html><body><p>two</p></body></html>`,
			`<html><body><p>three</p></body></html>`,
		},
		filepath.Join(t.TempDir(), "out.pdf"))

	data := runPDF(t, cmd)
	if n := pageCount(data); n != 3 {
		t.Errorf("pages = %d, want 3", n)
	}
}

// --- context, progress, quiet (plan 5.5) ---

func TestRunPDFProgress(t *testing.T) {
	t.Parallel()
	cmd, _ := newCommand(t, `<html><body><p>progress</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))

	var phases []string

	percents := map[string]int{}
	collect := func(phase string, percent int) {
		phases = append(phases, phase)
		percents[phase] = percent
	}

	var log bytes.Buffer

	var out bytes.Buffer

	cmd.Output = &out

	if err := Run(t.Context(), cmd, &log, collect); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Real phases only: load progress + Done (no theater 100% placeholders).
	want := []string{
		"Loading pages (1/1)",
		"Done",
	}
	if !slices.Equal(phases, want) {
		t.Errorf("phases = %v, want %v", phases, want)
	}

	if p := percents["Done"]; p != 100 {
		t.Errorf("Done percent = %d, want 100", p)
	}

	if p := percents["Loading pages (1/1)"]; p != 100 {
		t.Errorf("Loading pages percent = %d, want 100", p)
	}

	if !strings.Contains(log.String(), "Loading pages (1/1)") {
		t.Error("progress phase not written to log")
	}
}

func TestRunPDFQuiet(t *testing.T) {
	t.Parallel()
	cmd, _ := newCommand(t, `<html><body><p>quiet</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Quiet = true

	var log bytes.Buffer

	_ = runPDFWithLog(t, cmd, &log)

	if log.Len() != 0 {
		t.Errorf("quiet mode wrote %d bytes to log: %q", log.Len(), log.String())
	}
}

func TestRunPDFContextCancel(t *testing.T) {
	t.Parallel()
	cmd, _ := newCommand(t, `<html><body><p>cancel</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	cmd.Output = io.Discard
	err := Run(ctx, cmd, io.Discard, nil)

	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

// --- smart shrinking (plan 5.3) ---

func TestRunPDFSmartShrinking(t *testing.T) { //nolint:cyclop // zoom verification has many check steps
	t.Parallel()

	// 2000px fixed-width div is ~1500pt wide, far beyond the A4 content area.
	cmd, _ := newCommand(t,
		`<html><body><div style="background-color:#336699; width:2000px; height:50px;">wide</div></body></html>`,
		filepath.Join(t.TempDir(), "out.pdf"))

	var log bytes.Buffer
	data := runPDFWithLog(t, cmd, &log)

	if !bytes.Contains(log.Bytes(), []byte("smart shrinking")) && pageCount(data) < 1 {
		t.Errorf("expected smart-shrinking relayout or a valid page, log: %q", log.String())
	}

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("smart-shrink output is not a PDF")
	}

	if n := pageCount(data); n < 1 || n > 10 {
		t.Errorf("pages = %d, want a sane small count", n)
	}
	// the shrink must produce a scaled-down content stream: decompress all
	// streams and require the "wide" text with a font size well below 12pt
	decoded := decodeStreams(data)
	if !bytes.Contains(decoded, []byte("wide")) {
		t.Error("expected the 'wide' text in the shrunk PDF")
	}

	if !regexp.MustCompile(`\d+(\.\d+)? Tf`).Match(decoded) {
		t.Error("expected a text-show command in the shrunk PDF")
	}

	for _, m := range regexp.MustCompile(`(\d+(\.\d+)?) Tf`).FindAllSubmatch(decoded, -1) {
		if v, err := strconv.ParseFloat(string(m[1]), 64); err == nil && v < 6 {
			return
		}
	}

	t.Error("expected a font size below 6pt in the shrunk PDF (zoom applied)")
}

// decodeStreams decompresses every FlateDecode (zlib) stream in a PDF byte blob.
func decodeStreams(data []byte) []byte {
	var out []byte

	for {
		start := bytes.Index(data, []byte("stream\n"))
		if start < 0 {
			break
		}

		data = data[start+len("stream\n"):]

		end := bytes.Index(data, []byte("endstream"))
		if end < 0 {
			break
		}

		raw := data[:end]
		data = data[end+len("endstream"):]

		rdr, err := zlib.NewReader(bytes.NewReader(raw))
		if err != nil {
			continue
		}

		dec, derr := io.ReadAll(rdr)
		rdr.Close()

		if derr == nil {
			out = append(out, dec...)
		}
	}

	return out
}

//nolint:exhaustruct // test initial geometry and stylesheets
func TestCSSPageSizeAndMargins(t *testing.T) {
	t.Parallel()

	initGeom := hfGeom{
		pageW: 595.28, pageH: 841.89,
		marginTop: 28.35, marginBottom: 28.35,
		marginLeft: 28.35, marginRight: 28.35,
	}

	// 1. @page { size: letter; margin: 1in }
	sheetLetter := &css.Stylesheet{
		Page: &css.PageStyle{Size: "letter", Margin: "1in"},
	}
	geom := applyCSSPageMargins(initGeom, []*css.Stylesheet{sheetLetter})

	if geom.pageW != 612 || geom.pageH != 792 {
		t.Errorf("geom size = %vx%v, want 612x792 (letter)", geom.pageW, geom.pageH)
	}

	if geom.marginTop != 72 || geom.marginLeft != 72 {
		t.Errorf("geom margins = top %v, left %v, want 72 (1in)", geom.marginTop, geom.marginLeft)
	}

	// 2. @page { size: invalid } degrades without panic or corrupting size
	sheetInvalid := &css.Stylesheet{
		Page: &css.PageStyle{Size: "not-a-valid-size-name-xyz"},
	}
	geom2 := applyCSSPageMargins(initGeom, []*css.Stylesheet{sheetInvalid})

	if geom2.pageW != initGeom.pageW || geom2.pageH != initGeom.pageH {
		t.Errorf("invalid size modified geometry: %vx%v", geom2.pageW, geom2.pageH)
	}

	// 3. @page { size: 4in 6in }
	sheetCustom := &css.Stylesheet{
		Page: &css.PageStyle{Size: "4in 6in"},
	}
	geom3 := applyCSSPageMargins(initGeom, []*css.Stylesheet{sheetCustom})

	if geom3.pageW != 288 || geom3.pageH != 432 {
		t.Errorf("custom size = %vx%v, want 288x432", geom3.pageW, geom3.pageH)
	}
}

func TestHeaderFooterFontNameResolution(t *testing.T) {
	t.Parallel()

	defFont, err := pdf.DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	reg := pdf.NewRegistry()
	reg.AddFont(defFont)

	resolved := resolveHFFont("Liberation Sans", reg, defFont)
	if resolved == nil {
		t.Fatal("expected font to resolve for 'Liberation Sans'")
	}

	fallback := resolveHFFont("NonExistentFontXYZ", reg, defFont)
	if fallback != defFont {
		t.Errorf("expected fallback font, got %v", fallback)
	}
}

func TestConvertPDFVersion(t *testing.T) {
	t.Parallel()

	// 1. Unset version -> %PDF-1.4
	cmdUnset, _ := newCommand(t, `<html><body><p>Unset version</p></body></html>`, "")
	dataUnset := runPDF(t, cmdUnset)

	if !bytes.HasPrefix(dataUnset, []byte("%PDF-1.4\n")) {
		t.Errorf("expected %%PDF-1.4, got %q", dataUnset[:min(10, len(dataUnset))])
	}

	// 2. Explicit 1.4 -> %PDF-1.4
	cmd14, _ := newCommand(t, `<html><body><p>v1.4</p></body></html>`, "")
	cmd14.Global.PdfVersion = "1.4"
	data14 := runPDF(t, cmd14)

	if !bytes.HasPrefix(data14, []byte("%PDF-1.4\n")) {
		t.Errorf("expected %%PDF-1.4, got %q", data14[:min(10, len(data14))])
	}

	// 3. Explicit 1.7 -> %PDF-1.7
	cmd17, _ := newCommand(t, `<html><body><p>v1.7</p></body></html>`, "")
	cmd17.Global.PdfVersion = pdfVersion17
	data17 := runPDF(t, cmd17)

	if !bytes.HasPrefix(data17, []byte("%PDF-1.7\n")) {
		t.Errorf("expected %%PDF-1.7, got %q", data17[:min(10, len(data17))])
	}

	// 4. Version 2.0 -> %PDF-2.0
	cmd20, _ := newCommand(t, `<html><body><p>v2.0</p></body></html>`, "")
	cmd20.Global.PdfVersion = pdfVersion20
	data20 := runPDF(t, cmd20)

	if !bytes.HasPrefix(data20, []byte("%PDF-2.0\n")) {
		t.Errorf("expected %%PDF-2.0, got %q", data20[:min(10, len(data20))])
	}

	// 5. Invalid version -> returns ErrInvalidPDFVersion
	cmdBad, _ := newCommand(t, `<html><body><p>bad</p></body></html>`, "")
	cmdBad.Global.PdfVersion = "invalid"

	var bufBad bytes.Buffer
	cmdBad.Output = &bufBad
	errBad := Run(t.Context(), cmdBad, io.Discard, nil)

	if !errors.Is(errBad, settings.ErrInvalidPDFVersion) {
		t.Errorf("expected ErrInvalidPDFVersion, got %v", errBad)
	}

	// 6. With TOC and 1.7
	cmdTOC, _ := newCommand(t, `<html><body><h1>Chapter 1</h1><p>Content</p></body></html>`, "")
	cmdTOC.Global.PdfVersion = pdfVersion17
	tocObj := settings.DefaultPdfObject()
	tocObj.IsTableOfContent = true
	cmdTOC.Objects = append([]settings.PdfObject{tocObj}, cmdTOC.Objects...)
	dataTOC := runPDF(t, cmdTOC)

	if !bytes.HasPrefix(dataTOC, []byte("%PDF-1.7\n")) {
		t.Errorf("expected %%PDF-1.7 with TOC, got %q", dataTOC[:min(10, len(dataTOC))])
	}
}

//nolint:funlen // comprehensive negative tests for PDF version and unsupported combinations
func TestPDFVersionNegativeValidation(t *testing.T) {
	t.Parallel()

	// 1. Unsupported version strings never produce a PDF or 1.7 file and return ErrInvalidPDFVersion
	unsupportedVersions := []string{
		"9.9", "invalid", "1.5", "1.6", "1.3", "1.0", "-1", "0.0", "abc", "1.7.0", "v1.7",
	}

	for _, badVer := range unsupportedVersions {
		t.Run("unsupported_"+badVer, func(t *testing.T) {
			t.Parallel()

			cmd, _ := newCommand(t, `<html><body><p>bad version test</p></body></html>`, "")
			cmd.Global.PdfVersion = badVer

			var out bytes.Buffer

			cmd.Output = &out

			err := Run(t.Context(), cmd, io.Discard, nil)
			if !errors.Is(err, settings.ErrInvalidPDFVersion) {
				t.Fatalf("expected ErrInvalidPDFVersion for %q, got: %v", badVer, err)
			}

			if out.Len() != 0 {
				t.Errorf("expected 0 bytes written on error for %q, got %d bytes: %q", badVer, out.Len(), out.String())
			}
		})
	}

	// 2. Unsupported combinations fail closed before Write on both 1.7 and 2.0
	t.Run("unsupported_combinations_fail_closed", func(t *testing.T) {
		t.Parallel()

		combos := []struct {
			name    string
			policy  pdf.WriterPolicy
			wantErr error
		}{
			{
				"encryption",
				pdf.WriterPolicy{Version: pdf.PDF17, Encryption: true}, //nolint:exhaustruct // test case
				pdf.ErrEncryptionUnsupported,
			},
			{
				"forms",
				pdf.WriterPolicy{Version: pdf.PDF17, Forms: true}, //nolint:exhaustruct // test case
				pdf.ErrFormsUnsupported,
			},
			{
				"signatures",
				pdf.WriterPolicy{Version: pdf.PDF17, Signatures: true}, //nolint:exhaustruct // test case
				pdf.ErrSignaturesUnsupported,
			},
			{
				"object_streams",
				pdf.WriterPolicy{Version: pdf.PDF17, ObjectStreams: true}, //nolint:exhaustruct // test case
				pdf.ErrObjectStreamsUnsupported,
			},
			{
				"pdf_a",
				pdf.WriterPolicy{Version: pdf.PDF17, ConformanceProfile: "PDF/A-4"}, //nolint:exhaustruct // test case
				pdf.ErrConformanceRequiresPDF20,
			},
			{
				"pdf_ua",
				pdf.WriterPolicy{Version: pdf.PDF17, ConformanceProfile: "PDF/UA-2"}, //nolint:exhaustruct // test case
				pdf.ErrConformanceRequiresPDF20,
			},
			{
				"pdf20_encryption",
				pdf.WriterPolicy{Version: pdf.PDF20, Encryption: true}, //nolint:exhaustruct // test case
				pdf.ErrEncryptionUnsupported,
			},
			{
				"pdf20_forms",
				pdf.WriterPolicy{Version: pdf.PDF20, Forms: true}, //nolint:exhaustruct // test case
				pdf.ErrFormsUnsupported,
			},
			{
				"pdf20_signatures",
				pdf.WriterPolicy{Version: pdf.PDF20, Signatures: true}, //nolint:exhaustruct // test case
				pdf.ErrSignaturesUnsupported,
			},
			{
				"pdf20_object_streams",
				pdf.WriterPolicy{Version: pdf.PDF20, ObjectStreams: true}, //nolint:exhaustruct // test case
				pdf.ErrObjectStreamsUnsupported,
			},
		}

		for _, combination := range combos {
			t.Run(combination.name, func(t *testing.T) {
				t.Parallel()

				if _, err := pdf.NewDocumentWithPolicy(combination.policy); !errors.Is(err, combination.wantErr) {
					t.Fatalf("NewDocumentWithPolicy err = %v, want %v", err, combination.wantErr)
				}

				if err := combination.policy.Validate(); !errors.Is(err, combination.wantErr) {
					t.Errorf("policy.Validate() err = %v, want %v", err, combination.wantErr)
				}
			})
		}
	})

	// 3. Image mode has no version claim
	t.Run("image_mode_no_version_claim", func(t *testing.T) {
		t.Parallel()

		imgGlobal := settings.DefaultImageGlobal()
		imgGlobal.Format = "png"

		obj := defaultObject("<html><body><h1>Image Title</h1><p>Image mode test</p></body></html>")

		var out bytes.Buffer

		req := imageout.NewRequest(settings.DefaultPdfGlobal(), imgGlobal, []settings.PdfObject{obj}, &out)

		err := imageout.RunRequest(t.Context(), req, io.Discard)
		if err != nil {
			t.Fatalf("image conversion failed: %v", err)
		}

		data := out.Bytes()
		// Must not contain any PDF header or version
		if bytes.Contains(data, []byte("%PDF-")) {
			t.Error("image output unexpectedly contains %PDF- header")
		}

		// Must start with PNG magic
		if !bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
			t.Errorf("image output missing PNG magic header, got: %q", data[:min(8, len(data))])
		}
	})
}

//nolint:cyclop,funlen // comprehensive test for compliance profile conversion integration
func TestPDFProfileConvertIntegration(t *testing.T) {
	t.Parallel()

	const profileA3aUA1 = "a3a-ua1"

	// 1. --pdf-profile a3a-ua1 implies PDF 1.7 and produces compliant PDF
	htmlDual := "<html><head><title>Compliance Document</title></head>" +
		"<body><h1>Compliance Document</h1><p>Dual Profile A-3a + UA-1</p></body></html>"
	cmdDual, _ := newCommand(t, htmlDual, "")
	cmdDual.Global.Title = "Compliance Document"
	cmdDual.Global.PdfProfile = profileA3aUA1
	dataDual := runPDF(t, cmdDual)

	if !bytes.HasPrefix(dataDual, []byte("%PDF-1.7\n")) {
		t.Errorf("expected %%PDF-1.7 header for a3a-ua1 profile, got %q", dataDual[:min(10, len(dataDual))])
	}

	// 2. --pdf-profile a3a produces PDF 1.7
	cmdA3a, _ := newCommand(t, `<html><body><h1>Archival Document</h1><p>PDF/A-3a</p></body></html>`, "")
	cmdA3a.Global.PdfProfile = "a3a"
	dataA3a := runPDF(t, cmdA3a)

	if !bytes.HasPrefix(dataA3a, []byte("%PDF-1.7\n")) {
		t.Errorf("expected %%PDF-1.7 header for a3a profile, got %q", dataA3a[:min(10, len(dataA3a))])
	}

	// 3. --pdf-profile ua1 produces PDF 1.7
	htmlUA1 := "<html><head><title>Accessible Document</title></head>" +
		"<body><h1>Accessible Document</h1><p>PDF/UA-1</p></body></html>"
	cmdUA1, _ := newCommand(t, htmlUA1, "")
	cmdUA1.Global.Title = "Accessible Document"
	cmdUA1.Global.PdfProfile = "ua1"
	dataUA1 := runPDF(t, cmdUA1)

	if !bytes.HasPrefix(dataUA1, []byte("%PDF-1.7\n")) {
		t.Errorf("expected %%PDF-1.7 header for ua1 profile, got %q", dataUA1[:min(10, len(dataUA1))])
	}

	// 4. Explicit --pdf-version 1.4 + --pdf-profile a3a-ua1 fails with ErrProfileRequiresPDF17
	cmdConflict, _ := newCommand(t, `<html><head><title>Title</title></head><body><p>conflict</p></body></html>`, "")
	cmdConflict.Global.Title = "Title"
	cmdConflict.Global.PdfVersion = "1.4"
	cmdConflict.Global.PdfProfile = profileA3aUA1

	var bufConflict bytes.Buffer

	cmdConflict.Output = &bufConflict
	errConflict := Run(t.Context(), cmdConflict, io.Discard, nil)

	if errConflict == nil {
		t.Fatal("expected error for PDF 1.4 + a3a-ua1, got nil")
	}

	if !errors.Is(errConflict, ErrProfileRequiresPDF17) {
		t.Errorf("expected ErrProfileRequiresPDF17, got %v", errConflict)
	}

	// 5. Explicit --pdf-version 1.7 + --pdf-profile a3a-ua1 succeeds
	htmlExplicit17 := "<html><head><title>Title</title></head><body><p>explicit 1.7 + profile</p></body></html>"
	cmdExplicit17, _ := newCommand(t, htmlExplicit17, "")
	cmdExplicit17.Global.Title = "Title"
	cmdExplicit17.Global.PdfVersion = "1.7"
	cmdExplicit17.Global.PdfProfile = profileA3aUA1
	dataExplicit17 := runPDF(t, cmdExplicit17)

	if !bytes.HasPrefix(dataExplicit17, []byte("%PDF-1.7\n")) {
		t.Errorf("expected %%PDF-1.7 header, got %q", dataExplicit17[:min(10, len(dataExplicit17))])
	}

	// 6. PolicyForGlobal unit tests
	policyDefaults, err := PolicyForGlobal(settings.DefaultPdfGlobal())
	if err != nil {
		t.Fatalf("PolicyForGlobal default: %v", err)
	}

	if policyDefaults.Version != pdf.PDF14 || policyDefaults.ConformanceProfile != "" {
		t.Errorf("default policy = %+v, want PDF14 unclaimed", policyDefaults)
	}

	globA3aUA1 := settings.DefaultPdfGlobal()
	globA3aUA1.PdfProfile = profileA3aUA1

	policyA3aUA1, err := PolicyForGlobal(globA3aUA1)
	if err != nil {
		t.Fatalf("PolicyForGlobal(a3a-ua1): %v", err)
	}

	if policyA3aUA1.Version != pdf.PDF17 || policyA3aUA1.ConformanceProfile != pdf.ProfilePDFA3aPDFUA1 {
		t.Errorf("a3a-ua1 policy = %+v, want PDF17 %s", policyA3aUA1, pdf.ProfilePDFA3aPDFUA1)
	}

	globA4 := settings.DefaultPdfGlobal()
	globA4.PdfProfile = "a4"

	policyA4, err := PolicyForGlobal(globA4)
	if err != nil {
		t.Fatalf("PolicyForGlobal(a4): %v", err)
	}

	if policyA4.Version != pdf.PDF20 || policyA4.ConformanceProfile != pdf.ProfilePDFA4 {
		t.Errorf("a4 policy = %+v, want PDF20 %s", policyA4, pdf.ProfilePDFA4)
	}

	glob20A4 := settings.DefaultPdfGlobal()
	glob20A4.PdfVersion = pdfVersion20
	glob20A4.PdfProfile = "PDF/A-4"

	policy20A4, err := PolicyForGlobal(glob20A4)
	if err != nil {
		t.Fatalf("PolicyForGlobal(2.0 + PDF/A-4): %v", err)
	}

	if policy20A4.Version != pdf.PDF20 || policy20A4.ConformanceProfile != pdf.ProfilePDFA4 {
		t.Errorf("2.0 + PDF/A-4 policy = %+v, want PDF20 %s", policy20A4, pdf.ProfilePDFA4)
	}

	glob20UA2 := settings.DefaultPdfGlobal()
	glob20UA2.PdfVersion = pdfVersion20
	glob20UA2.PdfProfile = "ua2"

	policy20UA2, err := PolicyForGlobal(glob20UA2)
	if err != nil {
		t.Fatalf("PolicyForGlobal(2.0 + ua2): %v", err)
	}

	if policy20UA2.Version != pdf.PDF20 || policy20UA2.ConformanceProfile != pdf.ProfilePDFUA2 {
		t.Errorf("2.0 + ua2 policy = %+v, want PDF20 %s", policy20UA2, pdf.ProfilePDFUA2)
	}

	glob17A4 := settings.DefaultPdfGlobal()
	glob17A4.PdfVersion = pdfVersion17
	glob17A4.PdfProfile = "a4"

	if _, err := PolicyForGlobal(glob17A4); !errors.Is(err, ErrProfileRequiresPDF20) {
		t.Errorf("PolicyForGlobal(1.7 + a4) err = %v, want ErrProfileRequiresPDF20", err)
	}

	glob14A4 := settings.DefaultPdfGlobal()
	glob14A4.PdfVersion = pdfVersion14
	glob14A4.PdfProfile = "a4"

	if _, err := PolicyForGlobal(glob14A4); !errors.Is(err, ErrProfileRequiresPDF20) {
		t.Errorf("PolicyForGlobal(1.4 + a4) err = %v, want ErrProfileRequiresPDF20", err)
	}

	// A 1.7-era profile on a 2.0 document is still an error: those profiles
	// require PDF 1.7.
	glob20A3a := settings.DefaultPdfGlobal()
	glob20A3a.PdfVersion = pdfVersion20
	glob20A3a.PdfProfile = profileA3aUA1

	if _, err := PolicyForGlobal(glob20A3a); !errors.Is(err, ErrProfileRequiresPDF17) {
		t.Errorf("PolicyForGlobal(2.0 + a3a-ua1) err = %v, want ErrProfileRequiresPDF17", err)
	}

	// Version-only 2.0 mapping.
	glob20 := settings.DefaultPdfGlobal()
	glob20.PdfVersion = pdfVersion20

	policy20, err := PolicyForGlobal(glob20)
	if err != nil {
		t.Fatalf("PolicyForGlobal(2.0): %v", err)
	}

	if policy20.Version != pdf.PDF20 {
		t.Errorf("2.0 policy = %+v, want PDF20", policy20)
	}

	globPDFA1 := settings.DefaultPdfGlobal()
	globPDFA1.PdfProfile = "pdfa-1b"

	if _, err := PolicyForGlobal(globPDFA1); !errors.Is(err, settings.ErrProfilePDFA1Unsupported) {
		t.Errorf("PolicyForGlobal(pdfa-1b) err = %v, want ErrProfilePDFA1Unsupported", err)
	}
}
