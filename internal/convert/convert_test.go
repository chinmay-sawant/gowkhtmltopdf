package convert

import (
	"bytes"
	"compress/flate"
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
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/layout"
	"gowkhtmltopdf/internal/settings"
)

// newCommand writes html into a temp dir and returns a cli.Command pointing
// at it, with local file access enabled (the frozen ACL default blocks local
// reads unless the user opts in).
func newCommand(t *testing.T, html string, output string) (*cli.Command, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "input.html")
	if err := os.WriteFile(path, []byte(html), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	cmd := &cli.Command{
		Global: settings.DefaultPdfGlobal(),
		Objects: []settings.PdfObject{
			{Page: path, Load: settings.DefaultLoadPage()},
		},
		Output: output,
	}
	// --enable-local-file-access: global flag on, object-level block off.
	cmd.Global.EnableLocalFileAccess = true
	cmd.Objects[0].Load.BlockLocalFileAccess = false
	cmd.Global.Size = settings.Size{PageSize: cmd.Global.PageSize}
	return cmd, dir
}

// runPDF runs RunPDF and returns the produced PDF bytes. When output is "-"
// the PDF lands on os.Stdout, so the caller must have redirected it.
func runPDF(t *testing.T, cmd *cli.Command) []byte {
	t.Helper()
	var log bytes.Buffer
	if err := RunPDF(cmd, &log); err != nil {
		t.Fatalf("RunPDF: %v", err)
	}
	data, err := os.ReadFile(cmd.Output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	return data
}

// pageCount counts page dicts. The pages tree root is emitted as
// "/Type /Pages\n", which never matches "/Type /Page\n".
func pageCount(data []byte) int {
	return bytes.Count(data, []byte("/Type /Page\n"))
}

func TestRunPDFSinglePageA4(t *testing.T) {
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
	var sb strings.Builder
	sb.WriteString("<html><body>")
	for i := 0; i < 200; i++ {
		sb.WriteString("<p>paragraph of text number ")
		sb.WriteString(string(rune('a' + i%26)))
		sb.WriteString(" with some words to wrap</p>")
	}
	sb.WriteString("</body></html>")
	cmd, _ := newCommand(t, sb.String(), filepath.Join(t.TempDir(), "out.pdf"))
	data := runPDF(t, cmd)
	if n := pageCount(data); n < 2 {
		t.Errorf("pages = %d, want >= 2", n)
	}
}

func TestRunPDFStyleTableImage(t *testing.T) {
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

func TestRunPDFLinkedStylesheet(t *testing.T) {
	cmd, dir := newCommand(t, `<html><head><link rel="stylesheet" href="style.css"></head><body><div class="box">styled</div></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	if err := os.WriteFile(filepath.Join(dir, "style.css"), []byte(".box { background-color: #000000; }"), 0o644); err != nil {
		t.Fatalf("write css: %v", err)
	}
	data := runPDF(t, cmd)
	if n := pageCount(data); n != 1 {
		t.Errorf("pages = %d, want 1", n)
	}
}

func TestRunPDFScreenOnlyStylesheetExcluded(t *testing.T) {
	cmd, dir := newCommand(t, `<html><head><link rel="stylesheet" href="screen.css" media="screen"></head><body><div class="box">styled</div></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	if err := os.WriteFile(filepath.Join(dir, "screen.css"), []byte(".box { background-color: #000000; }"), 0o644); err != nil {
		t.Fatalf("write css: %v", err)
	}
	if err := RunPDF(cmd, &bytes.Buffer{}); err != nil {
		t.Fatalf("RunPDF: %v", err)
	}
}

func TestRunPDFOutputStdout(t *testing.T) {
	cmd, _ := newCommand(t, `<html><body><p>stdout test</p></body></html>`, "-")
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	runErr := RunPDF(cmd, &bytes.Buffer{})
	w.Close()
	os.Stdout = old
	if runErr != nil {
		t.Fatalf("RunPDF: %v", runErr)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout capture: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("stdout output is not a PDF")
	}
	if n := pageCount(data); n != 1 {
		t.Errorf("pages = %d, want 1", n)
	}
}

func TestRunPDFMissingFile(t *testing.T) {
	cmd, _ := newCommand(t, `<html><body>x</body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Objects[0].Page = filepath.Join(t.TempDir(), "does-not-exist.html")
	if err := RunPDF(cmd, &bytes.Buffer{}); err == nil {
		t.Fatal("expected error for missing input file, got nil")
	}
}

// pngDataURL builds a minimal valid RGBA PNG as a data: URL.
func pngDataURL(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
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
func newCommandMulti(t *testing.T, htmls []string, output string) *cli.Command {
	t.Helper()
	cmd := &cli.Command{
		Global: settings.DefaultPdfGlobal(),
		Output: output,
	}
	cmd.Global.EnableLocalFileAccess = true
	cmd.Global.Size = settings.Size{PageSize: cmd.Global.PageSize}
	for _, h := range htmls {
		dir := t.TempDir()
		path := filepath.Join(dir, "input.html")
		if err := os.WriteFile(path, []byte(h), 0o644); err != nil {
			t.Fatalf("write input: %v", err)
		}
		obj := settings.PdfObject{Page: path, Load: settings.DefaultLoadPage()}
		obj.Load.BlockLocalFileAccess = false
		cmd.Objects = append(cmd.Objects, obj)
	}
	return cmd
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
// builds only — the body is plain text).
func objectDict(t *testing.T, data []byte, ref int) []byte {
	t.Helper()
	marker := fmt.Sprintf("%d 0 obj\n", ref)
	i := bytes.Index(data, []byte(marker))
	if i < 0 {
		t.Fatalf("object %d not found", ref)
	}
	end := bytes.Index(data[i:], []byte("\nendobj"))
	if end < 0 {
		t.Fatalf("object %d has no endobj", ref)
	}
	return data[i : i+end]
}

// pageLabel resolves a page object to the marker text in its content stream
// ("AAA" → "A", "BBB" → "B"), so the /Kids order can be asserted
// semantically instead of by raw object id.
func pageLabel(t *testing.T, data []byte, pageRef int) string {
	t.Helper()
	dict := objectDict(t, data, pageRef)
	m := regexp.MustCompile(`/Contents (\d+) 0 R`).FindSubmatch(dict)
	if m == nil {
		t.Fatalf("page %d dict has no /Contents: %q", pageRef, dict)
	}
	contentRef, err := strconv.Atoi(string(m[1]))
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
	var labels []string
	for _, ref := range kidsRefs(t, data) {
		labels = append(labels, pageLabel(t, data, ref))
	}
	return labels
}

func TestRunPDFCopiesCollate(t *testing.T) {
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
	cmd, _ := newCommand(t, `<html><body><p>progress</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	var phases []string
	percents := map[string]int{}
	collect := func(phase string, percent int) {
		phases = append(phases, phase)
		percents[phase] = percent
	}
	var log bytes.Buffer
	if err := RunPDFContext(context.Background(), cmd, &log, collect); err != nil {
		t.Fatalf("RunPDFContext: %v", err)
	}
	want := []string{
		"Loading pages (1/1)",
		"Counting pages (1/1)",
		"Resolving links (1/1)",
		"Printing pages (1/1)",
		"Done",
	}
	if !slices.Equal(phases, want) {
		t.Errorf("phases = %v, want %v", phases, want)
	}
	if p := percents["Done"]; p != 100 {
		t.Errorf("Done percent = %d, want 100", p)
	}
	for _, phase := range want {
		if p := percents[phase]; p < 0 || p > 100 {
			t.Errorf("phase %q percent %d out of range", phase, p)
		}
	}
	if !strings.Contains(log.String(), "Loading pages (1/1)") {
		t.Error("progress phase not written to log")
	}
}

func TestRunPDFQuiet(t *testing.T) {
	cmd, _ := newCommand(t, `<html><body><p>quiet</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Quiet = true
	var log bytes.Buffer
	if err := RunPDFContext(context.Background(), cmd, &log, nil); err != nil {
		t.Fatalf("RunPDFContext: %v", err)
	}
	if log.Len() != 0 {
		t.Errorf("quiet mode wrote %d bytes to log: %q", log.Len(), log.String())
	}
}

func TestRunPDFContextCancel(t *testing.T) {
	cmd, _ := newCommand(t, `<html><body><p>cancel</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := RunPDFContext(ctx, cmd, &bytes.Buffer{}, nil)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

// --- smart shrinking (plan 5.3) ---

// layoutZoomAvailable reports whether internal/layout.Options has gained the
// Zoom field (added by the parallel layout agent). The re-layout code in
// convert.go and this test activate once it lands.
func layoutZoomAvailable() bool {
	_, ok := reflect.TypeOf(layout.Options{}).FieldByName("Zoom")
	return ok
}

func TestRunPDFSmartShrinking(t *testing.T) {
	if !layoutZoomAvailable() {
		t.Skip("smart shrinking re-layout needs internal/layout.Options.Zoom (TODO); only the over-width warning is emitted today")
	}
	// 2000px fixed-width div is ~1500pt wide, far beyond the A4 content area.
	cmd, _ := newCommand(t,
		`<html><body><div style="background-color:#336699; width:2000px; height:50px;">wide</div></body></html>`,
		filepath.Join(t.TempDir(), "out.pdf"))
	var log bytes.Buffer
	if err := RunPDF(cmd, &log); err != nil {
		t.Fatalf("RunPDF: %v", err)
	}
	if !bytes.Contains(log.Bytes(), []byte("smart shrinking")) {
		t.Errorf("expected a smart-shrinking log line, log: %q", log.String())
	}
	data, err := os.ReadFile(cmd.Output)
	if err != nil {
		t.Fatalf("read output: %v", err)
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

// decodeStreams decompresses every FlateDecode stream in a PDF byte blob.
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
		r := flate.NewReader(bytes.NewReader(raw))
		dec, derr := io.ReadAll(r)
		r.Close()
		if derr == nil {
			out = append(out, dec...)
		}
	}
	return out
}
