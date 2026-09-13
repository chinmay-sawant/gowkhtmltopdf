package convert

import (
	"context"
	"fmt"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

func renderIndependentBlocks(
	ctx context.Context,
	doc *pdf.Document,
	state *objectState,
	root *html.Node,
	blocks []*html.Node,
	render objectRenderContext,
) error {
	opts := state.bodyLayoutOpts(render)
	styles, err := layout.ResolveStyles(ctx, root, opts)

	if err != nil {
		return fmt.Errorf("independent block styles: %w", err)
	}

	start := doc.PageCount()
	workspace := &layout.Workspace{}

	for _, section := range blocks {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("independent block assembly: %w", err)
		}

		res, err := layout.NodeWithWorkspace(ctx, section, opts, styles, workspace)
		if err != nil {
			return fmt.Errorf("layout independent block: %w", err)
		}

		if err := applyBodyPolicies(ctx, state, render, res); err != nil {
			workspace.Release(res)

			return fmt.Errorf("policy independent block: %w", err)
		}

		if err := paintIndependentResult(ctx, doc, state, render, section, res, start); err != nil {
			workspace.Release(res)

			return err
		}

		workspace.Release(res)
	}

	state.pages = doc.PageCount() - start
	state.offset = start

	return nil
}

func paintIndependentResult(
	ctx context.Context,
	doc *pdf.Document,
	state *objectState,
	render objectRenderContext,
	section *html.Node,
	res *layout.Result,
	start int,
) error {
	before := doc.PageCount()

	if err := layout.PaintContext(ctx, doc, res, paintOptions(state.geom)); err != nil {
		return fmt.Errorf("paint independent block: %w", err)
	}

	pageOffset := before - start
	state.geom.pageNames = mergePageNames(
		state.geom.pageNames,
		layout.PageNames(res, state.geom.contentH),
		pageOffset,
	)

	for _, heading := range collectObjectHeadings(section, res, *render.obj) {
		heading.Page += pageOffset
		state.headings = append(state.headings, heading)
	}

	appendBlockNavigation(
		&state.navigation,
		collectBodyNavigation(res),
		pageOffset,
		state.geom.contentH,
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

func appendBlockNavigation(dst *bodyNavigation, src bodyNavigation, pageOffset int, contentH float64) {
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
