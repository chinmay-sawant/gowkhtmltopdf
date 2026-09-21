//nolint:all // targeted unit tests for font-size-adjust
package layout

import (
	"math"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

func TestFontSizeAdjustScalesUsedSize(t *testing.T) {
	t.Parallel()

	style := initialStyle()
	if !applyFontSizeAdjustProps(&style, "font-size-adjust", "0.5", 12, nil, nil, false) {
		t.Fatal("font-size-adjust not owned")
	}

	if !style.FontSizeAdjustSet || style.FontSizeAdjust != 0.5 {
		t.Fatalf("adjust = set=%v value=%v", style.FontSizeAdjustSet, style.FontSizeAdjust)
	}

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	face := faces.Regular
	aspect := face.XHeightAspect()
	if aspect <= 0 {
		t.Fatal("Liberation Sans must expose OS/2 sxHeight")
	}

	style.FontSize = 12
	got := usedFontSize(&style, face)
	want := 12 * (0.5 / aspect)

	if math.Abs(got-want) > 0.01 {
		t.Fatalf("usedFontSize = %v, want %v (aspect %v)", got, want, aspect)
	}

	doc := parseTestHTML(t, `<html><body style="margin:0;font-size:16px">`+
		`<p style="font-size-adjust:0.5">adjusted</p>`+
		`<p>plain</p>`+
		`</body></html>`)

	res, err := Layout(doc, Options{Width: 500, Height: 400})
	if err != nil {
		t.Fatal(err)
	}

	var adjusted, plain float64

	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.Kind != OpText {
			continue
		}

		if strings.Contains(paintOp.Text, "adjusted") {
			adjusted = paintOp.Size
		}

		if strings.Contains(paintOp.Text, "plain") {
			plain = paintOp.Size
		}
	}

	if adjusted == 0 || plain == 0 {
		t.Fatalf("missing sizes adjusted=%v plain=%v", adjusted, plain)
	}

	if !(adjusted < plain) {
		t.Fatalf("adjusted size %v should be smaller than plain %v", adjusted, plain)
	}
}
