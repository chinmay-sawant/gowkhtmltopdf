package markup_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/markup"
)

func TestParseDivAndScript(t *testing.T) {
	t.Parallel()

	root, err := markup.Parse([]byte(
		`<div id="user" data-action="focus">Ada</div><script>alert(1)</script>`,
	))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	div := find(root, func(node *markup.Node) bool {
		return node.Type == markup.TypeElement && node.Name == "div"
	})
	if div == nil {
		t.Fatal("div not found")
	}

	if div.Attrs["id"] != "user" || div.Attrs["data-action"] != "focus" {
		t.Fatalf("attrs = %#v", div.Attrs)
	}

	if text := nodeText(div); text != "Ada" {
		t.Fatalf("div text = %q", text)
	}

	script := find(root, func(node *markup.Node) bool {
		return node.Type == markup.TypeElement && node.Name == "script"
	})
	if script == nil {
		t.Fatal("script not found")
	}

	if text := nodeText(script); text != "alert(1)" {
		t.Fatalf("script text = %q", text)
	}
}

func find(node *markup.Node, match func(*markup.Node) bool) *markup.Node {
	if node == nil {
		return nil
	}

	if match(node) {
		return node
	}

	for _, child := range node.Children {
		if found := find(child, match); found != nil {
			return found
		}
	}

	return nil
}

func nodeText(node *markup.Node) string {
	if node == nil {
		return ""
	}

	text := node.Text

	for _, child := range node.Children {
		text += nodeText(child)
	}

	return text
}
