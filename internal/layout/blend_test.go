//nolint:wsl // white-box compositing tests use private style and layout seams
package layout

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

func TestBlendModeParsing(t *testing.T) {
	t.Parallel()

	style := initialStyle()
	if !applyAdvancedProps(&style, "mix-blend-mode", " Multiply ", 12) {
		t.Fatal("mix-blend-mode was not accepted")
	}
	if style.MixBlendMode != blendMultiply {
		t.Fatalf("MixBlendMode = %q, want %q", style.MixBlendMode, blendMultiply)
	}
	if !applyAdvancedProps(&style, "background-blend-mode", "screen, multiply", 12) {
		t.Fatal("background-blend-mode was not accepted")
	}
	if style.BackgroundBlendMode != "screen, multiply" {
		t.Fatalf("BackgroundBlendMode = %q", style.BackgroundBlendMode)
	}
	if !applyAdvancedProps(&style, "isolation", "isolate", 12) {
		t.Fatal("isolation was not accepted")
	}
	if style.Isolation != "isolate" {
		t.Fatalf("Isolation = %q, want isolate", style.Isolation)
	}
	if applyAdvancedProps(&style, "mix-blend-mode", "unsupported-mode", 12) {
		t.Fatal("unsupported blend mode was accepted")
	}
	if applyAdvancedProps(&style, "mix-blend-mode", "plus-lighter", 12) {
		t.Fatal("plus-lighter was accepted before additive compositing support")
	}
}

func TestBackgroundBlendModeForLayerRepeatsLastMode(t *testing.T) {
	t.Parallel()

	if got := backgroundBlendModeForLayer("screen, multiply", 0); got != blendScreen {
		t.Fatalf("layer 0 mode = %q, want %q", got, blendScreen)
	}
	if got := backgroundBlendModeForLayer("screen, multiply", 1); got != blendMultiply {
		t.Fatalf("layer 1 mode = %q, want %q", got, blendMultiply)
	}
	if got := backgroundBlendModeForLayer("screen, multiply", 2); got != blendScreen {
		t.Fatalf("layer 2 mode = %q, want %q (cycle)", got, blendScreen)
	}
	if got := backgroundBlendModeForLayer("screen, multiply", 4); got != blendScreen {
		t.Fatalf("layer 4 mode = %q, want %q (cycle)", got, blendScreen)
	}
	if got := backgroundBlendModeForLayer("", 0); got != blendNormal {
		t.Fatalf("empty mode = %q, want %q", got, blendNormal)
	}
}

func TestBlendColorMultiply(t *testing.T) {
	t.Parallel()

	got := BlendColor(blendMultiply, [3]float64{0.5, 0.5, 0.5}, [3]float64{1, 0, 0})
	want := [3]float64{0.5, 0, 0}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("channel %d = %v, want %v", index, got[index], want[index])
		}
	}
}

// TestMixBlendModeReachesDisplayList checks the group shape the painters
// consume for an element blend mode.
//
//nolint:cyclop // op-kind census needs one switch arm per kind
func TestMixBlendModeReachesDisplayList(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><div style="mix-blend-mode:multiply;background:#fff">blended</div></body></html>`)

	var group *BlendGroup

	textGrouped, backgroundGrouped := false, false

	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.IsGroupBegin() {
			group = paintOp.BlendGroup
		}

		switch paintOp.Kind {
		case OpText:
			textGrouped = paintOp.BlendGroup == group && paintOp.BlendMode == ""
		case OpFillRect:
			backgroundGrouped = paintOp.BlendGroup == group && paintOp.BlendMode == ""
		case OpUnknown, OpStrokeRect, OpLine, OpImage, OpLinkURI, OpBullet, OpGridRun, opKindNoop:
		}
	}

	if group == nil || group.Mode != blendMultiply {
		t.Fatalf("begin marker group = %+v, want a multiply group", group)
	}

	if !group.Isolate {
		t.Fatal("HTML blend group must be isolated (CSS Compositing 3.2)")
	}

	if !textGrouped {
		t.Fatal("display list text is not inside the multiply group")
	}

	if !backgroundGrouped {
		t.Fatal("display list background is not inside the multiply group")
	}
}

func TestIsolationClearsInheritedOperationBlendScope(t *testing.T) {
	t.Parallel()

	source := `<html><body><div style="mix-blend-mode:multiply">` +
		`<span style="isolation:isolate">isolated</span></div></body></html>`
	res := layoutHTML(t, source)

	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.Kind != OpText || paintOp.Text != "isolated" {
			continue
		}

		if paintOp.BlendMode != "" {
			t.Fatalf("isolated text blend mode = %q, want empty", paintOp.BlendMode)
		}

		if paintOp.BlendGroup == nil || paintOp.BlendGroup.Mode != "" {
			t.Fatalf("isolated text group = %+v, want an isolation group", paintOp.BlendGroup)
		}

		if paintOp.BlendGroup.Parent == nil || paintOp.BlendGroup.Parent.Mode != blendMultiply {
			t.Fatalf("isolation group parent = %+v, want the multiply group", paintOp.BlendGroup.Parent)
		}

		return
	}

	t.Fatal("display list has no isolated text")
}

// TestBlendGroupMarkersNestAroundSubtree pins the display-list shape painters
// rely on: balanced begin/end markers around exactly the element's ops, with
// inner groups nested inside outer groups.
//
//nolint:cyclop,funlen,varnamelen // marker census and nesting walk are explicit
func TestBlendGroupMarkersNestAroundSubtree(t *testing.T) {
	t.Parallel()

	source := `<html><body>` +
		`<div style="mix-blend-mode:multiply;background:#f00">` +
		`<div style="isolation:isolate;background:#00f">x</div>` +
		`</div></body></html>`
	res := layoutHTML(t, source)

	type mark struct {
		idx   int
		begin bool
		group *BlendGroup
	}

	marks := make([]mark, 0, 4)

	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.GroupMark == 0 {
			continue
		}

		marks = append(marks, mark{idx: i, begin: paintOp.IsGroupBegin(), group: paintOp.BlendGroup})
	}

	if len(marks) != 4 {
		t.Fatalf("group markers = %d, want 4 (two balanced pairs)", len(marks))
	}

	if !marks[0].begin || !marks[1].begin || marks[2].begin || marks[3].begin {
		t.Fatalf("marker order = %+v, want outer begin, inner begin, inner end, outer end", marks)
	}

	outer, inner := marks[0].group, marks[1].group
	if inner.Parent != outer {
		t.Fatalf("inner group parent = %p, want outer group %p", inner.Parent, outer)
	}

	if marks[2].group != inner || marks[3].group != outer {
		t.Fatal("end markers are not paired with their begin markers")
	}

	if outer.Mode != blendMultiply || inner.Mode != "" {
		t.Fatalf("group modes = %q / %q, want multiply / isolation", outer.Mode, inner.Mode)
	}

	insideInner := false
	insideOuter := false

	for i := marks[0].idx + 1; i < marks[3].idx; i++ {
		paintOp := &res.Ops[i]
		if paintOp.GroupMark != 0 {
			insideInner = true

			continue
		}

		if paintOp.BlendGroup == inner {
			insideInner = true

			continue
		}

		if paintOp.BlendGroup == outer {
			insideOuter = true

			continue
		}

		t.Fatalf("op %d is outside both nested groups", i)
	}

	if !insideInner || !insideOuter {
		t.Fatalf("nested subtree coverage: inner=%v outer=%v", insideInner, insideOuter)
	}
}

// TestBlendGroupEmitsTransparencyForm checks the PDF plumbing end to end: an
// element with mix-blend-mode becomes one Form XObject with a transparency
// group, painted once through an ExtGState with /BM /Multiply, and the form
// font subset carries the form's glyphs, not just the space fallback.
func TestBlendGroupEmitsTransparencyForm(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><div style="mix-blend-mode:multiply;background:#f00">grouped</div></body></html>`)

	raw := paintToPDF(t, res)

	for _, want := range []string{
		"/Subtype /Form",
		"/Group << /S /Transparency /I true /CS /DeviceRGB >>",
		"/BM /Multiply",
		"/Fm0 Do",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("PDF lacks %q", want)
		}
	}

	if strings.Contains(raw, "/FirstChar 32 /LastChar 32") {
		t.Fatal("transparency form embedded a space-only font subset; its text would render as .notdef boxes")
	}

	semantic, err := pdf.ParseSemantic([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(semantic.DocumentText(), "grouped") {
		t.Fatal("semantic text did not reach into the transparency form")
	}
}

// TestNestedBlendGroupsEmitNestedForms pins one Form XObject per group, with
// the inner form referenced from the outer form's resources.
func TestNestedBlendGroupsEmitNestedForms(t *testing.T) {
	t.Parallel()

	source := `<html><body>` +
		`<div style="mix-blend-mode:multiply;background:#f00">` +
		`<div style="isolation:isolate;background:#00f">nested</div>` +
		`</div></body></html>`
	raw := paintToPDF(t, layoutHTML(t, source))

	if got := strings.Count(raw, "/Subtype /Form"); got != 2 {
		t.Fatalf("Form XObject count = %d, want 2 (one per group)", got)
	}

	if got := strings.Count(raw, " Do\n"); got != 2 {
		t.Fatalf("Do operator count = %d, want 2 (one per group)", got)
	}
}

// TestIsolationOnlyGroupUsesNoBlendExtGState keeps isolation:isolate honest:
// it creates the group and composites it normally, with no /BM entry.
func TestIsolationOnlyGroupUsesNoBlendExtGState(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><div style="isolation:isolate;background:#0f0;padding:4px">iso</div></body></html>`)
	raw := paintToPDF(t, res)

	if !strings.Contains(raw, "/Subtype /Form") {
		t.Fatal("isolation did not create a form group")
	}

	if !strings.Contains(raw, "/Group << /S /Transparency /I true /CS /DeviceRGB >>") {
		t.Fatal("isolation group is not marked isolated")
	}

	if strings.Contains(raw, "/BM /") {
		t.Fatal("isolation-only group registered a blend mode")
	}

	if !strings.Contains(raw, " Do\n") {
		t.Fatal("isolation group is never painted")
	}
}

// TestBlendGroupSplitsAcrossPagesAsFragments documents the page-boundary
// subset: a grouped element whose ops split across pages is composited once
// per page fragment (never dropped, never left un-composited).
func TestBlendGroupSplitsAcrossPagesAsFragments(t *testing.T) {
	t.Parallel()

	source := `<html><body>` +
		`<div style="mix-blend-mode:multiply;background:#f00;height:1600pt">tall</div>` +
		`</body></html>`
	res := layoutHTML(t, source)

	doc := pdf.NewDocument()
	doc.SetCompression(false)

	opts := PaintOptions{PageWidth: 300, PageHeight: 400}

	if err := Paint(doc, res, opts); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatal(err)
	}

	raw := buf.String()

	if got := strings.Count(raw, "/Subtype /Form"); got < 2 {
		t.Fatalf("page-fragment Form XObject count = %d, want at least 2", got)
	}

	if got := strings.Count(raw, " Do\n"); got < 2 {
		t.Fatalf("page-fragment Do count = %d, want at least 2", got)
	}

	if !strings.Contains(raw, "(tall) Tj") {
		t.Fatal("grouped text was dropped by page-fragment splitting")
	}
}

// TestPaintRegistersFontsOnEveryPage guards the page-local font-name
// bookkeeping introduced with grouped painting: clearing the name map per
// page must not leave later pages without /Font resources.
func TestPaintRegistersFontsOnEveryPage(t *testing.T) {
	t.Parallel()

	source := `<html><body><p>first page</p>` +
		`<div style="page-break-before:always">second page</div></body></html>`
	res := layoutHTML(t, source)

	doc := pdf.NewDocument()
	doc.SetCompression(false)

	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatal(err)
	}

	semantic, err := pdf.ParseSemantic(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	if semantic.PageCount() < 2 {
		t.Fatalf("page count = %d, want at least 2", semantic.PageCount())
	}

	for i, page := range semantic.Pages {
		if len(page.Fonts) == 0 {
			t.Fatalf("page %d has no /Font resources; text would be invisible", i+1)
		}
	}
}

// TestBlendGroupScopeStaysOnElementSubtree checks the layout-side scope for
// the formatting contexts the group work touched: a blended flex item, an
// inline-block, and a grid item must own exactly their own ops, and the
// following plain sibling must keep a nil group so it paints into the parent
// stream.
//
//nolint:cyclop // three formatting-context subtests share one op census
func TestBlendGroupScopeStaysOnElementSubtree(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		html  string
		plain string
	}{
		{
			name: "flex",
			html: `<html><body style="margin:0"><div style="display:flex">` +
				`<div style="mix-blend-mode:multiply;background:#08f">blended</div>` +
				`<div>flexplain</div></div></body></html>`,
			plain: "flexplain",
		},
		{
			name: "inline-block",
			html: `<html><body>before <span style="display:inline-block;` +
				`mix-blend-mode:multiply;background:#08f">mid</span> ibplain</body></html>`,
			plain: "ibplain",
		},
		{
			name: "grid",
			html: `<html><body style="margin:0"><div style="display:grid">` +
				`<div style="mix-blend-mode:multiply;background:#08f">blended</div>` +
				`<div>gridplain</div></div></body></html>`,
			plain: "gridplain",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, testCase.html)

			blendedSeen, plainSeen := false, false

			for idx := range res.Ops {
				paintOp := &res.Ops[idx]
				switch {
				case paintOp.Kind == OpText && strings.Contains(paintOp.Text, testCase.plain):
					if paintOp.Group() != nil {
						t.Fatalf("plain sibling text carries group %d", paintOp.Group().ID)
					}

					plainSeen = true
				case paintOp.Kind == OpFillRect && paintOp.R < 0.1 && paintOp.B > 0.9 && paintOp.Group() != nil:
					blendedSeen = true
				}
			}

			if !plainSeen {
				t.Fatalf("plain sibling text %q not found", testCase.plain)
			}

			if !blendedSeen {
				t.Fatal("blended fill is not inside its element group")
			}
		})
	}
}

// TestBlendGroupKeepsPlainFlexSiblingOutOfForm pins fixture-62 cell 16: the
// plain flex sibling's background sorts before the blended item's text in
// paint order, so an innermost-frame painter routed it into the still-open
// multiply buffer and multiplied it. Every operation must route by its own
// group chain: a plain operation paints into the page stream even while an
// unrelated group is open.
//
//nolint:cyclop // fill census, group ancestry, and PDF resource assertions
func TestBlendGroupKeepsPlainFlexSiblingOutOfForm(t *testing.T) {
	t.Parallel()

	source := `<html><body style="margin:0">` +
		`<div style="background:#f80;padding:6px;display:flex;gap:8px;font-size:7pt;color:#fff">` +
		`<div style="mix-blend-mode:multiply;background:#08f;padding:3px 7px">multiply</div>` +
		`<div style="background:#0f0;padding:3px 7px">normal</div>` +
		`</div></body></html>`
	res := layoutHTML(t, source)

	blended, plain := false, false

	for idx := range res.Ops {
		paintOp := &res.Ops[idx]
		if paintOp.Kind != OpFillRect {
			continue
		}

		switch {
		case near(paintOp.R, 0) && near(paintOp.G, 1) && near(paintOp.B, 0): // #0f0
			if paintOp.Group() != nil {
				t.Fatalf("plain sibling fill carries group %d", paintOp.Group().ID)
			}

			plain = true
		case near(paintOp.R, 0) && paintOp.G > 0.5 && paintOp.G < 0.55 && near(paintOp.B, 1): // #08f
			if paintOp.Group() == nil {
				t.Fatal("blended fill carries no group")
			}

			blended = true
		}
	}

	if !blended || !plain {
		t.Fatalf("fills not found: blended=%v plain=%v", blended, plain)
	}

	for idx, form := range transparencyFormStreams(paintToPDF(t, res)) {
		if strings.Contains(form, "0 1 0 rg") {
			t.Fatalf("plain sibling fill leaked into transparency form %d", idx)
		}
	}
}

// TestSiblingBlendGroupsEmitSiblingForms pins fixture-61 cell 99: the left
// multiply chip and the right isolation wrapper are sibling groups, so the
// right chip's form must be invoked from the page, not nested inside the left
// chip's multiply form. Before the routing fix the wrapper's frame parented
// onto the left frame and both chips multiplied against each other.
//
//nolint:cyclop,funlen // group ancestry, PDF page resources, and text assertions
func TestSiblingBlendGroupsEmitSiblingForms(t *testing.T) {
	t.Parallel()

	source := `<html><body style="margin:0">` +
		`<div style="background:#f80;padding:5px;display:flex;gap:10px">` +
		`<div style="width:74px;height:26px;position:relative">` +
		`<div style="position:absolute;left:0;top:0;width:74px;height:26px;` +
		`background:#08f;mix-blend-mode:multiply">auto</div>` +
		`</div>` +
		`<div style="width:74px;height:26px;position:relative;isolation:isolate">` +
		`<div style="position:absolute;left:0;top:0;width:74px;height:26px;` +
		`background:#0f0;mix-blend-mode:multiply">isolate</div>` +
		`</div></div></body></html>`
	res := layoutHTML(t, source)

	var left, right *BlendGroup

	for idx := range res.Ops {
		paintOp := &res.Ops[idx]
		if paintOp.Kind != OpFillRect {
			continue
		}

		switch {
		case near(paintOp.R, 0) && paintOp.G > 0.5 && paintOp.G < 0.55 && near(paintOp.B, 1): // #08f
			left = paintOp.Group()
		case near(paintOp.R, 0) && near(paintOp.G, 1) && near(paintOp.B, 0): // #0f0
			right = paintOp.Group()
		}
	}

	if left == nil || right == nil || right.Parent == nil {
		t.Fatalf("chip groups not found: left=%v right=%v", left, right)
	}

	if left.Parent != nil {
		t.Fatalf("left chip group parent = %d, want a root group", left.Parent.ID)
	}

	if right.Parent.Parent != nil {
		t.Fatalf("right wrapper group parent = %d, want a root group", right.Parent.Parent.ID)
	}

	if right.Parent == left {
		t.Fatal("isolated wrapper group nested inside the left multiply group")
	}

	raw := paintToPDF(t, res)

	semantic, err := pdf.ParseSemantic([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}

	if got := len(semantic.Pages[0].Images); got < 2 {
		t.Fatalf("page invokes %d XObjects, want at least 2 sibling forms (left chip + isolated wrapper)", got)
	}

	for _, want := range []string{"auto", "isolate"} {
		if !strings.Contains(semantic.DocumentText(), want) {
			t.Fatalf("chip text %q lost", want)
		}
	}

	for idx, form := range transparencyFormStreams(raw) {
		if strings.Contains(form, "0 0.533") && strings.Contains(form, "0 1 0 rg") {
			t.Fatalf("transparency form %d holds both sibling chips", idx)
		}
	}
}

// transparencyFormStreams returns the stream bodies of every Form XObject in
// an uncompressed PDF, so a test can assert which operators landed in a group
// buffer.
func transparencyFormStreams(raw string) []string {
	var streams []string

	for {
		marker := strings.Index(raw, "/Subtype /Form")
		if marker < 0 {
			return streams
		}

		raw = raw[marker:]

		streamStart := strings.Index(raw, "stream\n")
		if streamStart < 0 {
			return streams
		}

		streamStart += len("stream\n")

		streamEnd := strings.Index(raw[streamStart:], "\nendstream")
		if streamEnd < 0 {
			return streams
		}

		streams = append(streams, raw[streamStart:streamStart+streamEnd])
		raw = raw[streamStart+streamEnd:]
	}
}

// paintToPDF lays out res, paints one document with plain (uncompressed)
// streams so tests can inspect content operators, and returns the PDF text.
func paintToPDF(t *testing.T, res *Result) string {
	t.Helper()

	doc := pdf.NewDocument()
	doc.SetCompression(false)

	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatal(err)
	}

	return buf.String()
}
