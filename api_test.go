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

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/settings"
)

// Shared literal values asserted across several tests.
const (
	htmlOriginal       = "<p>original</p>"
	valueBefore        = "before"
	valueAfter         = "after"
	mutatedNetworkHost = "mutated.example.test"
	valueMutated       = "mutated"
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

func TestPdfGlobalOptionsBuildSnapshot(t *testing.T) {
	t.Parallel()

	builder := NewPdfGlobalOptions().
		WithPageSize("Letter").
		WithMargins(1, 2, 3, 4).
		WithTitle(valueBefore).
		WithCopies(2, false).
		WithOutline(false, 6).
		WithSmartShrinking(false).
		WithBackground(false).
		WithCompression(false).
		WithResolveRelativeLinks(false)
	if _, err := builder.WithSetting("fontpath", "font-before"); err != nil {
		t.Fatalf("WithSetting(fontpath): %v", err)
	}

	if _, err := builder.WithSetting("dpi", valueBefore); err != nil {
		t.Fatalf("WithSetting(dpi): %v", err)
	}

	snapshot := builder.Build()
	if snapshot == nil {
		t.Fatal("Build() returned nil")
	}

	builder.WithPageSize("A4").WithTitle(valueAfter)

	if got, _ := snapshot.Get("size.pagesize"); got != "Letter" {
		t.Fatalf("snapshot page size = %q, want Letter", got)
	}

	if got, _ := snapshot.Get("title"); got != valueBefore {
		t.Fatalf("snapshot title = %q, want %q", got, valueBefore)
	}

	if got, _ := snapshot.Get("margin.top"); got != "1" {
		t.Fatalf("snapshot top margin = %q, want 1", got)
	}

	assertSnapshotFields(t, snapshot, []snapshotExpectation{
		{name: "copies", want: "2"},
		{name: "collate", want: "false"},
		{name: "outline", want: "false"},
		{name: "outlinedepth", want: "6"},
		{name: "smartshrinking", want: "false"},
		{name: "background", want: "false"},
		{name: "usecompression", want: "false"},
		{name: "resolverelativelinks", want: "false"},
		{name: "fontpath", want: "font-before"},
		{name: "dpi", want: valueBefore},
	})

	mutateAndVerifyGlobalOptionsSnapshot(t, builder, snapshot)
}

type snapshotExpectation struct {
	name, want string
}

func assertSnapshotFields(t *testing.T, snapshot *GlobalSettings, expectations []snapshotExpectation) {
	t.Helper()

	for _, testCase := range expectations {
		if got, ok := snapshot.Get(testCase.name); !ok || got != testCase.want {
			t.Errorf("snapshot Get(%q) = %q, %v; want %q, true", testCase.name, got, ok, testCase.want)
		}
	}
}

func mutateAndVerifyGlobalOptionsSnapshot(t *testing.T, builder *PdfGlobalOptions, snapshot *GlobalSettings) {
	t.Helper()

	if err := snapshot.Set("title", valueAfter); err != nil {
		t.Fatalf("mutate snapshot title: %v", err)
	}

	if _, err := builder.WithSetting("fontpath", "font-after"); err != nil {
		t.Fatalf("mutate builder fontpath: %v", err)
	}

	if _, err := builder.WithSetting("dpi", valueAfter); err != nil {
		t.Fatalf("mutate builder dpi: %v", err)
	}

	if got, _ := builder.Build().Get("title"); got != valueAfter {
		t.Fatalf("builder title = %q, want %q after fluent mutation", got, valueAfter)
	}

	if got, _ := builder.Build().Get("size.pagesize"); got != "A4" {
		t.Fatalf("builder page size = %q, want A4 after fluent mutation", got)
	}

	if got, _ := snapshot.Get("fontpath"); got != "font-before" {
		t.Fatalf("snapshot fontpath = %q, want font-before after builder mutation", got)
	}

	if got, _ := snapshot.Get("dpi"); got != valueBefore {
		t.Fatalf("snapshot dpi = %q, want %q after builder mutation", got, valueBefore)
	}
}

func TestPdfGlobalOptionsBuildsConverter(t *testing.T) {
	t.Parallel()

	global := NewPdfGlobalOptions().
		WithPageSize("Letter").
		WithTitle("builder integration").
		Build()
	setGlobalSettings(t, global, map[string]string{
		"fontpath": "font-before",
		"dpi":      valueBefore,
	})

	conv := NewConverter().
		WithGlobal(global).
		AddObject(NewObjectSettings().SetBody(
			[]byte("<html><body><p>builder integration</p></body></html>"), "",
		))

	setGlobalSettings(t, global, map[string]string{
		"size.pagesize": "A4",
		"title":         "mutated source",
		"fontpath":      "font-after",
		"dpi":           valueAfter,
	})

	assertGlobalSettings(t, conv, map[string]string{
		"size.pagesize": "Letter",
		"title":         "builder integration",
		"fontpath":      "font-before",
		"dpi":           valueBefore,
	})

	if err := conv.Convert(t.Context()); err != nil {
		t.Fatalf("Convert: %v", err)
	}

	data := conv.Output()
	if !bytes.Contains(data, []byte("/MediaBox [0 0 612 792]")) {
		t.Fatalf("converter output did not use Letter geometry")
	}

	if !bytes.Contains(data, []byte("/Title (builder integration)")) {
		t.Fatalf("converter output did not use builder title")
	}
}

func TestConverterWithGlobalNilPolicy(t *testing.T) {
	t.Parallel()

	conv := NewConverter()
	original := conv.Global()

	if got := conv.WithGlobal(nil); got != conv {
		t.Fatalf("WithGlobal(nil) returned %p, want converter %p", got, conv)
	}

	if conv.Global() != original {
		t.Fatal("WithGlobal(nil) replaced converter settings")
	}

	var nilConverter *Converter
	if got := nilConverter.WithGlobal(NewGlobalSettings()); got != nil {
		t.Fatalf("nil converter WithGlobal returned %p, want nil", got)
	}
}

func setGlobalSettings(t *testing.T, global *GlobalSettings, settings map[string]string) {
	t.Helper()

	for key, value := range settings {
		if err := global.Set(key, value); err != nil {
			t.Fatalf("set %s: %v", key, err)
		}
	}
}

func assertGlobalSettings(t *testing.T, conv *Converter, wants map[string]string) {
	t.Helper()

	for key, want := range wants {
		got, _ := conv.Global().Get(key)
		if got != want {
			t.Fatalf("converter %s = %q, want %q", key, got, want)
		}
	}
}

func TestPdfGlobalOptionsNilReceiverPanics(t *testing.T) {
	t.Parallel()

	var options *PdfGlobalOptions

	for _, testCase := range []struct {
		name string
		call func()
	}{
		{name: "page size", call: func() { options.WithPageSize("A4") }},
		{name: "margins", call: func() { options.WithMargins(1, 2, 3, 4) }},
		{name: "title", call: func() { options.WithTitle("invalid") }},
		{name: "copies", call: func() { options.WithCopies(1, true) }},
		{name: "outline", call: func() { options.WithOutline(true, 4) }},
		{name: "smart shrinking", call: func() { options.WithSmartShrinking(true) }},
		{name: "background", call: func() { options.WithBackground(true) }},
		{name: "compression", call: func() { options.WithCompression(true) }},
		{name: "relative links", call: func() { options.WithResolveRelativeLinks(true) }},
		{name: "setting", call: func() { _, _ = options.WithSetting("title", "invalid") }},
		{name: "build", call: func() { _ = options.Build() }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if recover() == nil {
					t.Fatal("nil builder call did not panic")
				}
			}()
			testCase.call()
		})
	}
}

func TestPdfGlobalOptionsRejectsUnknownPageSize(t *testing.T) {
	t.Parallel()

	defer func() {
		if recovered, ok := recover().(error); !ok || !errors.Is(recovered, ErrInvalidPageSize) {
			t.Fatal("invalid page size did not panic")
		}
	}()

	NewPdfGlobalOptions().WithPageSize("not-a-page-size")
}

func TestPdfGlobalOptionsValidationPaths(t *testing.T) {
	t.Parallel()

	if _, err := NewPdfGlobalOptions().WithSetting("size.pagesize", "not-a-page-size"); !errors.Is(
		err, ErrInvalidPageSize,
	) {
		t.Fatalf("WithSetting invalid page size = %v, want errors.Is(..., %v)", err, ErrInvalidPageSize)
	}

	if _, err := NewPdfGlobalOptions().WithSetting("copies", "0"); !errors.Is(err, ErrInvalidPDFCopies) {
		t.Fatalf("WithSetting invalid copies = %v, want errors.Is(..., %v)", err, ErrInvalidPDFCopies)
	}

	global := NewGlobalSettings()
	if err := global.Set("size.pagesize", ""); err != nil {
		t.Fatalf("empty page size: %v", err)
	}

	if got, ok := global.Get("size.pagesize"); !ok || got != "A4" {
		t.Fatalf("empty page size Get = %q, %v; want A4, true", got, ok)
	}

	if err := global.Set("copies", "0"); !errors.Is(err, ErrInvalidPDFCopies) {
		t.Fatalf("GlobalSettings.Set invalid copies = %v, want errors.Is(..., %v)", err, ErrInvalidPDFCopies)
	}

	if err := global.Set("margin.top", "-1"); err != nil {
		t.Fatalf("negative header/footer margin: %v", err)
	}

	if got, ok := global.Get("margin.top"); !ok || !strings.HasPrefix(got, "-") {
		t.Fatalf("negative margin Get = %q, %v; want a negative value", got, ok)
	}

	assertPanicsWith(t, ErrInvalidPDFCopies, func() {
		NewPdfGlobalOptions().WithCopies(0, true)
	})
}

func assertPanicsWith(t *testing.T, want error, call func()) {
	t.Helper()

	defer func() {
		recovered, ok := recover().(error)
		if !ok || !errors.Is(recovered, want) {
			t.Fatalf("panic = %v, want errors.Is(..., %v)", recovered, want)
		}
	}()
	call()
}

//nolint:cyclop,wsl,lll,funlen // validation error error-reporting assertions
func TestConverterValidationErrorsReachOnError(t *testing.T) {
	t.Parallel()

	var got string

	conv := NewConverter()
	conv.OnError = func(message string) { got = message }

	if err := conv.Convert(t.Context()); !errors.Is(err, ErrNoRenderablePDFObjects) {
		t.Fatalf("Convert() = %v, want %v", err, ErrNoRenderablePDFObjects)
	}

	if got != ErrNoRenderablePDFObjects.Error() {
		t.Fatalf("validation callback = %q, want %q", got, ErrNoRenderablePDFObjects)
	}

	got = ""
	if err := conv.ConvertTo(t.Context(), nil); !errors.Is(err, ErrMissingPDFOutput) {
		t.Fatalf("ConvertTo(nil writer) = %v, want %v", err, ErrMissingPDFOutput)
	}

	if got != ErrMissingPDFOutput.Error() {
		t.Fatalf("nil writer callback = %q, want %q", got, ErrMissingPDFOutput)
	}

	got = ""
	conv.AddHTML([]byte("<h1>test</h1>"), "")
	var buf bytes.Buffer
	if err := conv.ConvertTo(nil, &buf); !errors.Is(err, ErrNilContext) { //nolint:staticcheck // testing nil context preflight
		t.Fatalf("ConvertTo(nil ctx) = %v, want %v", err, ErrNilContext)
	}

	if got != ErrNilContext.Error() {
		t.Fatalf("nil ctx callback = %q, want %q", got, ErrNilContext)
	}

	var imageError string

	imageConv := NewImageConverter()
	imageConv.OnError = func(message string) { imageError = message }

	if err := imageConv.Convert(t.Context()); !errors.Is(err, ErrNoInputPageAdded) {
		t.Fatalf("image Convert() = %v, want %v", err, ErrNoInputPageAdded)
	}

	if imageError != ErrNoInputPageAdded.Error() {
		t.Fatalf("image validation callback = %q, want %q", imageError, ErrNoInputPageAdded)
	}

	imageError = ""
	if err := imageConv.ConvertTo(t.Context(), nil); !errors.Is(err, ErrMissingImageOutput) {
		t.Fatalf("image ConvertTo(nil writer) = %v, want %v", err, ErrMissingImageOutput)
	}

	if imageError != ErrMissingImageOutput.Error() {
		t.Fatalf("image nil writer callback = %q, want %q", imageError, ErrMissingImageOutput)
	}

	imageError = ""
	imageConv.AddObject("test.html")
	var imgBuf bytes.Buffer

	if err := imageConv.ConvertTo(nil, &imgBuf); !errors.Is(err, ErrNilContext) { //nolint:staticcheck // testing nil context preflight
		t.Fatalf("image ConvertTo(nil ctx) = %v, want %v", err, ErrNilContext)
	}

	if imageError != ErrNilContext.Error() {
		t.Fatalf("image nil ctx callback = %q, want %q", imageError, ErrNilContext)
	}
}

var dottedKeyParityCases = []struct { //nolint:gochecknoglobals // immutable test corpus
	key, value, want string
}{
	{"orientation", "Landscape", "Landscape"},
	{"colormode", "grayscale", "grayscale"},
	{"grayscale", "true", "true"},
	{"pageoffset", "2", "2"},
	{"copies", "2", "2"},
	{"collate", "false", "false"},
	{"outline", "false", "false"},
	{"outlinedepth", "6", "6"},
	{"dumpoutline", "true", "true"},
	{"dumpoutlinewithdefaulttocxsl", "true", "true"},
	{"usecompression", "false", "false"},
	{"title", "parity", "parity"},
	{"smartshrinking", "false", "false"},
	{"background", "false", "false"},
	{"web.background", "false", "false"},
	{"enablelocalfileaccess", "true", "true"},
	{"excludefromoutline", "h1", "h1"},
	{"quiet", "true", "true"},
	{"proxy", "http://proxy.example.test:8080", "http://proxy.example.test:8080"},
	{"usesystemfonts", "true", "true"},
	{"resolverelativelinks", "false", "false"},
	{"fontpath", "/fonts", "/fonts"},
	{"allow", "/srv/html", "/srv/html"},
	{"margin.top", "12mm", "12"},
	{"margin.bottom", "12mm", "12"},
	{"margin.left", "12mm", "12"},
	{"margin.right", "12mm", "12"},
	{"size.pagesize", "Letter", "Letter"},
	{"size.width", "210mm", "210"},
	{"size.height", "297mm", "297"},
	{"header.fontsize", "14", "14"},
	{"header.fontname", "Arial", "Arial"},
	{"header.left", "header-left", "header-left"},
	{"header.right", "header-right", "header-right"},
	{"header.center", "header-center", "header-center"},
	{"header.line", "true", "true"},
	{"header.spacing", "2", "2"},
	{"header.htmlurl", "/header.html", "/header.html"},
	{"footer.fontsize", "14", "14"},
	{"footer.fontname", "Arial", "Arial"},
	{"footer.left", "footer-left", "footer-left"},
	{"footer.right", "footer-right", "footer-right"},
	{"footer.center", "footer-center", "footer-center"},
	{"footer.line", "true", "true"},
	{"footer.spacing", "2", "2"},
	{"footer.htmlurl", "/footer.html", "/footer.html"},
	{"toc.fontscale", "0.9", "0.9"},
	{"toc.indentation", " 2 ", "2"},
	{"toc.dottedlines", "false", "false"},
	{"toc.captiontext", "Contents", "Contents"},
	{"toc.forwardlinks", "true", "true"},
	{"toc.backlinks", "true", "true"},
	{"toc.xslstylesheet", "/toc.xsl", "/toc.xsl"},
	{"web.images", "false", "false"},
	{"web.printmediatype", "true", "true"},
	{"web.mediatype", "print", "print"},
	{"web.simplifydom", "true", "true"},
	{"web.simplifydomprofile", "safe", "safe"},
	{"web.printlinkunderline", "false", "false"},
	{"dpi", "96", "96"},
	{"resolution", "96", "96"},
	{"imagedpi", "96", "96"},
	{"imagequality", "90", "90"},
	{"lowquality", "true", "true"},
	{"usexserver", "true", "true"},
	{"readargsfromstdin", "true", "true"},
	{"log-level", "info", "info"},
	{"loglevel", "info", "info"},
	{"cookiejar", "cookies.txt", "cookies.txt"},
	{"defaultencoding", "utf-8", "utf-8"},
	{"produceforms", "true", "true"},
	{"loaderrorhandling", "skip", "skip"},
	{"web.javascript", "true", "true"},
	{"web.java", "true", "true"},
	{"web.plugins", "true", "true"},
	{"web.minimumfontsize", "8", "8"},
	{"web.defaultencoding", "utf-8", "utf-8"},
	{"web.userstylesheet", "/style.css", "/style.css"},
	{"web.loadimages", "true", "true"},
}

func TestGlobalSettingsDottedKeyParity(t *testing.T) {
	t.Parallel()

	for _, testCase := range dottedKeyParityCases {
		t.Run(testCase.key, func(t *testing.T) {
			t.Parallel()

			global := NewGlobalSettings()
			if err := global.Set(testCase.key, testCase.value); err != nil {
				t.Fatalf("Set(%q): %v", testCase.key, err)
			}

			got, ok := global.Get(testCase.key)
			if !ok || got != testCase.want {
				t.Fatalf("Get(%q) = %q, %v; want %q, true", testCase.key, got, ok, testCase.want)
			}
		})
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

var typedPDFRequestPreflightCases = []struct { //nolint:gochecknoglobals // immutable test corpus
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

func TestTypedPDFRequestPreflight(t *testing.T) {
	t.Parallel()

	for _, testCase := range typedPDFRequestPreflightCases {
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

func TestGlobalNetworkPolicyIsCopied(t *testing.T) {
	t.Parallel()

	policy := RestrictedNetworkPolicy()
	policy.AllowedHosts = []string{"reports.example.test"}

	global := NewGlobalSettings()
	if err := global.SetNetworkPolicy(policy); err != nil {
		t.Fatal(err)
	}

	policy.AllowedSchemes[0] = "ftp"
	policy.AllowedHosts[0] = mutatedNetworkHost

	if !global.g.Load.NetworkPolicySet {
		t.Fatal("explicit network policy was not recorded")
	}

	if got := global.g.Load.NetworkAllowedSchemes[0]; got != "http" {
		t.Errorf("allowed scheme = %q, want copied http value", got)
	}

	if got := global.g.Load.NetworkAllowedHosts[0]; got != "reports.example.test" {
		t.Errorf("allowed host = %q, want copied host value", got)
	}

	if !global.g.Load.NetworkBlockPrivate || !global.g.Load.NetworkBlockCrossHost {
		t.Error("restricted network policy flags were not copied")
	}
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

func TestCLIVersionMatchesVERSIONFile(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("VERSION")
	if err != nil {
		t.Fatalf("read VERSION: %v", err)
	}

	want := strings.TrimSpace(string(raw))
	if want == "" {
		t.Fatal("VERSION file is empty")
	}

	if cli.Version != want && !strings.HasPrefix(cli.Version, want+"-") {
		t.Fatalf("cli.Version = %q, want VERSION %q or a suffix-stamped form", cli.Version, want)
	}

	// LibraryVersion is the upstream wkhtmltopdf compatibility id, not the
	// project release stamped from VERSION.
	if LibraryVersion == want {
		t.Fatalf("LibraryVersion %q must not equal the project release %q", LibraryVersion, want)
	}

	if !strings.HasPrefix(LibraryVersion, "0.12.") {
		t.Fatalf("LibraryVersion = %q, want an upstream 0.12.x compatibility id", LibraryVersion)
	}
}

func TestNilRequestSentinelsAreDistinct(t *testing.T) {
	t.Parallel()

	if errors.Is(ErrNilPDFRequest, ErrNilConverter) {
		t.Fatal("ErrNilPDFRequest must not alias ErrNilConverter")
	}

	if errors.Is(ErrNilImageRequest, ErrNilImageConverter) {
		t.Fatal("ErrNilImageRequest must not alias ErrNilImageConverter")
	}

	if err := RunPDF(t.Context(), nil); !errors.Is(err, ErrNilPDFRequest) {
		t.Fatalf("RunPDF(nil) = %v, want ErrNilPDFRequest", err)
	}

	if err := RunImage(t.Context(), nil); !errors.Is(err, ErrNilImageRequest) {
		t.Fatalf("RunImage(nil) = %v, want ErrNilImageRequest", err)
	}

	var conv *Converter
	if err := conv.Convert(t.Context()); !errors.Is(err, ErrNilConverter) {
		t.Fatalf("(*Converter)(nil).Convert = %v, want ErrNilConverter", err)
	}

	var img *ImageConverter
	if err := img.Convert(t.Context()); !errors.Is(err, ErrNilImageConverter) {
		t.Fatalf("(*ImageConverter)(nil).Convert = %v, want ErrNilImageConverter", err)
	}
}

func TestConvertToWritesPDFWithoutOutput(t *testing.T) {
	t.Parallel()

	conv := NewConverter()
	conv.AddObject(NewObjectSettings().SetBody(
		[]byte("<html><body><p>writer first</p></body></html>"), "",
	))

	var buf bytes.Buffer
	if err := conv.ConvertTo(t.Context(), &buf); err != nil {
		t.Fatalf("ConvertTo: %v", err)
	}

	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Fatalf("ConvertTo output = %q, want %%PDF-", buf.Bytes()[:min(len(buf.Bytes()), 16)])
	}

	if got := conv.Output(); got != nil {
		t.Fatalf("ConvertTo must not require Output(); got %d bytes", len(got))
	}
}

func TestImageConvertToWritesWithoutOutput(t *testing.T) {
	t.Parallel()

	conv := NewImageConverter()
	conv.Object().SetBody(
		[]byte(`<html><body><div style="width:20px;height:20px;background:#000"></div></body></html>`),
		"",
	)

	if err := conv.Set("width", "40"); err != nil {
		t.Fatalf("Set(width): %v", err)
	}

	var buf bytes.Buffer
	if err := conv.ConvertTo(t.Context(), &buf); err != nil {
		t.Fatalf("ConvertTo: %v", err)
	}

	if _, err := png.Decode(bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatalf("ConvertTo image is not a PNG: %v", err)
	}

	if got := conv.Output(); got != nil {
		t.Fatalf("ConvertTo must not require Output(); got %d bytes", len(got))
	}
}

func TestConvertSnapshotsGlobalAndObjectSettings(t *testing.T) { //nolint:cyclop // snapshot mutation matrix
	t.Parallel()

	conv := NewConverter()
	if err := conv.Global().Set("title", "snapshot-title"); err != nil {
		t.Fatalf("set title: %v", err)
	}

	policy := RestrictedNetworkPolicy()
	policy.AllowedHosts = []string{"reports.example.test"}

	if err := conv.Global().SetNetworkPolicy(policy); err != nil {
		t.Fatal(err)
	}

	if err := conv.Global().Set("allow", "/tmp/before"); err != nil {
		t.Fatalf("set allow: %v", err)
	}

	if err := conv.Global().Set("header.left", "before"); err != nil {
		t.Fatalf("set header.left: %v", err)
	}

	conv.Global().g.Header.Replace = map[string]string{"name": "before"}
	conv.AddObject(NewObjectSettings().SetBody(
		[]byte("<html><body><h1>snapshot</h1></body></html>"), "",
	))

	if err := conv.Convert(t.Context()); err != nil {
		t.Fatalf("Convert: %v", err)
	}

	first := conv.Output()
	if !bytes.Contains(first, []byte("/Title (snapshot-title)")) {
		t.Fatalf("first PDF missing snapshot title")
	}

	if err := conv.Global().Set("title", "mutated-after"); err != nil {
		t.Fatalf("mutate title: %v", err)
	}

	conv.Global().g.Load.Allow[0] = "/tmp/mutated"
	conv.Global().g.Load.NetworkAllowedHosts[0] = mutatedNetworkHost
	conv.Global().g.Header.Replace["name"] = "mutated"

	second := conv.Output()
	if !bytes.Equal(first, second) {
		t.Fatal("mutating Global() after Convert changed stored Output()")
	}

	if err := conv.Convert(t.Context()); err != nil {
		t.Fatalf("second Convert: %v", err)
	}

	if bytes.Contains(conv.Output(), []byte("/Title (snapshot-title)")) {
		t.Fatal("second Convert should use the mutated title")
	}
}

func TestImageConvertSnapshotsSettings(t *testing.T) {
	t.Parallel()

	conv := NewImageConverter()
	conv.Object().SetBody([]byte("<html><body><p>img snapshot</p></body></html>"), "")

	if err := conv.Set("width", "80"); err != nil {
		t.Fatalf("Set(width): %v", err)
	}

	if err := conv.Global().Set("allow", "/tmp/before"); err != nil {
		t.Fatalf("set allow: %v", err)
	}

	policy := RestrictedNetworkPolicy()
	policy.AllowedHosts = []string{"img.example.test"}

	if err := conv.Global().SetNetworkPolicy(policy); err != nil {
		t.Fatal(err)
	}

	if err := conv.Convert(t.Context()); err != nil {
		t.Fatalf("Convert: %v", err)
	}

	first := conv.Output()
	conv.Global().g.Load.Allow[0] = "/tmp/mutated"
	conv.Global().g.Load.NetworkAllowedHosts[0] = mutatedNetworkHost

	if !bytes.Equal(first, conv.Output()) {
		t.Fatal("mutating Global() after image Convert changed stored Output()")
	}
}

func TestNewTOCObjectAndCoverObjectSetRegisteredKeys(t *testing.T) {
	t.Parallel()

	toc := NewTOCObject()
	if got, ok := toc.Get("istableofcontent"); !ok || got != "true" {
		t.Fatalf("NewTOCObject istableofcontents = %q, %v", got, ok)
	}

	if !toc.o.IsTableOfContent {
		t.Fatal("NewTOCObject did not set IsTableOfContent")
	}

	cover := NewCoverObject()
	if got, ok := cover.Get("iscover"); !ok || got != "true" {
		t.Fatalf("NewCoverObject iscover = %q, %v", got, ok)
	}

	if !cover.o.IsCover || cover.o.IncludeInOutline {
		t.Fatalf("NewCoverObject cover=%v outline=%v", cover.o.IsCover, cover.o.IncludeInOutline)
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

func TestImageConverterAddObjectReplacesInput(t *testing.T) {
	t.Parallel()

	conv := NewImageConverter()
	conv.AddObject("first.html")
	conv.AddObject("second.html")

	got, ok := conv.Object().Get("page")
	if !ok {
		t.Fatal("Object().Get(page) did not return the configured input")
	}

	if got != "second.html" {
		t.Fatalf("Object().Get(page) = %q, want most recently added page", got)
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
