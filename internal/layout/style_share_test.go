package layout

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// This file is the PDF-01 contract for immutable style reuse. PDF-02 interns
// equal resolved styles at the styleStore boundary, so each test pins two
// things:
//
//  1. Value correctness after the whole resolution pass. A later element must
//     not mutate an earlier element's stored record, so tests re-read earlier
//     records after every sibling resolved.
//  2. Pointer identity. Identical declared styles must resolve to one shared
//     *ResolvedStyle; genuinely different declarations must stay distinct.
//
// The pointer assertions are red until PDF-02 lands. That red state is the
// required failing half of PDF-01, not a broken test.

// elementsByClass returns every element whose class list contains className,
// in document order. The existing styleByClass helper reads a map with
// nondeterministic iteration order and returns one record, so it cannot be
// used for repeated classes.
func elementsByClass(root *html.Node, className string) []*html.Node {
	var out []*html.Node

	root.Walk(func(node *html.Node) {
		if node.Type != html.ElementNode {
			return
		}

		if strings.Contains(" "+node.Attribute("class")+" ", " "+className+" ") {
			out = append(out, node)
		}
	})

	return out
}

// styleRecordsByClass returns the stored styles for every element with the
// class, in document order.
func styleRecordsByClass(
	t *testing.T, root *html.Node, styles map[*html.Node]*ResolvedStyle, className string,
) []*ResolvedStyle {
	t.Helper()

	nodes := elementsByClass(root, className)
	if len(nodes) == 0 {
		t.Fatalf("no element with class %q", className)
	}

	out := make([]*ResolvedStyle, 0, len(nodes))

	for _, node := range nodes {
		sty := styles[node]
		if sty == nil {
			t.Fatalf("class %q has no stored style", className)
		}

		out = append(out, sty)
	}

	return out
}

// requireSharedRecords is the PDF-02 sharing contract: equal declared styles
// must resolve to one immutable *ResolvedStyle, not one copy per element.
func requireSharedRecords(t *testing.T, label string, records []*ResolvedStyle) {
	t.Helper()

	if len(records) < 2 {
		t.Fatalf("%s: sharing check needs at least 2 records, got %d", label, len(records))
	}

	for i := 1; i < len(records); i++ {
		if records[i] != records[0] {
			t.Errorf(
				"%s: record[%d] = %p, want shared record %p",
				label, i, records[i], records[0],
			)
		}
	}
}

// requireDistinctRecords guards intern collisions: different declarations must
// not collapse onto one record.
func requireDistinctRecords(t *testing.T, label string, a, b *ResolvedStyle) {
	t.Helper()

	if a == b {
		t.Errorf("%s: different declarations share record %p", label, a)
	}
}

// mapIdentity returns the runtime header address of a map so tests can assert
// that two records reference the same custom-property map.
func mapIdentity(m map[string]string) uintptr {
	return reflect.ValueOf(m).Pointer()
}

// requireSameMapIdentity asserts two records reference one custom-property map
// (the no-property inheritance path).
func requireSameMapIdentity(t *testing.T, label string, left, right map[string]string) {
	t.Helper()

	if mapIdentity(left) != mapIdentity(right) {
		t.Errorf(
			"%s: map identities 0x%x and 0x%x differ, want the same map",
			label, mapIdentity(left), mapIdentity(right),
		)
	}
}

// requireDistinctMapIdentity asserts two declarations did not alias one map.
func requireDistinctMapIdentity(t *testing.T, label string, a, b map[string]string) {
	t.Helper()

	if mapIdentity(a) == mapIdentity(b) {
		t.Errorf("%s: maps share identity 0x%x, want independent maps", label, mapIdentity(a))
	}
}

// reportTableRows is the repeated row count shared by the fixture and the
// per-cell count assertions.
const reportTableRows = 12

// reportTableHTML builds a report-shaped table: identical data rows with one
// right-aligned amount cell and one special-styled flag cell per row.
func reportTableHTML() string {
	var markup strings.Builder

	markup.WriteString(`<html><body><table><thead><tr>`)

	for _, header := range []string{"Line", "SKU", "Description", "Quantity", "Amount", "Flag"} {
		markup.WriteString("<th>" + header + "</th>")
	}

	markup.WriteString(`</tr></thead><tbody>`)

	for row := range reportTableRows {
		fmt.Fprintf(
			&markup,
			`<tr><td class="plain">%03d</td><td class="plain">SKU-%03d</td>`+
				`<td class="plain">Widget</td><td class="plain">%d</td>`+
				`<td class="amount">$19.98</td><td class="special">rush</td></tr>`,
			row, row, row+1,
		)
	}

	markup.WriteString(`</tbody></table></body></html>`)

	return markup.String()
}

// TestRepeatedReportCellsStyleReuseImmutable models the benchmark report table:
// many identical rows and cells. Equal resolved styles must intern to one
// record, while a changed color or text-align must keep its own record.
func TestRepeatedReportCellsStyleReuseImmutable(t *testing.T) {
	t.Parallel()

	root := mustParse(t, reportTableHTML())
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		body { color: #172033; font-family: sans-serif; font-size: 9pt; margin: 0 }
		table { border-collapse: collapse; width: 100% }
		th, td { border: 1px solid #a8b5c5; padding: 1.5mm 2mm; white-space: nowrap }
		th { background: #e6eef7; text-align: left }
		td.amount { text-align: right }
		td.special { color: #b00020; font-size: 14pt }
	`)}, "print", testViewport, 800)

	plain := styleRecordsByClass(t, root, styles, "plain")
	amount := styleRecordsByClass(t, root, styles, "amount")
	special := styleRecordsByClass(t, root, styles, "special")

	if len(plain) != 4*reportTableRows || len(amount) != reportTableRows || len(special) != reportTableRows {
		t.Fatalf(
			"cell counts plain=%d amount=%d special=%d, want %d/%d/%d",
			len(plain), len(amount), len(special),
			4*reportTableRows, reportTableRows, reportTableRows,
		)
	}

	assertReportPlainCells(t, plain)

	if amount[0].TextAlign != floatRight {
		t.Fatalf("amount cell text-align = %q, want %q", amount[0].TextAlign, floatRight)
	}

	if !near(special[0].FontSize, 14) || special[0].Color[0] < 0.5 {
		t.Fatalf("special cell font-size=%v color=%v, want 14pt and red-ish", special[0].FontSize, special[0].Color)
	}

	requireSharedRecords(t, "repeated td.plain", plain)
	requireSharedRecords(t, "repeated td.amount", amount)
	requireSharedRecords(t, "repeated td.special", special)
	requireDistinctRecords(t, "td.plain vs td.amount", plain[0], amount[0])
	requireDistinctRecords(t, "td.plain vs td.special", plain[0], special[0])
}

// assertReportPlainCells requires every td.plain cell to carry the same
// resolved typography and color as the first one.
func assertReportPlainCells(t *testing.T, plain []*ResolvedStyle) {
	t.Helper()

	for cellIdx, style := range plain {
		if !near(style.FontSize, 9) {
			t.Fatalf("plain cell[%d] font-size = %v, want 9pt", cellIdx, style.FontSize)
		}

		if style.TextAlign != plain[0].TextAlign || style.Color != plain[0].Color {
			t.Fatalf(
				"plain cell[%d] diverged from cell[0]: text-align=%q/%q color=%v/%v",
				cellIdx, style.TextAlign, plain[0].TextAlign, style.Color, plain[0].Color,
			)
		}
	}
}

// TestInheritedFontFamilyStyleReuse pins inherited FontFamily reuse. Sibling
// cells inherit one family list from their parent; a later sibling that
// declares its own list must not change what the earlier cells hold.
func TestInheritedFontFamilyStyleReuse(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="report">
			<span class="cell">one</span>
			<span class="cell">two</span>
			<span class="other">three</span>
		</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.report { font-family: "DejaVu Sans", Helvetica, sans-serif }
		.other { font-family: Georgia, serif }
	`)}, "print", testViewport, 800)

	cells := styleRecordsByClass(t, root, styles, "cell")
	others := styleRecordsByClass(t, root, styles, "other")
	want := []string{"DejaVu Sans", "Helvetica", "sans-serif"}

	for cellIdx, sty := range cells {
		if !slices.Equal(sty.FontFamily, want) {
			t.Fatalf("cell[%d] FontFamily = %q, want %q", cellIdx, sty.FontFamily, want)
		}

		if sty.famHash != hashFontFamily(want) {
			t.Fatalf("cell[%d] famHash = %d, want %d", cellIdx, sty.famHash, hashFontFamily(want))
		}
	}

	if slices.Equal(others[0].FontFamily, want) {
		t.Fatalf(".other FontFamily = %q, want a distinct list", others[0].FontFamily)
	}

	// .other resolved after both .cell spans; re-read the cells afterwards.
	if !slices.Equal(cells[0].FontFamily, want) || !slices.Equal(cells[1].FontFamily, want) {
		t.Fatalf(
			"cell FontFamily changed after a later sibling resolved: %q / %q",
			cells[0].FontFamily, cells[1].FontFamily,
		)
	}

	if cells[0].famHash != cells[1].famHash {
		t.Fatalf("cell famHash mismatch: %d vs %d", cells[0].famHash, cells[1].famHash)
	}

	requireSharedRecords(t, "sibling .cell spans", cells)
	requireDistinctRecords(t, ".cell vs .other", cells[0], others[0])
}

// TestCustomPropsStyleReuseIndependent covers custom-property inheritance: a
// no-property child shares the parent map, equal declarations stay independent
// of a later different declaration, and later resolution does not rewrite
// earlier maps.
func TestCustomPropsStyleReuseIndependent(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="parent">
			<div class="inherit">a</div>
			<div class="one">b</div>
			<div class="one">c</div>
			<div class="later">d</div>
		</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.parent { --x: 1 }
		.one { --x: 1 }
		.later { --x: 2 }
	`)}, "print", testViewport, 800)

	parents := styleRecordsByClass(t, root, styles, "parent")
	inherits := styleRecordsByClass(t, root, styles, "inherit")
	ones := styleRecordsByClass(t, root, styles, "one")
	laters := styleRecordsByClass(t, root, styles, "later")

	if parents[0].CustomProps["--x"] != "1" {
		t.Fatalf("parent --x = %q, want 1", parents[0].CustomProps["--x"])
	}

	if inherits[0].CustomProps["--x"] != "1" {
		t.Fatalf("no-property child --x = %q, want inherited 1", inherits[0].CustomProps["--x"])
	}

	// The no-property child resolves before any other element declares the
	// same --x, so the parent map is the only equal map it can adopt.
	requireSameMapIdentity(t, "no-property child vs parent", inherits[0].CustomProps, parents[0].CustomProps)

	for i, sty := range ones {
		if sty.CustomProps["--x"] != "1" {
			t.Fatalf(".one[%d] --x = %q, want 1", i, sty.CustomProps["--x"])
		}
	}

	if laters[0].CustomProps["--x"] != "2" {
		t.Fatalf(".later --x = %q, want 2", laters[0].CustomProps["--x"])
	}

	// .later resolved after the earlier elements: their maps must be unchanged.
	if inherits[0].CustomProps["--x"] != "1" || ones[0].CustomProps["--x"] != "1" || ones[1].CustomProps["--x"] != "1" {
		t.Errorf("resolving .later mutated an earlier custom-property map")
	}

	requireDistinctMapIdentity(t, ".one vs .later", ones[0].CustomProps, laters[0].CustomProps)
	requireSameMapIdentity(t, "repeated .one divs", ones[0].CustomProps, ones[1].CustomProps)

	requireSharedRecords(t, "repeated .one divs", ones)
	requireDistinctRecords(t, ".one vs .later", ones[0], laters[0])
}

// TestStrokeDashArrayStyleReuse pins slice-valued stroke state. Equal dash
// arrays must share a record; a different array must not alias the first.
func TestStrokeDashArrayStyleReuse(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="dash">a</div>
		<div class="dash">b</div>
		<div class="different">c</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.dash { stroke-dasharray: 4 2 }
		.different { stroke-dasharray: 8 4 }
	`)}, "print", testViewport, 800)

	dashes := styleRecordsByClass(t, root, styles, "dash")
	differents := styleRecordsByClass(t, root, styles, "different")

	// Unitless stroke lengths convert through pxToPt (probed: 4 2 -> 3 1.5).
	wantDash := []float64{pxToPt(4), pxToPt(2)}
	wantDifferent := []float64{pxToPt(8), pxToPt(4)}

	for i, sty := range dashes {
		if !slices.Equal(sty.StrokeDashArray, wantDash) {
			t.Fatalf("dash[%d] StrokeDashArray = %v, want %v", i, sty.StrokeDashArray, wantDash)
		}
	}

	if !slices.Equal(differents[0].StrokeDashArray, wantDifferent) {
		t.Fatalf("different StrokeDashArray = %v, want %v", differents[0].StrokeDashArray, wantDifferent)
	}

	// Re-read the first dash after the different element resolved.
	if !slices.Equal(dashes[0].StrokeDashArray, wantDash) {
		t.Fatalf("first dash array changed after later resolution: %v", dashes[0].StrokeDashArray)
	}

	requireSharedRecords(t, "repeated .dash divs", dashes)
	requireDistinctRecords(t, ".dash vs .different", dashes[0], differents[0])
}

// TestNamedPageStyleReuse pins the page-name override. Each named element keeps
// its own PageName across the pass, and equal declarations share one record.
func TestNamedPageStyleReuse(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="chapter">one</div>
		<div class="cover">two</div>
		<div class="plain">three</div>
		<div class="cover">four</div>
		<div class="chapter">five</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.cover { page: cover }
		.chapter { page: chapter }
	`)}, "print", testViewport, 800)

	covers := styleRecordsByClass(t, root, styles, "cover")
	chapters := styleRecordsByClass(t, root, styles, "chapter")
	plains := styleRecordsByClass(t, root, styles, "plain")

	for i, sty := range covers {
		if sty.PageName != "cover" {
			t.Fatalf("cover[%d] PageName = %q, want cover", i, sty.PageName)
		}
	}

	for i, sty := range chapters {
		if sty.PageName != chapterPageName {
			t.Fatalf("chapter[%d] PageName = %q, want chapter", i, sty.PageName)
		}
	}

	if plains[0].PageName != "" {
		t.Fatalf("plain PageName = %q, want empty", plains[0].PageName)
	}

	// The last chapter resolved after both covers: re-read everything for
	// leakage in either direction.
	if covers[0].PageName != "cover" || chapters[0].PageName != chapterPageName || plains[0].PageName != "" {
		t.Errorf(
			"page-name leakage: cover[0]=%q chapter[0]=%q plain=%q",
			covers[0].PageName, chapters[0].PageName, plains[0].PageName,
		)
	}

	requireSharedRecords(t, "repeated .cover divs", covers)
	requireSharedRecords(t, "repeated .chapter divs", chapters)
	requireDistinctRecords(t, ".cover vs .chapter", covers[0], chapters[0])
	requireDistinctRecords(t, ".cover vs .plain", covers[0], plains[0])
}

// TestNamedPageStyleReuseCloneIsolation pins the dedicated page-name override
// clone. applyNamedPageBreaks must clone a shared record before stamping
// page-break-before; the sibling box that still points at the original record
// must keep its own values.
func TestNamedPageStyleReuseCloneIsolation(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div class="chapter">one</div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.chapter { page: chapter }
	`)}, "print", testViewport, 800)
	shared := styleRecordsByClass(t, root, styles, "chapter")[0]

	first := &box{style: shared}
	second := &box{style: shared}

	forcePageBreakBefore(first)

	if first.style == second.style {
		t.Errorf("page-name override mutated the shared record %p in place", shared)
	}

	if first.style.PageBreakBefore != pageBreakAlways {
		t.Errorf("first box PageBreakBefore = %q, want %q", first.style.PageBreakBefore, pageBreakAlways)
	}

	if second.style.PageBreakBefore != "" {
		t.Errorf("sibling box PageBreakBefore = %q, want untouched", second.style.PageBreakBefore)
	}

	if second.style.PageName != chapterPageName || first.style.PageName != chapterPageName {
		t.Errorf(
			"page-name override changed PageName: first=%q second=%q",
			first.style.PageName, second.style.PageName,
		)
	}

	if shared.PageBreakBefore != "" {
		t.Errorf("source record PageBreakBefore = %q, want untouched", shared.PageBreakBefore)
	}
}

// TestTransformStyleReuseDistinct pins Matrix2D state. Equal transforms share
// one record; a different matrix keeps its own and does not bleed.
func TestTransformStyleReuseDistinct(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="move">a</div>
		<div class="move">b</div>
		<div class="shift">c</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.move { transform: translate(10px, 5px) }
		.shift { transform: translate(1px, 2px) }
	`)}, "print", testViewport, 800)

	moves := styleRecordsByClass(t, root, styles, "move")
	shifts := styleRecordsByClass(t, root, styles, "shift")

	assertMoveTransforms(t, moves)

	if !near(shifts[0].Transform.E, pxToPt(1)) || !near(shifts[0].Transform.F, pxToPt(2)) {
		t.Fatalf(
			"shift translate = (%v, %v), want (%v, %v)",
			shifts[0].Transform.E, shifts[0].Transform.F, pxToPt(1), pxToPt(2),
		)
	}

	// Re-read the first move after the different transform resolved.
	if !near(moves[0].Transform.E, pxToPt(10)) || !near(moves[0].Transform.F, pxToPt(5)) {
		t.Errorf(
			"first move matrix changed after later resolution: E=%v F=%v",
			moves[0].Transform.E, moves[0].Transform.F,
		)
	}

	requireSharedRecords(t, "repeated .move divs", moves)
	requireDistinctRecords(t, ".move vs .shift", moves[0], shifts[0])
}

// assertMoveTransforms requires every .move record to hold the shared
// translate(10px, 5px) matrix with identity scale.
func assertMoveTransforms(t *testing.T, moves []*ResolvedStyle) {
	t.Helper()

	for moveIdx, sty := range moves {
		if !sty.HasTransform {
			t.Fatalf("move[%d] HasTransform = false, want true", moveIdx)
		}

		if !near(sty.Transform.E, pxToPt(10)) || !near(sty.Transform.F, pxToPt(5)) {
			t.Fatalf(
				"move[%d] translate = (%v, %v), want (%v, %v)",
				moveIdx, sty.Transform.E, sty.Transform.F, pxToPt(10), pxToPt(5),
			)
		}

		if !near(sty.Transform.A, 1) || !near(sty.Transform.D, 1) {
			t.Fatalf("move[%d] scale = (%v, %v), want identity", moveIdx, sty.Transform.A, sty.Transform.D)
		}
	}
}

// TestContainerRecascadeStyleReuse pins the second-pass @container cascade.
// Repeated matching declarations share one record, the non-matching element
// keeps its own, and the first-pass map is not rewritten by the second pass.
func TestContainerRecascadeStyleReuse(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
		.card { container: card / inline-size; font-size: 12pt }
		.wide { width: 400px }
		.narrow { width: 100px }
		@container card (inline-size > 20em) {
			.title { color: red; font-size: 24pt }
		}
	`)
	root := mustParse(t, `<html><body>
		<div class="card wide"><p class="title wtitle">one</p></div>
		<div class="card wide"><p class="title wtitle">two</p></div>
		<div class="card narrow"><p class="title ntitle">three</p></div>
	</body></html>`)

	pass1 := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", testViewport, 800)

	cinfo := measureSizeContainers(root, pass1, testViewport)
	if len(cinfo) == 0 {
		t.Fatal("fixture produced no size containers")
	}

	pass2 := resolveStylesWithContainers(root, []*css.Stylesheet{cssSheet}, "print", testViewport, 800, cinfo)
	wides := styleRecordsByClass(t, root, pass2, "wtitle")
	narrows := styleRecordsByClass(t, root, pass2, "ntitle")

	assertWideTitleContainers(t, wides, styleRecordsByClass(t, root, pass1, "wtitle")[0])

	if narrows[0].Color[0] > 0.1 {
		t.Fatalf("narrow title color = %v, want black (no @container match)", narrows[0].Color)
	}

	if narrows[0].FontSize > 13 {
		t.Fatalf("narrow title font-size = %v, want inherited 12pt", narrows[0].FontSize)
	}

	requireSharedRecords(t, "repeated .wide .title", wides)
	requireDistinctRecords(t, "wide title vs narrow title", wides[0], narrows[0])
}

// assertWideTitleContainers requires every wide .title record to have picked
// up the @container declaration and to still hold it after the narrow title
// resolved. pass1Wide is the first-pass record, which must not have changed.
func assertWideTitleContainers(t *testing.T, wides []*ResolvedStyle, pass1Wide *ResolvedStyle) {
	t.Helper()

	for wideIdx, sty := range wides {
		if sty.Color[0] < 0.9 {
			t.Fatalf("wide title[%d] color = %v, want red from @container", wideIdx, sty.Color)
		}

		if !near(sty.FontSize, 24) {
			t.Fatalf("wide title[%d] font-size = %v, want 24pt", wideIdx, sty.FontSize)
		}
	}

	// Re-read both wide records after the narrow element resolved.
	if wides[0].Color[0] < 0.9 || wides[1].Color[0] < 0.9 ||
		!near(wides[0].FontSize, 24) || !near(wides[1].FontSize, 24) {
		t.Errorf("wide title records changed after the narrow title resolved")
	}

	if pass1Wide.Color[0] > 0.1 {
		t.Errorf("first-pass record changed across the @container re-cascade: color=%v", pass1Wide.Color)
	}
}

// styleValueSnapshot is a deep copy of the style fields the reuse tests care
// about. Copies, not aliases, are stored so a later pass mutating a slice or
// map in place is visible to the comparison.
type styleValueSnapshot struct {
	display   string
	fontSize  float64
	color     [3]float64
	fontFam   []string
	dash      []float64
	pageName  string
	transform Matrix2D
	hasXform  bool
	textAlign string
	custom    map[string]string
}

func snapshotStyleValues(style *ResolvedStyle) styleValueSnapshot {
	return styleValueSnapshot{
		display:   style.Display,
		fontSize:  style.FontSize,
		color:     style.Color,
		fontFam:   slices.Clone(style.FontFamily),
		dash:      slices.Clone(style.StrokeDashArray),
		pageName:  style.PageName,
		transform: style.Transform,
		hasXform:  style.HasTransform,
		textAlign: style.TextAlign,
		custom:    maps.Clone(style.CustomProps),
	}
}

func (snap styleValueSnapshot) equal(other styleValueSnapshot) bool {
	return snap.display == other.display &&
		snap.fontSize == other.fontSize &&
		snap.color == other.color &&
		slices.Equal(snap.fontFam, other.fontFam) &&
		slices.Equal(snap.dash, other.dash) &&
		snap.pageName == other.pageName &&
		snap.transform == other.transform &&
		snap.hasXform == other.hasXform &&
		snap.textAlign == other.textAlign &&
		maps.Equal(snap.custom, other.custom)
}

// TestStoredStyleValuesImmutableAcrossResolutions proves stored records keep
// their values after a second, differently styled document resolves. This is
// the cross-pass immutability half of "treat all stored styles as immutable
// after insertion"; it passes before PDF-02 and must keep passing after.
func TestStoredStyleValuesImmutableAcrossResolutions(t *testing.T) {
	t.Parallel()

	fixture := `<html><body>
		<p class="target">t</p>
		<div class="parent"><span class="child">c</span></div>
	</body></html>`
	firstCSS := `
		body { color: #123456; font-size: 11pt }
		.target { page: cover; stroke-dasharray: 4 2; transform: translate(10px, 5px) }
		.parent { --x: 1 }
	`

	rootA := mustParse(t, fixture)
	stylesA := resolveStyles(rootA, []*css.Stylesheet{sheet(t, firstCSS)}, "print", testViewport, 800)
	target := findElementByClass(rootA, "target")
	before := snapshotStyleValues(stylesA[target])

	rootB := mustParse(t, fixture)
	_ = resolveStyles(rootB, []*css.Stylesheet{sheet(t, `
		body { color: #ff0000; font-size: 20pt }
		.target { page: chapter; stroke-dasharray: 1pt; transform: scale(2) }
		.parent { --x: 9 }
	`)}, "print", testViewport, 800)

	after := snapshotStyleValues(stylesA[target])
	if !before.equal(after) {
		t.Errorf("stored style changed after a second resolution pass:\nbefore %+v\nafter  %+v", before, after)
	}
}
