package prepare

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/line"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

const (
	maxStylesheetRules = 1_000_000
	maxImportDepth     = 8
)

var errStylesheetLimit = errors.New("convert: stylesheet rule limit exceeded")

// SheetOptions configures stylesheet viewport/media gating and warning labels.
type SheetOptions struct {
	ViewportW, ViewportH float64
	MediaType            string
	ObjectIndex          int
}

// CollectSheets gathers inline and linked stylesheets in document order.
//
//nolint:lll // stylesheet collection flow
func CollectSheets(ctx context.Context, loader *load.Loader, root *html.Node, base string, loadPage settings.LoadPage, opts SheetOptions, log io.Writer) []*css.Stylesheet {
	if loader == nil {
		return nil
	}
	if ctx == nil { //nolint:wsl // nil-context warning is a separate preflight branch.
		if log != nil {
			line.Emit(log, line.Warn, "stylesheet collection: nil context")
		}

		return nil
	}

	resources := loader.ForResource(&load.Resource{Base: base}, loadPage) //nolint:exhaustruct,lll // base-only resource reference
	sheets, err := collectSheets(ctx, resources, root, opts, log)

	if err != nil && log != nil {
		line.Emit(log, line.Warn, "stylesheet collection: %v", err)
	}

	return sheets
}

type sheetCollector struct {
	resources load.ResourceContext
	opts      SheetOptions
	log       io.Writer
	sheets    []*css.Stylesheet
	seen      map[string]struct{}
	rules     int
	visits    uint32
	err       error
}

//nolint:wsl,lll // stylesheet collection flow
func collectSheets(ctx context.Context, resources load.ResourceContext, root *html.Node, opts SheetOptions, log io.Writer) ([]*css.Stylesheet, error) {
	collector := sheetCollector{ //nolint:exhaustruct // empty result state
		resources: resources, opts: opts, log: log, seen: make(map[string]struct{}),
	}
	if root != nil {
		root.Walk(func(node *html.Node) { collector.visit(ctx, node) })
	}

	const softRuleWarn = 25_000

	if collector.rules >= softRuleWarn {
		collector.warn("large stylesheet volume (%d rules); print may be slow", collector.rules)
	}
	if collector.err != nil {
		return collector.sheets, collector.err
	}

	return collector.sheets, nil
}

func (collector *sheetCollector) visit(ctx context.Context, node *html.Node) {
	if collector.err != nil || node == nil || node.Type != html.ElementNode {
		return
	}

	collector.visits++
	if collector.visits&63 == 0 {
		if err := ctx.Err(); err != nil {
			collector.err = err

			return
		}
	}

	switch node.Name {
	case "style":
		collector.collectStyle(ctx, node)
	case "link":
		collector.collectLink(ctx, node)
	}
}

//nolint:wsl,nlreturn // collector traversal flow
func (collector *sheetCollector) collectStyle(ctx context.Context, node *html.Node) {
	if collector.err != nil {
		return
	}

	sheet, err := css.Parse(styleText(node))
	if err != nil {
		collector.warn("skipping <style>: %v", err)
		return
	}
	collector.addWithImports(ctx, sheet, collector.resources.Base(), 0)
}

//nolint:wsl,nlreturn // collector traversal flow
func (collector *sheetCollector) collectLink(ctx context.Context, node *html.Node) {
	if collector.err != nil {
		return
	}
	if err := ctx.Err(); err != nil {
		collector.err = err

		return
	}

	if !linkStylesheet(node, collector.opts.ViewportW, collector.opts.ViewportH, collector.opts.MediaType) {
		return
	}

	resource, err := collector.resources.Fetch(ctx, node.Attribute("href"))
	if err != nil {
		collector.warn("skipping <link href=%q>: %v", node.Attribute("href"), err)
		return
	}

	sheet, err := css.Parse(string(resource.Body))
	if err != nil {
		collector.warn("skipping <link href=%q>: %v", node.Attribute("href"), err)
		return
	}
	collector.noteSeen(resource.URL)
	collector.addWithImports(ctx, sheet, resourceBase(resource), 0)
}

// addWithImports appends imported sheets (recursively) before sheet so @import
// rules precede the importer, matching CSS cascade order.
//
//nolint:wsl,nlreturn // collector import flow
func (collector *sheetCollector) addWithImports(ctx context.Context, sheet *css.Stylesheet, base string, depth int) {
	if sheet == nil || collector.err != nil {
		return
	}
	collector.fetchImports(ctx, sheet, base, depth)
	if collector.err != nil {
		return
	}
	collector.add(sheet)
}

//nolint:wsl,nlreturn // collector import flow
func (collector *sheetCollector) fetchImports(ctx context.Context, sheet *css.Stylesheet, base string, depth int) {
	if collector.err != nil || sheet == nil || len(sheet.Imports) == 0 {
		return
	}
	if depth >= maxImportDepth {
		collector.warn("skipping nested @import: depth exceeds %d", maxImportDepth)
		return
	}
	for _, rule := range sheet.Imports {
		collector.fetchOneImport(ctx, rule, base, depth)
	}
}

//nolint:wsl,nlreturn,lll // collector import flow
func (collector *sheetCollector) fetchOneImport(ctx context.Context, rule css.ImportRule, base string, depth int) {
	if collector.err != nil {
		return
	}
	if err := ctx.Err(); err != nil {
		collector.err = err
		return
	}

	ref := importRef(rule.URL)
	if ref == "" {
		return
	}
	if rule.Media != "" && !css.MediaMatches(rule.Media, collector.opts.MediaType, collector.opts.ViewportW, collector.opts.ViewportH) {
		return
	}

	resolved := resolvedRef(base, ref)
	if collector.seenURL(resolved) {
		return
	}
	collector.noteSeen(resolved)

	resource, err := collector.fetchRef(ctx, base, ref)
	if err != nil {
		collector.warn("skipping @import %q: %v", ref, err)
		return
	}
	if resource == nil || resource.Skip {
		collector.warn("skipping @import %q: resource skipped", ref)
		return
	}
	collector.noteSeen(resource.URL)

	sheet, err := css.Parse(string(resource.Body))
	if err != nil {
		collector.warn("skipping @import %q: %v", ref, err)
		return
	}
	collector.addWithImports(ctx, sheet, resourceBase(resource), depth+1)
}

// fetchRef loads ref with the same ACL as <link rel=stylesheet>. Relative
// imports resolve against the current sheet base, not the document base.
//
//nolint:wsl,nlreturn // collector import flow
func (collector *sheetCollector) fetchRef(ctx context.Context, base, ref string) (*load.Resource, error) {
	resources := collector.resources
	if base != "" && base != resources.Base() {
		if loader := resources.Loader(); loader != nil {
			resources = loader.ForResource(&load.Resource{Base: base}, resources.PageLoad()) //nolint:exhaustruct,lll // base-only resource reference
		}
	}

	return resources.Fetch(ctx, ref)
}

func (collector *sheetCollector) seenURL(raw string) bool {
	if raw == "" || collector.seen == nil {
		return false
	}
	_, ok := collector.seen[raw]

	return ok
}

func (collector *sheetCollector) noteSeen(raw string) {
	if raw == "" {
		return
	}
	if collector.seen == nil {
		collector.seen = make(map[string]struct{})
	}

	collector.seen[raw] = struct{}{}
}

func resourceBase(resource *load.Resource) string {
	if resource == nil {
		return ""
	}
	if resource.Base != "" {
		return resource.Base
	}

	return resource.URL
}

func importRef(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) >= 4 && strings.EqualFold(raw[:4], "url(") && strings.HasSuffix(raw, ")") {
		raw = strings.TrimSpace(raw[4 : len(raw)-1])
	}

	return strings.Trim(raw, `"' `)
}

func resolvedRef(base, ref string) string {
	parsed, err := url.Parse(strings.TrimSpace(ref))
	if err != nil {
		return strings.TrimSpace(ref)
	}
	if parsed.IsAbs() || strings.TrimSpace(base) == "" {
		return parsed.String()
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return parsed.String()
	}

	return baseURL.ResolveReference(parsed).String()
}

//nolint:wsl,nlreturn // collector accumulation flow
func (collector *sheetCollector) add(sheet *css.Stylesheet) {
	if sheet == nil {
		return
	}

	collector.rules += len(sheet.Rules)
	if collector.rules > maxStylesheetRules {
		collector.err = fmt.Errorf("%w: got %d, limit %d", errStylesheetLimit, collector.rules, maxStylesheetRules)
		return
	}
	collector.sheets = append(collector.sheets, sheet)
}

//nolint:wsl,nlreturn // warning routing flow
func (collector *sheetCollector) warn(format string, args ...any) {
	if collector.log == nil {
		return
	}
	if collector.opts.ObjectIndex > 0 {
		line.Emit(collector.log, line.Warn, "object %d: "+format, append([]any{collector.opts.ObjectIndex}, args...)...)
		return
	}
	line.Emit(collector.log, line.Warn, format, args...)
}

//nolint:wsl,nlreturn // text extraction flow
func styleText(node *html.Node) string {
	var out strings.Builder
	for _, child := range node.Children {
		if child.Type == html.TextNode {
			out.WriteString(child.Text)
		}
	}
	return out.String()
}

//nolint:wsl,nlreturn // media predicate
func linkStylesheet(node *html.Node, viewportW, viewportH float64, mediaType string) bool {
	if node.Name != "link" || node.Attribute("href") == "" {
		return false
	}
	if !strings.Contains(strings.ToLower(node.Attribute("rel")), "stylesheet") {
		return false
	}
	media := node.Attribute("media")
	return media == "" || css.MediaMatches(media, mediaType, viewportW, viewportH)
}

// LinkStylesheet retains the stylesheet media predicate as a small seam for
// the conversion package's white-box compatibility tests.
func LinkStylesheet(node *html.Node, viewportW, viewportH float64, mediaType string) bool {
	return linkStylesheet(node, viewportW, viewportH, mediaType)
}

// MergeFontFaces loads supported @font-face sources into registry.
//
//nolint:wsl,lll // font-face collection flow
func MergeFontFaces(ctx context.Context, loader *load.Loader, registry *pdf.Registry, sheets []*css.Stylesheet, base string, loadPage settings.LoadPage, idx int, log io.Writer) *pdf.Registry {
	if loader == nil {
		return registry
	}
	resources := loader.ForResource(&load.Resource{Base: base}, loadPage) //nolint:exhaustruct,lll // base-only resource reference

	return mergeFontFaces(ctx, resources, registry, sheets, idx, log)
}

//nolint:wsl,nlreturn,lll // font-face collection flow
func mergeFontFaces(ctx context.Context, resources load.ResourceContext, registry *pdf.Registry, sheets []*css.Stylesheet, idx int, log io.Writer) *pdf.Registry {
	for _, sheet := range sheets {
		if ctx != nil && ctx.Err() != nil {
			return registry
		}
		if sheet == nil {
			continue
		}
		for _, face := range sheet.FontFaces {
			if ctx != nil && ctx.Err() != nil {
				return registry
			}
			registry = mergeFontFace(ctx, resources, registry, face, idx, log)
		}
	}
	return registry
}

//nolint:wsl,nlreturn,lll // font-face collection flow
func mergeFontFace(ctx context.Context, resources load.ResourceContext, registry *pdf.Registry, face css.FontFace, idx int, log io.Writer) *pdf.Registry {
	for _, uri := range css.FontFaceURLs(face.Src) {
		font, ok := fetchFontFace(ctx, resources, uri, idx, log)
		if !ok {
			continue
		}
		if face.Family != "" {
			font.PostScriptName = strings.ReplaceAll(face.Family, " ", "")
		}
		if registry == nil {
			registry = pdf.NewRegistry()
		}
		registry.AddFont(font)
		if face.Family != "" {
			registry.AddFamilyAlias(face.Family, font)
		}
	}
	return registry
}

//nolint:wsl,nlreturn,lll // font-face collection flow
func fetchFontFace(ctx context.Context, resources load.ResourceContext, uri string, idx int, log io.Writer) (*pdf.Font, bool) {
	lower := strings.ToLower(uri)
	if strings.HasSuffix(lower, ".woff2") || strings.HasSuffix(lower, ".eot") {
		line.Emit(log, line.Warn,
			"object %d: @font-face src %q skipped (WOFF2/EOT unsupported; WOFF1/TTF/OTF only)", idx, uri)
		return nil, false
	}
	if strings.HasPrefix(lower, "data:") {
		line.Emit(log, line.Warn, "object %d: @font-face data: src skipped", idx, uri)
		return nil, false
	}

	resource, err := resources.Fetch(ctx, uri)
	if err != nil {
		line.Emit(log, line.Warn, "object %d: @font-face src %q: %v", idx, uri, err)
		return nil, false
	}
	font, err := pdf.ParseFontBytes(resource.Body)
	if err != nil {
		line.Emit(log, line.Warn, "object %d: @font-face src %q: %v", idx, uri, err)
		return nil, false
	}
	return font, true
}
