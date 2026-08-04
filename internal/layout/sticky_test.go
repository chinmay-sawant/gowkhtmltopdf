package layout

import (
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

func TestStickyClampYTop(t *testing.T) {
	b := &box{
		sticky:       true,
		stickyTopSet: true,
		stickyTop:    0,
		h:            20,
	}
	// Natural y=50; page [200,400): stick to page top.
	y := clampStickyY(50, 20, 0, 500, 200, 400, b)
	if y != 200 {
		t.Fatalf("clampStickyY = %.1f, want 200", y)
	}
	// Already below sticky edge: no move.
	y = clampStickyY(250, 20, 0, 500, 200, 400, b)
	if y != 250 {
		t.Fatalf("clampStickyY natural = %.1f, want 250", y)
	}
}

func TestStickyClampYContainingBlockLimit(t *testing.T) {
	b := &box{
		sticky:       true,
		stickyTopSet: true,
		stickyTop:    0,
		h:            30,
	}
	// Would stick at 200, but CB ends at 220 → clamp to 190.
	y := clampStickyY(50, 30, 0, 220, 200, 400, b)
	if y != 190 {
		t.Fatalf("CB limit = %.1f, want 190", y)
	}
}

func TestStickyTopContinuationPages(t *testing.T) {
	// Section CB spans multiple pages; sticky bar at section top must appear
	// near page tops on continuation pages (print scrollport = contentH).
	var body strings.Builder
	body.WriteString(`<html><body>
<div class="sec">
  <div class="stick">STICKY</div>
`)
	for i := 0; i < 30; i++ {
		body.WriteString(`<p>row ` + itoa(i) + ` filler text for pagination</p>`)
	}
	body.WriteString(`</div></body></html>`)

	s := sheet(t, `
.sec { background: #f5f5f5; }
.stick {
  position: sticky;
  top: 0;
  background: #c62828;
  color: #fff;
  padding: 6pt;
  height: 24pt;
}
p { margin: 4pt 0; font-size: 12pt; }
`)
	root, err := html.Parse(body.String())
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{
		Width: 400, Height: 200, Background: true,
		Sheets: []*css.Stylesheet{s}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}
	doc := pdf.NewDocument()
	opts := PaintOptions{
		PageWidth: 500, PageHeight: 280,
		MarginTop: 20, MarginBottom: 20, MarginLeft: 20, MarginRight: 20,
	}
	if err := Paint(doc, res, opts); err != nil {
		t.Fatal(err)
	}
	contentH := 280.0 - 40.0
	if doc.PageCount() < 2 {
		t.Fatalf("expected multi-page sticky section, got %d pages", doc.PageCount())
	}
	pagesWithSticky := map[int]bool{}
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}
		if strings.Contains(op.Text, "STICKY") {
			page := int(op.Y / contentH)
			pagesWithSticky[page] = true
			pageTop := float64(page) * contentH
			if op.Y < pageTop-0.5 || op.Y > pageTop+40 {
				t.Errorf("STICKY on page %d at y=%.1f, want near pageTop=%.1f", page, op.Y, pageTop)
			}
		}
	}
	if len(pagesWithSticky) < 2 {
		t.Fatalf("sticky text on %d page band(s), want ≥2: %v", len(pagesWithSticky), pagesWithSticky)
	}
}

func TestStickyContainingBlockStops(t *testing.T) {
	// Short CB: sticky must not paint past the section end on later pages.
	src := `<html><body>
<div class="sec">
  <div class="stick">STICKY</div>
  <p>short</p>
</div>
<div class="after">` + strings.Repeat(`<p>after row</p>`, 40) + `</div>
</body></html>`
	s := sheet(t, `
.sec { height: 80pt; background: #eee; }
.stick { position: sticky; top: 0; height: 20pt; background: #333; color: #fff; }
p { margin: 2pt 0; font-size: 11pt; }
`)
	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{
		Width: 400, Height: 200, Background: true,
		Sheets: []*css.Stylesheet{s}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}
	doc := pdf.NewDocument()
	opts := PaintOptions{
		PageWidth: 500, PageHeight: 280,
		MarginTop: 20, MarginBottom: 20, MarginLeft: 20, MarginRight: 20,
	}
	if err := Paint(doc, res, opts); err != nil {
		t.Fatal(err)
	}
	contentH := 240.0
	// Find section CB via sticky box after paint walk would have set cb*.
	var stick *box
	var walk func(b *box)
	walk = func(b *box) {
		if b.sticky {
			stick = b
			return
		}
		for _, c := range b.children {
			walk(c)
		}
	}
	walk(res.root)
	if stick == nil {
		t.Fatal("no sticky box")
	}
	cbBottom := stick.cbY + stick.cbH
	for _, op := range res.Ops {
		if op.Kind != OpText || !strings.Contains(op.Text, "STICKY") {
			continue
		}
		if op.Y > cbBottom+0.5 {
			t.Errorf("sticky op y=%.1f past CB bottom %.1f", op.Y, cbBottom)
		}
		// Must not appear on pages that start after the CB ends.
		page := int(op.Y / contentH)
		pageTop := float64(page) * contentH
		if pageTop >= cbBottom-1e-6 {
			t.Errorf("sticky painted on page %d (pageTop=%.1f) after CB end %.1f", page, pageTop, cbBottom)
		}
	}
}

func TestStickyNotFixedReplication(t *testing.T) {
	// Sticky in a short first section must not stamp onto later pages the way
	// position:fixed does.
	src := `<html><body>
<div class="sec"><div class="stick">STICKY</div><p>a</p></div>
` + strings.Repeat(`<p>pad</p>`, 50) + `
</body></html>`
	s := sheet(t, `
.sec { height: 60pt; }
.stick { position: sticky; top: 0; height: 18pt; background: #900; color: #fff; }
p { margin: 3pt 0; font-size: 12pt; }
`)
	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{
		Width: 400, Height: 200, Background: true,
		Sheets: []*css.Stylesheet{s}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}
	doc := pdf.NewDocument()
	opts := PaintOptions{
		PageWidth: 500, PageHeight: 280,
		MarginTop: 20, MarginBottom: 20, MarginLeft: 20, MarginRight: 20,
	}
	if err := Paint(doc, res, opts); err != nil {
		t.Fatal(err)
	}
	if doc.PageCount() < 2 {
		t.Fatalf("need ≥2 pages to compare with fixed, got %d", doc.PageCount())
	}
	contentH := 240.0
	pages := map[int]bool{}
	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "STICKY") {
			pages[int(op.Y/contentH)] = true
			if op.Fixed {
				t.Error("sticky ops must not be marked Fixed")
			}
		}
	}
	if len(pages) != 1 {
		t.Fatalf("sticky on %d page(s) %v, want exactly 1 (not fixed-style stamp)", len(pages), pages)
	}
}

func TestStickyNotRelativeOffsetAtLayout(t *testing.T) {
	// top inset must not shift the box during layout the way relative does.
	s := sheet(t, `.s { position: sticky; top: 40pt; } .r { position: relative; top: 40pt; }`)
	resS := layoutHTML(t, `<html><body><div class="s">S</div></body></html>`, s)
	resR := layoutHTML(t, `<html><body><div class="r">R</div></body></html>`, s)
	var sy, ry float64
	var gotS, gotR bool
	for _, op := range resS.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "S") {
			sy, gotS = op.Y, true
		}
	}
	for _, op := range resR.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "R") {
			ry, gotR = op.Y, true
		}
	}
	if !gotS || !gotR {
		t.Fatalf("missing text ops sticky=%v relative=%v", gotS, gotR)
	}
	if sy >= ry-1 {
		t.Fatalf("sticky y=%.1f should be above relative y=%.1f (relative applies top at layout)", sy, ry)
	}
}
