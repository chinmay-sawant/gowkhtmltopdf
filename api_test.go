package gowkhtmltopdf //nolint:testpackage // tests inspect unexported Converter/ObjectSettings state

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"gowkhtmltopdf/internal/settings"
)

// Shared literal values asserted across several tests.
const (
	htmlOriginal = "<p>original</p>"
	valueBefore  = "before"
	valueAfter   = "after"
	valueMutated = "mutated"
)

// writeHTML writes a temp HTML file and returns its path.
func writeHTML(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "input.html")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	return path
}

// newPDFConverter returns a converter wired for local-file input: global
// enable flag on and object-level block flag off (the frozen ACL pair).
func newPDFConverter(t *testing.T, path string) *Converter {
	t.Helper()

	conv := NewConverter()
	if err := conv.Global().Set("enablelocalfileaccess", "true"); err != nil {
		t.Fatalf("global set: %v", err)
	}

	obj := NewObjectSettings().SetPage(path)
	if err := obj.Set("load.blocklocalfileaccess", "false"); err != nil {
		t.Fatalf("object set: %v", err)
	}

	conv.AddObject(obj)

	return conv
}

func TestConvertPDFToBytes(t *testing.T) {
	t.Parallel()
	path := writeHTML(t, "<html><body><h1>Hello</h1><p>world</p></body></html>")

	conv := newPDFConverter(t, path)
	if err := conv.Convert(t.Context()); err != nil {
		t.Fatalf("Convert: %v", err)
	}

	data := conv.Output()
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("output does not start with %%PDF- (got %q)", data[:min(len(data), 8)])
	}

	if !bytes.Contains(data, []byte("/Type /Page")) {
		t.Error("output does not contain /Type /Page")
	}
	// A fresh converter has no output yet.
	if got := NewConverter().Output(); got != nil {
		t.Errorf("fresh Converter.Output() = %d bytes, want nil", len(got))
	}
}

func TestConvertHTMLHelper(t *testing.T) {
	t.Parallel()

	pdf, err := ConvertHTML(t.Context(), []byte("<html><body><p>inline html</p></body></html>"), nil)
	if err != nil {
		t.Fatalf("ConvertHTML: %v", err)
	}

	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("ConvertHTML output is not a PDF")
	}

	global := NewGlobalSettings()
	_ = global.Set("size.pagesize", "Letter")

	pdf2, err := ConvertHTML(t.Context(), []byte("<html><body><p>letter</p></body></html>"), global)
	if err != nil {
		t.Fatalf("ConvertHTML with global: %v", err)
	}

	if !bytes.HasPrefix(pdf2, []byte("%PDF-")) {
		t.Fatal("ConvertHTML+global output is not a PDF")
	}
}

func TestRunPDFTypedRequest(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	req := &PDFRequest{
		Global: NewGlobalSettings(),
		Objects: []*ObjectSettings{
			NewObjectSettings().SetBody(
				[]byte("<html><body><h1>typed PDF</h1></body></html>"),
				"",
			),
		},
		Now:           nil,
		Output:        &output,
		OutlineOutput: nil,
	}

	if err := RunPDF(t.Context(), req); err != nil {
		t.Fatalf("RunPDF: %v", err)
	}

	if !bytes.HasPrefix(output.Bytes(), []byte("%PDF-")) {
		t.Fatalf("typed PDF output does not start with %%PDF-")
	}
}

func TestRunImageTypedRequest(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	imageSettings := NewImageSettings()

	if err := imageSettings.Set("format", "png"); err != nil {
		t.Fatalf("image settings: %v", err)
	}

	req := &ImageRequest{
		Global: NewGlobalSettings(),
		Image:  imageSettings,
		Object: NewObjectSettings().SetBody([]byte("<html><body>typed image</body></html>"), ""),
		Now:    nil,
		Output: &output,
	}

	if err := RunImage(t.Context(), req); err != nil {
		t.Fatalf("RunImage: %v", err)
	}

	if !bytes.HasPrefix(output.Bytes(), []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatalf("typed image output does not start with PNG signature")
	}
}

func TestImageSettingsBackgroundAliasesRoundTrip(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"background", "WEB.BACKGROUND", " web.background "} {
		for _, value := range []string{"false", "true"} {
			t.Run(name+"/"+value, func(t *testing.T) {
				t.Parallel()

				settings := NewImageSettings()
				if err := settings.Set(name, value); err != nil {
					t.Fatalf("Set(%q): %v", name, err)
				}

				for _, alias := range []string{"background", "web.background", " WEB.BACKGROUND "} {
					got, ok := settings.Get(alias)
					if !ok || got != value {
						t.Errorf("Get(%q) = %q, %v; want %q, true", alias, got, ok, value)
					}
				}
			})
		}
	}
}

func TestTypedImageBackgroundAliasAffectsPixels(t *testing.T) {
	t.Parallel()

	const html = `<html><body style="margin:0;background-color:#336699;width:100px;height:100px"></body></html>`

	render := func(background string) color.NRGBA {
		var output bytes.Buffer

		imageSettings := NewImageSettings()

		if err := imageSettings.Set(" WEB.BACKGROUND ", background); err != nil {
			t.Fatalf("Set background: %v", err)
		}

		if err := imageSettings.Set("width", "100"); err != nil {
			t.Fatalf("Set width: %v", err)
		}

		if err := imageSettings.Set("height", "100"); err != nil {
			t.Fatalf("Set height: %v", err)
		}

		request := &ImageRequest{ //nolint:exhaustruct // focused typed request
			Image:  imageSettings,
			Object: NewObjectSettings().SetBody([]byte(html), ""),
			Output: &output,
		}
		if err := RunImage(t.Context(), request); err != nil {
			t.Fatalf("RunImage(%s): %v", background, err)
		}

		decoded, err := png.Decode(bytes.NewReader(output.Bytes()))
		if err != nil {
			t.Fatalf("decode PNG: %v", err)
		}

		return pixelAt(decoded, 50, 50)
	}

	if got := render("true"); got != (color.NRGBA{R: 0x33, G: 0x66, B: 0x99, A: 0xff}) {
		t.Fatalf("background=true center pixel = %v, want #336699", got)
	}

	if got := render("false"); got == (color.NRGBA{R: 0x33, G: 0x66, B: 0x99, A: 0xff}) {
		t.Fatalf("background=false retained body background pixel %v", got)
	}
}

//nolint:funlen // matrix cases are intentionally explicit
func TestTypedPDFRequestPreflight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		make func(*countingTestWriter) *PDFRequest
		want error
	}{
		{
			name: "no objects",
			make: func(output *countingTestWriter) *PDFRequest {
				return &PDFRequest{Output: output} //nolint:exhaustruct // focused invalid request
			},
			want: ErrNoRenderablePDFObjects,
		},
		{
			name: "toc only",
			make: func(output *countingTestWriter) *PDFRequest {
				return &PDFRequest{ //nolint:exhaustruct // focused invalid request
					Objects: []*ObjectSettings{{o: settings.PdfObject{IsTableOfContent: true}}}, //nolint:exhaustruct
					Output:  output,
				}
			},
			want: ErrNoRenderablePDFObjects,
		},
		{
			name: "empty object",
			make: func(output *countingTestWriter) *PDFRequest {
				return &PDFRequest{Objects: []*ObjectSettings{NewObjectSettings()}, Output: output} //nolint:exhaustruct
			},
			want: ErrNoRenderablePDFObjects,
		},
		{
			name: "missing output",
			make: func(*countingTestWriter) *PDFRequest {
				object := NewObjectSettings().SetBody([]byte("<p>body</p>"), "")

				return &PDFRequest{Objects: []*ObjectSettings{object}} //nolint:exhaustruct
			},
			want: ErrMissingPDFOutput,
		},
		{
			name: "missing outline output",
			make: func(output *countingTestWriter) *PDFRequest {
				global := NewGlobalSettings()
				global.g.DumpOutline = true

				return &PDFRequest{ //nolint:exhaustruct // focused invalid request
					Global:  global,
					Objects: []*ObjectSettings{NewObjectSettings().SetBody([]byte("<p>body</p>"), "")},
					Output:  output,
				}
			},
			want: ErrMissingPDFOutlineOutput,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var output countingTestWriter
			req := testCase.make(&output)

			if err := req.ValidatePDF(); !errors.Is(err, testCase.want) {
				t.Fatalf("ValidatePDF() = %v, want errors.Is(..., %v)", err, testCase.want)
			}

			if err := RunPDF(t.Context(), req); !errors.Is(err, testCase.want) {
				t.Fatalf("RunPDF() = %v, want errors.Is(..., %v)", err, testCase.want)
			}

			if output.Writes != 0 {
				t.Fatalf("invalid request wrote %d times", output.Writes)
			}
		})
	}
}

func TestTypedImageRequestPreflight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		make func(*countingTestWriter) *ImageRequest
		want error
	}{
		{
			name: "no object",
			make: func(output *countingTestWriter) *ImageRequest {
				return &ImageRequest{Image: NewImageSettings(), Output: output} //nolint:exhaustruct
			},
			want: ErrNoInputPageAdded,
		},
		{
			name: "toc only",
			make: func(output *countingTestWriter) *ImageRequest {
				return &ImageRequest{ //nolint:exhaustruct // focused invalid request
					Image:  NewImageSettings(),
					Object: &ObjectSettings{o: settings.PdfObject{IsTableOfContent: true}}, //nolint:exhaustruct
					Output: output,
				}
			},
			want: ErrNoInputPageAdded,
		},
		{
			name: "missing output",
			make: func(*countingTestWriter) *ImageRequest {
				return &ImageRequest{ //nolint:exhaustruct // focused invalid request
					Image:  NewImageSettings(),
					Object: NewObjectSettings().SetBody([]byte("<p>body</p>"), ""),
				}
			},
			want: ErrMissingImageOutput,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var output countingTestWriter
			req := testCase.make(&output)

			if err := req.ValidateImage(); !errors.Is(err, testCase.want) {
				t.Fatalf("ValidateImage() = %v, want errors.Is(..., %v)", err, testCase.want)
			}

			if err := RunImage(t.Context(), req); !errors.Is(err, testCase.want) {
				t.Fatalf("RunImage() = %v, want errors.Is(..., %v)", err, testCase.want)
			}

			if output.Writes != 0 {
				t.Fatalf("invalid request wrote %d times", output.Writes)
			}
		})
	}
}

type countingTestWriter struct {
	bytes.Buffer
	Writes int
}

func (w *countingTestWriter) Write(payload []byte) (int, error) {
	w.Writes++

	return w.Buffer.Write(payload) //nolint:wrapcheck // counting writer is test-only
}

func TestGlobalSettingsGetSetRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct{ name, value string }{
		{"size.pagesize", "Letter"},
		{"orientation", "Landscape"},
		{"margin.top", "12.5"},
		{"web.background", "false"},
		{"grayscale", "true"},
		{"title", "Round Trip"},
		{"enablelocalfileaccess", "true"},
		{"header.center", "[title]"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assertGlobalSetGetRoundTrip(t, testCase.name, testCase.value)
		})
	}

	assertGlobalDefaults(t)
}

// assertGlobalSetGetRoundTrip pins one Set/Get round trip for a global key.
func assertGlobalSetGetRoundTrip(t *testing.T, name, value string) {
	t.Helper()

	global := NewGlobalSettings()
	if err := global.Set(name, value); err != nil {
		t.Fatalf("Set(%q): %v", name, err)
	}

	got, ok := global.Get(name)
	if !ok {
		t.Fatalf("Get(%q) not found after Set", name)
	}

	if got != value {
		t.Errorf("Get(%q) = %q, want %q", name, got, value)
	}
}

// assertGlobalDefaults pins that defaults are readable without any Set call
// and that bogus keys fail.
func assertGlobalDefaults(t *testing.T) {
	t.Helper()

	global := NewGlobalSettings()
	for name, want := range map[string]string{
		"size.pagesize":  "A4",
		"orientation":    "Portrait",
		"margin.top":     "10",
		"margin.bottom":  "10",
		"web.background": "true",
	} {
		got, ok := global.Get(name)
		if !ok || got != want {
			t.Errorf("default Get(%q) = %q, %v; want %q, true", name, got, ok, want)
		}
	}

	if _, ok := global.Get("bogus.key"); ok {
		t.Error("Get(bogus.key) found, want not found")
	}

	if err := global.Set("bogus.key", "x"); err == nil {
		t.Error("Set(bogus.key) succeeded, want error")
	}

	if err := global.Set("margin.top", "not-a-length"); err == nil {
		t.Error("Set(margin.top, not-a-length) succeeded, want error")
	}
}

func TestObjectSettingsGetSet(t *testing.T) {
	t.Parallel()

	obj := NewObjectSettings().SetPage("in.html")
	if got, _ := obj.Get("page"); got != "in.html" {
		t.Errorf("Get(page) = %q, want in.html", got)
	}

	if err := obj.Set("load.blocklocalfileaccess", "false"); err != nil {
		t.Fatalf("Set(load.blocklocalfileaccess): %v", err)
	}

	if got, ok := obj.Get("load.blocklocalfileaccess"); !ok || got != "false" {
		t.Errorf("Get(load.blocklocalfileaccess) = %q, %v; want false, true", got, ok)
	}

	if err := obj.Set("footer.right", "[page]"); err != nil {
		t.Fatalf("Set(footer.right): %v", err)
	}

	if got, ok := obj.Get("footer.right"); !ok || got != "[page]" {
		t.Errorf("Get(footer.right) = %q, %v; want [page], true", got, ok)
	}

	if err := obj.Set("unknown.key", "x"); err == nil {
		t.Error("Set(unknown.key) succeeded, want error")
	}
}

func TestObjectSettingsSetBodyCopiesInput(t *testing.T) {
	t.Parallel()

	html := []byte(htmlOriginal)
	obj := NewObjectSettings().SetBody(html, "https://example.test/")
	html[4] = 'X'

	if got := string(obj.o.Load.InlineHTML); got != htmlOriginal {
		t.Fatalf("SetBody retained caller buffer: %q", got)
	}
}

func TestConverterAddObjectDeepCopiesNestedData(t *testing.T) {
	t.Parallel()

	source := NewObjectSettings().SetBody([]byte(htmlOriginal), "https://example.test/")
	source.o.Load.CustomHeaders = map[string]string{"X-Request": valueBefore}
	source.o.Load.Cookies = map[string]string{"session": valueBefore}
	source.o.Load.Post = []settings.PostItem{{Name: "q", Value: valueBefore}}
	source.o.Header.Replace = map[string]string{"[name]": valueBefore}
	source.o.Footer.Replace = map[string]string{"[page]": valueBefore}
	source.o.Ignored = map[string]string{"stub": valueBefore}

	conv := NewConverter().AddObject(source)

	source.o.Load.InlineHTML[3] = 'X'
	source.o.Load.CustomHeaders["X-Request"] = valueAfter
	source.o.Load.CustomHeaders["X-New"] = valueAfter
	source.o.Load.Cookies["session"] = valueAfter
	source.o.Load.Post[0].Value = valueAfter
	source.o.Header.Replace["[name]"] = valueAfter
	source.o.Footer.Replace["[page]"] = valueAfter
	source.o.Ignored["stub"] = valueAfter
	source.o.Page = "after.html"

	assertSnapshotUntouched(t, conv.objects[0].o)
}

// assertSnapshotUntouched pins that the AddObject copy was not affected by
// mutating the source object afterwards.
func assertSnapshotUntouched(t *testing.T, got settings.PdfObject) {
	t.Helper()

	if string(got.Load.InlineHTML) != htmlOriginal {
		t.Errorf("inline HTML changed through source mutation: %q", got.Load.InlineHTML)
	}

	if got.Load.CustomHeaders["X-Request"] != valueBefore || got.Load.CustomHeaders["X-New"] != "" {
		t.Errorf("custom headers were not copied: %v", got.Load.CustomHeaders)
	}

	if got.Load.Cookies["session"] != valueBefore {
		t.Errorf("cookies were not copied: %v", got.Load.Cookies)
	}

	if len(got.Load.Post) != 1 || got.Load.Post[0].Value != valueBefore {
		t.Errorf("post data was not copied: %v", got.Load.Post)
	}

	assertSnapshotHeaderFooterUntouched(t, got)

	if got.Ignored["stub"] != valueBefore || got.Page != "" {
		t.Errorf("object snapshot changed through source mutation: %+v", got)
	}
}

// assertSnapshotHeaderFooterUntouched pins the header/footer replace copies.
func assertSnapshotHeaderFooterUntouched(t *testing.T, got settings.PdfObject) {
	t.Helper()

	if got.Header.Replace["[name]"] != valueBefore || got.Footer.Replace["[page]"] != valueBefore {
		t.Errorf(
			"header/footer replacements were not copied: header=%v footer=%v",
			got.Header.Replace, got.Footer.Replace,
		)
	}
}

func TestConverterAddObjectSnapshotIsIndependentUnderMutation(t *testing.T) {
	t.Parallel()

	source := NewObjectSettings().SetBody([]byte(htmlOriginal), "")
	source.o.Load.CustomHeaders = map[string]string{"X-Request": valueBefore}
	source.o.Load.Post = []settings.PostItem{{Name: "q", Value: valueBefore}}
	source.o.Header.Replace = map[string]string{"[name]": valueBefore}
	conv := NewConverter().AddObject(source)
	snapshot := conv.objects[0].o

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		for i := range 1000 {
			source.o.Load.InlineHTML[0] = byte('a' + i%26)
			source.o.Load.CustomHeaders["X-Request"] = valueMutated
			source.o.Load.Post[0].Value = valueMutated
			source.o.Header.Replace["[name]"] = valueMutated
		}
	}()

	for range 1000 {
		_ = snapshot.Load.InlineHTML[0]
		_ = snapshot.Load.CustomHeaders["X-Request"]
		_ = snapshot.Load.Post[0].Value
		_ = snapshot.Header.Replace["[name]"]
	}

	waitGroup.Wait()

	if got := string(snapshot.Load.InlineHTML); got != htmlOriginal {
		t.Errorf("snapshot changed during source mutation: %q", got)
	}

	if snapshot.Load.CustomHeaders["X-Request"] != valueBefore ||
		snapshot.Load.Post[0].Value != valueBefore || snapshot.Header.Replace["[name]"] != valueBefore {
		t.Errorf("snapshot nested data changed during source mutation: %+v", snapshot)
	}
}

func TestConvertContextCancel(t *testing.T) {
	t.Parallel()
	path := writeHTML(t, "<html><body><p>cancel me</p></body></html>")
	conv := newPDFConverter(t, path)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if err := conv.Convert(ctx); err == nil {
		t.Fatal("Convert with canceled context succeeded, want error")
	}
}

func TestConverterCallbacks(t *testing.T) {
	t.Parallel()
	path := writeHTML(t, "<html><body><h1>callbacks</h1></body></html>")
	conv := newPDFConverter(t, path)

	var phases, infos []string

	var progs []int

	conv.OnPhase = func(p string) { phases = append(phases, p) }
	conv.OnProgress = func(p int) { progs = append(progs, p) }
	conv.OnInfo = func(line string) { infos = append(infos, line) }

	if err := conv.Convert(t.Context()); err != nil {
		t.Fatalf("Convert: %v", err)
	}

	if len(phases) == 0 {
		t.Error("OnPhase never called")
	}

	if len(progs) == 0 {
		t.Error("OnProgress never called")
	}

	if len(infos) == 0 {
		t.Error("OnInfo never called")
	}

	if last := progs[len(progs)-1]; last != 100 {
		t.Errorf("last progress = %d, want 100", last)
	}

	if !bytes.HasPrefix(conv.Output(), []byte("%PDF-")) {
		t.Error("Output after Convert is not a PDF")
	}
}

func TestImageConverterPNG(t *testing.T) {
	t.Parallel()
	path := writeHTML(t, `<html><body><div style="width:120px;height:80px;background-color:#336699"></div></body></html>`)

	conv := NewImageConverter()
	if err := conv.Global().Set("enablelocalfileaccess", "true"); err != nil {
		t.Fatalf("global set: %v", err)
	}

	conv.AddObject(path)

	if err := conv.Object().Set("load.blocklocalfileaccess", "false"); err != nil {
		t.Fatalf("object set: %v", err)
	}

	if err := conv.Set("width", "200"); err != nil {
		t.Fatalf("Set(width): %v", err)
	}

	if err := conv.Convert(t.Context()); err != nil {
		t.Fatalf("Convert: %v", err)
	}

	img, err := png.Decode(bytes.NewReader(conv.Output()))
	if err != nil {
		t.Fatalf("output is not a decodable PNG: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 200 {
		t.Errorf("width = %d, want 200 (viewport)", bounds.Dx())
	}

	if bounds.Dy() < 80 {
		t.Errorf("height = %d, want >= 80 (content + body margins)", bounds.Dy())
	}
	// The 120x80 #336699 div sits below the 8px body margin.
	if got := pixelAt(img, 60, 40); got != (color.NRGBA{R: 0x33, G: 0x66, B: 0x99, A: 0xff}) {
		t.Errorf("div pixel = %v, want #336699", got)
	}
}

func TestImageConverterJPEG(t *testing.T) {
	t.Parallel()
	path := writeHTML(t, `<html><body><p>hello image</p></body></html>`)

	conv := NewImageConverter()
	if err := conv.Global().Set("enablelocalfileaccess", "true"); err != nil {
		t.Fatalf("global set: %v", err)
	}

	conv.AddObject(path)

	if err := conv.Object().Set("load.blocklocalfileaccess", "false"); err != nil {
		t.Fatalf("object set: %v", err)
	}

	if err := conv.Set("format", "jpg"); err != nil {
		t.Fatalf("Set(format): %v", err)
	}

	if err := conv.Convert(t.Context()); err != nil {
		t.Fatalf("Convert: %v", err)
	}

	if _, err := jpeg.Decode(bytes.NewReader(conv.Output())); err != nil {
		t.Fatalf("output is not a decodable JPEG: %v", err)
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()

	v := Version()
	if !strings.Contains(v, LibraryVersion) {
		t.Errorf("Version() = %q, want it to contain LibraryVersion %q", v, LibraryVersion)
	}
}

// TestLineLogSeverity pins the marker-prefix severity protocol: the
// grammar lives in internal/line, the callback mapping stays in api.go,
// and a line whose *message* mentions "error" is not an error line.
func TestLineLogSeverity(t *testing.T) {
	t.Parallel()

	var infos, warns, errs []string

	logWriter := &lineLog{ //nolint:exhaustruct // intentional zero/partial fields
		onInfo:  func(l string) { infos = append(infos, l) },
		onWarn:  func(l string) { warns = append(warns, l) },
		onError: func(l string) { errs = append(errs, l) },
	}
	_, _ = logWriter.Write([]byte("Loading pages (1/1)\n"))
	_, _ = logWriter.Write([]byte("warning: object 1: large stylesheet volume\n"))
	_, _ = logWriter.Write([]byte("error: failed to load http://x\n"))
	_, _ = logWriter.Write([]byte("info: load error policy is skip, omitting\n"))

	if len(infos) != 2 {
		t.Errorf("infos = %v, want 2 lines", infos)
	}

	if len(warns) != 1 || warns[0] != "warning: object 1: large stylesheet volume" {
		t.Errorf("warns = %v", warns)
	}

	if len(errs) != 1 || errs[0] != "error: failed to load http://x" {
		t.Errorf("errs = %v", errs)
	}
}

func TestImageConverterNeedsPage(t *testing.T) {
	t.Parallel()

	conv := NewImageConverter()
	if err := conv.Convert(t.Context()); err == nil {
		t.Fatal("Convert without a page succeeded, want error")
	}
}

// TestImageConverterSetBody: P2-04 InlineHTML source kind works for image mode
// via Object().SetBody (no temp file, no URL guessing).
func TestImageConverterSetBody(t *testing.T) {
	t.Parallel()

	conv := NewImageConverter()
	conv.Object().SetBody(
		[]byte(`<html><body><div style="width:40px;height:30px;background-color:#112233"></div></body></html>`),
		"",
	)

	if err := conv.Set("width", "100"); err != nil {
		t.Fatalf("Set(width): %v", err)
	}

	if err := conv.Convert(t.Context()); err != nil {
		t.Fatalf("Convert: %v", err)
	}

	img, err := png.Decode(bytes.NewReader(conv.Output()))
	if err != nil {
		t.Fatalf("output is not a decodable PNG: %v", err)
	}

	if img.Bounds().Dx() != 100 {
		t.Errorf("width = %d, want 100", img.Bounds().Dx())
	}
}

func TestConverterNeedsObject(t *testing.T) {
	t.Parallel()

	if err := NewConverter().Convert(t.Context()); err == nil {
		t.Fatal("Convert without objects succeeded, want error")
	}
}

// pixelAt returns the NRGBA colour at (x, y).
func pixelAt(img image.Image, x, y int) color.NRGBA {
	c, ok := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
	if !ok {
		return color.NRGBA{} //nolint:exhaustruct // impossible for NRGBAModel; zero colour fallback
	}

	return c
}
