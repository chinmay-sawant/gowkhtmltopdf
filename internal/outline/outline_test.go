package outline

import (
	"bytes"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/layout"
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

// fakeLocations builds element locations for the given nodes at the given
// (page, y, x) positions.
func fakeLocations(nodes []*html.Node, page int, y, x float64) []layout.ElementLocation {
	var locs []layout.ElementLocation
	for _, n := range nodes {
		locs = append(locs, layout.ElementLocation{Node: n, Page: page, X: x, Y: y, W: 100, H: 20})
		y += 40
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
	// h1 > h2 > h1: the second h1 closes the h2 and returns to the root.
	root := treeHTML(t, "<h1>One</h1><h2>Sub</h2><h1>Two</h1>")
	nodes := headNodes(t, root)
	hs := CollectHeadings(root)
	hs = Lookup(hs, fakeLocations(nodes, 0, 10, 0))
	AssignAnchors(hs)
	tree := BuildTree(hs, Options{})

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
	// Document order is h2, h4, h1; the h4 sits mid-page below the h2 and
	// jumps two levels, clamping to a child of h2 at level 3.
	root := treeHTML(t, "<h2>A</h2><h4>Deep</h4><h1>B</h1>")
	nodes := headNodes(t, root)
	hs := CollectHeadings(root)
	hs = Lookup(hs, fakeLocations(nodes, 0, 10, 0))
	tree := BuildTree(hs, Options{})

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
	// Page 1 ends inside a chapter; page 2 starts with its h2: the h2 nests
	// under the h1 from page 1 (page order governs, not document order).
	root := treeHTML(t, "<h1>Ch</h1><h2>Late</h2>")
	nodes := headNodes(t, root)
	locs := []layout.ElementLocation{
		{Node: nodes[0], Page: 0, X: 0, Y: 10, W: 100, H: 20},
		{Node: nodes[1], Page: 1, X: 0, Y: 10, W: 100, H: 20},
	}
	hs := Lookup(CollectHeadings(root), locs)

	tree := BuildTree(hs, Options{})
	if len(tree.Children) != 1 || len(tree.Children[0].Children) != 1 {
		t.Fatalf("tree = %+v, want Ch > Late", tree)
	}
}

func TestOutlineDepth(t *testing.T) {
	root := treeHTML(t, "<h1>One</h1><h2>Sub</h2><h2>Sub2</h2>")
	nodes := headNodes(t, root)
	hs := Lookup(CollectHeadings(root), fakeLocations(nodes, 0, 10, 0))

	tree := BuildTree(hs, Options{MaxDepth: 1})
	if len(tree.Children) != 1 {
		t.Fatalf("root children = %d, want 1", len(tree.Children))
	}

	if len(tree.Children[0].Children) != 0 {
		t.Errorf("MaxDepth=1 must drop level-2 headings")
	}
}

func TestOutlineExclude(t *testing.T) {
	root := treeHTML(t, `<h1>Keep</h1><h1 class="hidden">Drop</h1><h1 id="x">Also</h1>`)
	nodes := headNodes(t, root)
	hs := Lookup(CollectHeadings(root), fakeLocations(nodes, 0, 10, 0))

	tree := BuildTree(hs, Options{Exclude: []css.Selector{
		parseSel(t, ".hidden"),
		parseSel(t, "#x"),
	}})
	if len(tree.Children) != 1 || tree.Children[0].Heading.Title != "Keep" {
		t.Fatalf("tree = %+v, want [Keep]", tree.Children)
	}
}

func TestOutlineExcludeNestedAndEmpty(t *testing.T) {
	// Descendant/element selectors drop via css.Match; empty Exclude is a no-op.
	root := treeHTML(t, `<div class="nav"><h2>Nav</h2></div><h1>Body</h1><h2>Sec</h2>`)
	nodes := headNodes(t, root)
	hs := Lookup(CollectHeadings(root), fakeLocations(nodes, 0, 10, 0))

	// Document order Nav(h2), Body(h1), Sec(h2): Nav clamps to a root child,
	// Body is a sibling root child, Sec nests under Body.
	tree := BuildTree(hs, Options{})
	if len(tree.Children) != 2 || tree.Children[0].Heading.Title != "Nav" || tree.Children[1].Heading.Title != "Body" {
		t.Fatalf("empty Exclude: root = %+v, want [Nav, Body]", titles(tree.Children))
	}

	if len(tree.Children[1].Children) != 1 || tree.Children[1].Children[0].Heading.Title != "Sec" {
		t.Fatalf("empty Exclude: Body children = %+v, want [Sec]", titles(tree.Children[1].Children))
	}

	tree = BuildTree(hs, Options{Exclude: []css.Selector{parseSel(t, ".nav h2")}})
	if len(tree.Children) != 1 || tree.Children[0].Heading.Title != "Body" {
		t.Fatalf("after .nav h2: root = %+v, want [Body]", titles(tree.Children))
	}

	if len(tree.Children[0].Children) != 1 || tree.Children[0].Children[0].Heading.Title != "Sec" {
		t.Fatalf("after .nav h2: Body children = %+v, want [Sec]", titles(tree.Children[0].Children))
	}

	// Element selector drops every h2, leaving only the h1.
	tree = BuildTree(hs, Options{Exclude: []css.Selector{parseSel(t, "h2")}})
	if len(tree.Children) != 1 || tree.Children[0].Heading.Title != "Body" || len(tree.Children[0].Children) != 0 {
		t.Fatalf("after h2: tree = %+v, want [Body] with no children", titles(tree.Children))
	}
}

// titles returns heading titles for a slice of outline nodes (test helper).
func titles(ns []*Node) []string {
	out := make([]string, len(ns))

	for i, n := range ns {
		if n.Heading != nil {
			out[i] = n.Heading.Title
		}
	}

	return out
}

func TestSortHeadings(t *testing.T) {
	root := treeHTML(t, "<h1>A</h1><h2>B</h2><h1>C</h1>")
	nodes := headNodes(t, root)
	hs := CollectHeadings(root)
	locs := []layout.ElementLocation{
		{Node: nodes[0], Page: 1, X: 5, Y: 30, W: 100, H: 20},
		{Node: nodes[1], Page: 0, X: 0, Y: 10, W: 100, H: 20},
		{Node: nodes[2], Page: 1, X: 5, Y: 10, W: 100, H: 20},
	}
	hs = Lookup(hs, locs)

	// Deliberately out of order: page 1 before page 0; C (y=10) before A (y=30).
	reversed := []*Heading{hs[0], hs[2], hs[1]}
	SortHeadings(reversed)

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
	root := treeHTML(t, "<h1>Intro</h1><h2>Deep</h2><h1>Body</h1>")
	nodes := headNodes(t, root)
	hs := Lookup(CollectHeadings(root), []layout.ElementLocation{
		{Node: nodes[0], Page: 0, X: 0, Y: 10, W: 100, H: 20},
		{Node: nodes[1], Page: 0, X: 0, Y: 50, W: 100, H: 20},
		{Node: nodes[2], Page: 1, X: 0, Y: 10, W: 100, H: 20},
	})
	SortHeadings(hs)

	cases := []struct {
		page       int
		section    string
		subsection string
	}{
		{0, "Intro", "Deep"},
		{1, "Intro", "Body"},
		{2, "Intro", "Body"},
		{-1, "", ""},
	}
	for _, c := range cases {
		sec, sub := SectionOf(hs, c.page)
		if sec != c.section || sub != c.subsection {
			t.Errorf("SectionOf(page %d) = %q, %q; want %q, %q", c.page, sec, sub, c.section, c.subsection)
		}
	}
}

func TestCollapseWS(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"  Two   Words ", "Two Words"},
		{"a\t\nb\rc\fd", "a b c d"},
		{" already ", "already"},
		{"no-space", "no-space"},
	}
	for _, tc := range cases {
		if got := CollapseWS(tc.in); got != tc.want {
			t.Errorf("CollapseWS(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestOutlineCollectTitlesAndAnchors(t *testing.T) {
	root := treeHTML(t, "<h1>  Two   Words </h1><p>text</p><h2>Deep</h2>")

	hs := CollectHeadings(root)
	if len(hs) != 2 {
		t.Fatalf("headings = %d, want 2", len(hs))
	}

	if hs[0].Title != "Two Words" || hs[1].Title != "Deep" {
		t.Errorf("titles = %q, %q", hs[0].Title, hs[1].Title)
	}

	AssignAnchors(hs)

	if !strings.HasPrefix(hs[0].Anchor, "__WKANCHOR_") || hs[0].Anchor == hs[1].Anchor {
		t.Errorf("anchors = %q, %q", hs[0].Anchor, hs[1].Anchor)
	}
}

func TestLookupSkipsMissingLocations(t *testing.T) {
	root := treeHTML(t, "<h1>Shown</h1><h1>Hidden</h1>")
	nodes := headNodes(t, root)
	hs := CollectHeadings(root)
	// Only the first heading has a location.
	locs := fakeLocations(nodes[:1], 0, 10, 0)

	hs = Lookup(hs, locs)
	if len(hs) != 1 || hs[0].Title != "Shown" {
		t.Fatalf("lookup = %+v, want [Shown]", hs)
	}
}

func TestDumpOutlineXML(t *testing.T) {
	root := treeHTML(t, "<h1>A & B</h1><h2>Sub</h2>")
	nodes := headNodes(t, root)
	hs := Lookup(CollectHeadings(root), fakeLocations(nodes, 0, 10, 0))
	AssignAnchors(hs)
	tree := BuildTree(hs, Options{})
	xml := DumpOutlineXML(tree)

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
	root := treeHTML(t, "<h1>First in document</h1><h1>Second in document</h1>")
	nodes := headNodes(t, root)
	hs := Lookup(CollectHeadings(root), []layout.ElementLocation{
		{Node: nodes[0], Page: 3, X: 0, Y: 10, W: 100, H: 20},
		{Node: nodes[1], Page: 0, X: 0, Y: 10, W: 100, H: 20},
	})
	hs[0].DocPage = 0
	hs[1].DocPage = 2
	localPages := []int{hs[0].Page, hs[1].Page}

	tree := BuildTreeBy(hs, Options{}, DocumentPage)
	if got := tree.Children[0].Heading.Title; got != "First in document" {
		t.Fatalf("first heading = %q, want First in document", got)
	}

	if got := tree.Children[1].Heading.Title; got != "Second in document" {
		t.Fatalf("second heading = %q, want Second in document", got)
	}

	if hs[0].Page != localPages[0] || hs[1].Page != localPages[1] {
		t.Fatalf("document ordering mutated local pages: got %d,%d want %d,%d", hs[0].Page, hs[1].Page, localPages[0], localPages[1])
	}

	SortHeadingsBy(hs, DocumentPage)

	section, subsection := SectionOfBy(hs, 1, DocumentPage)
	if section != "First in document" || subsection != "First in document" {
		t.Errorf("SectionOfBy = %q, %q", section, subsection)
	}

	xml := DumpOutlineXMLBy(tree, 0, DocumentPage)
	if !bytes.Contains(xml, []byte(`title="First in document" page="1"`)) {
		t.Errorf("document-page XML missing page 1:\n%s", xml)
	}

	if !bytes.Contains(xml, []byte(`title="Second in document" page="3"`)) {
		t.Errorf("document-page XML missing page 3:\n%s", xml)
	}
}

func TestNilPageAccessorUsesLocalPage(t *testing.T) {
	root := treeHTML(t, "<h1>Local</h1>")
	node := headNodes(t, root)[0]
	hs := Lookup(CollectHeadings(root), []layout.ElementLocation{{Node: node, Page: 4, X: 0, Y: 0, W: 1, H: 1}})
	SortHeadingsBy(hs, nil)

	xml := DumpOutlineXMLBy(BuildTreeBy(hs, Options{}, nil), 0, nil)
	if !bytes.Contains(xml, []byte(`page="5"`)) {
		t.Errorf("nil accessor should use local page:\n%s", xml)
	}
}
