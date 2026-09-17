package layout

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// Phase 1 image-pipeline safety: a color-only background shorthand and a
// non-image fetch payload must never become an OpImage or an embed error.

func TestBackgroundShorthandWithoutImageToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     string
		wantColor bool
	}{
		{name: "hex color", value: "#f4f6fd", wantColor: true},
		{name: "named color", value: "red", wantColor: true},
		{name: "rgb color", value: "rgb(244, 246, 253)", wantColor: true},
		{name: "position and size", value: "center / cover"},
		{name: "position and size unspaced", value: "center/cover"},
		{name: "repeat keyword", value: "no-repeat"},
		{name: "length", value: "10px"},
		{name: "percentage", value: "50%"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			style := ResolvedStyle{}
			applyBackgroundShorthand(&style, testCase.value)

			if style.BackgroundImage != "" {
				t.Fatalf("background %q stored BackgroundImage = %q, want empty", testCase.value, style.BackgroundImage)
			}

			if gotColor := style.BGColor[3] > 0; gotColor != testCase.wantColor {
				t.Fatalf("background %q color set = %v, want %v (BGColor=%v)",
					testCase.value, gotColor, testCase.wantColor, style.BGColor)
			}
		})
	}
}

func TestBackgroundShorthandKeepsImageTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "url with color", value: `url("x.png") #fff`, want: "x.png"},
		{name: "url with position", value: `url("x.png") no-repeat center / cover`, want: "x.png"},
		{name: "gradient", value: "linear-gradient(red, blue)", want: "linear-gradient(red, blue)"},
		{name: "bare path", value: "logo.png", want: "logo.png"},
		{name: "bare path with slash", value: "images/logo.png", want: "images/logo.png"},
		{name: "multi url layers", value: `url("a.png"), url("b.png")`, want: `url("a.png"), url("b.png")`},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			style := ResolvedStyle{}
			applyBackgroundShorthand(&style, testCase.value)

			if style.BackgroundImage != testCase.want {
				t.Fatalf("background %q stored BackgroundImage = %q, want %q",
					testCase.value, style.BackgroundImage, testCase.want)
			}
		})
	}
}

func TestBackgroundShorthandColorOnlyEmitsNoImage(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `div.x { background: #f4f6fd; width: 100pt; height: 50pt; }`)
	root := mustParse(t, `<html><body><div class="x">hi</div></body></html>`)
	fetched := false
	opts := Options{
		Width: testViewport, Height: 800, Sheets: []*css.Stylesheet{cssSheet}, Background: true,
		Images: func(string) ([]byte, error) {
			fetched = true

			return tinyPNG(2, 2), nil
		},
	}

	styles, _, err := resolveStylesForLayoutContext(t.Context(), root, opts)
	if err != nil {
		t.Fatalf("resolve styles: %v", err)
	}

	divStyle := firstDivResolvedStyle(t, styles)

	if divStyle.BackgroundImage != "" {
		t.Fatalf("color-only background stored BackgroundImage = %q, want empty", divStyle.BackgroundImage)
	}

	if divStyle.BGColor[3] == 0 {
		t.Fatal("color-only background did not set BGColor")
	}

	res, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	if fetched {
		t.Fatal("color-only background fetched an image")
	}

	if imgs := opsOfKind(res, OpImage); len(imgs) != 0 {
		t.Fatalf("color-only background emitted image ops: %+v", imgs)
	}
}

// firstDivResolvedStyle returns the resolved style of the first div in styles.
func firstDivResolvedStyle(t *testing.T, styles map[*html.Node]*ResolvedStyle) *ResolvedStyle {
	t.Helper()

	for node, st := range styles {
		if node != nil && node.Name == divElementName {
			return st
		}
	}

	t.Fatal("no div style resolved")

	return nil
}

// TestBackgroundShorthandNthChildColorOnly mirrors the learncpp.com baseline
// rule `.lessontable-row:nth-child(odd){background:#f4f6fd}`: odd rows get the
// color, no row fetches an image, and no OpImage is emitted.
func TestBackgroundShorthandNthChildColorOnly(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `.lessontable-row:nth-child(odd) { background:#f4f6fd }`)
	root := mustParse(t, `<html><body>`+
		`<div class="lessontable-row">one</div>`+
		`<div class="lessontable-row">two</div>`+
		`<div class="lessontable-row">three</div>`+
		`<div class="lessontable-row">four</div>`+
		`</body></html>`)
	fetched := false
	opts := Options{
		Width: testViewport, Height: 800, Sheets: []*css.Stylesheet{cssSheet}, Background: true,
		Images: func(string) ([]byte, error) {
			fetched = true

			return tinyPNG(2, 2), nil
		},
	}

	res, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	if fetched {
		t.Fatal("odd-row color shorthand fetched an image")
	}

	if imgs := opsOfKind(res, OpImage); len(imgs) != 0 {
		t.Fatalf("odd-row color shorthand emitted image ops: %+v", imgs)
	}

	if fills := opsOfKind(res, OpFillRect); len(fills) == 0 {
		t.Fatal("odd-row color shorthand painted no fills")
	}
}

func TestResolveImageRejectsNonImagePayloads(t *testing.T) {
	t.Parallel()

	webp := append([]byte("RIFF\x24\x00\x00\x00WEBP"), make([]byte, 32)...)
	// ICO magic with a truncated/garbage directory must still reject.
	garbageICO := []byte{0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x10, 0x10}
	tests := []struct {
		name string
		data []byte
	}{
		{name: "html page", data: []byte("<!doctype html><html><body>homepage</body></html>")},
		{name: "webp", data: webp},
		{name: "empty", data: nil},
		{name: "garbage ico", data: garbageICO},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var warnings []string

			eng := &engine{
				opts: Options{
					Images: func(string) ([]byte, error) {
						return testCase.data, nil
					},
					Warnf: func(format string, args ...any) {
						warnings = append(warnings, fmt.Sprintf(format, args...))
					},
				},
				scale: 1,
			}

			if ref := eng.resolveImage("bad.png"); ref != nil {
				t.Fatalf("resolveImage kept unsupported payload: %+v", ref)
			}
			// The second call must hit the nil cache sentinel, not warn again.
			if ref := eng.resolveImage("bad.png"); ref != nil {
				t.Fatalf("resolveImage returned a ref from the miss cache: %+v", ref)
			}

			if len(warnings) != 1 {
				t.Fatalf("warnings = %d (%v), want exactly 1", len(warnings), warnings)
			}

			if !strings.Contains(warnings[0], "bad.png") {
				t.Fatalf("warning %q does not name the src", warnings[0])
			}
		})
	}
}

func TestResolveImageKeepsEmbeddablePayloads(t *testing.T) {
	t.Parallel()

	gifData := mustEncodeGIF(t, image.Rect(0, 0, 3, 2))
	pngPayload := tinyPNG(4, 3)
	icoData := buildTestPNGICO(pngPayload, 4, 3)

	tests := []struct {
		name      string
		src       string
		data      []byte
		wantW     int
		wantH     int
		wantBytes string
	}{
		{name: "png", src: "x.png", data: pngPayload, wantW: 4, wantH: 3, wantBytes: "\x89PNG"},
		{name: "gif", src: "x.gif", data: gifData, wantW: 3, wantH: 2, wantBytes: "\x89PNG"},
		{name: "png-in-ico", src: "x.ico", data: icoData, wantW: 4, wantH: 3, wantBytes: "\x89PNG"},
		{
			name: "svg", src: "x.svg", wantW: 20, wantH: 20, wantBytes: "\x89PNG",
			data: []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20"><rect width="10" height="10"/></svg>`),
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var warnings []string

			eng := &engine{
				opts: Options{
					Images: func(string) ([]byte, error) {
						return testCase.data, nil
					},
					Warnf: func(format string, args ...any) {
						warnings = append(warnings, fmt.Sprintf(format, args...))
					},
				},
				scale: 1,
			}

			ref := eng.resolveImage(testCase.src)
			if ref == nil {
				t.Fatal("resolveImage dropped a supported payload")
			}

			if ref.w != testCase.wantW || ref.h != testCase.wantH {
				t.Fatalf("intrinsic = %dx%d, want %dx%d", ref.w, ref.h, testCase.wantW, testCase.wantH)
			}

			if !bytes.HasPrefix(ref.data, []byte(testCase.wantBytes)) {
				t.Fatalf("payload = %q, want prefix %q", ref.data, testCase.wantBytes)
			}

			if len(warnings) != 0 {
				t.Fatalf("supported payload warned: %v", warnings)
			}
		})
	}
}

func TestGifBorderImageSourceStillCrops(t *testing.T) {
	t.Parallel()

	gifData := mustEncodeGIF(t, image.Rect(0, 0, 4, 4))

	eng := &engine{
		opts: Options{
			Images: func(string) ([]byte, error) {
				return gifData, nil
			},
		},
		scale: 1,
	}

	ops := eng.appendBorderImage(nil, newBorderImageStretchStyle(), 0, 0, 100, 80)
	if len(ops) != 8 {
		t.Fatalf("GIF border-image ops = %d, want 8 sliced border ops", len(ops))
	}

	for _, paintOp := range ops {
		if paintOp.Kind != OpImage || !bytes.HasPrefix(paintOp.Image, []byte("\x89PNG")) {
			t.Fatalf("GIF border-image slice is not a PNG op: kind=%v", paintOp.Kind)
		}
	}
}

func TestLayoutSkipsNonImagePayloadWithOneWarning(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `div.x { background-image: url("page.html"); width: 100pt; height: 10pt; }`)
	root := mustParse(t, `<html><body>`+
		`<div class="x">one</div><div class="x">two</div>`+
		`<div class="x">three</div><div class="x">four</div>`+
		`</body></html>`)

	var warnings []string

	opts := Options{
		Width: testViewport, Height: 800, Sheets: []*css.Stylesheet{cssSheet}, Background: true,
		Images: func(src string) ([]byte, error) {
			return []byte("<!doctype html><html><body>not an image: " + src + "</body></html>"), nil
		},
		Warnf: func(format string, args ...any) {
			warnings = append(warnings, fmt.Sprintf(format, args...))
		},
	}

	res, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	if imgs := opsOfKind(res, OpImage); len(imgs) != 0 {
		t.Fatalf("non-image payload emitted image ops: %+v", imgs)
	}

	if len(warnings) != 1 {
		t.Fatalf("warnings = %d (%v), want exactly 1", len(warnings), warnings)
	}

	if !strings.Contains(warnings[0], "page.html") {
		t.Fatalf("warning %q does not name the src", warnings[0])
	}
}

func TestDrawImageEmbedErrorNamesSrc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		src     string
		wantSub string
	}{
		{name: "fetched src", src: "https://example.com/bad.png", wantSub: "https://example.com/bad.png"},
		{name: "synthetic payload", src: "", wantSub: "inline"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			op := (Op{Kind: OpImage, W: 10, H: 10}).withImage([]byte("<html>not an image"), 4, 4, "", testCase.src)

			err := drawImage(nil, pdf.NewContent(), &op, 0, 100,
				PaintOptions{PageWidth: 595, PageHeight: 842}, 842, "I0")
			if err == nil {
				t.Fatal("drawImage accepted non-image bytes")
			}

			if !strings.Contains(err.Error(), testCase.wantSub) {
				t.Fatalf("embed error %q does not contain %q", err, testCase.wantSub)
			}

			if strings.Contains(err.Error(), "I0") {
				t.Fatalf("embed error %q still names the page-local counter", err)
			}
		})
	}
}

// mustEncodeGIF encodes a two-color palette GIF at the given bounds and
// returns its bytes.
func mustEncodeGIF(t *testing.T, bounds image.Rectangle) []byte {
	t.Helper()

	var buf bytes.Buffer

	palette := []color.Color{color.RGBA{R: 255, A: 255}, color.RGBA{B: 255, A: 255}}
	if err := gif.Encode(&buf, image.NewPaletted(bounds, palette), nil); err != nil {
		t.Fatalf("encode gif: %v", err)
	}

	return buf.Bytes()
}
