package css

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func findID(n *html.Node, targetID string) *html.Node {
	if n.Attribute("id") == targetID {
		return n
	}

	for _, c := range n.Children {
		if got := findID(c, targetID); got != nil {
			return got
		}
	}

	return nil
}

func stateNode(t *testing.T, src, targetID string) *html.Node {
	t.Helper()

	root, err := html.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	node := findID(root, targetID)
	if node == nil {
		t.Fatalf("no node id %q", targetID)
	}

	return node
}

func mustSelector(t *testing.T, src string) Selector {
	t.Helper()

	sel, ok := parseSelector(src)
	if !ok {
		t.Fatalf("parse selector %q", src)
	}

	return sel
}

func TestMatchStateFocusHoverActive(t *testing.T) {
	t.Parallel()

	node := stateNode(t, `<div id="e"></div>`, "e")

	focus := mustSelector(t, "#e:focus")
	if !(MatchState{Focus: "e"}).Matches(focus, node) {
		t.Fatal("focus state did not match")
	}

	if (MatchState{}).Matches(focus, node) || (MatchState{Focus: "x"}).Matches(focus, node) {
		t.Fatal("focus matched without the id")
	}

	visible := mustSelector(t, "#e:focus-visible")
	if !(MatchState{Focus: "e"}).Matches(visible, node) {
		t.Fatal("focus-visible did not match")
	}

	hover := mustSelector(t, "#e:hover")
	if !(MatchState{Hover: "e"}).Matches(hover, node) {
		t.Fatal("hover did not match")
	}

	active := mustSelector(t, "#e:active")
	if !(MatchState{Active: "e"}).Matches(active, node) {
		t.Fatal("active did not match")
	}
}

func TestMatchChecked(t *testing.T) {
	t.Parallel()

	sel := mustSelector(t, "input:checked")

	checked := stateNode(t, `<input id="c" type="checkbox" checked>`, "c")
	if !Match(sel, checked) {
		t.Fatal("checked box did not match")
	}

	plain := stateNode(t, `<input id="c" type="checkbox">`, "c")
	if Match(sel, plain) {
		t.Fatal("unchecked box matched")
	}

	text := stateNode(t, `<input id="c" type="text" checked>`, "c")
	if Match(sel, text) {
		t.Fatal("text input matched :checked")
	}
}
