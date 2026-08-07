package layout

import (
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

func TestOrphansWidowsParseAndInherit(t *testing.T) {
	s := sheet(t, `
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
	styles := resolveStyles(root, []*css.Stylesheet{s}, "", testViewport, 800)

	var gotO4, gotBad, gotInner, gotBody *ResolvedStyle

	for n, st := range styles {
		if n.Type != html.ElementNode {
			continue
		}

		switch n.Attribute("class") {
		case "o4":
			gotO4 = &st
		case "bad":
			gotBad = &st
		case "inner":
			gotInner = &st
		}

		if n.Name == "body" {
			gotBody = &st
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
	// Synthetic IFC leaf: 2 lines before the boundary, 3 after. orphans:4 makes
	// that Class B break illegal → whole block shifts to the next page.
	contentH := 100.0
	ops := []Op{
		{Kind: OpText, Y: 70, Text: "1", Size: 10},
		{Kind: OpText, Y: 85, Text: "2", Size: 10},
		{Kind: OpText, Y: 105, Text: "3", Size: 10},
		{Kind: OpText, Y: 120, Text: "4", Size: 10},
		{Kind: OpText, Y: 135, Text: "5", Size: 10},
	}
	root := &box{
		kind: "block", y: 60, h: 90, opStart: 0, opEnd: 4,
		style: ResolvedStyle{Orphans: 4, Widows: 2},
	}

	res := &Result{Ops: ops, root: root}
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
		{Kind: OpText, Y: 70, Text: "1", Size: 10},
		{Kind: OpText, Y: 85, Text: "2", Size: 10},
		{Kind: OpText, Y: 105, Text: "3", Size: 10},
		{Kind: OpText, Y: 120, Text: "4", Size: 10},
		{Kind: OpText, Y: 135, Text: "5", Size: 10},
	}
	root2 := &box{
		kind: "block", y: 60, h: 90, opStart: 0, opEnd: 4,
		style: ResolvedStyle{Orphans: 2, Widows: 2},
	}

	res2 := &Result{Ops: ops2, root: root2}
	if orphansWidows(res2, contentH) {
		t.Fatal("legal 2|3 split with orphans:2 widows:2 must not shift")
	}
}

func TestOrphansWidowsCSSIntegration(t *testing.T) {
	// End-to-end: parsed orphans:4 survives layout+paint; forced multi-line
	// paragraph text ends on a single page near the boundary.
	s := sheet(t, `
		body { margin: 0; font-size: 12pt; line-height: 14pt }
		.pad { height: 800pt }
		.keep { orphans: 4; widows: 2; width: 120pt; margin: 0 }
	`)
	src := `<html><body>
		<div class="pad"></div>
		<p class="keep">Line one here. Line two here. Line three here.
		Line four here. Line five here. Line six here. Line seven here.</p>
	</body></html>`
	res := layoutHTML(t, src, s)

	var keep *box

	var find func(b *box)
	find = func(b *box) {
		if b == nil {
			return
		}

		if b.node != nil && b.node.Attribute("class") == "keep" {
			keep = b
		}

		for _, c := range b.children {
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
		op := res.Ops[i]
		if op.Kind != OpText {
			continue
		}

		nLines++
		pages[int(op.Y/contentH)] = true
	}

	if nLines < 4 {
		t.Fatalf("expected ≥4 text ops, got %d", nLines)
	}

	if len(pages) != 1 {
		t.Fatalf("keep text spans %d pages, want 1", len(pages))
	}
}

func TestOrphansWidowsHeuristicFallback(t *testing.T) {
	// Geometric fallback: short straddling block (~14–60pt) moves wholly when
	// it fits the next page (no line boxes required).
	res := &Result{
		Ops: []Op{{Kind: OpFillRect, Y: 830, H: 30}},
	}
	b := &box{
		kind: "block", y: 830, h: 30, opStart: 0, opEnd: 0,
		style: ResolvedStyle{Orphans: 2, Widows: 2},
	}

	if !orphansWidowsHeuristic(res, b, 842) {
		t.Fatal("heuristic should shift short straddling block that fits")
	}

	if res.Ops[0].Y < 842-1e-6 {
		t.Fatalf("op Y = %v, want ≥ 842 after shift", res.Ops[0].Y)
	}
	// Outside the short-band → no heuristic move.
	b2 := &box{kind: "block", y: 800, h: 80, opStart: 0, opEnd: 0}
	if orphansWidowsHeuristic(res, b2, 842) {
		t.Fatal("heuristic must not move tall blocks (>60pt)")
	}
}

func TestBreakBeforeAlwaysIgnoresOrphans(t *testing.T) {
	// Forced break-before:always must land on a new page even when a preceding
	// paragraph has orphans:4 (forced breaks override widow/orphan limits).
	s := sheet(t, `
		body { margin: 0; font-size: 11pt }
		.para { orphans: 4; widows: 2; margin: 0 0 8px 0 }
		.force { break-before: always }
	`)

	var b strings.Builder

	b.WriteString(`<html><body>`)

	for range 40 {
		b.WriteString(`<p class="para">Forced-break filler paragraph number `)
		b.WriteString(strings.Repeat("word ", 12))
		b.WriteString(`MARKER-PRE.</p>`)
	}

	b.WriteString(`<div class="force"><p>FORCED-PAGE-START</p></div>`)
	b.WriteString(`</body></html>`)
	res := layoutHTML(t, b.String(), s)
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
