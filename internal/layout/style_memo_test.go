package layout

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// resolveStylesMemoDisabled runs one resolution pass with the style memo
// switched off, so a differential test can compare the cached and uncached
// value of every node.
func resolveStylesMemoDisabled(
	root *html.Node, opts Options, containers map[*html.Node]sizeContainer,
) map[*html.Node]*ResolvedStyle {
	ctx := &styleContext{
		ctx:                context.Background(),
		sheets:             opts.Sheets,
		media:              opts.Media,
		viewportW:          opts.Width,
		viewportH:          opts.Height,
		printLinkUnderline: opts.PrintLinkUnderline,
		containers:         containers,
		memo:               styleResolutionMemo{disabled: true},
	}

	styles, _ := resolveStylesCtx(root, ctx)

	return styles
}

// requireSameResolvedValues compares the memoized and uncached resolutions of
// one document node by node. Pointers differ between passes by design, so the
// comparison is by value.
func requireSameResolvedValues(
	t *testing.T, label string, memoized, uncached map[*html.Node]*ResolvedStyle,
) {
	t.Helper()

	for node, want := range uncached {
		got, ok := memoized[node]
		if !ok {
			t.Errorf("%s: node %s missing from the memoized result", label, describeNode(node))

			continue
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf(
				"%s: node %s resolved differently:\nmemo  %+v\nplain %+v",
				label, describeNode(node), got, want,
			)
		}
	}

	if len(memoized) != len(uncached) {
		t.Errorf("%s: memoized result has %d nodes, uncached has %d", label, len(memoized), len(uncached))
	}
}

// describeNode renders an element or text node for failure messages.
func describeNode(node *html.Node) string {
	if node == nil {
		return "<nil>"
	}

	switch node.Type {
	case html.ElementNode:
		return "<" + node.Name + ">"
	case html.TextNode:
		return "text(" + strings.TrimSpace(node.Text) + ")"
	case html.CommentNode, html.DoctypeNode, html.NodeUnknown:
		return fmt.Sprintf("node(%d)", node.Type)
	}

	return fmt.Sprintf("node(%d)", node.Type)
}

// styleMemoFixture is one differential memo fixture.
type styleMemoFixture struct {
	name      string
	markup    string
	css       string
	underline bool
	container bool
}

// styleMemoFixtures returns the differential corpus. Each entry covers one
// selector or policy family the memo key must capture.
//
//nolint:funlen // the coverage list reads better in one place
func styleMemoFixtures() []styleMemoFixture {
	return []styleMemoFixture{
		{
			name: "nth-and-first-child",
			markup: `<html><body><ul>
				<li class="row">a</li><li class="row">b</li><li class="row">c</li>
				<li class="row">d</li><li class="row">e</li>
			</ul><ol>
				<li class="one">a</li><li class="one">b</li><li class="one">c</li>
			</ol></body></html>`,
			css: `
				.row { color: #111; }
				.row:nth-child(2n) { color: #222; }
				.row:nth-child(3) { font-weight: bold; }
				.one:first-child { text-transform: uppercase; }
				.one:last-child { color: #333; }
			`,
		},
		{
			name: "sibling-combinators",
			markup: `<html><body>
				<h1 class="title">t</h1><p class="body">p1</p><p class="body">p2</p>
				<span class="tag">s1</span><span class="tag">s2</span>
				<div class="wrap"><p class="body">p3</p></div>
				<div class="wrap"><p class="body">p4</p></div>
			</body></html>`,
			css: `
				.title + .body { margin-top: 0; }
				.body { color: #010203; font-size: 11pt; }
				.body ~ .tag { color: #040506; }
				.tag + .tag { font-style: italic; }
				.wrap p { text-indent: 4pt; }
			`,
		},
		{
			name: "has-and-not",
			markup: `<html><body>
				<div class="card"><span class="flag">x</span><p class="txt">a</p></div>
				<div class="card"><p class="txt">b</p></div>
				<div class="card"><span class="flag">y</span><p class="txt">c</p></div>
				<ul><li class="item">one</li><li class="item pick">two</li></ul>
			</body></html>`,
			css: `
				.card:has(.flag) .txt { color: #0a0b0c; }
				.card:not(:has(.flag)) .txt { color: #0d0e0f; }
				.item:not(.pick) { color: #101112; }
			`,
		},
		{
			name: "inline-styles",
			markup: `<html><body>
				<div class="cell" style="color: #ff0000; margin: 2px">a</div>
				<div class="cell" style="color: #00ff00">b</div>
				<div class="cell" style="color: #ff0000; margin: 2px">c</div>
				<div class="cell">d</div>
			</body></html>`,
			css: `.cell { font-size: 10pt; padding: 1px }`,
		},
		{
			name: "anchor-policy",
			markup: `<html><body>
				<a class="link" href="https://example.com/a">a</a>
				<a class="link" href="https://example.com/b">b</a>
				<a class="link">c</a>
				<a class="link" href="   ">d</a>
				<span class="note">e</span>
			</body></html>`,
			css:       `.link, .note { color: #123456 }`,
			underline: true,
		},
		{
			name: "custom-props-var",
			markup: `<html><body>
				<div class="theme"><span class="use">a</span><span class="use later">b</span></div>
				<div class="theme alt"><span class="use">c</span></div>
			</body></html>`,
			css: `
				.theme { --brand: #204080; --pad: 3pt }
				.theme.alt { --brand: #802040 }
				.use { color: var(--brand); padding: var(--pad, 0) }
				.later { padding: var(--pad) }
			`,
		},
		{
			name: "container-recascade",
			markup: `<html><body>
				<div class="card wide"><p class="title wtitle">one</p></div>
				<div class="card wide"><p class="title wtitle">two</p></div>
				<div class="card narrow"><p class="title ntitle">three</p></div>
			</body></html>`,
			css: `
				.card { container: card / inline-size; font-size: 12pt }
				.wide { width: 400px }
				.narrow { width: 100px }
				@container card (inline-size > 20em) {
					.title { color: red; font-size: 24pt }
				}
			`,
			container: true,
		},
	}
}

// runStyleMemoFixture resolves one fixture with and without the memo and
// compares every node.
func runStyleMemoFixture(t *testing.T, fixture styleMemoFixture) {
	t.Helper()

	root := mustParse(t, fixture.markup)
	opts := Options{
		Sheets:             []*css.Stylesheet{sheet(t, fixture.css)},
		Media:              "print",
		Width:              testViewport,
		Height:             800,
		PrintLinkUnderline: fixture.underline,
	}

	var containers map[*html.Node]sizeContainer

	if fixture.container {
		pass1 := resolveStylesWith(root, opts, nil)
		containers = measureContainers(t, root, pass1)

		if len(containers) == 0 {
			t.Fatal("fixture produced no size containers")
		}
	}

	memoized := resolveStylesWith(root, opts, containers)
	uncached := resolveStylesMemoDisabled(root, opts, containers)

	requireSameResolvedValues(t, fixture.name, memoized, uncached)
}

// TestStyleMemoMatchesUncachedResolution is the differential gate for the
// memo: documents with positional and relational selectors, inline styles,
// anchors with href under the link-underline policy, custom properties with
// var(), and @container re-cascades must resolve to the same values with and
// without the cache.
func TestStyleMemoMatchesUncachedResolution(t *testing.T) {
	t.Parallel()

	for _, fixture := range styleMemoFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()

			runStyleMemoFixture(t, fixture)
		})
	}
}

// TestStyleMemoSharesRepeatedElements pins the sharing contract under the
// memo: repeated declaration shapes return one pointer, and a changed shape
// keeps its own record.
func TestStyleMemoSharesRepeatedElements(t *testing.T) {
	t.Parallel()

	const rows = 64

	root, opts, styles := repeatedTableStyles(t, rows)

	requireSameResolvedValues(t, "memo shares repeated elements", styles, resolveStylesMemoDisabled(root, opts, nil))

	plain := styleRecordsByClass(t, root, styles, "plain")
	amount := styleRecordsByClass(t, root, styles, "amount")

	if len(plain) != rows || len(amount) != rows {
		t.Fatalf("cell counts plain=%d amount=%d, want %d each", len(plain), len(amount), rows)
	}

	requireSharedRecords(t, "memoized td.plain", plain)
	requireSharedRecords(t, "memoized td.amount", amount)
	requireDistinctRecords(t, "td.plain vs td.amount", plain[0], amount[0])
}

// TestStyleMemoLookupRejectsDifferentHits guards the collision-safety half:
// two elements with identical parents, names and inline styles but different
// matched rules must not share a record.
func TestStyleMemoLookupRejectsDifferentHits(t *testing.T) {
	t.Parallel()

	var markup strings.Builder

	markup.WriteString(`<html><body><ul>`)

	for i := range 6 {
		fmt.Fprintf(&markup, `<li class="row">%d</li>`, i)
	}

	markup.WriteString(`</ul></body></html>`)

	root := mustParse(t, markup.String())
	opts := Options{
		Sheets: []*css.Stylesheet{sheet(t, `
			.row { color: #111 }
			.row:nth-child(2n) { color: #222 }
			.row:nth-child(3) { font-weight: 700 }
		`)},
		Media:  "print",
		Width:  testViewport,
		Height: 800,
	}
	styles := resolveStylesWith(root, opts, nil)

	rows := styleRecordsByClass(t, root, styles, "row")
	uncached := resolveStylesMemoDisabled(root, opts, nil)

	requireSameResolvedValues(t, "nth-child rows", styles, uncached)

	// The even rows and the third row have different declaration shapes, so
	// the memo must hand them distinct records even though they share a class.
	if rows[1] == rows[2] {
		t.Errorf("row[1] and row[2] share record %p despite different nth-child declarations", rows[1])
	}
}
