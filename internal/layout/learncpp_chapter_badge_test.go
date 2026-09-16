package layout

import (
	"strings"
	"testing"
)

// learncpp.com chapter header badge (Phase 10). The live page carries the
// global border-box setup (html{box-sizing:border-box}, *{box-sizing:inherit})
// and .main{width:100%;padding:0 22pt}; the Chapter N pill is a float:right
// inside .lessontable-header. Before the fix the ignore of box-sizing:inherit
// made .main content-box, 44pt wider than the page content box, so the card
// and the pill were laid out past the overflow:hidden clip at the right page
// edge: the badge background was clamped (56.2pt instead of 68pt) and the
// white "0" glyph painted on the page margin, invisible.
const learnCppBadgeHTML = `<html><body>
<div class="badge-frame">
  <div class="badge-main">
    <div class="badge-card">
      <div class="badge-header">
        <div class="badge-pill">Chapter` + "\u00a0" + `0</div>
        <div class="badge-title">Introduction / Getting Started</div>
      </div>
    </div>
  </div>
</div>
</body></html>`

const learnCppBadgeCSS = `
html { box-sizing: border-box; }
* { box-sizing: inherit; }
body { margin: 0; font-size: 14px; }
.badge-frame { overflow: hidden; width: 500pt; }
.badge-main { width: 100%; padding: 0 22pt; }
.badge-card { padding: 0 11px 9px 11px; }
.badge-pill {
  color: #fff;
  font-size: 1.2em;
  float: right;
  background-color: #6daaf3;
  padding: 0 8px;
  border-bottom-left-radius: 8px;
  border-bottom-right-radius: 8px;
}
.badge-title { color: #3b4c5a; font-size: 1.4em; font-weight: 700; }
`

// badgeTextOps returns the painted text ops that start inside the pill's
// horizontal band, in order. Position membership rather than a text prefix
// keeps the probe working before the fix, when the unbreakable run is still
// split; the pill's left offset excludes the in-flow chapter title.
func badgeTextOps(t *testing.T, res *Result, pill *box) []*Op {
	t.Helper()

	var ops []*Op

	for i := range res.Ops {
		op := &res.Ops[i]
		if op.Kind == OpText && op.X >= pill.x-0.5 && op.Y >= pill.y-1 && op.Y <= pill.y+pill.height+1 {
			ops = append(ops, op)
		}
	}

	if len(ops) == 0 {
		t.Fatal("missing Chapter badge text op")
	}

	return ops
}

func badgePaintedText(ops []*Op) string {
	var builder strings.Builder

	for _, op := range ops {
		builder.WriteString(op.Text)
	}

	return builder.String()
}

func badgeTextExtent(ops []*Op) (float64, float64) {
	width, right := 0.0, 0.0

	for _, op := range ops {
		width += op.W

		if opRight := op.X + op.W; opRight > right {
			right = opRight
		}
	}

	return width, right
}

// TestLearnCppChapterBadgeWidthAndClip: the pill keeps the full unbreakable
// NBSP run and the badge (box, background, text) stays inside the
// overflow:hidden page content clip.
func TestLearnCppChapterBadgeWidthAndClip(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, learnCppBadgeHTML, sheet(t, learnCppBadgeCSS))

	pill := findBoxByClass(t, res, "badge-pill")
	frame := findBoxByClass(t, res, "badge-frame")
	clip := paddingBoxOfTest(frame)
	ops := badgeTextOps(t, res, pill)
	textWidth, textRight := badgeTextExtent(ops)

	pillPad := 12.0 // padding: 0 8px -> 6pt each side

	t.Logf("badge box x=%.2f w=%.2f right=%.2f; text width=%.2f right=%.2f; clip x=%.2f right=%.2f",
		pill.x, pill.w, pill.x+pill.w, textWidth, textRight, clip.x, clip.x+clip.w)

	if pill.w+0.01 < textWidth+pillPad {
		t.Fatalf("badge box width %.2fpt below text+padding %.2fpt; unbreakable NBSP run was shrunk",
			pill.w, textWidth+pillPad)
	}

	if pill.x+pill.w > clip.x+clip.w+0.01 {
		t.Fatalf("badge box right edge %.2fpt past the clip right edge %.2fpt",
			pill.x+pill.w, clip.x+clip.w)
	}

	if textRight > clip.x+clip.w+0.01 {
		t.Fatalf("badge text right edge %.2fpt past the clip right edge %.2fpt; white glyph paints on the page margin",
			textRight, clip.x+clip.w)
	}

	// Chapter N fully visible: the painted run is inside the pill's padding box
	// and still carries the number after the NBSP.
	for _, op := range ops {
		if op.X < pill.x+pillPad/2-0.01 || op.X+op.W > pill.x+pill.w-pillPad/2+0.01 {
			t.Fatalf("badge text %q at x=%.2f w=%.2f overflows the pill padding box [%.2f, %.2f]",
				op.Text, op.X, op.W, pill.x+pillPad/2, pill.x+pill.w-pillPad/2)
		}
	}

	if painted := badgePaintedText(ops); !strings.Contains(painted, "Chapter") || !strings.Contains(painted, "0") {
		t.Fatalf("badge painted text %q lost the Chapter number", painted)
	}
}

// TestLearnCppChapterBadgeKeepsUnbreakableRun: a float narrower than its
// unbreakable text keeps min-content (CSS 2.1 10.3.5 shrink-to-fit floor)
// instead of collapsing to the available width and splitting the run.
func TestLearnCppChapterBadgeKeepsUnbreakableRun(t *testing.T) {
	t.Parallel()

	const cssSheet = `
body { margin: 0; font-size: 14px; }
.badge-frame { overflow: hidden; width: 120pt; }
.badge-card { width: 48pt; margin-left: 30pt; }
.badge-pill {
  color: #fff;
  font-size: 1.2em;
  float: right;
  background-color: #6daaf3;
  padding: 0 8px;
}
`

	const src = `<html><body>
<div class="badge-frame"><div class="badge-card">
  <div class="badge-pill">Chapter` + "\u00a0" + `0</div>
</div></div>
</body></html>`

	res := layoutHTML(t, src, sheet(t, cssSheet))
	pill := findBoxByClass(t, res, "badge-pill")
	ops := badgeTextOps(t, res, pill)
	textWidth, _ := badgeTextExtent(ops)

	t.Logf("narrow containing block: badge box w=%.2f right=%.2f; text width=%.2f", pill.w, pill.x+pill.w, textWidth)

	if pill.w+0.01 < textWidth+12 {
		t.Fatalf("badge box width %.2fpt below text+padding %.2fpt when available width is 48pt",
			pill.w, textWidth+12)
	}

	if painted := badgePaintedText(ops); !strings.Contains(painted, "Chapter") || !strings.Contains(painted, "0") {
		t.Fatalf("badge painted text %q lost the Chapter number", painted)
	}

	for _, op := range ops {
		if op.X < pill.x+6-0.01 || op.X+op.W > pill.x+pill.w-6+0.01 {
			t.Fatalf("badge text %q at x=%.2f w=%.2f overflows the pill padding box [%.2f, %.2f]",
				op.Text, op.X, op.W, pill.x+6, pill.x+pill.w-6)
		}
	}
}
