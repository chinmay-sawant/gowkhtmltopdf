package convert

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/blinkengine"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/prepare"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/line"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// blinkImportDepth caps the @import chain the blinkless bridge fetches,
// mirroring the native collector's cap.
const blinkImportDepth = 8

// paintBlinklessBody renders one body object through the blinkless engine.
// It is the blinkless counterpart of the renderObject tail: adapt the drawing
// list, apply the body policies, paint through the shared paginator and PDF
// painter, and record the page bookkeeping later passes need. The native
// pipeline is not touched: this runs only when the request selects
// settings.EngineBlinkless.
//
// Headings and navigation stay empty: a blinkless result carries no element
// box tree, so the outline and TOC cover only objects rendered by the native
// engine. Headers, footers, link annotations, page plans, and copies still
// apply because those passes read the painted document and this result's
// link operations, not the box tree.
func (run *runContext) paintBlinklessBody(
	ctx context.Context,
	state *objectState,
	render objectRenderContext,
	prep *prepare.Prepared,
) error {
	media := state.media
	if media == "" {
		media = mediaPrint
	}

	res, err := blinkengine.Layout(ctx, prep.Resource.Body, blinkengine.Options{
		WidthPt:  state.geom.contentW,
		HeightPt: state.geom.contentH,
		Media:    media,
		Sheets:   run.externalSheets(ctx, prep, state, media),
		Images:   state.imagesFn,
	})
	if err != nil {
		return fmt.Errorf("blinkless: %w", err)
	}

	if err := applyBodyPolicies(ctx, state, render, res); err != nil {
		return fmt.Errorf("body policy: %w", err)
	}

	before := run.doc.PageCount()

	if err := layout.PaintContext(ctx, run.doc, res, paintOptions(state.geom)); err != nil {
		return fmt.Errorf("paint: %w", err)
	}

	state.geom.pageNames = layout.PageNames(res, state.geom.contentH)
	state.navigation = collectBodyNavigation(res)
	state.pages = run.doc.PageCount() - before
	state.offset = before

	return nil
}

// externalSheets fetches the document's <link rel=stylesheet> sheets with the
// same loader policy the native collector uses and returns their CSS text in
// cascade order: each sheet's @imports first, then the sheet. Blinkless
// applies <style> blocks itself but cannot resolve relative <link> hrefs
// (its own loader has no document base), so the bridge passes the link sheets
// in explicitly.
func (run *runContext) externalSheets(
	ctx context.Context,
	prep *prepare.Prepared,
	state *objectState,
	media string,
) []string {
	if prep == nil || prep.Root == nil {
		return nil
	}

	seen := map[string]bool{}

	var sheets []string

	prep.Root.Walk(func(node *html.Node) {
		if !blinkLinkStylesheet(node, media, state.geom.contentW, state.geom.contentH) {
			return
		}

		run.collectLinkedSheet(
			ctx, prep.Resources, state.obj.Load, node.Attribute("href"),
			media, state.geom, run.log, seen, 0, &sheets,
		)
	})

	return sheets
}

// blinkLinkStylesheet mirrors the native collector's linkStylesheet gate: a
// stylesheet link whose media matches the active print media.
func blinkLinkStylesheet(node *html.Node, media string, widthPt, heightPt float64) bool {
	if node.Name != "link" || node.Attribute("href") == "" {
		return false
	}

	if !strings.Contains(strings.ToLower(node.Attribute("rel")), "stylesheet") {
		return false
	}

	linkMedia := node.Attribute("media")

	return linkMedia == "" || css.MediaMatches(linkMedia, media, widthPt, heightPt)
}

// collectLinkedSheet fetches one stylesheet, recurses into its @imports, and
// appends its CSS text. Fetch and parse failures warn and skip, matching the
// native collector; a cycle is stopped by the seen set and the depth cap.
func (run *runContext) collectLinkedSheet(
	ctx context.Context,
	resources prepare.ResourceContext,
	objLoad settings.LoadPage,
	ref string,
	media string,
	geom hfGeom,
	log io.Writer,
	seen map[string]bool,
	depth int,
	out *[]string,
) {
	if depth > blinkImportDepth {
		line.Emit(log, line.Warn, "blinkless: skipping nested @import %q: depth exceeds %d", ref, blinkImportDepth)

		return
	}

	resource, err := resources.Fetch(ctx, ref)
	if err != nil {
		line.Emit(log, line.Warn, "blinkless: skipping stylesheet %q: %v", ref, err)

		return
	}

	if seen[resource.URL] {
		return
	}

	seen[resource.URL] = true

	sheet, err := css.ParseBytes(resource.Body)
	if err != nil {
		line.Emit(log, line.Warn, "blinkless: skipping stylesheet %q: %v", ref, err)

		return
	}

	base := resource.Base
	if base == "" {
		base = resource.URL
	}

	if len(sheet.Imports) > 0 {
		imported := prepare.NewResourceContext(run.loader, base, objLoad)

		for index := range sheet.Imports {
			rule := &sheet.Imports[index]
			if rule.Media != "" && !css.MediaMatches(rule.Media, media, geom.contentW, geom.contentH) {
				continue
			}

			run.collectLinkedSheet(ctx, imported, objLoad, rule.URL, media, geom, log, seen, depth+1, out)
		}
	}

	*out = append(*out, string(resource.Body))
}
