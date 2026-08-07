package layout

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

func TestStickyOverflowScrollportNoPageClone(t *testing.T) {
	// Sticky inside overflow:auto must use that box as scrollport at offset 0
	// (no page-edge continuation clones). Without overflow, the same sticky
	// would stick across pages (print scrollport).
	var body strings.Builder

	body.WriteString(`<html><body>
<div class="scroller">
  <div class="stick">STICKY</div>
`)

	for i := range 40 {
		body.WriteString(`<p>row ` + strconv.Itoa(i) + ` filler text for pagination</p>`)
	}

	body.WriteString(`</div></body></html>`)

	cssSheet := sheet(t, `
.scroller { overflow: auto; background: #f5f5f5; }
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

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 400, Height: 200, Background: true,
		Sheets: []*css.Stylesheet{cssSheet}, Media: "print",
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

	if doc.PageCount() < 2 {
		t.Fatalf("expected multi-page scroller content, got %d pages", doc.PageCount())
	}

	pagesWithSticky := map[int]bool{}

	var stick *box

	var walk func(b *box)
	walk = func(boxNode *box) {
		if boxNode.sticky {
			stick = boxNode

			return
		}

		for _, c := range boxNode.children {
			walk(c)
		}
	}
	walk(res.root)

	if stick == nil {
		t.Fatal("no sticky box")
	}

	if stick.stickyPort == nil {
		t.Fatal("expected overflow stickyPort on sticky box")
	}

	if !overflowCreatesStickyScrollport(stick.stickyPort.style.Overflow) {
		t.Fatalf("stickyPort overflow=%q, want auto/scroll/hidden", stick.stickyPort.style.Overflow)
	}

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		if strings.Contains(paintOp.Text, "STICKY") {
			pagesWithSticky[int(paintOp.Y/contentH)] = true
		}
	}

	if len(pagesWithSticky) != 1 {
		t.Fatalf("overflow sticky on %d page band(s) %v, want exactly 1 (no page clones)",
			len(pagesWithSticky), pagesWithSticky)
	}
}

func TestStickyOverflowClampAtOffsetZero(t *testing.T) {
	t.Parallel()
	// At scroll offset 0, sticky top:0 inside overflow stays at natural Y when
	// already below the scrollport top (no spurious jump to page top).
	cssSheet := sheet(t, `
.scroller { overflow: hidden; width: 200pt; padding-top: 20pt; background:#eee }
.stick { position: sticky; top: 0; height: 16pt; background:#333; color:#fff }
`)
	res := layoutHTML(t, `<html><body>
<div class="scroller"><div class="stick">S</div><p>below</p></div>
</body></html>`, cssSheet)
	doc := pdf.NewDocument()

	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatal(err)
	}

	var startY float64

	var found bool

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "S") {
			startY, found = op.Y, true

			break
		}
	}

	if !found {
		t.Fatal("sticky text missing")
	}
	// Natural position is below scroller padding — must not jump to y≈0 page top.
	if startY < 10 {
		t.Fatalf("sticky y=%.1f jumped toward page top; overflow scrollport at offset 0 should keep natural Y", startY)
	}
}

func TestStickyClampYTop(t *testing.T) {
	t.Parallel()
	boxNode := &box{ //nolint:exhaustruct // intentional zero fields
		sticky:       true,
		stickyTopSet: true,
		stickyTop:    0,
		height:       20,
	}
	// Natural y=50; page [200,400): stick to page top.
	posY := clampStickyY(50, 20, 0, 500, 200, 400, boxNode)
	if posY != 200 {
		t.Fatalf("clampStickyY = %.1f, want 200", posY)
	}
	// Already below sticky edge: no move.
	posY = clampStickyY(250, 20, 0, 500, 200, 400, boxNode)
	if posY != 250 {
		t.Fatalf("clampStickyY natural = %.1f, want 250", posY)
	}
}

func TestStickyClampYContainingBlockLimit(t *testing.T) {
	t.Parallel()
	boxNode := &box{ //nolint:exhaustruct // intentional zero fields
		sticky:       true,
		stickyTopSet: true,
		stickyTop:    0,
		height:       30,
	}
	// Would stick at 200, but CB ends at 220 → clamp to 190.
	y := clampStickyY(50, 30, 0, 220, 200, 400, boxNode)
	if y != 190 {
		t.Fatalf("CB limit = %.1f, want 190", y)
	}
}

func TestStickyTopContinuationPages(t *testing.T) {
	// Section CB spans multiple pages; print pagination must keep sticky in its
	// containing block instead of stamping it on continuation pages like fixed.
	var body strings.Builder

	body.WriteString(`<html><body>
<div class="sec">
  <div class="stick">STICKY</div>
`)

	for i := range 30 {
		body.WriteString(`<p>row `)
		body.WriteString(strconv.Itoa(i))
		body.WriteString(` filler text for pagination</p>`)
	}

	body.WriteString(`</div></body></html>`)

	cssSheet := sheet(t, `
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

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 400, Height: 200, Background: true,
		Sheets: []*css.Stylesheet{cssSheet}, Media: "print",
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

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		if strings.Contains(paintOp.Text, "STICKY") {
			page := int(paintOp.Y / contentH)
			pagesWithSticky[page] = true
		}
	}

	if len(pagesWithSticky) != 1 || !pagesWithSticky[0] {
		t.Fatalf("sticky text on %d page band(s) %v, want only its natural page", len(pagesWithSticky), pagesWithSticky)
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
	cssSheet := sheet(t, `
.sec { height: 80pt; background: #eee; }
.stick { position: sticky; top: 0; height: 20pt; background: #333; color: #fff; }
p { margin: 2pt 0; font-size: 11pt; }
`)

	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 400, Height: 200, Background: true,
		Sheets: []*css.Stylesheet{cssSheet}, Media: "print",
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
	walk = func(boxNode *box) {
		if boxNode.sticky {
			stick = boxNode

			return
		}

		for _, c := range boxNode.children {
			walk(c)
		}
	}
	walk(res.root)

	if stick == nil {
		t.Fatal("no sticky box")
	}

	cbBottom := stick.cbY + stick.cbH

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText || !strings.Contains(paintOp.Text, "STICKY") {
			continue
		}

		if paintOp.Y > cbBottom+0.5 {
			t.Errorf("sticky op y=%.1f past CB bottom %.1f", paintOp.Y, cbBottom)
		}
		// Must not appear on pages that start after the CB ends.
		page := int(paintOp.Y / contentH)
		pageTop := float64(page) * contentH

		if pageTop >= cbBottom-1e-6 {
			t.Errorf("sticky painted on page %d (pageTop=%.1f) after CB end %.1f", page, pageTop, cbBottom)
		}
	}
}

func TestStickyNotFixedReplication(t *testing.T) {
	t.Parallel()
	// Sticky in a short first section must not stamp onto later pages the way
	// position:fixed does.
	src := `<html><body>
<div class="sec"><div class="stick">STICKY</div><p>a</p></div>
` + strings.Repeat(`<p>pad</p>`, 50) + `
</body></html>`
	cssSheet := sheet(t, `
.sec { height: 60pt; }
.stick { position: sticky; top: 0; height: 18pt; background: #900; color: #fff; }
p { margin: 3pt 0; font-size: 12pt; }
`)

	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 400, Height: 200, Background: true,
		Sheets: []*css.Stylesheet{cssSheet}, Media: "print",
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
	t.Parallel()
	// top inset must not shift the box during layout the way relative does.
	s := sheet(t, `.s { position: sticky; top: 40pt; } .r { position: relative; top: 40pt; }`)
	resS := layoutHTML(t, `<html><body><div class="s">S</div></body></html>`, s)
	resR := layoutHTML(t, `<html><body><div class="r">R</div></body></html>`, s)

	var startY, ryVal float64

	var gotS, gotR bool

	for _, op := range resS.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "S") {
			startY, gotS = op.Y, true
		}
	}

	for _, op := range resR.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "R") {
			ryVal, gotR = op.Y, true
		}
	}

	if !gotS || !gotR {
		t.Fatalf("missing text ops sticky=%v relative=%v", gotS, gotR)
	}

	if startY >= ryVal-1 {
		t.Fatalf("sticky y=%.1f should be above relative y=%.1f (relative applies top at layout)", startY, ryVal)
	}
}

// TestStickyFixture31DoesNotCloneAcrossPages ensures the sticky header stays
// inside its containing block and does not become a position:fixed stamp.
func TestStickyFixture31DoesNotCloneAcrossPages(t *testing.T) {
	res, contentH, doc := paintFixture31(t)
	if doc.PageCount() < 2 {
		t.Fatalf("fixture-31 expected ≥2 pages, got %d", doc.PageCount())
	}

	pagesWithSticky := map[int]bool{}

	for _, op := range res.Ops {
		if op.Kind != OpText || !strings.Contains(op.Text, "Section header") {
			continue
		}

		pagesWithSticky[int(op.Y/contentH)] = true
	}

	if len(pagesWithSticky) != 1 || !pagesWithSticky[0] {
		t.Fatalf("fixture-31 sticky header on page band(s) %v, want only page 0", pagesWithSticky)
	}

	var row28Y, row29Y, row35Y float64

	var found28, found29, found35 bool

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		switch {
		case strings.Contains(paintOp.Text, "Row 28"):
			found28, row28Y = true, paintOp.Y
		case strings.Contains(paintOp.Text, "Row 29"):
			found29, row29Y = true, paintOp.Y
		case strings.Contains(paintOp.Text, "Row 35"):
			found35, row35Y = true, paintOp.Y
		}
	}

	if !found28 || !found29 || !found35 {
		t.Fatalf("fixture-31 rows missing: row28=%v row29=%v row35=%v", found28, found29, found35)
	}

	if int(row28Y/contentH) < 1 {
		t.Fatalf("Row 28 on page %d, want continuation", int(row28Y/contentH))
	}

	if gap := row29Y - row28Y; gap < 20 || gap > 35 {
		t.Errorf("Row 28→29 spacing = %.2f, want ~25pt (got Row28=%.2f Row29=%.2f)", gap, row28Y, row29Y)
	}

	if gap := row35Y - row29Y; gap < 100 || gap > 180 {
		t.Errorf("Row 29→35 spacing = %.2f, want natural six-row continuation", gap)
	}

	row35Enclosed := false

	for _, op := range res.Ops {
		if op.Kind != OpLine || op.W > 1 || op.H < 40 || !nearRGB(&op, 0.271, 0.353, 0.392) {
			continue
		}

		if op.Y <= row35Y && op.Y+op.H >= row35Y+10 {
			row35Enclosed = true

			break
		}
	}

	if !row35Enclosed {
		t.Fatalf("section side border does not enclose Row 35 at y=%.2f", row35Y)
	}
}

// TestStickyFixture31SplitFillsPreservePaintOrder ensures page-split section/row
// fills are not appended after continuation text (which would wash out rows).
func TestStickyFixture31SplitFillsPreservePaintOrder(t *testing.T) {
	res, contentH, doc := paintFixture31(t)
	if doc.PageCount() < 2 {
		t.Fatalf("fixture-31 expected ≥2 pages, got %d", doc.PageCount())
	}

	var row28Idx int = -1

	var row28Y float64

	for i, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "Row 28") {
			row28Idx = i
			row28Y = op.Y

			break
		}
	}

	if row28Idx < 0 {
		t.Fatal("Row 28 text missing")
	}

	if int(row28Y/contentH) < 1 {
		t.Fatalf("Row 28 not on continuation page (y=%.2f)", row28Y)
	}

	// Section gray continuation must appear before Row 28 text in the display
	// list (insert-after-original), so document order alone keeps it under ink.
	sectionBefore := false

	for i := range row28Idx {
		paintOp := res.Ops[i]
		if paintOp.Kind != OpFillRect || paintOp.StickyID != 0 {
			continue
		}

		if int(paintOp.Y/contentH) != 1 {
			continue
		}
		// Gray section background (#eceff1).
		if paintOp.R > 0.9 && paintOp.G > 0.9 && paintOp.B > 0.9 && paintOp.H > 50 &&
			paintOp.Y <= row28Y && paintOp.Y+paintOp.H >= row28Y {
			sectionBefore = true

			break
		}
	}

	if !sectionBefore {
		t.Error("section continuation fill must precede Row 28 text in Ops")
	}

	// At paint time, equal-z fills under Row 28's baseline must paint before
	// the text (sortPaintIndices chrome-under-content).
	pageIdxs := make([]int, 0, 32)

	for idx, op := range res.Ops {
		if op.Fixed {
			continue
		}

		if int(op.Y/contentH) != 1 {
			continue
		}

		pageIdxs = append(pageIdxs, idx)
	}

	sortPaintIndices(res.Ops, pageIdxs)

	row28Paint := -1

	for pi, idx := range pageIdxs {
		if idx == row28Idx {
			row28Paint = pi

			break
		}
	}

	if row28Paint < 0 {
		t.Fatal("Row 28 not in continuation page paint list")
	}

	for pidx := row28Paint + 1; pidx < len(pageIdxs); pidx++ {
		paintOp := res.Ops[pageIdxs[pidx]]
		if paintOp.Kind != OpFillRect {
			continue
		}

		if paintOp.Y >= row28Y || paintOp.Y+paintOp.H <= row28Y {
			continue
		}

		opZ := 0
		if paintOp.ZIndexSet {
			opZ = paintOp.ZIndex
		}

		if opZ >= 1 {
			continue
		}

		t.Errorf("paint-order: fill op[%d] after Row 28 covers baseline with z=%d",
			pageIdxs[pidx], opZ)
	}
}

// TestStickyFixture31AfterSectionNotCovered ensures the After-section note is
// present after Row 35 (not overlapping it) and not buried under section fill.
func TestStickyFixture31AfterSectionNotCovered(t *testing.T) {
	res, contentH, doc := paintFixture31(t)
	if doc.PageCount() < 2 {
		t.Fatalf("fixture-31 expected ≥2 pages, got %d", doc.PageCount())
	}

	var afterIdx int = -1

	var afterY, row35Y float64

	var afterText string

	var found35 bool

	for i, paintOp := range res.Ops {
		if paintOp.Kind == OpText && strings.Contains(paintOp.Text, "After the section") {
			afterIdx = i
			afterY = paintOp.Y
			afterText = paintOp.Text
		}

		if paintOp.Kind == OpText && strings.Contains(paintOp.Text, "Row 35") {
			row35Y = paintOp.Y
			found35 = true
		}
	}

	if afterIdx < 0 {
		t.Fatal("After-section text missing from display list")
	}

	if !found35 {
		t.Fatal("Row 35 text missing")
	}

	if int(afterY/contentH) < 1 {
		t.Fatalf("After-section expected on continuation page, y=%.2f", afterY)
	}

	if !strings.Contains(afterText, "sticky must not replicate") {
		t.Errorf("After text incomplete: %q", afterText)
	}
	// After must sit below Row 35 in the natural continuation flow.
	if afterY < row35Y+8 {
		t.Errorf("After overlaps Row 35: afterY=%.2f row35Y=%.2f", afterY, row35Y)
	}

	// Section gray must end at its own bottom border and never continue behind
	// the following sibling's margin/box.
	for idx := range len(res.Ops) {
		paintOp := res.Ops[idx]
		if paintOp.Kind != OpFillRect || !nearRGB(&paintOp, 0.925, 0.937, 0.945) {
			continue
		}

		if paintOp.Y >= afterY || paintOp.Y+paintOp.H <= afterY {
			continue
		}

		t.Errorf("op[%d] section fill covers After-text band (y=%.2f h=%.2f afterY=%.2f)",
			idx, paintOp.Y, paintOp.H, afterY)
	}

	var sectionBottom *Op

	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.Kind != OpLine || paintOp.H >= 1 || paintOp.W < 500 || !nearRGB(paintOp, 0.271, 0.353, 0.392) || paintOp.Y >= afterY {
			continue
		}

		if sectionBottom == nil || paintOp.Y > sectionBottom.Y {
			sectionBottom = paintOp
		}
	}

	if sectionBottom == nil {
		t.Fatal("section bottom border missing before After-section box")
	}
}

func paintFixture31(t *testing.T) (*Result, float64, *pdf.Document) {
	t.Helper()

	src, err := os.ReadFile("../../testdata/golden/fixture-31-sticky-top.html")
	if err != nil {
		t.Fatal(err)
	}

	htmlSrc := string(src)
	sidx := strings.Index(htmlSrc, "<style>")
	sjd := strings.Index(htmlSrc, "</style>")

	if sidx < 0 || sjd < 0 {
		t.Fatal("fixture missing <style>")
	}

	sheet, err := css.Parse(htmlSrc[sidx+7 : sjd])
	if err != nil {
		t.Fatal(err)
	}

	root, err := html.Parse(htmlSrc)
	if err != nil {
		t.Fatal(err)
	}

	pageW, pageH := 595.28, 841.89
	mat := 28.35
	contentW := pageW - 2*mat
	contentH := pageH - 2*mat

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: contentW, Height: contentH, Background: true,
		Sheets: []*css.Stylesheet{sheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}

	doc := pdf.NewDocument()
	if err := Paint(doc, res, PaintOptions{
		PageWidth: pageW, PageHeight: pageH,
		MarginTop: mat, MarginBottom: mat, MarginLeft: mat, MarginRight: mat,
	}); err != nil {
		t.Fatal(err)
	}

	return res, contentH, doc
}

func TestStickyFixture31Row28HasWhiteBackground(t *testing.T) {
	res, contentH, _ := paintFixture31(t)

	var row28Y float64

	found := false

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "Row 28") {
			row28Y, found = op.Y, true

			break
		}
	}

	if !found {
		t.Fatal("Row 28 text missing")
	}

	if int(row28Y/contentH) < 1 {
		t.Fatalf("Row 28 still on page 0 (y=%.2f)", row28Y)
	}

	covered := false

	var white Op

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpFillRect || paintOp.H < 5 || paintOp.H > 40 {
			continue
		}

		if paintOp.R < 0.99 || paintOp.G < 0.99 || paintOp.B < 0.99 {
			continue
		}
		// Must cover the text band AND start above the baseline so section
		// gray cannot show through ascenders/padding (was clamped to text Y).
		if paintOp.Y <= row28Y-2 && paintOp.Y+paintOp.H >= row28Y+4 {
			covered = true
			white = paintOp

			break
		}
	}

	if !covered {
		t.Errorf("Row 28 at y=%.2f has no white row background starting above baseline", row28Y)
	} else if white.Y >= row28Y-0.5 {
		t.Errorf("Row 28 white fill y=%.2f ≈ text y=%.2f; gray will show through", white.Y, row28Y)
	}
}

func TestStickyFixture31NoOrphanRowsOnPage1(t *testing.T) {
	res, contentH, _ := paintFixture31(t)

	var last0 float64

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText || paintOp.Fixed || int(paintOp.Y/contentH) != 0 {
			continue
		}

		if paintOp.Y > last0 {
			last0 = paintOp.Y
		}
	}

	if last0 == 0 {
		t.Fatal("no page-0 text")
	}

	lastBot := last0 + 14

	for idx, paintOp := range res.Ops {
		if paintOp.Fixed || paintOp.StickyID != 0 || int(paintOp.Y/contentH) != 0 {
			continue
		}

		if paintOp.Kind == OpFillRect && paintOp.H > 0.5 && paintOp.H <= 40 && paintOp.Y+paintOp.H/2 > lastBot+1 {
			t.Errorf("op[%d] orphan row fill y=%.2f h=%.2f below last text bot %.2f",
				idx, paintOp.Y, paintOp.H, lastBot)
		}
	}

	for idx, paintOp := range res.Ops {
		if paintOp.Fixed || paintOp.Kind != OpFillRect || int(paintOp.Y/contentH) != 0 ||
			!nearRGB(&paintOp, 0.925, 0.937, 0.945) || paintOp.H <= 40 {
			continue
		}

		if paintOp.Y+paintOp.H > last0+20 {
			t.Errorf("op[%d] sticky section fill extends %.2fpt below last page-0 row", idx, paintOp.Y+paintOp.H-last0)
		}
	}

	for idx, paintOp := range res.Ops {
		if paintOp.Fixed || paintOp.Kind != OpLine || int(paintOp.Y/contentH) != 0 || paintOp.W > 1 ||
			paintOp.H <= 40 || !nearRGB(&paintOp, 0.271, 0.353, 0.392) {
			continue
		}

		if paintOp.Y+paintOp.H > last0+20 {
			t.Errorf("op[%d] sticky section side border extends %.2fpt below last page-0 row", idx, paintOp.Y+paintOp.H-last0)
		}
	}

	closed := false

	for _, paintOp := range res.Ops {
		if paintOp.Fixed || paintOp.Kind != OpLine || int(paintOp.Y/contentH) != 0 || paintOp.H >= 1 || paintOp.W < 500 ||
			!nearRGB(&paintOp, 0.271, 0.353, 0.392) {
			continue
		}

		if paintOp.Y >= last0 && paintOp.Y <= last0+20 {
			closed = true

			break
		}
	}

	if !closed {
		t.Errorf("sticky section has no visible bottom border near last page-0 row at y=%.2f", last0)
	}
}
