package outline_test

import (
	"bytes"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/layout"
	"gowkhtmltopdf/internal/outline"
)

const (
	testBodyTitle     = "Body"
	testFirstDocTitle = "First in document"
)

// treeHTML parses an HTML fragment into a document tree.
func treeHTML(t *testing.T, src string) *html.Node {
	t.Helper()

	root, err := html.Parse("<html><body>" + src + "</body></html>")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	return root
}

// parseSel parses one CSS selector via the stylesheet parser.
func parseSel(t *testing.T, sel string) css.Selector {
	t.Helper()

	sheet, err := css.Parse(sel + "{}")
	if err != nil || sheet == nil || len(sheet.Rules) == 0 || len(sheet.Rules[0].Selectors) == 0 {
		t.Fatalf("parse selector %q: %v", sel, err)
	}

	return sheet.Rules[0].Selectors[0]
}

// fakeLocations builds element locations for the given nodes, starting at
// y=10 and stepping 40 units down the page.
func fakeLocations(nodes []*html.Node) []layout.ElementLocation {
	locs := make([]layout.ElementLocation, 0, len(nodes))
	for i, node := range nodes {
		locs = append(locs, layout.ElementLocation{Node: node, Page: 0, X: 0, Y: 10 + 40*float64(i), W: 100, H: 20})
	}

	return locs
}

// headNodes returns the h1..h6 element nodes of a tree in document order.
func headNodes(t *testing.T, root *html.Node) []*html.Node {
	t.Helper()

	var nodes []*html.Node

	root.Walk(func(n *html.Node) {
		if n.Type == html.ElementNode && n.Name >= "h1" && n.Name <= "h6" {
			nodes = append(nodes, n)
		}
	})

	return nodes
}

func TestOutlineTreeNesting(t *testing.T) {
	t.Parallel()
	// h1 > h2 > h1: the second h1 closes the h2 and returns to the root.
	root := treeHTML(t, "<h1>One</h1><h2>Sub</h2><h1>Two</h1>")
	nodes := headNodes(t, root)
	headings := outline.CollectHeadings(root)
	headings = outline.Lookup(headings, fakeLocations(nodes))
	outline.AssignAnchors(headings)
	tree := outline.BuildTree(headings, outline.Options{}) //nolint:exhaustruct // intentional zero/partial fields

	if len(tree.Children) != 2 {
		t.Fatalf("root children = %d, want 2", len(tree.Children))
	}

	if tree.Children[0].Heading.Title != "One" || tree.Children[1].Heading.Title != "Two" {
		t.Errorf("root children = %q, %q", tree.Children[0].Heading.Title, tree.Children[1].Heading.Title)
	}

	sub := tree.Children[0].Children
	if len(sub) != 1 || sub[0].Heading.Title != "Sub" {
		t.Fatalf("h1 One children = %d, want [Sub]", len(sub))
	}

	if len(tree.Children[1].Children) != 0 {
		t.Errorf("h1 Two should have no children")
	}
}

func TestOutlineTreeSortAndClamp(t *testing.T) {
	t.Parallel()
	// Document order is h2, h4, h1; the h4 sits mid-page below the h2 and
	// jumps two levels, clamping to a child of h2 at level 3.
	root := treeHTML(t, "<h2>A</h2><h4>Deep</h4><h1>B</h1>")
	nodes := headNodes(t, root)
	headings := outline.CollectHeadings(root)
	headings = outline.Lookup(headings, fakeLocations(nodes))
	tree := outline.BuildTree(headings, outline.Options{}) //nolint:exhaustruct // intentional zero/partial fields

	if len(tree.Children) != 2 || tree.Children[0].Heading.Title != "A" {
		t.Fatalf("root = %v, want [A B]", tree.Children)
	}
	// The h4 is a child of the h2 (clamped to level 3), and the h1 B is a
	// root child even though it comes later in document order: the tree is
	// sorted by (page, y, x), and B is higher on the page than A.
	a := tree.Children[0]
	if len(a.Children) != 1 || a.Children[0].Heading.Title != "Deep" {
		t.Fatalf("A children = %v, want [Deep]", a.Children)
	}

	if got := a.Children[0].Heading.Level; got != 4 {
		t.Errorf("Deep raw level = %d, want 4 (clamp affects tree depth, not the heading)", got)
	}
}

func TestOutlineTreeLevelStackAcrossPages(t *testing.T) {
	t.Parallel()
	// Page 1 ends inside a chapter; page 2 starts with its h2: the h2 nests
	// under the h1 from page 1 (page order governs, not document order).
	root := treeHTML(t, "<h1>Ch</h1><h2>Late</h2>")
	nodes := headNodes(t, root)
	locs := []layout.ElementLocation{
		{Node: nodes[0], Page: 0, X: 0, Y: 10, W: 100, H: 20},
		{Node: nodes[1], Page: 1, X: 0, Y: 10, W: 100, H: 20},
	}
	headings := outline.Lookup(outline.CollectHeadings(root), locs)

	tree := outline.BuildTree(headings, outline.Options{}) //nolint:exhaustruct // intentional zero/partial fields
	if len(tree.Children) != 1 || len(tree.Children[0].Children) != 1 {
		t.Fatalf("tree = %+v, want Ch > Late", tree)
	}
}

func TestOutlineDepth(t *testing.T) {
	t.Parallel()
	root := treeHTML(t, "<h1>One</h1><h2>Sub</h2><h2>Sub2</h2>")
	nodes := headNodes(t, root)
	headings := outline.Lookup(outline.CollectHeadings(root), fakeLocations(nodes))

	tree := outline.BuildTree(headings, outline.Options{ //nolint:exhaustruct // intentional zero/partial fields
		MaxDepth: 1,
	})
	if len(tree.Children) != 1 {
		t.Fatalf("root children = %d, want 1", len(tree.Children))
	}

	if len(tree.Children[0].Children) != 0 {
		t.Errorf("MaxDepth=1 must drop level-2 headings")
	}
}

func TestOutlineExclude(t *testing.T) {
	t.Parallel()
	root := treeHTML(t, `<h1>Keep</h1><h1 class="hidden">Drop</h1><h1 id="x">Also</h1>`)
	nodes := headNodes(t, root)
	headings := outline.Lookup(outline.CollectHeadings(root), fakeLocations(nodes))

	tree := outline.BuildTree(headings, outline.Options{ //nolint:exhaustruct // intentional zero/partial fields
		Exclude: []css.Selector{
			parseSel(t, ".hidden"),
			parseSel(t, "#x"),
		},
	})
	if len(tree.Children) != 1 || tree.Children[0].Heading.Title != "Keep" {
		t.Fatalf("tree = %+v, want [Keep]", tree.Children)
	}
}

func TestOutlineExcludeNestedAndEmpty(t *testing.T) {
	t.Parallel()
	// Descendant/element selectors drop via css.Match; empty Exclude is a no-op.
	root := treeHTML(t, `<div class="nav"><h2>Nav</h2></div><h1>`+testBodyTitle+`</h1><h2>Sec</h2>`)
	nodes := headNodes(t, root)
	headings := outline.Lookup(outline.CollectHeadings(root), fakeLocations(nodes))

	t.Run("empty exclude is a no-op", func(t *testing.T) {
		t.Parallel()
		// Document order Nav(h2), Body(h1), Sec(h2): Nav clamps to a root
		// child, Body is a sibling root child, Sec nests under Body.
		tree := excludeTree(t, headings)
		wantTitles(t, tree.Children, "Nav", testBodyTitle)
		wantTitles(t, tree.Children[1].Children, "Sec")
	})

	t.Run("descendant selector", func(t *testing.T) {
		t.Parallel()
		tree := excludeTree(t, headings, parseSel(t, ".nav h2"))
		wantTitles(t, tree.Children, testBodyTitle)
		wantTitles(t, tree.Children[0].Children, "Sec")
	})

	t.Run("element selector drops every h2", func(t *testing.T) {
		t.Parallel()
		tree := excludeTree(t, headings, parseSel(t, "h2"))
		wantTitles(t, tree.Children, testBodyTitle)

		if len(tree.Children[0].Children) != 0 {
			t.Fatalf("after h2: tree = %+v, want [Body] with no children", titles(tree.Children))
		}
	})
}

// excludeTree builds an outline tree with the given exclude selectors
// (test helper).
func excludeTree(t *testing.T, headings []*outline.Heading, sels ...css.Selector) *outline.Node {
	t.Helper()

	return outline.BuildTree(headings, outline.Options{ //nolint:exhaustruct // intentional zero/partial fields
		Exclude: sels,
	})
}

// wantTitles fails the test unless ns has exactly the given titles in order
// (test helper).
func wantTitles(t *testing.T, ns []*outline.Node, want ...string) {
	t.Helper()

	got := titles(ns)
	if len(got) != len(want) {
		t.Fatalf("titles = %+v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("titles = %+v, want %v", got, want)
		}
	}
}

// titles returns heading titles for a slice of outline nodes (test helper).
func titles(ns []*outline.Node) []string {
	out := make([]string, len(ns))

	for i, node := range ns {
		if node.Heading != nil {
			out[i] = node.Heading.Title
		}
	}

	return out
}

func TestSortHeadings(t *testing.T) {
	t.Parallel()
	root := treeHTML(t, "<h1>A</h1><h2>B</h2><h1>C</h1>")
	nodes := headNodes(t, root)
	headings := outline.CollectHeadings(root)
	locs := []layout.ElementLocation{
		{Node: nodes[0], Page: 1, X: 5, Y: 30, W: 100, H: 20},
		{Node: nodes[1], Page: 0, X: 0, Y: 10, W: 100, H: 20},
		{Node: nodes[2], Page: 1, X: 5, Y: 10, W: 100, H: 20},
	}
	headings = outline.Lookup(headings, locs)

	// Deliberately out of order: page 1 before page 0; C (y=10) before A (y=30).
	reversed := []*outline.Heading{headings[0], headings[2], headings[1]}
	outline.SortHeadings(reversed)

	got := []string{}
	for _, h := range reversed {
		got = append(got, h.Title)
	}

	want := []string{"B", "C", "A"} // page 0 first; within page 1: y-down, then x
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SortHeadings order = %v, want %v", got, want)
		}
	}
}

func TestSectionOf(t *testing.T) {
	t.Parallel()
	root := treeHTML(t, "<h1>Intro</h1><h2>Deep</h2><h1>"+testBodyTitle+"</h1>")
	nodes := headNodes(t, root)
	headings := outline.Lookup(outline.CollectHeadings(root), []layout.ElementLocation{
		{Node: nodes[0], Page: 0, X: 0, Y: 10, W: 100, H: 20},
		{Node: nodes[1], Page: 0, X: 0, Y: 50, W: 100, H: 20},
		{Node: nodes[2], Page: 1, X: 0, Y: 10, W: 100, H: 20},
	})
	outline.SortHeadings(headings)

	cases := []struct {
		page       int
		section    string
		subsection string
	}{
		{0, "Intro", "Deep"},
		{1, "Intro", testBodyTitle},
		{2, "Intro", testBodyTitle},
		{-1, "", ""},
	}
	for _, c := range cases {
		sec, sub := outline.SectionOf(headings, c.page)
		if sec != c.section || sub != c.subsection {
			t.Errorf("SectionOf(page %d) = %q, %q; want %q, %q", c.page, sec, sub, c.section, c.subsection)
		}
	}
}

func TestCollapseWS(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"  Two   Words ", "Two Words"},
		{"a\t\nb\rc\fd", "a b c d"},
		{" already ", "already"},
		{"no-space", "no-space"},
	}
	for _, testCase := range cases {
		if got := outline.CollapseWS(testCase.in); got != testCase.want {
			t.Errorf("CollapseWS(%q) = %q, want %q", testCase.in, got, testCase.want)
		}
	}
}

func TestOutlineCollectTitlesAndAnchors(t *testing.T) {
	t.Parallel()
	root := treeHTML(t, "<h1>  Two   Words </h1><p>text</p><h2>Deep</h2>")

	headings := outline.CollectHeadings(root)
	if len(headings) != 2 {
		t.Fatalf("headings = %d, want 2", len(headings))
	}

	if headings[0].Title != "Two Words" || headings[1].Title != "Deep" {
		t.Errorf("titles = %q, %q", headings[0].Title, headings[1].Title)
	}

	outline.AssignAnchors(headings)

	if !strings.HasPrefix(headings[0].Anchor, "__WKANCHOR_") || headings[0].Anchor == headings[1].Anchor {
		t.Errorf("anchors = %q, %q", headings[0].Anchor, headings[1].Anchor)
	}
}

func TestLookupSkipsMissingLocations(t *testing.T) {
	t.Parallel()
	root := treeHTML(t, "<h1>Shown</h1><h1>Hidden</h1>")
	nodes := headNodes(t, root)
	headings := outline.CollectHeadings(root)
	// Only the first heading has a location.
	locs := fakeLocations(nodes[:1])

	headings = outline.Lookup(headings, locs)
	if len(headings) != 1 || headings[0].Title != "Shown" {
		t.Fatalf("lookup = %+v, want [Shown]", headings)
	}
}

func TestDumpOutlineXML(t *testing.T) {
	t.Parallel()
	root := treeHTML(t, "<h1>A & B</h1><h2>Sub</h2>")
	nodes := headNodes(t, root)
	headings := outline.Lookup(outline.CollectHeadings(root), fakeLocations(nodes))
	outline.AssignAnchors(headings)
	tree := outline.BuildTree(headings, outline.Options{}) //nolint:exhaustruct // intentional zero/partial fields
	xml := outline.DumpOutlineXML(tree)

	for _, want := range []string{
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>",
		"<outline xmlns=\"http://wkhtmltopdf.org/outline\">",
		`<item title="A &amp; B" page="1" link="__WKANCHOR_0" backLink="__WKANCHOR_0">`,
		`<item title="Sub" page="1" link="__WKANCHOR_1" backLink="__WKANCHOR_1"/>`,
		"</outline>",
	} {
		if !bytes.Contains(xml, []byte(want)) {
			t.Errorf("dump missing %q in:\n%s", want, xml)
		}
	}
}

func TestExplicitDocumentPageOrderingDoesNotMutateLocalPage(t *testing.T) {
	t.Parallel()
	root := treeHTML(t, "<h1>"+testFirstDocTitle+"</h1><h1>Second in document</h1>")
	nodes := headNodes(t, root)
	headings := outline.Lookup(outline.CollectHeadings(root), []layout.ElementLocation{
		{Node: nodes[0], Page: 3, X: 0, Y: 10, W: 100, H: 20},
		{Node: nodes[1], Page: 0, X: 0, Y: 10, W: 100, H: 20},
	})
	headings[0].DocPage = 0
	headings[1].DocPage = 2
	localPages := []int{headings[0].Page, headings[1].Page}

	tree := outline.BuildTreeBy(
		headings, outline.Options{}, //nolint:exhaustruct // intentional zero/partial fields
		outline.DocumentPage,
	)
	if got := tree.Children[0].Heading.Title; got != testFirstDocTitle {
		t.Fatalf("first heading = %q, want First in document", got)
	}

	if got := tree.Children[1].Heading.Title; got != "Second in document" {
		t.Fatalf("second heading = %q, want Second in document", got)
	}

	if headings[0].Page != localPages[0] || headings[1].Page != localPages[1] {
		t.Fatalf(
			"document ordering mutated local pages: got %d,%d want %d,%d",
			headings[0].Page, headings[1].Page, localPages[0], localPages[1],
		)
	}

	outline.SortHeadingsBy(headings, outline.DocumentPage)

	section, subsection := outline.SectionOfBy(headings, 1, outline.DocumentPage)
	if section != testFirstDocTitle || subsection != testFirstDocTitle {
		t.Errorf("SectionOfBy = %q, %q", section, subsection)
	}

	xml := outline.DumpOutlineXMLBy(tree, 0, outline.DocumentPage)
	if !bytes.Contains(xml, []byte(`title="First in document" page="1"`)) {
		t.Errorf("document-page XML missing page 1:\n%s", xml)
	}

	if !bytes.Contains(xml, []byte(`title="Second in document" page="3"`)) {
		t.Errorf("document-page XML missing page 3:\n%s", xml)
	}
}

func TestNilPageAccessorUsesLocalPage(t *testing.T) {
	t.Parallel()
	root := treeHTML(t, "<h1>Local</h1>")
	node := headNodes(t, root)[0]
	headings := outline.Lookup(outline.CollectHeadings(root), []layout.ElementLocation{
		{Node: node, Page: 4, X: 0, Y: 0, W: 1, H: 1},
	})
	outline.SortHeadingsBy(headings, nil)

	xml := outline.DumpOutlineXMLBy(
		outline.BuildTreeBy(headings, outline.Options{}, nil), //nolint:exhaustruct // intentional zero/partial fields
		0,
		nil,
	)
	if !bytes.Contains(xml, []byte(`page="5"`)) {
		t.Errorf("nil accessor should use local page:\n%s", xml)
	}
}
