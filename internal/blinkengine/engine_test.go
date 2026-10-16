package blinkengine

import (
	"errors"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

// TestLayoutMapsInvoiceOps runs the blinkless pipeline on a small invoice and
// checks the adapter output: canvas bounds from the options, text operations
// with an attached face, and the invoice text on the list.
func TestLayoutMapsInvoiceOps(t *testing.T) {
	t.Parallel()

	source := []byte(`<!DOCTYPE html><html><head><style>
h1 { color: #1a3d6d; font-size: 18pt }
table { width: 100%; border-collapse: collapse }
</style></head><body>
<h1>Invoice</h1>
<table><tbody><tr><td>Total</td><td>234.40 EUR</td></tr></tbody></table>
<div class="footer"><p>IBAN DE00 0000 0000 0000 0000 00</p></div>
</body></html>`)

	res, err := Layout(t.Context(), source, Options{
		WidthPt:  538.6,
		HeightPt: 785.2,
		Media:    "print",
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	if res.Width != 538.6 {
		t.Errorf("Width = %g, want 538.6", res.Width)
	}

	if res.Height <= 0 {
		t.Fatalf("Height = %g, want positive", res.Height)
	}

	if len(res.Ops) == 0 {
		t.Fatal("no operations mapped")
	}

	text, textOps := mappedText(t, res)
	if textOps == 0 {
		t.Fatal("no text operations mapped")
	}

	for _, needle := range []string{"Invoice", "234.40"} {
		if !strings.Contains(text, needle) {
			t.Errorf("mapped text %q missing %q", text, needle)
		}
	}
}

// mappedText concatenates the mapped text runs and counts them, checking each
// carries a face.
func mappedText(t *testing.T, res *layout.Result) (string, int) {
	t.Helper()

	var text strings.Builder

	textOps := 0

	for idx := range res.Ops {
		textOp := &res.Ops[idx]
		if textOp.Kind != layout.OpText {
			continue
		}

		textOps++

		if textOp.Font == nil {
			t.Errorf("text op %d has no face", idx)
		}

		text.WriteString(textOp.Text)
	}

	return text.String(), textOps
}

func TestLayoutRejectsBadInput(t *testing.T) {
	t.Parallel()

	if _, err := Layout(nil, []byte("<p>x</p>"), Options{WidthPt: 500, HeightPt: 700}); !errors.Is(err, errNilContext) {
		t.Fatalf("nil context err = %v, want %v", err, errNilContext)
	}

	for _, opts := range []Options{
		{WidthPt: 0, HeightPt: 700},
		{WidthPt: 500, HeightPt: -1},
	} {
		if _, err := Layout(t.Context(), []byte("<p>x</p>"), opts); !errors.Is(err, errBadViewport) {
			t.Fatalf("viewport %+v err = %v, want %v", opts, err, errBadViewport)
		}
	}
}
