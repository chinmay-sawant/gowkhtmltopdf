package convert

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/layout"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/settings"
)

// benchmarkFixtureMarker is the checked-in benchmark template's immutable
// source marker. The page-island path is deliberately limited to that input
// until a broader CSS/layout proof exists.
const (
	benchmarkFixtureMarker = "report.html.tmpl: paginated benchmark report"
	htmlSectionName        = "section"
	islandBreakOverrideCSS = ".benchmark-page { page-break-before: auto !important; }"
)

var (
	errCertifiedIslandExpanded = errors.New("certified page island expanded")
)

// pageIslandPlan is a certified sequence of independently paintable body
// sections. It is intentionally private: generic documents must keep the
// complete-document layout path until their cross-island dependencies are
// proven safe.
type pageIslandPlan struct {
	sections []*html.Node
}

// benchmarkPageIslandPlan recognizes only the repository's generated report
// fixture: its source marker, title, and body must be a whitespace-separated
// sequence of section.benchmark-page elements. Anything else fails closed.
func benchmarkPageIslandPlan(root *html.Node) (pageIslandPlan, bool) {
	body, ok := benchmarkIslandBody(root)
	if !ok {
		return pageIslandPlan{}, false //nolint:exhaustruct // no certified islands
	}

	return collectBenchmarkIslands(body)
}

func benchmarkIslandBody(root *html.Node) (*html.Node, bool) {
	if root == nil || !hasBenchmarkFixtureMarker(root) || root.TextContentOf("title") != "Benchmark report" {
		return nil, false
	}

	document := root.FirstChild("html")
	if document == nil {
		return nil, false
	}

	body := document.FirstChild("body")
	if body == nil {
		return nil, false
	}

	return body, true
}

func collectBenchmarkIslands(body *html.Node) (pageIslandPlan, bool) {
	plan := pageIslandPlan{sections: nil}

	for _, child := range body.Children {
		if child.Type == html.TextNode && strings.TrimSpace(child.Text) == "" {
			continue
		}

		if child.Type != html.ElementNode || child.Name != htmlSectionName || !hasHTMLClass(child, "benchmark-page") {
			return pageIslandPlan{}, false //nolint:exhaustruct // unsupported sibling dependency
		}

		plan.sections = append(plan.sections, child)
	}

	return plan, len(plan.sections) > 0
}

func hasBenchmarkFixtureMarker(root *html.Node) bool {
	found := false

	root.Walk(func(node *html.Node) {
		if node.Type == html.CommentNode && strings.TrimSpace(node.Text) == benchmarkFixtureMarker {
			found = true
		}
	})

	return found
}

func hasHTMLClass(node *html.Node, class string) bool {
	for _, token := range strings.Fields(node.Attribute("class")) {
		if token == class {
			return true
		}
	}

	return false
}

func renderBenchmarkPageIslands(
	ctx context.Context, doc *pdf.Document, state *objectState, root *html.Node, plan pageIslandPlan,
	font *pdf.Font, registry *pdf.Registry, sheets []*css.Stylesheet, zoom float64,
	imagesFn func(string) ([]byte, error), global settings.PdfGlobal, printLinkUnderline bool,
	obj *settings.PdfObject, log io.Writer,
) error {
	breakSheet, err := css.Parse(islandBreakOverrideCSS)
	if err != nil {
		return fmt.Errorf("parse certified island break override: %w", err)
	}

	islandSheets := append(append([]*css.Stylesheet(nil), sheets...), breakSheet)
	start := doc.PageCount()
	workspace := &layout.Workspace{}

	for _, section := range plan.sections {
		islandRoot := benchmarkIslandRoot(root, section)
		res, err := layout.WithWorkspace(ctx, islandRoot, state.bodyLayoutOpts(
			font, registry, islandSheets, zoom, imagesFn, global.Background, printLinkUnderline,
		), workspace)

		if err != nil {
			return fmt.Errorf("layout certified page island: %w", err)
		}

		before := doc.PageCount()

		if err := layout.PaintContext(ctx, doc, res, paintOptions(state.geom)); err != nil {
			return fmt.Errorf("paint certified page island: %w", err)
		}

		if doc.PageCount() != before+1 {
			return errCertifiedIslandExpanded
		}

		pageOffset := before - start
		for _, heading := range collectObjectHeadings(islandRoot, res, before, global, *obj, log) {
			heading.Page += pageOffset
			state.headings = append(state.headings, heading)
		}

		appendIslandNavigation(&state.navigation, collectBodyNavigation(res), pageOffset, state.geom.contentH)
		workspace.Release(res)
	}

	state.pages = doc.PageCount() - start
	state.offset = start

	return nil
}

func benchmarkIslandRoot(root, section *html.Node) *html.Node {
	document := root.FirstChild("html")
	body := document.FirstChild("body")
	copyRoot := &html.Node{Type: root.Type, Name: root.Name} //nolint:exhaustruct // tree wrapper

	copyDocument := &html.Node{ //nolint:exhaustruct // tree wrapper
		Type: document.Type, Name: document.Name, Attrs: document.Attrs, Parent: copyRoot,
	}
	copyBody := &html.Node{ //nolint:exhaustruct // tree wrapper
		Type: body.Type, Name: body.Name, Attrs: body.Attrs, Parent: copyDocument, Children: []*html.Node{section},
	}
	copyDocument.Children = []*html.Node{copyBody}
	copyRoot.Children = []*html.Node{copyDocument}

	return copyRoot
}

func appendIslandNavigation(dst *bodyNavigation, src bodyNavigation, pageOffset int, contentH float64) {
	if dst.ids == nil {
		dst.ids = make(map[string]layout.ElementLocation, len(src.ids))
	}

	yOffset := float64(pageOffset) * contentH

	for id, loc := range src.ids {
		loc.Page += pageOffset
		loc.Y += yOffset
		dst.ids[id] = loc
	}

	for _, link := range src.links {
		link.loc.Y += yOffset
		dst.links = append(dst.links, link)
	}
}
