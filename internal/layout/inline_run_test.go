//nolint:testpackage // tests exercise unexported package internals
package layout

import (
	"testing"

	"gowkhtmltopdf/internal/html"
)

func TestCollectInlineRunUsesContiguousChildren(t *testing.T) {
	root := mustParse(t, `<div>one<span>two</span> three</div>`)
	container := firstElementNamed(root, "div")
	span := firstElementNamed(root, "span")
	eng := &engine{ //nolint:exhaustruct // only style lookup is needed
		styles: map[*html.Node]*ResolvedStyle{
			span: {Display: cssDisplayInline, Float: cssDisplayNone},
		},
	}

	run, next := collectInlineRun(container.Children, 0, eng)
	if next != len(container.Children) || len(run) != len(container.Children) {
		t.Fatalf(
			"run len/next = %d/%d, want %d/%d",
			len(run),
			next,
			len(container.Children),
			len(container.Children),
		)
	}

	if &run[0] != &container.Children[0] || &run[len(run)-1] != &container.Children[len(container.Children)-1] {
		t.Fatal("contiguous inline run did not reuse the child slice")
	}
}

func TestCollectInlineRunFiltersDisplayNoneChildren(t *testing.T) {
	root := mustParse(t, `<div>one<span>hidden</span>two</div>`)
	container := firstElementNamed(root, "div")
	span := firstElementNamed(root, "span")
	eng := &engine{ //nolint:exhaustruct // only style lookup is needed
		styles: map[*html.Node]*ResolvedStyle{
			span: {Display: cssDisplayNone, Float: cssDisplayNone},
		},
	}

	run, next := collectInlineRun(container.Children, 0, eng)
	if next != len(container.Children) || len(run) != 2 {
		t.Fatalf(
			"run len/next = %d/%d, want 2/%d",
			len(run),
			next,
			len(container.Children),
		)
	}

	if run[0] == span || run[1] == span {
		t.Fatal("display:none child remained in inline run")
	}
}

func firstElementNamed(root *html.Node, name string) *html.Node {
	if root.Type == html.ElementNode && root.Name == name {
		return root
	}

	for _, child := range root.Children {
		if found := firstElementNamed(child, name); found != nil {
			return found
		}
	}

	return nil
}
