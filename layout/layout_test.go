package layout_test

import (
	"math"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/layout"
)

func TestLayUsesParsedHTMLAndCSS(t *testing.T) {
	t.Parallel()

	placed := mustLay(t)
	requirePicture(t, placed)
	requireBox(t, findBox(t, placed.Boxes(), "box"))
}

func mustLay(t *testing.T) *layout.Result {
	t.Helper()

	const page = `<!DOCTYPE html><html><head></head><body>` +
		`<div id="box" data-action="go">Hi</div></body></html>`

	doc, err := html.Parse([]byte(page))
	if err != nil {
		t.Fatal(err)
	}

	sheet, err := css.Parse("body { margin: 0 } #box { width: 80px; height: 30px; background: #abcdef }")
	if err != nil {
		t.Fatal(err)
	}

	styled, err := css.Apply(t.Context(), doc, css.Options{
		WidthPx:  200,
		HeightPx: 100,
		Media:    "screen",
		Extra:    []*css.Sheet{sheet},
	})
	if err != nil {
		t.Fatal(err)
	}

	placed, err := layout.Lay(t.Context(), styled)
	if err != nil {
		t.Fatal(err)
	}

	return placed
}

func requirePicture(t *testing.T, placed *layout.Result) {
	t.Helper()

	if placed.Image() == nil {
		t.Fatal("image is nil")
	}

	width, height := placed.Size()
	if width != 200 || height < 100 {
		t.Fatalf("size = %d x %d", width, height)
	}
}

func requireBox(t *testing.T, box layout.Box) {
	t.Helper()

	if math.Abs(box.W-80) > 2 || math.Abs(box.H-30) > 2 {
		t.Fatalf("box size %.2f x %.2f", box.W, box.H)
	}

	if box.Action != "go" || box.Text != "Hi" {
		t.Fatalf("box action %q text %q", box.Action, box.Text)
	}
}

func TestLayRejectsNilDocument(t *testing.T) {
	t.Parallel()

	_, err := layout.Lay(t.Context(), nil)
	if err == nil {
		t.Fatal("expected an error")
	}
}

func findBox(t *testing.T, boxes []layout.Box, elementID string) layout.Box {
	t.Helper()

	for _, box := range boxes {
		if box.ID == elementID {
			return box
		}
	}

	t.Fatalf("box %q not found", elementID)

	return layout.Box{}
}
