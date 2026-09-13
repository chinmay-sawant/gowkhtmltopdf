//nolint:varnamelen // targeted paint tests for the color-adjust family
package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// layoutColorAdjustDoc lays out src with the given stylesheet and returns the
// display list. Background is off to pin the print-color-adjust economy.
func layoutColorAdjustDoc(t *testing.T, src string, background bool, rules string) *Result {
	t.Helper()

	root := mustParse(t, src)

	res, err := Layout(root, Options{
		Width: testViewport, Height: 800, Sheets: []*css.Stylesheet{sheet(t, rules)}, Background: background,
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	return res
}

// paintColorAdjustDoc paints res into a 595x842 page with 36pt margins.
func paintColorAdjustDoc(t *testing.T, res *Result) *pdf.Document {
	t.Helper()

	doc := pdf.NewDocument()

	err := Paint(doc, res, PaintOptions{
		PageWidth: 595, PageHeight: 842,
		MarginTop: 36, MarginBottom: 36, MarginLeft: 36, MarginRight: 36,
	})
	if err != nil {
		t.Fatalf("Paint: %v", err)
	}

	return doc
}

func pageContent(t *testing.T, doc *pdf.Document) string {
	t.Helper()

	if doc.PageCount() < 1 {
		t.Fatal("no pages painted")
	}

	return string(doc.PageAt(0).Content().Bytes())
}

// TestPrintColorAdjustExactPaintsBackgroundWithGlobalOff is the print-color
// consumer proof: the document setting drops backgrounds (economy), and the
// element's exact opt-out keeps this background painted anyway.
func TestPrintColorAdjustExactPaintsBackgroundWithGlobalOff(t *testing.T) {
	t.Parallel()

	res := layoutColorAdjustDoc(t, `<div class="card">card</div>`, false, `
.card {
	print-color-adjust: exact;
	background-image: linear-gradient(#ff0000, #ff0000);
	width: 100pt;
	height: 40pt;
}`)

	imgs := opsOfKind(res, OpImage)

	if len(imgs) == 0 {
		t.Fatal("print-color-adjust: exact painted no background with Background off")
	}

	if !imgs[0].IsBackground {
		t.Errorf("image op is not marked as a background: %+v", imgs[0])
	}
}

func TestPrintColorAdjustEconomyKeepsBackgroundOff(t *testing.T) {
	t.Parallel()

	res := layoutColorAdjustDoc(t, `<div class="card">card</div>`, false, `
.card {
	print-color-adjust: economy;
	background-image: linear-gradient(#ff0000, #ff0000);
	width: 100pt;
	height: 40pt;
}`)

	if imgs := opsOfKind(res, OpImage); len(imgs) != 0 {
		t.Fatalf("economy painted %d background image ops with Background off", len(imgs))
	}
}

// TestColorSchemeDarkPaintsCanvasAndDefaultText proves the dark scheme paints
// the #121212 canvas fill and substitutes the #e8e8e8 default text color when
// the author declares none.
func TestColorSchemeDarkPaintsCanvasAndDefaultText(t *testing.T) {
	t.Parallel()

	res := layoutColorAdjustDoc(t, `<html><body><p>Hello</p></body></html>`, true, `html { color-scheme: dark; }`)
	raw := pageContent(t, paintColorAdjustDoc(t, res))

	if !strings.Contains(raw, "0.071 0.071 0.071 rg") {
		t.Fatalf("dark canvas fill missing from content stream:\n%s", raw)
	}

	if !strings.Contains(raw, "0.91 0.91 0.91 rg") {
		t.Fatalf("dark default text color missing from content stream:\n%s", raw)
	}
}

func TestColorSchemeDarkKeepsAuthorTextColor(t *testing.T) {
	t.Parallel()

	res := layoutColorAdjustDoc(t, `<html><body><p>Hello</p></body></html>`, true,
		`html { color-scheme: dark; } p { color: #ff0000; }`)
	raw := pageContent(t, paintColorAdjustDoc(t, res))

	if !strings.Contains(raw, "1 0 0 rg") {
		t.Fatalf("author red text missing from content stream:\n%s", raw)
	}

	if strings.Contains(raw, "0.91 0.91 0.91 rg") {
		t.Fatalf("scheme default overrode the author text color:\n%s", raw)
	}
}

// TestForcedColorAdjustNoneKeepsAuthorTextColor pins the print reading of
// forced-color-adjust: with no forced-colors mode, none keeps author colors,
// which for undeclared text is the initial black.
func TestForcedColorAdjustNoneKeepsAuthorTextColor(t *testing.T) {
	t.Parallel()

	res := layoutColorAdjustDoc(t, `<html><body><p>Hello</p></body></html>`, true,
		`html { color-scheme: dark; forced-color-adjust: none; }`)
	raw := pageContent(t, paintColorAdjustDoc(t, res))

	if !strings.Contains(raw, "0 0 0 rg") {
		t.Fatalf("initial black text missing with forced-color-adjust: none:\n%s", raw)
	}

	if strings.Contains(raw, "0.91 0.91 0.91 rg") {
		t.Fatalf("forced-color-adjust: none still applied the dark text default:\n%s", raw)
	}
}

// TestDynamicRangeLimitClampsOutOfRangeChannels drives the sRGB clamp through
// Paint: standard clamps an out-of-range fill to the writer's range, no-limit
// passes it through.
func TestDynamicRangeLimitClampsOutOfRangeChannels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		limit string
		want  string
	}{
		{name: "standard clamps", limit: dynamicRangeLimitStandard, want: "1 0 0.5 rg"},
		{name: "constrained-high clamps", limit: dynamicRangeLimitConstrainedHigh, want: "1 0 0.5 rg"},
		{name: "no-limit passes through", limit: dynamicRangeLimitNoLimit, want: "1.5 -0.5 0.5 rg"},
		{name: "high passes through", limit: dynamicRangeLimitHigh, want: "1.5 -0.5 0.5 rg"},
	}

	for _, tc := range cases {
		rootStyle := initialStyle()
		rootStyle.DynamicRangeLimit = tc.limit
		root := &box{
			node:  &html.Node{Type: html.ElementNode, Name: htmlRootName},
			style: &rootStyle,
		}
		res := &Result{Ops: []Op{{
			Kind: OpFillRect, X: 0, Y: 0, W: 10, H: 10,
			R: 1.5, G: -0.5, B: 0.5, Alpha: 1,
		}}, root: root}

		doc := pdf.NewDocument()

		err := Paint(doc, res, PaintOptions{PageWidth: 100, PageHeight: 100})
		if err != nil {
			t.Fatalf("%s: Paint: %v", tc.name, err)
		}

		raw := string(doc.PageAt(0).Content().Bytes())
		if !strings.Contains(raw, tc.want) {
			t.Errorf("%s: content stream = %q, want %q", tc.name, raw, tc.want)
		}
	}
}

// TestColorSchemeDarkPaintIsIdempotent pins that the scheme consumers do not
// mutate the display list: two paints of one Result produce the same content.
func TestColorSchemeDarkPaintIsIdempotent(t *testing.T) {
	t.Parallel()

	res := layoutColorAdjustDoc(t, `<html><body><p>Hello</p></body></html>`, true, `html { color-scheme: dark; }`)

	first := pageContent(t, paintColorAdjustDoc(t, res))
	second := pageContent(t, paintColorAdjustDoc(t, res))

	if first != second {
		t.Fatalf("second paint differs:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}
