//nolint:testpackage,wsl,nlreturn // white-box fixture chrome regressions
package layout

import (
	"math"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

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

	// After blueprint refresh --accent2 is #2563eb.
	const blueR, blueG, blueB = 0.145, 0.388, 0.922
	pageTop := float64(int(box.y/contentH)) * contentH

	for i, op := range res.Ops {
		if !fixture56HasRGB(op, blueR, blueG, blueB) {
			continue
		}
		// Horizontal top edge near the section page start.
		isTopLine := (op.Kind == OpLine && op.H < 4 && op.W > 100) ||
			(op.Kind == OpStrokeRect && op.StrokeMask == StrokeMaskTop)
		if !isTopLine {
			continue
		}
		if op.Y >= pageTop-1 && op.Y <= pageTop+8 {
			t.Fatalf("op[%d] blue top rail on domain-03 page: %+v (pageTop=%.2f)", i, op, pageTop)
		}
	}

	if box.style != nil && box.style.BorderTop.Width > 1.5 {
		t.Fatalf("domain-03 resolved border-top width = %.2f, want frame 1px", box.style.BorderTop.Width)
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

	for _, id := range []string{"domain-01", "domain-02"} {
		node := fixture56Node(root, func(n *html.Node) bool { return n.Attribute("id") == id })
		boxNode := fixture56BoxByNode(res.root, node)
		if boxNode == nil || boxNode.style == nil {
			t.Fatalf("%s missing", id)
		}

		padB := boxNode.style.PaddingBottom
		if padB < 4 {
			t.Fatalf("%s padding-bottom = %.2f, want authored space", id, padB)
		}

		var textBot, fillBot float64
		for i := boxNode.opStart; i <= boxNode.opEnd && i < len(res.Ops); i++ {
			op := res.Ops[i]
			if op.Kind == OpText {
				if bot := op.Y + opVisibleInkHeight(op); bot > textBot {
					textBot = bot
				}
			}
			if op.Kind == OpFillRect && op.R > 0.99 && op.W > boxNode.w*0.9 {
				if bot := op.Y + op.H; bot > fillBot {
					fillBot = bot
				}
			}
		}

		gap := fillBot - textBot
		// Allow a little font-ink vs line-box slack, but not a double pad.
		if gap < padB-2 || gap > padB+8 {
			t.Fatalf("%s footer-to-border gap = %.2f, want ~padding-bottom %.2f (textBot=%.2f fillBot=%.2f)",
				id, gap, padB, textBot, fillBot)
		}
	}
}

// TestFixture56MultiPageSectionFrameClosesOnlyOnLastPage: domain-01 spans
// pages 1–2; the first fragment must leave the bottom open, and the last
// fragment must paint a bottom border under the section footer.
func TestFixture56MultiPageSectionFrameClosesOnlyOnLastPage(t *testing.T) { //nolint:paralleltest // shared fixture fonts
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

	var firstFrag, lastFrag *Op
	for i := box.opStart; i <= box.opEnd && i < len(res.Ops); i++ {
		op := &res.Ops[i]
		if op.Kind != OpStrokeRect || op.W < box.w-1 || op.H < 40 {
			continue
		}
		// Frame stroke is the neutral #cbd5e1 (not the teal left rail).
		if op.R > 0.5 && op.B > 0.5 {
			if firstFrag == nil {
				firstFrag = op
			}
			lastFrag = op
		}
	}
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
	for _, op := range res.Ops {
		if op.Kind != OpFillRect || !fixture56HasRGB(op, paperR, paperG, paperB) {
			continue
		}
		if op.Y < pageTop-1 || op.Y >= pageBot {
			continue
		}
		if bot := op.Y + op.H; bot > bestBot {
			bestBot = bot
		}
	}
	if bestBot < pageBot-2 {
		t.Fatalf("page-2 paper wash bottom = %.2f, want ≥ %.2f (full content height)", bestBot, pageBot-2)
	}
}

// TestFixture56Domain08PageHasProgressAndNoStrayBottomLine uses the fixture's
// authored @page 12mm margins: domain-08 ends mid-page with the progress bar,
// and domain-09 must not leave a zero-height top-stroke hairline on that page.
func TestFixture56Domain08PageHasProgressAndNoStrayBottomLine(t *testing.T) { //nolint:paralleltest // shared fixture fonts
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

	// Green value fill on .d0n-progress (after "callback:").
	progress := fixture56Node(root, func(n *html.Node) bool {
		return n.Attribute("class") == "d0n-progress"
	})
	pbox := fixture56BoxByNode(res.root, progress)
	if pbox == nil {
		t.Fatal("d0n-progress missing")
	}
	foundGreen := false
	for i := pbox.opStart; i <= pbox.opEnd && i < len(res.Ops); i++ {
		op := res.Ops[i]
		if op.Kind == OpFillRect && op.W > 0 && op.H > 0 &&
			op.G > 0.35 && op.G > op.R*1.5 && op.G > op.B*1.5 {
			foundGreen = true
			break
		}
	}
	if !foundGreen {
		t.Fatal("d0n-progress missing green value fill")
	}

	// domain-09 must not park a zero-height / hairline frame fragment on the
	// page that still holds domain-08's footer (the stray full-width underline).
	d08 := fixture56BoxByNode(res.root, fixture56Node(root, func(n *html.Node) bool {
		return n.Attribute("id") == "domain-08"
	}))
	d09 := fixture56BoxByNode(res.root, fixture56Node(root, func(n *html.Node) bool {
		return n.Attribute("id") == "domain-09"
	}))
	if d08 == nil || d09 == nil {
		t.Fatal("domain-08/09 missing")
	}
	if math.Abs(d09.y-res.Ops[d09.opStart].Y) > 1 {
		// After pagination the first wide chrome op should share the box page top.
		var fillY float64
		for i := d09.opStart; i <= d09.opEnd && i < len(res.Ops); i++ {
			op := res.Ops[i]
			if (op.Kind == OpFillRect || op.Kind == OpStrokeRect) && op.W > d09.w*0.9 {
				fillY = op.Y
				break
			}
		}
		if math.Abs(d09.y-fillY) > 1 {
			t.Fatalf("domain-09 box.y=%.2f desynced from chrome Y=%.2f (page-boundary float shift)", d09.y, fillY)
		}
	}

	d08Page := int(d08.y / contentH)
	// Last page of domain-08: no zero-height full-width stroke with a top mask
	// after the section's own bottom (domain-09 bleed).
	pageTop := float64(d08Page) * contentH
	// domain-08 may span two pages; use the page of its bottom.
	endPage := int((d08.y + d08.height - layoutEpsilon) / contentH)
	pageTop = float64(endPage) * contentH
	pageBot := float64(endPage+1) * contentH
	d08Bot := d08.y + d08.height
	for i, op := range res.Ops {
		if op.Kind != OpStrokeRect && op.Kind != OpFillRect {
			continue
		}
		if op.W < contentW*0.8 {
			continue
		}
		if op.Y < pageTop-1 || op.Y >= pageBot-1 {
			continue
		}
		// Hairline / empty fragment below domain-08 content.
		if op.H > 1 {
			continue
		}
		if op.Y+op.H < d08Bot-1 {
			continue
		}
		if op.Kind == OpStrokeRect && op.StrokeMask&StrokeMaskTop != 0 {
			t.Fatalf("op[%d] zero-height top-stroke on domain-08 end page: Y=%.2f H=%.4f mask=%#b (stray underline)",
				i, op.Y, op.H, op.StrokeMask)
		}
		if op.Kind == OpFillRect && op.H == 0 && op.R > 0.99 && op.G > 0.99 && op.B > 0.99 {
			t.Fatalf("op[%d] zero-height white fill on domain-08 end page: Y=%.2f (stray fragment)", i, op.Y)
		}
	}
}
