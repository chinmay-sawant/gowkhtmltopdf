package css //nolint:testpackage // exercises unexported parseSelector and Match

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// ofTypeFixture is mixed-tag siblings so of-type index differs from nth-child.
// Whitespace text nodes sit between the elements, as in real markup.
func ofTypeFixture(t *testing.T) map[string]*html.Node {
	t.Helper()

	root := treeFor(t, `<html><body><div id="box">
		<p id="p1">one</p>
		<span id="s1">s</span>
		<p id="p2">two</p>
		<span id="s2">s2</span>
		<p id="p3">three</p>
	</div></body></html>`)

	ids := []string{"box", "p1", "s1", "p2", "s2", "p3"}
	nodes := make(map[string]*html.Node, len(ids))

	for _, nodeID := range ids {
		node := byID(root, nodeID)
		if node == nil {
			t.Fatalf("missing #%s", nodeID)
		}

		nodes[nodeID] = node
	}

	return nodes
}

func TestFirstOfType(t *testing.T) {
	t.Parallel()

	nodes := ofTypeFixture(t)
	cases := []struct {
		sel  string
		id   string
		want bool
	}{
		{"p:first-of-type", "p1", true},
		{"p:first-of-type", "p2", false},
		{"p:first-of-type", "p3", false},
		{"span:first-of-type", "s1", true},
		{"span:first-of-type", "s2", false},
		{"span:first-of-type", "p1", false},
		{":first-of-type", "p1", true},
		{":first-of-type", "box", true},
		// first-of-type is nth-of-type(1); span is not first-child.
		{"span:nth-of-type(1)", "s1", true},
		{"span:first-child", "s1", false},
	}

	checkSelectorMatchIDs(t, nodes, cases)
}

func TestLastOfType(t *testing.T) {
	t.Parallel()

	nodes := ofTypeFixture(t)
	cases := []struct {
		sel  string
		id   string
		want bool
	}{
		{"p:last-of-type", "p3", true},
		{"p:last-of-type", "p1", false},
		{"p:last-of-type", "p2", false},
		{"span:last-of-type", "s2", true},
		{"span:last-of-type", "s1", false},
		{":last-of-type", "p3", true},
		{":last-of-type", "box", true},
		// last-of-type is nth-last-of-type(1); span is not last-child.
		{"span:nth-last-of-type(1)", "s2", true},
		{"span:last-child", "s2", false},
		{"p:last-child", "p3", true},
	}

	checkSelectorMatchIDs(t, nodes, cases)
}

func TestNthOfType(t *testing.T) {
	t.Parallel()

	nodes := ofTypeFixture(t)
	cases := []struct {
		sel  string
		id   string
		want bool
	}{
		{"p:nth-of-type(1)", "p1", true},
		{"p:nth-of-type(2)", "p2", true},
		{"p:nth-of-type(3)", "p3", true},
		{"p:nth-of-type(4)", "p3", false},
		{"p:nth-of-type(2)", "p1", false},
		// of-type index ignores other tags: p2 is child 3, of-type 2.
		{"p:nth-child(2)", "p2", false},
		{"p:nth-child(3)", "p2", true},
		{"p:nth-of-type(odd)", "p1", true},
		{"p:nth-of-type(odd)", "p2", false},
		{"p:nth-of-type(odd)", "p3", true},
		{"p:nth-of-type(even)", "p2", true},
		{"p:nth-of-type(even)", "p1", false},
		{"p:nth-of-type(2n)", "p2", true},
		{"p:nth-of-type(2n+1)", "p1", true},
		{"p:nth-of-type(2n+1)", "p3", true},
		{"p:nth-of-type(2n+1)", "p2", false},
		{"span:nth-of-type(2)", "s2", true},
		{"span:nth-of-type(2)", "s1", false},
		// invalid an+b never matches (same as nth-child).
		{"p:nth-of-type(foo)", "p1", false},
		{"p:nth-of-type(foo)", "p2", false},
		{"p:nth-of-type(0)", "p1", false},
	}

	checkSelectorMatchIDs(t, nodes, cases)
}

func TestNthLastOfType(t *testing.T) {
	t.Parallel()

	nodes := ofTypeFixture(t)
	cases := []struct {
		sel  string
		id   string
		want bool
	}{
		{"p:nth-last-of-type(1)", "p3", true},
		{"p:nth-last-of-type(2)", "p2", true},
		{"p:nth-last-of-type(3)", "p1", true},
		{"p:nth-last-of-type(1)", "p1", false},
		{"p:nth-last-of-type(4)", "p1", false},
		{"span:nth-last-of-type(1)", "s2", true},
		{"span:nth-last-of-type(2)", "s1", true},
		{"span:nth-last-of-type(2)", "s2", false},
		{"p:nth-last-of-type(odd)", "p3", true},
		{"p:nth-last-of-type(odd)", "p1", true},
		{"p:nth-last-of-type(odd)", "p2", false},
		{"p:nth-last-of-type(even)", "p2", true},
		{"p:nth-last-of-type(2n+1)", "p3", true},
		{"p:nth-last-of-type(foo)", "p3", false},
	}

	checkSelectorMatchIDs(t, nodes, cases)
}

func checkSelectorMatchIDs(t *testing.T, nodes map[string]*html.Node, cases []struct {
	sel  string
	id   string
	want bool
},
) {
	t.Helper()

	for _, testCase := range cases {
		sel, ok := parseSelector(testCase.sel)
		if !ok {
			t.Fatalf("parseSelector(%q) failed", testCase.sel)
		}

		node := nodes[testCase.id]
		if node == nil {
			t.Fatalf("missing #%s", testCase.id)
		}

		if got := Match(sel, node); got != testCase.want {
			t.Errorf("Match(%q, #%s) = %v, want %v", testCase.sel, testCase.id, got, testCase.want)
		}
	}
}
