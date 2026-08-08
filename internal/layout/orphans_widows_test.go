//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

func TestOrphansWidowsParseAndInherit(t *testing.T) { //nolint:cyclop
	t.Parallel()

	cssSheet := sheet(t, `
		.o4 { orphans: 4; widows: 2 }
		.bad { orphans: 0; widows: -1; orphans: 1.5; widows: auto }
		.outer { orphans: 5; widows: 3 }
	`)
	src := `<html><body>
		<p class="o4">a</p>
		<p class="bad">b</p>
		<div class="outer"><p class="inner">c</p></div>
	</body></html>`
	root := mustParse(t, src)
	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "", testViewport, 800)

	var gotO4, gotBad, gotInner, gotBody *ResolvedStyle

	for node, sty := range styles {
		if node.Type != html.ElementNode {
			continue
		}

		switch node.Attribute("class") {
		case "o4":
			gotO4 = sty
		case "bad":
			gotBad = sty
		case "inner":
			gotInner = sty
		}

		if node.Name == "body" {
			gotBody = sty
		}
	}

	if gotO4 == nil || gotO4.Orphans != 4 || gotO4.Widows != 2 {
		t.Fatalf("o4 = orphans %d widows %d, want 4/2",
			ptrOrphans(gotO4), ptrWidows(gotO4))
	}
	// Invalid values ignored → initial 2.
	if gotBad == nil || gotBad.Orphans != 2 || gotBad.Widows != 2 {
		t.Fatalf("bad = orphans %d widows %d, want initial 2/2",
			ptrOrphans(gotBad), ptrWidows(gotBad))
	}

	if gotBody == nil || gotBody.Orphans != 2 || gotBody.Widows != 2 {
		t.Fatalf("body initial = orphans %d widows %d, want 2/2",
			ptrOrphans(gotBody), ptrWidows(gotBody))
	}
	// Inherited from .outer (no own declaration).
	if gotInner == nil || gotInner.Orphans != 5 || gotInner.Widows != 3 {
		t.Fatalf("inner inherit = orphans %d widows %d, want 5/3",
			ptrOrphans(gotInner), ptrWidows(gotInner))
	}
}

func ptrOrphans(st *ResolvedStyle) int {
	if st == nil {
		return -1
	}

	return st.Orphans
}

func ptrWidows(st *ResolvedStyle) int {
	if st == nil {
		return -1
	}

	return st.Widows
}

func TestOrphansWidowsLineAwareKeepTogether(t *testing.T) {
	t.Parallel()
	// Synthetic IFC leaf: 2 lines before the boundary, 3 after. orphans:4 makes
	// that Class B break illegal → whole block shifts to the next page.
	contentH := 100.0
	ops := []Op{
		{Kind: OpText, Y: 70, Text: "1", Size: 10},  //nolint:exhaustruct // intentional zero fields
		{Kind: OpText, Y: 85, Text: "2", Size: 10},  //nolint:exhaustruct // intentional zero fields
		{Kind: OpText, Y: 105, Text: "3", Size: 10}, //nolint:exhaustruct // intentional zero fields
		{Kind: OpText, Y: 120, Text: "4", Size: 10}, //nolint:exhaustruct // intentional zero fields
		{Kind: OpText, Y: 135, Text: "5", Size: 10}, //nolint:exhaustruct // intentional zero fields
	}

	root := &box{ //nolint:exhaustruct // intentional zero fields
		kind: displayBlock, y: 60, height: 90, opStart: 0, opEnd: 4,
		style: &ResolvedStyle{Orphans: 4, Widows: 2}, //nolint:exhaustruct // intentional zero fields
	}

	res := &Result{Ops: ops, root: root} //nolint:exhaustruct // intentional zero fields
	if !orphansWidows(res, contentH) {
		t.Fatal("expected Rule 3 keep-together shift for orphans:4 with 2|3 split")
	}

	if root.y < contentH-1e-6 {
		t.Fatalf("box y=%.1f, want ≥ %.1f after keep-together", root.y, contentH)
	}

	for i, op := range res.Ops {
		if op.Y < contentH-1e-6 {
			t.Fatalf("op %d Y=%.1f still on prior page after keep-together", i, op.Y)
		}
	}

	// Legal split (orphans:2, widows:2 with 2|3) must not move.
	ops2 := []Op{
		{Kind: OpText, Y: 70, Text: "1", Size: 10},  //nolint:exhaustruct // intentional zero fields
		{Kind: OpText, Y: 85, Text: "2", Size: 10},  //nolint:exhaustruct // intentional zero fields
		{Kind: OpText, Y: 105, Text: "3", Size: 10}, //nolint:exhaustruct // intentional zero fields
		{Kind: OpText, Y: 120, Text: "4", Size: 10}, //nolint:exhaustruct // intentional zero fields
		{Kind: OpText, Y: 135, Text: "5", Size: 10}, //nolint:exhaustruct // intentional zero fields
	}

	root2 := &box{ //nolint:exhaustruct // intentional zero fields
		kind: displayBlock, y: 60, height: 90, opStart: 0, opEnd: 4,
		style: &ResolvedStyle{Orphans: 2, Widows: 2}, //nolint:exhaustruct // intentional zero fields
	}

	res2 := &Result{Ops: ops2, root: root2} //nolint:exhaustruct // intentional zero fields
	if orphansWidows(res2, contentH) {
		t.Fatal("legal 2|3 split with orphans:2 widows:2 must not shift")
	}
}

func TestOrphansWidowsCSSIntegration(t *testing.T) { //nolint:cyclop,funlen
	t.Parallel()
	// End-to-end: parsed orphans:4 survives layout+paint; forced multi-line
	// paragraph text ends on a single page near the boundary.
	cssSheet := sheet(t, `
		body { margin: 0; font-size: 12pt; line-height: 14pt }
		.pad { height: 800pt }
		.keep { orphans: 4; widows: 2; width: 120pt; margin: 0 }
	`)
	src := `<html><body>
		<div class="pad"></div>
		<p class="keep">Line one here. Line two here. Line three here.
		Line four here. Line five here. Line six here. Line seven here.</p>
	</body></html>`
	res := layoutHTML(t, src, cssSheet)

	var keep *box

	var find func(b *box)
	find = func(boxNode *box) {
		if boxNode == nil {
			return
		}

		if boxNode.node != nil && boxNode.node.Attribute("class") == "keep" {
			keep = boxNode
		}

		for _, c := range boxNode.children {
			find(c)
		}
	}
	find(res.root)

	if keep == nil || keep.style.Orphans != 4 || keep.style.Widows != 2 {
		t.Fatalf("keep style orphans/widows = %v, want 4/2", keep)
	}

	doc := pdf.NewDocument()
	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatal(err)
	}

	contentH := paintOpts().PageHeight
	pages := map[int]bool{}
	nLines := 0

	for i := keep.opStart; i <= keep.opEnd && i < len(res.Ops); i++ {
		paintOp := res.Ops[i]
		if paintOp.Kind != OpText {
			continue
		}

		nLines++
		pages[int(paintOp.Y/contentH)] = true
	}

	if nLines < 4 {
		t.Fatalf("expected ≥4 text ops, got %d", nLines)
	}

	if len(pages) != 1 {
		t.Fatalf("keep text spans %d pages, want 1", len(pages))
	}
}

func TestOrphansWidowsHeuristicFallback(t *testing.T) {
	t.Parallel()
	// Geometric fallback: short straddling block (~14–60pt) moves wholly when
	// it fits the next page (no line boxes required).
	res := &Result{ //nolint:exhaustruct // intentional zero fields
		Ops: []Op{{Kind: OpFillRect, Y: 830, H: 30}}, //nolint:exhaustruct // intentional zero fields
	}

	b := &box{ //nolint:exhaustruct // intentional zero fields
		kind: displayBlock, y: 830, height: 30, opStart: 0, opEnd: 0,
		style: &ResolvedStyle{Orphans: 2, Widows: 2}, //nolint:exhaustruct // intentional zero fields
	}

	if !orphansWidowsHeuristic(res, b, 842) {
		t.Fatal("heuristic should shift short straddling block that fits")
	}

	if res.Ops[0].Y < 842-1e-6 {
		t.Fatalf("op Y = %v, want ≥ 842 after shift", res.Ops[0].Y)
	}
	// Outside the short-band → no heuristic move.
	b2 := &box{kind: displayBlock, y: 800, height: 80, opStart: 0, opEnd: 0} //nolint:exhaustruct // zero fields
	if orphansWidowsHeuristic(res, b2, 842) {
		t.Fatal("heuristic must not move tall blocks (>60pt)")
	}
}

func TestBreakBeforeAlwaysIgnoresOrphans(t *testing.T) {
	t.Parallel()
	// Forced break-before:always must land on a new page even when a preceding
	// paragraph has orphans:4 (forced breaks override widow/orphan limits).
	cssSheet := sheet(t, `
		body { margin: 0; font-size: 11pt }
		.para { orphans: 4; widows: 2; margin: 0 0 8px 0 }
		.force { break-before: always }
	`)

	var boxNode strings.Builder

	boxNode.WriteString(`<html><body>`)

	for range 40 {
		boxNode.WriteString(`<p class="para">Forced-break filler paragraph number `)
		boxNode.WriteString(strings.Repeat("word ", 12))
		boxNode.WriteString(`MARKER-PRE.</p>`)
	}

	boxNode.WriteString(`<div class="force"><p>FORCED-PAGE-START</p></div>`)
	boxNode.WriteString(`</body></html>`)
	res := layoutHTML(t, boxNode.String(), cssSheet)
	doc := pdf.NewDocument()

	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatal(err)
	}

	pForce := pageOf(t, res, "FORCED-PAGE-START")
	pPre := pageOf(t, res, "MARKER-PRE")

	if pForce <= pPre {
		t.Fatalf("forced break page %d should be after preceding content page %d",
			pForce, pPre)
	}

	if pForce < 1 {
		t.Fatalf("forced break on page %d, want ≥1", pForce)
	}
}
