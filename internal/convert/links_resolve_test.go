package convert //nolint:testpackage // white-box tests need unexported access

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

func testBodyNavigationResult() *layout.Result {
	target := &html.Node{ //nolint:exhaustruct // test needs only id attributes
		Attrs: map[string]string{"id": "target"},
	}

	return &layout.Result{ //nolint:exhaustruct // test needs only navigation fields
		Locations: []layout.ElementLocation{
			{Node: target, Page: 2, X: 3, Y: 4, W: 5, H: 6},
		},
		Ops: []layout.Op{
			//nolint:exhaustruct // test needs only fragment geometry
			{Kind: layout.OpLinkURI, URI: "#target", X: 7, Y: 8, W: 9, H: 10},
			//nolint:exhaustruct // test needs only URI filtering
			{Kind: layout.OpLinkURI, URI: "https://example.com", X: 11, Y: 12},
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

	first := &objectState{ //nolint:exhaustruct // test needs only indexed destination
		navigation: bodyNavigation{
			ids: map[string]layout.ElementLocation{
				"duplicate": {Node: nil, Page: 0, X: 1, Y: 0, W: 0, H: 0},
			},
			idElems: nil,
			links:   nil,
		},
	}
	later := &objectState{ //nolint:exhaustruct // test needs only indexed destination
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
		{Kind: layout.OpLinkURI, URI: "docs/a.html"},           //nolint:exhaustruct // intentional zero-value fields
		{Kind: layout.OpLinkURI, URI: "#frag"},                 //nolint:exhaustruct // intentional zero-value fields
		{Kind: layout.OpLinkURI, URI: "https://example.com/x"}, //nolint:exhaustruct // intentional zero-value fields
		{Kind: layout.OpLinkURI, URI: "mailto:a@b.c"},          //nolint:exhaustruct // intentional zero-value fields
	}
	resolveRelativeLinkURIs(ops, "https://example.com/base/page.html")

	if ops[0].URI != "https://example.com/base/docs/a.html" {
		t.Errorf("relative = %q", ops[0].URI)
	}

	if ops[1].URI != "#frag" {
		t.Errorf("fragment mutated: %q", ops[1].URI)
	}

	if ops[2].URI != "https://example.com/x" {
		t.Errorf("absolute mutated: %q", ops[2].URI)
	}

	if ops[3].URI != "mailto:a@b.c" {
		t.Errorf("mailto mutated: %q", ops[3].URI)
	}
}
