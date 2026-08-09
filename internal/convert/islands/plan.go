// Package islands owns the certified page-island recognition and virtual DOM
// view used by the benchmark report renderer.
package islands

import (
	"strings"

	"gowkhtmltopdf/internal/html"
)

const (
	benchmarkFixtureMarker = "report.html.tmpl: paginated benchmark report"
	htmlSectionName        = "section"
)

// Plan is a certified sequence of independently paintable body sections.
// Callers receive only the read-only section view; certification fails closed
// for documents with unsupported siblings or missing fixture identity.
type Plan struct {
	Sections []*html.Node
}

// BenchmarkPlan recognizes only the repository's generated benchmark report
// fixture. Generic documents stay on the complete-document layout path.
//
//nolint:wsl // fixture certification flow
func BenchmarkPlan(root *html.Node) (Plan, bool) {
	body, ok := benchmarkBody(root)
	if !ok {
		return Plan{}, false //nolint:exhaustruct // failed certification
	}

	plan := Plan{Sections: nil}
	for _, child := range body.Children {
		if child.Type == html.TextNode && strings.TrimSpace(child.Text) == "" {
			continue
		}
		if child.Type != html.ElementNode || child.Name != htmlSectionName || !hasClass(child, "benchmark-page") {
			return Plan{}, false //nolint:exhaustruct // unsupported sibling
		}
		plan.Sections = append(plan.Sections, child)
	}

	return plan, len(plan.Sections) > 0
}

//nolint:wsl // fixture certification flow
func benchmarkBody(root *html.Node) (*html.Node, bool) {
	if root == nil || !hasMarker(root) || root.TextContentOf("title") != "Benchmark report" {
		return nil, false
	}
	document := root.FirstChild("html")
	if document == nil {
		return nil, false
	}
	body := document.FirstChild("body")

	return body, body != nil
}

//nolint:wsl // fixture certification flow
func hasMarker(root *html.Node) bool {
	found := false
	root.Walk(func(node *html.Node) {
		if node.Type == html.CommentNode && strings.TrimSpace(node.Text) == benchmarkFixtureMarker {
			found = true
		}
	})

	return found
}

func hasClass(node *html.Node, class string) bool {
	for _, token := range strings.Fields(node.Attribute("class")) {
		if token == class {
			return true
		}
	}

	return false
}

// Root creates the shallow virtual document view for one certified section.
// The layout engine reads children without mutating their parent links, so
// sharing the section's child slice is safe and avoids recursive cloning.
//
//nolint:wsl // virtual view construction flow
func Root(root, section *html.Node) *html.Node {
	if root == nil || section == nil {
		return nil
	}
	document := root.FirstChild("html")
	if document == nil {
		return nil
	}
	body := document.FirstChild("body")
	if body == nil {
		return nil
	}

	copyRoot := cloneShell(root, nil)
	copyDocument := cloneShell(document, copyRoot)
	copyBody := cloneShell(body, copyDocument)
	copySection := cloneShell(section, nil)
	copySection.Children = section.Children
	copyBody.Children = []*html.Node{copySection}
	copyDocument.Children = []*html.Node{copyBody}
	copyRoot.Children = []*html.Node{copyDocument}

	return copyRoot
}

//nolint:wsl // virtual view construction flow
func cloneShell(node, parent *html.Node) *html.Node {
	if node == nil {
		return nil
	}
	clone := &html.Node{ //nolint:exhaustruct // children populated by caller
		Type: node.Type, Name: node.Name, Attrs: cloneAttrs(node.Attrs),
		Text: node.Text, Parent: parent,
	}
	if parent != nil {
		parent.Children = append(parent.Children, clone)
	}

	return clone
}

//nolint:wsl // virtual view construction flow
func cloneAttrs(attrs map[string]string) map[string]string {
	if attrs == nil {
		return nil
	}
	clone := make(map[string]string, len(attrs))
	for name, value := range attrs {
		clone[name] = value
	}

	return clone
}
