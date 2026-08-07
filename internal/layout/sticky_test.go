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
	for i := 0; i < 40; i++ {
		body.WriteString(`<p>row ` + strconv.Itoa(i) + ` filler text for pagination</p>`)
	}
	body.WriteString(`</div></body></html>`)

	s := sheet(t, `
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
	if doc.PageCount() < 2 {
		t.Fatalf("expected multi-page scroller content, got %d pages", doc.PageCount())
	}
	pagesWithSticky := map[int]bool{}
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
	if stick.stickyPort == nil {
		t.Fatal("expected overflow stickyPort on sticky box")
	}
	if !overflowCreatesStickyScrollport(stick.stickyPort.style.Overflow) {
		t.Fatalf("stickyPort overflow=%q, want auto/scroll/hidden", stick.stickyPort.style.Overflow)
	}
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}
		if strings.Contains(op.Text, "STICKY") {
			pagesWithSticky[int(op.Y/contentH)] = true
		}
	}
	if len(pagesWithSticky) != 1 {
		t.Fatalf("overflow sticky on %d page band(s) %v, want exactly 1 (no page clones)",
			len(pagesWithSticky), pagesWithSticky)
	}
}

func TestStickyOverflowClampAtOffsetZero(t *testing.T) {
	// At scroll offset 0, sticky top:0 inside overflow stays at natural Y when
	// already below the scrollport top (no spurious jump to page top).
	s := sheet(t, `
.scroller { overflow: hidden; width: 200pt; padding-top: 20pt; background:#eee }
.stick { position: sticky; top: 0; height: 16pt; background:#333; color:#fff }
`)
	res := layoutHTML(t, `<html><body>
<div class="scroller"><div class="stick">S</div><p>below</p></div>
</body></html>`, s)
	doc := pdf.NewDocument()
	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatal(err)
	}
	var sy float64
	var found bool
	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "S") {
			sy, found = op.Y, true
			break
		}
	}
	if !found {
		t.Fatal("sticky text missing")
	}
	// Natural position is below scroller padding — must not jump to y≈0 page top.
	if sy < 10 {
		t.Fatalf("sticky y=%.1f jumped toward page top; overflow scrollport at offset 0 should keep natural Y", sy)
	}
}

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
		body.WriteString(`<p>row ` + strconv.Itoa(i) + ` filler text for pagination</p>`)
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

// TestStickyFixture31ContinuationClearsFlow ensures continuation-page sticky
// clones reserve space so flow starts just under the sticky bar (thead-style),
// row fills clear the sticky band, and Row 28/29 keep natural spacing.
func TestStickyFixture31ContinuationClearsFlow(t *testing.T) {
	res, contentH, doc := paintFixture31(t)
	if doc.PageCount() < 2 {
		t.Fatalf("fixture-31 expected ≥2 pages, got %d", doc.PageCount())
	}

	pt := contentH
	stickyBot := pt
	foundFill := false
	for _, op := range res.Ops {
		if op.Kind != OpFillRect || op.StickyID != 0 {
			continue
		}
		// Sticky clone fill: near page-1 top, bar-sized height.
		if op.Y < pt-1 || op.Y > pt+5 || op.H < 20 || op.H > 40 {
			continue
		}
		bot := op.Y + op.H
		if bot > stickyBot {
			stickyBot = bot
			foundFill = true
		}
	}
	if !foundFill {
		t.Fatal("no sticky clone fill on continuation page")
	}

	var row28Y, row29Y float64
	var found28, found29 bool
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}
		switch {
		case strings.Contains(op.Text, "Row 28"):
			found28, row28Y = true, op.Y
			if int(op.Y/contentH) < 1 {
				t.Errorf("Row 28 still on page %d (y=%.2f), want continuation", int(op.Y/contentH), op.Y)
			}
		case strings.Contains(op.Text, "Row 29"):
			found29, row29Y = true, op.Y
		}
	}
	if !found28 {
		t.Fatal("Row 28 text op missing")
	}
	if !found29 {
		t.Fatal("Row 29 text op missing")
	}
	// Natural row pitch is ~25pt; snapping Row 28 alone used to collapse this
	// to ~14pt so both lines looked like one cell.
	if gap := row29Y - row28Y; gap < 20 || gap > 35 {
		t.Errorf("Row 28→29 spacing = %.2f, want ~25pt (got Row28=%.2f Row29=%.2f)", gap, row28Y, row29Y)
	}
	// First continuation baseline sits under the sticky bottom (not through it).
	if gap := row28Y - stickyBot; gap < 6 || gap > 24 {
		t.Errorf("gap sticky→Row28 = %.2f (stickyBot=%.2f row28=%.2f), want ~10pt under bar", gap, stickyBot, row28Y)
	}

	// Split row fills must not sit in the sticky band. Tall page-leading
	// section chrome may remain under the sticky clone by design.
	for i, op := range res.Ops {
		if op.Kind != OpFillRect || op.StickyID != 0 {
			continue
		}
		if op.Y < pt-1 || op.Y >= stickyBot-0.5 {
			continue
		}
		if int(op.Y/contentH) != 1 {
			continue
		}
		// Ignore the sticky clone itself.
		if op.H >= 20 && op.H <= 40 && op.Y <= pt+5 {
			continue
		}
		if isPageLeadingBackground(&op, pt, stickyBot-pt) {
			continue
		}
		t.Errorf("op[%d] fill y=%.2f h=%.2f sits under sticky band [%.2f,%.2f)",
			i, op.Y, op.H, pt, stickyBot)
	}

	// Section side borders must extend past Row 35 after the sticky shift.
	var row35Y float64
	var found35 bool
	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "Row 35") {
			row35Y, found35 = op.Y, true
			break
		}
	}
	if !found35 {
		t.Fatal("Row 35 text missing")
	}
	borderCovers35 := false
	for _, op := range res.Ops {
		if op.Kind != OpLine || int(op.Y/contentH) != 1 {
			continue
		}
		if op.Y > pt+1 || op.H < 40 {
			continue
		}
		if op.Y+op.H >= row35Y-0.5 {
			borderCovers35 = true
			break
		}
	}
	if !borderCovers35 {
		t.Errorf("section border does not extend to Row 35 (y=%.2f)", row35Y)
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
	for i := 0; i < row28Idx; i++ {
		op := res.Ops[i]
		if op.Kind != OpFillRect || op.StickyID != 0 {
			continue
		}
		if int(op.Y/contentH) != 1 {
			continue
		}
		// Gray section background (#eceff1).
		if op.R > 0.9 && op.G > 0.9 && op.B > 0.9 && op.H > 50 &&
			op.Y <= row28Y && op.Y+op.H >= row28Y {
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
	for i, op := range res.Ops {
		if op.Fixed {
			continue
		}
		if int(op.Y/contentH) != 1 {
			continue
		}
		pageIdxs = append(pageIdxs, i)
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
	for pi := row28Paint + 1; pi < len(pageIdxs); pi++ {
		op := res.Ops[pageIdxs[pi]]
		if op.Kind != OpFillRect {
			continue
		}
		if op.Y >= row28Y || op.Y+op.H <= row28Y {
			continue
		}
		oz := 0
		if op.ZIndexSet {
			oz = op.ZIndex
		}
		if oz >= 1 {
			continue
		}
		t.Errorf("paint-order: fill op[%d] after Row 28 covers baseline with z=%d",
			pageIdxs[pi], oz)
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
	for i, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "After the section") {
			afterIdx = i
			afterY = op.Y
			afterText = op.Text
		}
		if op.Kind == OpText && strings.Contains(op.Text, "Row 35") {
			row35Y = op.Y
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
	// After must sit below Row 35 (sticky reserve shifts both equally).
	if afterY < row35Y+8 {
		t.Errorf("After overlaps Row 35: afterY=%.2f row35Y=%.2f", afterY, row35Y)
	}

	// Late section gray must not follow After text while covering its Y
	// (would hide the cream box / note the way position:fixed would not).
	for i := afterIdx + 1; i < len(res.Ops); i++ {
		op := res.Ops[i]
		if op.Kind != OpFillRect {
			continue
		}
		if op.Y >= afterY || op.Y+op.H <= afterY {
			continue
		}
		// Gray section (#eceff1), not sticky blue / cream after-box.
		if op.R > 0.9 && op.G > 0.9 && op.B > 0.92 && op.B < 0.97 && op.H > 50 {
			t.Errorf("op[%d] section fill after After-text covers it (y=%.2f h=%.2f)",
				i, op.Y, op.H)
		}
	}
}

func paintFixture31(t *testing.T) (*Result, float64, *pdf.Document) {
	t.Helper()
	src, err := os.ReadFile("../../testdata/golden/fixture-31-sticky-top.html")
	if err != nil {
		t.Fatal(err)
	}
	htmlSrc := string(src)
	si := strings.Index(htmlSrc, "<style>")
	sj := strings.Index(htmlSrc, "</style>")
	if si < 0 || sj < 0 {
		t.Fatal("fixture missing <style>")
	}
	sheet, err := css.Parse(htmlSrc[si+7 : sj])
	if err != nil {
		t.Fatal(err)
	}
	root, err := html.Parse(htmlSrc)
	if err != nil {
		t.Fatal(err)
	}
	pageW, pageH := 595.28, 841.89
	m := 28.35
	contentW := pageW - 2*m
	contentH := pageH - 2*m
	res, err := Layout(root, Options{
		Width: contentW, Height: contentH, Background: true,
		Sheets: []*css.Stylesheet{sheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}
	doc := pdf.NewDocument()
	if err := Paint(doc, res, PaintOptions{
		PageWidth: pageW, PageHeight: pageH,
		MarginTop: m, MarginBottom: m, MarginLeft: m, MarginRight: m,
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
	for _, op := range res.Ops {
		if op.Kind != OpFillRect || op.H < 5 || op.H > 40 {
			continue
		}
		if op.R < 0.99 || op.G < 0.99 || op.B < 0.99 {
			continue
		}
		// Must cover the text band AND start above the baseline so section
		// gray cannot show through ascenders/padding (was clamped to text Y).
		if op.Y <= row28Y-2 && op.Y+op.H >= row28Y+4 {
			covered = true
			white = op
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
	for _, op := range res.Ops {
		if op.Kind != OpText || op.Fixed || int(op.Y/contentH) != 0 {
			continue
		}
		if op.Y > last0 {
			last0 = op.Y
		}
	}
	if last0 == 0 {
		t.Fatal("no page-0 text")
	}
	lastBot := last0 + 14
	for i, op := range res.Ops {
		if op.Fixed || op.StickyID != 0 || int(op.Y/contentH) != 0 {
			continue
		}
		if op.Kind == OpFillRect && op.H > 0.5 && op.H <= 40 && op.Y+op.H/2 > lastBot+1 {
			t.Errorf("op[%d] orphan row fill y=%.2f h=%.2f below last text bot %.2f",
				i, op.Y, op.H, lastBot)
		}
	}
	// Section chrome should end near last content, not the page boundary.
	boundary := contentH
	for i, op := range res.Ops {
		if op.Kind != OpFillRect || op.H < 50 || int(op.Y/contentH) != 0 {
			continue
		}
		if op.Y+op.H > boundary-5 {
			t.Errorf("op[%d] section fill still reaches page bottom (bot=%.2f boundary=%.2f)",
				i, op.Y+op.H, boundary)
		}
		if op.Y+op.H > lastBot+40 {
			t.Errorf("op[%d] section fill bot=%.2f too far past last text %.2f",
				i, op.Y+op.H, lastBot)
		}
	}
}
