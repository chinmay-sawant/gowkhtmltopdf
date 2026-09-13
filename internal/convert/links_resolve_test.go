package convert

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

func TestCollectBodyNavigationCopiesOnlyPostPaintLinkData(t *testing.T) {
	t.Parallel()

	nav := collectBodyNavigation(testBodyNavigationResult())
	assertProjectedBodyID(t, nav)
	assertProjectedFragmentLink(t, nav)
}

func TestBodyNavigationProjectionIsIndependentOfLayoutResult(t *testing.T) {
	t.Parallel()

	result := testBodyNavigationResult()
	nav := collectBodyNavigation(result)
	result.Locations[0].X = 99
	result.Ops[0].SetURI("#changed")

	if got := nav.ids["target"].X; got != 3 {
		t.Fatalf("projected destination X = %v, want copied value 3", got)
	}

	if got := nav.links[0].uri; got != "#target" {
		t.Fatalf("projected link URI = %q, want copied value #target", got)
	}
}

func testLinkOp(uri string, x, y, w, h float64) layout.Op {
	op := layout.Op{Kind: layout.OpLinkURI, X: x, Y: y, W: w, H: h}
	op.SetURI(uri)

	return op
}

func testBodyNavigationResult() *layout.Result {
	target := &html.Node{
		Attrs: map[string]string{"id": "target"},
	}

	return &layout.Result{
		Locations: []layout.ElementLocation{
			{Node: target, Page: 2, X: 3, Y: 4, W: 5, H: 6},
		},
		Ops: []layout.Op{
			testLinkOp("#target", 7, 8, 9, 10),
			testLinkOp("https://example.com", 11, 12, 0, 0),
		},
	}
}

func assertProjectedBodyID(t *testing.T, nav bodyNavigation) {
	t.Helper()

	loc, ok := nav.ids["target"]

	if !ok {
		t.Fatal("target id was not projected")
	}

	if loc.Node != nil {
		t.Fatal("projected destination retained its DOM node")
	}

	if loc.Page != 2 || loc.X != 3 || loc.Y != 4 || loc.W != 5 || loc.H != 6 {
		t.Errorf("destination = %#v, want copied geometry", loc)
	}
}

func assertProjectedFragmentLink(t *testing.T, nav bodyNavigation) {
	t.Helper()

	if len(nav.links) != 1 {
		t.Fatalf("fragment links = %d, want 1", len(nav.links))
	}

	if got := nav.links[0]; got.uri != "#target" || got.loc.X != 7 || got.loc.Y != 8 || got.loc.W != 9 || got.loc.H != 10 {
		t.Errorf("fragment link = %#v, want copied #target geometry", got)
	}
}

func TestBuildBodyIDIndexKeepsLaterDuplicate(t *testing.T) {
	t.Parallel()

	first := &objectState{
		navigation: bodyNavigation{
			ids: map[string]layout.ElementLocation{
				"duplicate": {Node: nil, Page: 0, X: 1, Y: 0, W: 0, H: 0},
			},
			idElems: nil,
			links:   nil,
		},
	}
	later := &objectState{
		navigation: bodyNavigation{
			ids: map[string]layout.ElementLocation{
				"duplicate": {Node: nil, Page: 1, X: 2, Y: 0, W: 0, H: 0},
			},
			idElems: nil,
			links:   nil,
		},
	}

	dest := buildBodyIDIndex([]*objectState{first, later})["duplicate"]
	if dest.st != later || dest.loc.Page != 1 || dest.loc.X != 2 {
		t.Errorf("duplicate destination = %#v, want later object location", dest)
	}
}

func TestResolveRelativeLinkURIs(t *testing.T) {
	t.Parallel()

	ops := []layout.Op{
		testLinkOp("docs/a.html", 0, 0, 0, 0),
		testLinkOp("//cdn.example/a.css", 0, 0, 0, 0),
		testLinkOp("#frag", 0, 0, 0, 0),
		testLinkOp("https://example.com/x", 0, 0, 0, 0),
		testLinkOp("mailto:a@b.c", 0, 0, 0, 0),
	}
	resolveRelativeLinkURIs(ops, "https://example.com/base/page.html")

	if ops[0].URI != "https://example.com/base/docs/a.html" {
		t.Errorf("relative = %q", ops[0].URI)
	}

	if ops[1].URI != "https://cdn.example/a.css" {
		t.Errorf("protocol-relative = %q", ops[1].URI)
	}

	if ops[2].URI != "#frag" {
		t.Errorf("fragment mutated: %q", ops[2].URI)
	}

	if ops[3].URI != "https://example.com/x" {
		t.Errorf("absolute mutated: %q", ops[3].URI)
	}

	if ops[4].URI != "mailto:a@b.c" {
		t.Errorf("mailto mutated: %q", ops[4].URI)
	}
}
