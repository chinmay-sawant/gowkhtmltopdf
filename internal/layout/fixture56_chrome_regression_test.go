//nolint:testpackage,wsl,nlreturn // white-box fixture chrome regressions
package layout

import (
	"math"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

func isBlueTopRailOp(paintOp Op) bool {
	const blueR, blueG, blueB = 0.145, 0.388, 0.922
	if !fixture56HasRGB(paintOp, blueR, blueG, blueB) {
		return false
	}
	return (paintOp.Kind == OpLine && paintOp.H < 4 && paintOp.W > 100) ||
		(paintOp.Kind == OpStrokeRect && paintOp.StrokeMask == StrokeMaskTop)
}

// TestFixture56Domain03HasNoBlueTopRail: blueprint .domains > section border
// must override section.d03 border-top accent so page starts have no blue bar.
func TestFixture56Domain03HasNoBlueTopRail(t *testing.T) { //nolint:paralleltest // shared fixture fonts
	root, sheet := loadFixture56(t)
	const (
		pageW  = 595.28
		pageH  = 841.89
		margin = 10 * 72.0 / 25.4
	)
	contentH := pageH - 2*margin
	res, err := Layout(root, Options{ //nolint:exhaustruct
		Width: pageW - 2*margin, Height: contentH,
		Background: true, Sheets: []*css.Stylesheet{sheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Paint(pdf.NewDocument(), res, PaintOptions{
		PageWidth: pageW, PageHeight: pageH,
		MarginTop: margin, MarginBottom: margin, MarginLeft: margin, MarginRight: margin,
	}); err != nil {
		t.Fatal(err)
	}

	d03 := fixture56Node(root, func(node *html.Node) bool { return node.Attribute("id") == "domain-03" })
	box := fixture56BoxByNode(res.root, d03)
	if box == nil {
		t.Fatal("domain-03 missing")
	}

	pageTop := float64(int(box.y/contentH)) * contentH

	for opIdx, paintOp := range res.Ops {
		if !isBlueTopRailOp(paintOp) {
			continue
		}
		if paintOp.Y >= pageTop-1 && paintOp.Y <= pageTop+8 {
			t.Fatalf("op[%d] blue top rail on domain-03 page: %+v (pageTop=%.2f)", opIdx, paintOp, pageTop)
		}
	}

	if box.style != nil && box.style.BorderTop.Width > 1.5 {
		t.Fatalf("domain-03 resolved border-top width = %.2f, want frame 1px", box.style.BorderTop.Width)
	}
}

func findDomainFooterTextAndFillBottoms(boxNode *box, ops []Op) (float64, float64) {
	var textBot, fillBot float64
	for i := boxNode.opStart; i <= boxNode.opEnd && i < len(ops); i++ {
		paintOp := ops[i]
		if paintOp.Kind == OpText {
			if bot := paintOp.Y + opVisibleInkHeight(paintOp); bot > textBot {
				textBot = bot
			}
		}
		if paintOp.Kind == OpFillRect && paintOp.R > 0.99 && paintOp.W > boxNode.w*0.9 {
			if bot := paintOp.Y + paintOp.H; bot > fillBot {
				fillBot = bot
			}
		}
	}
	return textBot, fillBot
}

func verifyDomainFooterPadding(t *testing.T, res *Result, root *html.Node, domainID string) {
	t.Helper()

	node := fixture56Node(root, func(n *html.Node) bool { return n.Attribute("id") == domainID })
	boxNode := fixture56BoxByNode(res.root, node)
	if boxNode == nil || boxNode.style == nil {
		t.Fatalf("%s missing", domainID)
	}

	padB := boxNode.style.PaddingBottom
	if padB < 4 {
		t.Fatalf("%s padding-bottom = %.2f, want authored space", domainID, padB)
	}

	textBot, fillBot := findDomainFooterTextAndFillBottoms(boxNode, res.Ops)

	gap := fillBot - textBot
	// Allow a little font-ink vs line-box slack, but not a double pad.
	if gap < padB-2 || gap > padB+8 {
		t.Fatalf("%s footer-to-border gap = %.2f, want ~padding-bottom %.2f (textBot=%.2f fillBot=%.2f)",
			domainID, gap, padB, textBot, fillBot)
	}
}

// TestFixture56SectionFooterKeepsBottomPadding: after the domain footer text,
// the white card must keep authored padding-bottom before the closing border
// (pages 2 and 4 were closing flush under the last line).
func TestFixture56SectionFooterKeepsBottomPadding(t *testing.T) { //nolint:paralleltest // shared fixture fonts
	root, sheet := loadFixture56(t)
	const (
		pageW  = 595.28
		pageH  = 841.89
		margin = 10 * 72.0 / 25.4
	)
	contentH := pageH - 2*margin
	res, err := Layout(root, Options{ //nolint:exhaustruct
		Width: pageW - 2*margin, Height: contentH,
		Background: true, Sheets: []*css.Stylesheet{sheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Paint(pdf.NewDocument(), res, PaintOptions{
		PageWidth: pageW, PageHeight: pageH,
		MarginTop: margin, MarginBottom: margin, MarginLeft: margin, MarginRight: margin,
	}); err != nil {
		t.Fatal(err)
	}

	for _, domainID := range []string{"domain-01", "domain-02"} {
		verifyDomainFooterPadding(t, res, root, domainID)
	}
}

func findFirstAndLastFrameFragments(boxNode *box, ops []Op) (*Op, *Op) {
	var firstFrag, lastFrag *Op
	for i := boxNode.opStart; i <= boxNode.opEnd && i < len(ops); i++ {
		paintOp := &ops[i]
		if paintOp.Kind != OpStrokeRect || paintOp.W < boxNode.w-1 || paintOp.H < 40 {
			continue
		}
		// Frame stroke is the neutral #cbd5e1 (not the teal left rail).
		if paintOp.R > 0.5 && paintOp.B > 0.5 {
			if firstFrag == nil {
				firstFrag = paintOp
			}
			lastFrag = paintOp
		}
	}
	return firstFrag, lastFrag
}

func verifyMultiPageFrameMasks(t *testing.T, firstFrag, lastFrag *Op) {
	t.Helper()

	if firstFrag == nil || lastFrag == nil || firstFrag == lastFrag {
		t.Fatalf("expected multi-page domain-01 frame fragments, got first=%v last=%v", firstFrag, lastFrag)
	}

	// Continuation page edge: first fragment must not paint a bottom close.
	if firstFrag.StrokeMask&StrokeMaskBottom != 0 {
		t.Fatalf("first frame fragment still closes bottom: mask=%#b", firstFrag.StrokeMask)
	}
	// Final page: bottom border required so the card closes under the footer.
	if lastFrag.StrokeMask&StrokeMaskBottom == 0 && lastFrag.StrokeMask != 0 {
		t.Fatalf("last frame fragment missing bottom close: mask=%#b", lastFrag.StrokeMask)
	}
	// Continuation must not re-open with a top rail on the second fragment.
	if lastFrag.StrokeMask&StrokeMaskTop != 0 {
		t.Fatalf("last frame fragment still paints top close: mask=%#b", lastFrag.StrokeMask)
	}
}

// TestFixture56MultiPageSectionFrameClosesOnlyOnLastPage: domain-01 spans
// pages 1–2; the first fragment must leave the bottom open, and the last
// fragment must paint a bottom border under the section footer.
//
//nolint:paralleltest // shared fixture fonts
func TestFixture56MultiPageSectionFrameClosesOnlyOnLastPage(t *testing.T) {
	root, sheet := loadFixture56(t)
	const (
		pageW  = 595.28
		pageH  = 841.89
		margin = 10 * 72.0 / 25.4
	)
	contentH := pageH - 2*margin
	res, err := Layout(root, Options{ //nolint:exhaustruct
		Width: pageW - 2*margin, Height: contentH,
		Background: true, Sheets: []*css.Stylesheet{sheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Paint(pdf.NewDocument(), res, PaintOptions{
		PageWidth: pageW, PageHeight: pageH,
		MarginTop: margin, MarginBottom: margin, MarginLeft: margin, MarginRight: margin,
	}); err != nil {
		t.Fatal(err)
	}

	d01 := fixture56Node(root, func(node *html.Node) bool { return node.Attribute("id") == "domain-01" })
	box := fixture56BoxByNode(res.root, d01)
	if box == nil {
		t.Fatal("domain-01 missing")
	}

	firstFrag, lastFrag := findFirstAndLastFrameFragments(box, res.Ops)
	verifyMultiPageFrameMasks(t, firstFrag, lastFrag)
}

// TestFixture56ShortPageKeepsPaperWashToBottom: when a domain section ends
// mid-page, the html/body paper fill must still reach the content-box bottom
// (not stop with the last section ink).
func TestFixture56ShortPageKeepsPaperWashToBottom(t *testing.T) { //nolint:paralleltest // shared fixture fonts
	root, sheet := loadFixture56(t)
	const (
		pageW  = 595.28
		pageH  = 841.89
		margin = 10 * 72.0 / 25.4
	)
	contentH := pageH - 2*margin
	res, err := Layout(root, Options{ //nolint:exhaustruct
		Width: pageW - 2*margin, Height: contentH,
		Background: true, Sheets: []*css.Stylesheet{sheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Paint(pdf.NewDocument(), res, PaintOptions{
		PageWidth: pageW, PageHeight: pageH,
		MarginTop: margin, MarginBottom: margin, MarginLeft: margin, MarginRight: margin,
	}); err != nil {
		t.Fatal(err)
	}

	// Page 2 (index 1) ends domain-01 early — paper must reach content bottom.
	const paperR, paperG, paperB = 0.933, 0.949, 0.957 // #eef2f4
	pageTop := contentH
	pageBot := 2 * contentH
	var bestBot float64
	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpFillRect || !fixture56HasRGB(paintOp, paperR, paperG, paperB) {
			continue
		}
		if paintOp.Y < pageTop-1 || paintOp.Y >= pageBot {
			continue
		}
		if bot := paintOp.Y + paintOp.H; bot > bestBot {
			bestBot = bot
		}
	}
	if bestBot < pageBot-2 {
		t.Fatalf("page-2 paper wash bottom = %.2f, want ≥ %.2f (full content height)", bestBot, pageBot-2)
	}
}

func checkDomain08ProgressGreen(t *testing.T, res *Result, root *html.Node) {
	t.Helper()

	progress := fixture56Node(root, func(n *html.Node) bool {
		return n.Attribute("class") == "d0n-progress"
	})
	pbox := fixture56BoxByNode(res.root, progress)
	if pbox == nil {
		t.Fatal("d0n-progress missing")
	}
	for i := pbox.opStart; i <= pbox.opEnd && i < len(res.Ops); i++ {
		paintOp := res.Ops[i]
		if paintOp.Kind == OpFillRect && paintOp.W > 0 && paintOp.H > 0 &&
			paintOp.G > 0.35 && paintOp.G > paintOp.R*1.5 && paintOp.G > paintOp.B*1.5 {
			return
		}
	}
	t.Fatal("d0n-progress missing green value fill")
}

func checkDomain09TopChromeSync(t *testing.T, res *Result, root *html.Node) {
	t.Helper()

	d09 := fixture56BoxByNode(res.root, fixture56Node(root, func(n *html.Node) bool {
		return n.Attribute("id") == "domain-09"
	}))
	if d09 == nil {
		t.Fatal("domain-09 missing")
	}
	if math.Abs(d09.y-res.Ops[d09.opStart].Y) <= 1 {
		return
	}
	// After pagination the first wide chrome op should share the box page top.
	var fillY float64
	for i := d09.opStart; i <= d09.opEnd && i < len(res.Ops); i++ {
		paintOp := res.Ops[i]
		if (paintOp.Kind == OpFillRect || paintOp.Kind == OpStrokeRect) && paintOp.W > d09.w*0.9 {
			fillY = paintOp.Y
			break
		}
	}
	if math.Abs(d09.y-fillY) > 1 {
		t.Fatalf("domain-09 box.y=%.2f desynced from chrome Y=%.2f (page-boundary float shift)", d09.y, fillY)
	}
}

func isStrayUnderlineOp(paintOp Op, pageTop, pageBot, d08Bot, contentW float64) bool {
	if paintOp.Kind != OpStrokeRect && paintOp.Kind != OpFillRect {
		return false
	}
	if paintOp.W < contentW*0.8 {
		return false
	}
	if paintOp.Y < pageTop-1 || paintOp.Y >= pageBot-1 {
		return false
	}
	// Hairline / empty fragment below domain-08 content.
	return paintOp.H <= 1 && paintOp.Y+paintOp.H >= d08Bot-1
}

func checkDomain08NoStrayUnderline(t *testing.T, res *Result, d08 *box, contentW, contentH float64) {
	t.Helper()

	// domain-08 may span two pages; use the page of its bottom.
	endPage := int((d08.y + d08.height - layoutEpsilon) / contentH)
	pageTop := float64(endPage) * contentH
	pageBot := float64(endPage+1) * contentH
	d08Bot := d08.y + d08.height
	for opIdx, paintOp := range res.Ops {
		if !isStrayUnderlineOp(paintOp, pageTop, pageBot, d08Bot, contentW) {
			continue
		}
		if paintOp.Kind == OpStrokeRect && paintOp.StrokeMask&StrokeMaskTop != 0 {
			t.Fatalf("op[%d] zero-height top-stroke on domain-08 end page: Y=%.2f H=%.4f mask=%#b (stray underline)",
				opIdx, paintOp.Y, paintOp.H, paintOp.StrokeMask)
		}
		if paintOp.Kind == OpFillRect && paintOp.H == 0 && paintOp.R > 0.99 && paintOp.G > 0.99 && paintOp.B > 0.99 {
			t.Fatalf("op[%d] zero-height white fill on domain-08 end page: Y=%.2f (stray fragment)", opIdx, paintOp.Y)
		}
	}
}

// TestFixture56Domain08PageHasProgressAndNoStrayBottomLine uses the fixture's
// authored @page 12mm margins: domain-08 ends mid-page with the progress bar,
// and domain-09 must not leave a zero-height top-stroke hairline on that page.
//
//nolint:paralleltest // shared fixture fonts
func TestFixture56Domain08PageHasProgressAndNoStrayBottomLine(t *testing.T) {
	root, sheet := loadFixture56(t)
	const (
		pageW  = 595.28
		pageH  = 841.89
		margin = 12 * 72.0 / 25.4 // match fixture @page { margin: 12mm }
	)
	contentH := pageH - 2*margin
	contentW := pageW - 2*margin
	res, err := Layout(root, Options{ //nolint:exhaustruct
		Width: contentW, Height: contentH,
		Background: true, Sheets: []*css.Stylesheet{sheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Paint(pdf.NewDocument(), res, PaintOptions{
		PageWidth: pageW, PageHeight: pageH,
		MarginTop: margin, MarginBottom: margin, MarginLeft: margin, MarginRight: margin,
	}); err != nil {
		t.Fatal(err)
	}

	checkDomain08ProgressGreen(t, res, root)

	d08 := fixture56BoxByNode(res.root, fixture56Node(root, func(n *html.Node) bool {
		return n.Attribute("id") == "domain-08"
	}))
	if d08 == nil {
		t.Fatal("domain-08 missing")
	}

	checkDomain09TopChromeSync(t, res, root)
	checkDomain08NoStrayUnderline(t, res, d08, contentW, contentH)
}
