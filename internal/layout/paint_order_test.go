package layout

import (
	"fmt"
	"strings"
	"testing"
)

func TestPaintOrderSharedPolicyKeepsStableMetadataOrder(t *testing.T) {
	t.Parallel()

	ops := []Op{
		{Kind: OpText, ZIndexSet: true},
		{Kind: OpFillRect, ZIndexSet: true},
		{Kind: OpText, ZIndexSet: true},
		{Kind: OpFillRect, ZIndexSet: true},
		{Kind: OpLinkURI, ZIndexSet: true},
	}
	// The test intentionally sets the actual z-index values below after the
	// compact literals keep the operation kinds easy to scan.
	ops[0].ZIndex = 0
	ops[1].ZIndex = 0
	ops[2].ZIndex = 2
	ops[3].ZIndex = -1
	ops[4].ZIndex = 0

	if got, want := PaintOrder(ops), []int{3, 1, 0, 4, 2}; !sameIndices(got, want) {
		t.Fatalf("paint order = %v, want %v", got, want)
	}

	subset := []int{4, 0, 1, 3}
	if got, want := paintOrderSubset(ops, subset), []int{3, 1, 0, 4}; !sameIndices(got, want) {
		t.Fatalf("subset paint order = %v, want %v", got, want)
	}
}

// chainedText builds a text op inside frame, mirroring what engine.add stamps
// for an element nested in a stacking context.
func chainedText(frame *paintCtxFrame, z int) Op {
	op := Op{Kind: OpText, ZIndexSet: true}
	op.ZIndex = z
	op.setZChain(frame)

	return op
}

// chainedChrome builds an element-chrome fill inside frame.
func chainedChrome(frame *paintCtxFrame, z int) Op {
	op := chainedText(frame, z)
	op.Kind = OpFillRect
	op.R, op.G, op.B = 1, 1, 1
	op.setChrome(chainDepth(frame))

	return op
}

// TestPaintStackingOrderNestedContexts pins the comparator contract for ops
// that carry stacking-context chains. A flat z sort paints main#main's z=1
// chrome after article.hentry's z=0 text; the chain keeps ancestor chrome
// and ancestor content below the nested context.
func TestPaintStackingOrderNestedContexts(t *testing.T) {
	t.Parallel()

	main := &paintCtxFrame{z: 1, seq: 1, depth: 1}
	identity := &paintCtxFrame{parent: main, z: 0, seq: 2, depth: 2}
	behind := &paintCtxFrame{parent: main, z: -1, seq: 3, depth: 2}
	front := &paintCtxFrame{z: 5, seq: 4, depth: 1}
	back := &paintCtxFrame{z: 1, seq: 5, depth: 1}
	firstSibling := &paintCtxFrame{z: 2, seq: 6, depth: 1}
	secondSibling := &paintCtxFrame{z: 2, seq: 7, depth: 1}

	cases := []struct {
		name        string
		left, right Op
		want        []int
	}{
		{"ancestor chrome before identity-transform text", chainedText(identity, 0), chainedChrome(main, 1), []int{1, 0}},
		{"ancestor content before identity-transform text", chainedText(identity, 0), chainedText(main, 1), []int{1, 0}},
		{"ancestor content after negative-z context", chainedText(main, 1), chainedText(behind, -1), []int{1, 0}},
		{"ancestor chrome before negative-z context", chainedChrome(main, 1), chainedText(behind, -1), []int{0, 1}},
		{"lower sibling context first", chainedText(front, 5), chainedText(back, 1), []int{1, 0}},
		{"equal-z siblings keep creation order", chainedText(secondSibling, 2), chainedText(firstSibling, 2), []int{1, 0}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ops := []Op{testCase.left, testCase.right}
			if got := PaintOrder(ops); !sameIndices(got, testCase.want) {
				t.Fatalf("paint order = %v, want %v", got, testCase.want)
			}
		})
	}
}

// learnCPPTriggerCSS is the verified minimal repro from the Phase 2
// reconnaissance: an ancestor with z-index plus a descendant whose transform
// resolves to identity.
const learnCPPTriggerCSS = `
html, body { margin:0; padding:0 }
main#main { position:relative; z-index:1; background:#fff; border-radius:15px;
            box-shadow:0 4px 16px -4px #0000007d }
article.hentry { display:block; position:relative; margin-bottom:2em;
                 opacity:1; transform:translateY(0) scale(1,1) }
p { margin:0 0 12px; font-size:16px; color:#000 }
`

// learnCPPReproHTML builds the trigger document with count paragraphs.
func learnCPPReproHTML(count int) string {
	var body strings.Builder

	body.WriteString(`<html><body><main id="main" class="main">`)
	body.WriteString(`<article id="post-8" class="post-8 page status-publish hentry">`)
	body.WriteString(`<div class="article-inner"><div class="entry-content">`)

	for i := range count {
		fmt.Fprintf(&body, "<p>Paragraph number %d carries enough words to paint a visible line of text.</p>", i)
	}

	body.WriteString(`</div></div></article></main></body></html>`)

	return body.String()
}

// orderPositions maps each op index to its paint-order position.
func orderPositions(ops []Op) map[int]int {
	positions := make(map[int]int, len(ops))
	for pos, idx := range PaintOrder(ops) {
		positions[idx] = pos
	}

	return positions
}

// TestTransformedDescendantTextPaintsAfterAncestorChrome replicates Agent B's
// learncpp repro: main#main keeps z-index:1 while article.hentry creates a
// nested context through an identity transform. Every chrome op must sort
// before every text op, and the identity transform must still create a
// context (z=0 is recorded, not ignored).
func TestTransformedDescendantTextPaintsAfterAncestorChrome(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, learnCPPReproHTML(60), sheet(t, learnCPPTriggerCSS))
	positions := orderPositions(res.Ops)

	chromeMax, textMin, chromeOps, textOps := paintOrderBounds(t, res.Ops, positions)

	if chromeOps == 0 || textOps == 0 {
		t.Fatalf("ops chrome=%d text=%d, want both nonzero", chromeOps, textOps)
	}

	assertChromeDepth(t, res.Ops)

	if chromeMax >= textMin {
		t.Fatalf("chrome order %d >= first text order %d; ancestor chrome covered by text", chromeMax, textMin)
	}
}

// paintOrderBounds returns the last chrome paint position, the first text
// position, and both op counts for ops carrying order positions. It also
// checks that every text op carries the identity-transform context chain
// (z=0 set, one frame below main).
func paintOrderBounds(t *testing.T, ops []Op, positions map[int]int) (int, int, int, int) {
	t.Helper()

	chromeMax, textMin := -1, len(ops)
	textOps, chromeOps := 0, 0

	for idx, paintOp := range ops {
		switch {
		case paintOp.isChrome():
			chromeOps++

			if positions[idx] > chromeMax {
				chromeMax = positions[idx]
			}
		case paintOp.Kind == OpText:
			textOps++

			if positions[idx] < textMin {
				textMin = positions[idx]
			}

			// The identity transform still creates a context: text carries
			// z=0 set and sits one frame below main.
			if !paintOp.ZIndexSet || paintOp.ZIndex != 0 || chainDepth(paintOp.zChain()) != 2 {
				t.Fatalf("text op %d: z=(%v,%d) chain depth=%d, want context z=0 depth 2",
					idx, paintOp.ZIndexSet, paintOp.ZIndex, chainDepth(paintOp.zChain()))
			}
		}
	}

	return chromeMax, textMin, chromeOps, textOps
}

// assertChromeDepth fails when a chrome op sits at a depth other than the
// ancestor context depth of 1 created by main#main in the repro.
func assertChromeDepth(t *testing.T, ops []Op) {
	t.Helper()

	// main#main creates the ancestor context, so its chrome depth is 1.
	for idx, paintOp := range ops {
		if paintOp.isChrome() && paintOp.chromeDepth() != 1 {
			t.Fatalf("chrome op %d depth = %d, want 1", idx, paintOp.chromeDepth())
		}
	}
}

// TestControlAncestorChromePaintsFirstWithoutContexts is the control for the
// repro: without z-index and transform no stacking context exists, all ops
// stay in the root chain, and chrome still paints first.
func TestControlAncestorChromePaintsFirstWithoutContexts(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
html, body { margin:0; padding:0 }
main#main { position:relative; background:#fff; border-radius:15px;
            box-shadow:0 4px 16px -4px #0000007d }
article.hentry { display:block; position:relative; margin-bottom:2em; opacity:1 }
p { margin:0 0 12px; font-size:16px; color:#000 }
`)
	res := layoutHTML(t, learnCPPReproHTML(60), cssSheet)
	positions := orderPositions(res.Ops)

	chromeMax, textMin := -1, len(res.Ops)
	chromeOps, textOps := 0, 0

	for idx, paintOp := range res.Ops {
		if paintOp.zChain() != nil {
			t.Fatalf("op %d carries a stacking frame without z-index/transform", idx)
		}

		switch {
		case paintOp.isChrome():
			chromeOps++

			if positions[idx] > chromeMax {
				chromeMax = positions[idx]
			}
		case paintOp.Kind == OpText:
			textOps++

			if positions[idx] < textMin {
				textMin = positions[idx]
			}
		}
	}

	if chromeOps == 0 || textOps == 0 {
		t.Fatalf("ops chrome=%d text=%d, want both nonzero", chromeOps, textOps)
	}

	if chromeMax >= textMin {
		t.Fatalf("chrome order %d >= first text order %d in control", chromeMax, textMin)
	}
}

func sameIndices(got, want []int) bool {
	if len(got) != len(want) {
		return false
	}

	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}

	return true
}
