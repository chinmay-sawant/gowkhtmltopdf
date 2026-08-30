package convert

import (
	"context"
	"fmt"
	"io"
	"runtime/debug"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/islands"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

const (
	benchmarkFixtureMarker = "report.html.tmpl: paginated benchmark report"
	htmlSectionName        = "section"
	islandBreakOverrideCSS = ".benchmark-page { page-break-before: auto !important; }"
	islandMemoryTrimEvery  = 4
)

// pageIslandPlan preserves the convert package's private benchmark shape;
// recognition and virtual-view construction live in internal/convert/islands.
type pageIslandPlan struct {
	sections []*html.Node
}

// benchmarkPageIslandPlan recognizes only the repository's generated report
// fixture: its source marker, title, and body must be a whitespace-separated
// sequence of section.benchmark-page elements. It is called only for the
// explicit benchmark request; normal requests never inspect HTML for this
// marker. Anything else fails closed to generic rendering.
func benchmarkPageIslandPlan(root *html.Node) (pageIslandPlan, bool) {
	plan, ok := islands.BenchmarkPlan(root)
	if !ok {
		return pageIslandPlan{}, false //nolint:exhaustruct // no certified islands
	}

	return pageIslandPlan{sections: plan.Sections}, true
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
		doc:       doc,
		state:     state,
		root:      root,
		start:     start,
		sheets:    islandSheets,
		renderCtx: render,
		workspace: workspace,
		log:       log,
	}

	for index, section := range plan.sections {
		if err := island.render(ctx, section); err != nil {
			return err
		}

		plan.sections[index] = nil

		if (index+1)%islandMemoryTrimEvery == 0 {
			debug.FreeOSMemory()
		}
	}

	state.pages = doc.PageCount() - start
	state.offset = start

	return nil
}

type pageIslandRenderContext struct {
	doc       *pdf.Document
	state     *objectState
	root      *html.Node
	start     int
	sheets    []*css.Stylesheet
	renderCtx objectRenderContext
	workspace *layout.Workspace
	log       io.Writer
}

func (island pageIslandRenderContext) render(ctx context.Context, section *html.Node) error {
	islandRoot := benchmarkIslandRoot(island.root, section)

	res, err := layout.WithWorkspace(ctx, islandRoot, island.state.bodyLayoutOpts(
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

	if err := layout.PaintContext(ctx, island.doc, res, paintOptions(island.state.geom)); err != nil {
		return fmt.Errorf("paint certified page island: %w", err)
	}

	pageOffset := before - island.start
	island.state.geom.pageNames = mergePageNames(
		island.state.geom.pageNames,
		layout.PageNames(res, island.state.geom.contentH),
		pageOffset,
	)

	for _, heading := range collectObjectHeadings(
		islandRoot,
		res,
		*island.renderCtx.obj,
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

func mergePageNames(dst []string, src []string, offset int) []string {
	if len(src) == 0 || offset < 0 {
		return dst
	}

	need := offset + len(src)
	if len(dst) < need {
		dst = append(dst, make([]string, need-len(dst))...)
	}

	copy(dst[offset:], src)

	return dst
}

func benchmarkIslandRoot(root, section *html.Node) *html.Node {
	return islands.Root(root, section)
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
