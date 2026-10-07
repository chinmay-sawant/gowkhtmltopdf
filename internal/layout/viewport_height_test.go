package layout

import "testing"

func TestRootPercentageHeightFillsTheViewport(t *testing.T) {
	t.Parallel()

	const viewportH = 640.0

	cssSheet := sheet(t, `
		html, body { margin: 0; height: 100%; }
	`)
	res := layoutHTMLAtViewport(t, `<html><body><p>Hi</p></body></html>`, 480, viewportH, cssSheet)
	htmlBox := findBox(t, res, "html")
	bodyBox := findBox(t, res, "body")

	if !near(htmlBox.height, viewportH) {
		t.Fatalf("html height = %.2f, want %.2f", htmlBox.height, viewportH)
	}

	if !near(bodyBox.height, viewportH) {
		t.Fatalf("body height = %.2f, want %.2f", bodyBox.height, viewportH)
	}
}

func TestNestedPercentageHeightStaysContentSized(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
		html, body { margin: 0; }
		#card { height: 100%; }
		p { margin: 0; }
	`)
	res := layoutHTMLAtViewport(t, `<html><body><div id="card"><p>Hi</p></div></body></html>`, 400, 600, cssSheet)

	card := findBoxByID(res.root, "card")
	if card == nil {
		t.Fatal("no #card box")
	}

	if card.height > 80 {
		t.Fatalf("card height = %.2f, want the content height", card.height)
	}
}

func TestFlexCentersACardInTheViewport(t *testing.T) {
	t.Parallel()

	const (
		viewportW = 400.0
		viewportH = 600.0
		cardW     = 100.0
		cardH     = 80.0
	)

	cssSheet := sheet(t, `
		html, body { margin: 0; height: 100%; }
		body { display: flex; align-items: center; justify-content: center; }
		#card { width: 100pt; height: 80pt; }
	`)
	res := layoutHTMLAtViewport(t, `<html><body><div id="card">Hi</div></body></html>`, viewportW, viewportH, cssSheet)

	card := findBoxByID(res.root, "card")
	if card == nil {
		t.Fatal("no #card box")
	}

	wantX := (viewportW - cardW) / 2

	wantY := (viewportH - cardH) / 2
	if !near(card.x, wantX) || !near(card.y, wantY) {
		t.Fatalf("card origin = %.2f, %.2f, want %.2f, %.2f", card.x, card.y, wantX, wantY)
	}

	if !near(card.w, cardW) || !near(card.height, cardH) {
		t.Fatalf("card size = %.2f x %.2f, want %.2f x %.2f", card.w, card.height, cardW, cardH)
	}
}

func TestFlexCenterMovesDescendantBoxes(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
		html, body { margin: 0; height: 100%; }
		body { display: flex; align-items: center; justify-content: center; }
		#card { width: 100pt; height: 80pt; }
		#field { width: 40pt; height: 20pt; }
	`)
	res := layoutHTMLAtViewport(
		t, `<html><body><div id="card"><div id="field"></div></div></body></html>`, 400, 600, cssSheet,
	)

	card := findBoxByID(res.root, "card")
	field := findBoxByID(res.root, "field")

	if card == nil || field == nil {
		t.Fatal("missing card or field")
	}

	if field.x < card.x-0.5 || field.y < card.y-0.5 {
		t.Fatalf("field origin = %.2f, %.2f, card origin = %.2f, %.2f", field.x, field.y, card.x, card.y)
	}

	if field.x+field.w > card.x+card.w+0.5 || field.y+field.height > card.y+card.height+0.5 {
		t.Fatalf("field extends outside the card: field %.2f,%.2f %.2fx%.2f card %.2f,%.2f %.2fx%.2f",
			field.x, field.y, field.w, field.height, card.x, card.y, card.w, card.height)
	}
}
