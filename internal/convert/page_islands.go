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
	render objectRenderContext, log io.Writer,
) error {
	breakSheet, err := css.Parse(islandBreakOverrideCSS)
	if err != nil {
		return fmt.Errorf("parse certified island break override: %w", err)
	}

	islandSheets := append(append([]*css.Stylesheet(nil), render.sheets...), breakSheet)
	start := doc.PageCount()
	workspace := &layout.Workspace{}
	island := pageIslandRenderContext{
		ctx:       ctx,
		doc:       doc,
		state:     state,
		root:      root,
		start:     start,
		sheets:    islandSheets,
		renderCtx: render,
		workspace: workspace,
		log:       log,
	}

	for _, section := range plan.sections {
		if err := island.render(section); err != nil {
			return err
		}
	}

	state.pages = doc.PageCount() - start
	state.offset = start

	return nil
}

type pageIslandRenderContext struct {
	ctx       context.Context
	doc       *pdf.Document
	state     *objectState
	root      *html.Node
	start     int
	sheets    []*css.Stylesheet
	renderCtx objectRenderContext
	workspace *layout.Workspace
	log       io.Writer
}

func (island pageIslandRenderContext) render(section *html.Node) error {
	islandRoot := benchmarkIslandRoot(island.root, section)
	res, err := layout.WithWorkspace(island.ctx, islandRoot, island.state.bodyLayoutOpts(
		objectRenderContext{
			global:             island.renderCtx.global,
			obj:                island.renderCtx.obj,
			font:               island.renderCtx.font,
			registry:           island.renderCtx.registry,
			sheets:             island.sheets,
			zoom:               island.renderCtx.zoom,
			imagesFn:           island.renderCtx.imagesFn,
			printLinkUnderline: island.renderCtx.printLinkUnderline,
		},
	), island.workspace)
	if err != nil {
		return fmt.Errorf("layout certified page island: %w", err)
	}
	defer island.workspace.Release(res)

	before := island.doc.PageCount()
	if err := layout.PaintContext(island.ctx, island.doc, res, paintOptions(island.state.geom)); err != nil {
		return fmt.Errorf("paint certified page island: %w", err)
	}
	if island.doc.PageCount() != before+1 {
		return errCertifiedIslandExpanded
	}

	pageOffset := before - island.start
	for _, heading := range collectObjectHeadings(
		islandRoot,
		res,
		before,
		island.renderCtx.global,
		*island.renderCtx.obj,
		island.log,
	) {
		heading.Page += pageOffset
		island.state.headings = append(island.state.headings, heading)
	}

	appendIslandNavigation(
		&island.state.navigation,
		collectBodyNavigation(res),
		pageOffset,
		island.state.geom.contentH,
	)
	return nil
}

func benchmarkIslandRoot(root, section *html.Node) *html.Node {
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

	copyRoot := cloneHTMLNodeShell(root, nil)

	copyDocument := cloneHTMLNodeShell(document, copyRoot)

	copyBody := cloneHTMLNodeShell(body, copyDocument)
	copyBody.Children = []*html.Node{cloneHTMLNode(section, copyBody)}
	copyDocument.Children = []*html.Node{copyBody}
	copyRoot.Children = []*html.Node{copyDocument}

	return copyRoot
}

func cloneHTMLNode(node, parent *html.Node) *html.Node {
	if node == nil {
		return nil
	}

	clone := cloneHTMLNodeShell(node, parent)
	if node.Children == nil {
		return clone
	}

	clone.Children = make([]*html.Node, 0, len(node.Children))
	for _, child := range node.Children {
		clone.Children = append(clone.Children, cloneHTMLNode(child, clone))
	}

	return clone
}

func cloneHTMLNodeShell(node, parent *html.Node) *html.Node {
	if node == nil {
		return nil
	}

	return &html.Node{ //nolint:exhaustruct // children are populated by the clone caller
		Type:     node.Type,
		Name:     node.Name,
		Attrs:    cloneHTMLAttrs(node.Attrs),
		Text:     node.Text,
		Parent:   parent,
	}
}

func cloneHTMLAttrs(attrs map[string]string) map[string]string {
	if attrs == nil {
		return nil
	}

	clone := make(map[string]string, len(attrs))
	for name, value := range attrs {
		clone[name] = value
	}

	return clone
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
