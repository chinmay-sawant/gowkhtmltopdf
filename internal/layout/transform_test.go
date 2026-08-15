//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

func TestParseTransformTranslateRotateScale(t *testing.T) {
	t.Parallel()

	m, has, ok := parseTransformList("translate(10pt, 20pt) rotate(90deg) scale(2)", 12)
	if !ok || !has {
		t.Fatalf("parse ok=%v has=%v", ok, has)
	}
	// scale(2) first on point, then rotate 90, then translate — M = T*R*S
	x, y := m.Apply(1, 0)
	// S: (2,0) → R90: (0,2) → T: (10,22)
	if math.Abs(x-10) > 1e-6 || math.Abs(y-22) > 1e-6 {
		t.Fatalf("Apply(1,0)=(%v,%v), want (10,22)", x, y)
	}
}

func TestTransformedInlineBorderArrowIsRebasedAfterFlowShift(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t,
		`<html><body><div class="flow"><span class="step">html</span>`+
			`<span class="arrow"></span><span class="step">css</span></div></body></html>`, sheet(t, `
body { margin: 0; }
.flow { display: block; }

.step {
 display: inline-block;
 padding: 0.22em 0.55em;
 border-radius: 6px;
 border: 1px solid #d8d1c2;
 background: #fff;
 font-family: monospace;
 font-size: 0.72rem;
}

.arrow {
 display: inline-block;
 width: 0.5em;
 height: 0.5em;
 border-top: 2px solid #5c6470;
 border-right: 2px solid #5c6470;
 transform: rotate(45deg);
 margin: 0 0.1em;
}
`))

	arrow := transformedLineOp(res)
	if arrow == nil {
		t.Fatal("missing transformed arrow stroke")
	}

	transformed := transformedBox(res.root)
	if transformed == nil {
		t.Fatal("missing transformed arrow box")
	}

	before := arrow.Xform
	transformed.y += 100
	arrow.Y += 100

	restampBoxTransforms(res.root, res.Ops)

	if math.Abs(arrow.Xform.F-before.F) < 20 {
		t.Fatalf("transformed arrow kept stale origin after flow shift: before=%+v after=%+v", before, arrow.Xform)
	}
}

func transformedLineOp(res *Result) *Op {
	for idx := range res.Ops {
		if res.Ops[idx].Kind == OpLine && res.Ops[idx].XformSet {
			return &res.Ops[idx]
		}
	}

	return nil
}

func transformedBox(root *box) *box {
	boxes := make([]*box, 0)
	flattenBoxes(root, &boxes)

	for _, candidate := range boxes {
		if candidate.style != nil && candidate.style.HasTransform {
			return candidate
		}
	}

	return nil
}

func TestParseTransformMatrixAndSkew(t *testing.T) {
	t.Parallel()

	m, has, isOK := parseTransformList("matrix(1, 0, 0, 1, 30, 0)", 12)
	if !isOK || !has {
		t.Fatal("matrix parse failed")
	}

	x, y := m.Apply(0, 0)
	// 30 CSS px → 22.5 pt
	want := pxToPt(30)
	if math.Abs(x-want) > 1e-6 || math.Abs(y) > 1e-6 {
		t.Fatalf("matrix translate got (%v,%v), want (%v,0)", x, y, want)
	}

	sk, has, isOK := parseTransformList("skewX(45deg)", 12)
	if !isOK || !has {
		t.Fatal("skewX parse failed")
	}

	sx, sy := sk.Apply(10, 10)
	if math.Abs(sx-20) > 1e-6 || math.Abs(sy-10) > 1e-6 {
		t.Fatalf("skewX Apply got (%v,%v)", sx, sy)
	}
}

func TestParseTransformNoneAnd3DRejected(t *testing.T) {
	t.Parallel()

	_, has, isOK := parseTransformList("none", 12)
	if !isOK || has {
		t.Fatalf("none: ok=%v has=%v", isOK, has)
	}

	_, _, isOK = parseTransformList("translate3d(1px,2px,3px)", 12)
	if isOK {
		t.Fatal("3d transform should be rejected")
	}
}

func TestParseTransformOriginKeywords(t *testing.T) {
	t.Parallel()

	spec, isOK := parseTransformOrigin("top left", 12)
	if !isOK {
		t.Fatal("origin parse failed")
	}

	if !spec.XPercent || !spec.YPercent || spec.X != 0 || spec.Y != 0 {
		t.Fatalf("top left → %+v", spec)
	}

	spec, isOK = parseTransformOrigin("50% 50%", 12)
	if !isOK || spec.X != 50 || spec.Y != 50 {
		t.Fatalf("50%% 50%% → %+v", spec)
	}
}

func TestTransformRotateBadgeSiblingsUnmoved(t *testing.T) { //nolint:cyclop,funlen
	t.Parallel()

	cssSheet := sheet(t, `
.row { font-size: 12pt; }
.badge {
  display: inline-block;
  transform: rotate(-15deg);
  transform-origin: center center;
  background: #c62828;
  color: #fff;
  padding: 4pt 8pt;
}
.plain { display: inline-block; background: #eee; padding: 4pt 8pt; }
`)
	res := layoutHTML(t, `<div class="row">`+
		`<span class="badge">NEW</span> <span class="plain">Sibling</span></div>`, cssSheet)

	var badge, plain, plainFill *Op

	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.Kind == OpFillRect && paintOp.R > 0.8 && paintOp.G > 0.8 && paintOp.B > 0.8 {
			plainFill = paintOp
		}

		if paintOp.Kind != OpText {
			continue
		}

		if strings.Contains(paintOp.Text, "NEW") {
			badge = paintOp
		}

		if strings.Contains(paintOp.Text, "Sibling") {
			plain = paintOp
		}
	}

	if badge == nil || plain == nil {
		t.Fatal("missing text ops")
	}

	if !badge.XformSet {
		t.Fatal("badge should have transform stamped")
	}

	if plain.XformSet {
		t.Fatal("sibling must not inherit paint transform on its own ops incorrectly — wait, sibling is not under badge")
	}

	if plainFill == nil {
		t.Fatal("plain sibling background missing")
	}

	if math.Abs(plainFill.X-(plain.X-8)) > 1 {
		t.Fatalf("plain sibling background x=%.1f, text x=%.1f; chrome was not moved with inline-block", plainFill.X, plain.X)
	}
	// Layout Y of sibling text should be on the same line band as badge
	// (transform does not change sibling flow).
	if math.Abs(badge.Y-plain.Y) > badge.Size {
		t.Fatalf("sibling flow shifted: badgeY=%.1f plainY=%.1f", badge.Y, plain.Y)
	}
	// Paint into PDF and ensure content stream contains cm (vector CTM).
	doc := pdf.NewDocument()
	if err := Paint(doc, res, PaintOptions{
		PageWidth: 612, PageHeight: 792, MarginTop: 36, MarginBottom: 36, MarginLeft: 36, MarginRight: 36,
	}); err != nil {
		t.Fatal(err)
	}

	if doc.PageCount() < 1 {
		t.Fatal("no pages")
	}

	raw := string(doc.PageAt(0).Content().Bytes())
	if !strings.Contains(raw, " cm\n") && !strings.Contains(raw, " cm") {
		// num() may format without always matching; look for cm operator
		if !strings.Contains(raw, "cm\n") {
			t.Fatalf("expected PDF cm transform in content stream, got:\n%s", raw[:min(400, len(raw))])
		}
	}
}

func TestTransformAbsposContainingBlockScale(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.host {
  position: relative;
  transform: scale(2);
  transform-origin: top left;
  width: 100pt;
  height: 80pt;
  background: #e3f2fd;
  padding: 10pt;
}
.child {
  position: absolute;
  top: 0;
  left: 0;
  width: 20pt;
  height: 20pt;
  background: #f44336;
}
`)
	res := layoutHTML(t, `<div class="host"><div class="child"></div></div>`, cssSheet)
	host := findBoxByClass(t, res, "host")
	child := findBoxByClass(t, res, "child")
	// Padding-box CB: child top-left at host padding edge.
	expectX := host.x + host.style.BorderLeft.Width
	expectY := host.y + host.style.BorderTop.Width

	if math.Abs(child.x-expectX) > 0.5 || math.Abs(child.y-expectY) > 0.5 {
		t.Fatalf("abs child at (%.2f,%.2f), want padding-box (%.2f,%.2f); host=(%.2f,%.2f) padL=%.1f",
			child.x, child.y, expectX, expectY, host.x, host.y, host.style.PaddingLeft)
	}

	var sawXform bool

	for i := child.opStart; i <= child.opEnd && i < len(res.Ops); i++ {
		if res.Ops[i].XformSet {
			sawXform = true

			break
		}
	}

	if !sawXform {
		t.Fatal("child ops should carry composed scale transform")
	}
}

func TestTransformNestedScaleTranslate(t *testing.T) {
	t.Parallel()

	s := sheet(t, `
.outer { transform: translate(10pt, 0); }
.inner { transform: scale(2); transform-origin: top left; width: 40pt; height: 20pt; background: #090; }
`)
	res := layoutHTML(t, `<div class="outer"><div class="inner">X</div></div>`, s)

	var text *Op

	for i := range res.Ops {
		if res.Ops[i].Kind == OpText && strings.Contains(res.Ops[i].Text, "X") {
			text = &res.Ops[i]

			break
		}
	}

	if text == nil || !text.XformSet {
		t.Fatal("inner text missing composed transform")
	}
	// Parent translate then child scale about child's origin — identity check:
	// composed must not be pure identity.
	if text.Xform.IsIdentity() {
		t.Fatal("composed transform is identity")
	}
}

func TestTransformKeyframesStaticCascaded(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
.badge {
  transform: rotate(45deg);
  animation: spin 1s linear infinite;
  display: inline-block;
}
`)
	res := layoutHTML(t, `<span class="badge">A</span>`, cssSheet)

	var text *Op

	for i := range res.Ops {
		if res.Ops[i].Kind == OpText {
			text = &res.Ops[i]

			break
		}
	}

	if text == nil || !text.XformSet {
		t.Fatal("expected static rotate(45deg) stamped")
	}
	// rotate(45°) about center — A and D ≈ cos(45)
	a := text.Xform.A
	if math.Abs(a) < 0.1 {
		t.Fatalf("expected rotation matrix component, A=%v", a)
	}
}

func TestOpacityExtGState(t *testing.T) {
	t.Parallel()

	s := sheet(t, `.faded { opacity: 0.5; background: #00f; width: 40pt; height: 20pt; }`)
	res := layoutHTML(t, `<div class="faded">Hi</div>`, s)

	var saw bool

	for _, op := range res.Ops {
		if op.PaintOpacity > 0 && op.PaintOpacity < 1 {
			saw = true

			if math.Abs(op.PaintOpacity-0.5) > 1e-6 {
				t.Fatalf("opacity=%v", op.PaintOpacity)
			}
		}
	}

	if !saw {
		t.Fatal("no PaintOpacity stamped")
	}

	doc := pdf.NewDocument()
	_ = Paint(doc, res, PaintOptions{
		PageWidth: 612, PageHeight: 792, MarginTop: 36, MarginBottom: 36, MarginLeft: 36, MarginRight: 36,
	})

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatal(err)
	}

	raw := buf.String()
	if !strings.Contains(raw, "/opacity") && !strings.Contains(raw, "gs") {
		t.Fatalf("expected ExtGState opacity, got %q", raw[:min(300, len(raw))])
	}
}

func findBoxByClass(t *testing.T, res *Result, class string) *box {
	t.Helper()

	var found *box

	var walk func(b *box)
	walk = func(boxNode *box) {
		if boxNode == nil || found != nil {
			return
		}

		if boxNode.node != nil {
			if strings.Contains(boxNode.node.Attribute("class"), class) {
				found = boxNode

				return
			}
		}

		for _, c := range boxNode.children {
			walk(c)
		}
	}
	walk(res.root)

	if found == nil {
		t.Fatalf("box class %q not found", class)
	}

	return found
}
