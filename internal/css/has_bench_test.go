package css //nolint:testpackage // exercises unexported selector matching via Match

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// deepHasHTML builds a chain of depth nested <div> elements ending in a single
// <span class="target">. Matching div:has(.target) on the outermost div forces
// a full descendant walk with no early exit: the shape the iterator change in
// elementDescendants targets (no per-subject subtree slice, stop on match).
func deepHasHTML(depth int) string {
	var builder strings.Builder

	builder.WriteString("<html><body>")

	for range depth {
		builder.WriteString("<div>")
	}

	builder.WriteString(`<span class="target">t</span>`)

	for range depth {
		builder.WriteString("</div>")
	}

	builder.WriteString("</body></html>")

	return builder.String()
}

func parseBenchTree(b *testing.B, src string) *html.Node {
	b.Helper()

	root, err := html.Parse(src)
	if err != nil {
		b.Fatal(err)
	}

	return root
}

func hasBenchSubject(b *testing.B, src string) (Selector, *html.Node) {
	b.Helper()

	root := parseBenchTree(b, src)

	sel, ok := parseSelector("div:has(.target)")
	if !ok {
		b.Fatal("parse selector")
	}

	outer := root.FirstChild("html").FirstChild("body").FirstChild("div")

	return sel, outer
}

// BenchmarkHasDeepSubtreeMatch walks a 300-deep subtree to find the matching
// leaf: the worst case for :has(), no early exit possible.
func BenchmarkHasDeepSubtreeMatch(b *testing.B) {
	sel, outer := hasBenchSubject(b, deepHasHTML(300))

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		if !Match(sel, outer) {
			b.Fatal("expected match")
		}
	}
}

// BenchmarkHasDeepSubtreeEarlyExit matches a :has() whose target sits directly
// under the subject: the walk stops after the first yielded descendant.
func BenchmarkHasDeepSubtreeEarlyExit(b *testing.B) {
	src := "<html><body><div>" + `<span class="target">t</span>` + "</div></body></html>"
	sel, outer := hasBenchSubject(b, src)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		if !Match(sel, outer) {
			b.Fatal("expected match")
		}
	}
}
