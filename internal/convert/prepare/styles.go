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
	// PageBoxViewport, when set, replaces the gating viewport after inline
	// <style> sheets are parsed. Callers use it to apply inline @page geometry
	// before linked and imported media queries are gated, so size features
	// see the final page box the cascade will use. nil keeps ViewportW/H.
	PageBoxViewport func(inline []*css.Stylesheet) (width, height float64)
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
	// inlineSheets / inlineByNode cache the <style> parses from the gating
	// pre-pass so the main walk does not parse inline styles twice.
	inlineSheets []*css.Stylesheet
	inlineByNode map[*html.Node]*css.Stylesheet
}

//nolint:wsl,lll // stylesheet collection flow
func collectSheets(ctx context.Context, resources load.ResourceContext, root *html.Node, opts SheetOptions, log io.Writer) ([]*css.Stylesheet, error) {
	collector := sheetCollector{ //nolint:exhaustruct // empty result state
		resources: resources, opts: opts, log: log, seen: make(map[string]struct{}),
	}
	if root != nil {
		if collector.opts.PageBoxViewport != nil {
			collector.preparseInlineStyles(ctx, root)

			if collector.err == nil {
				collector.opts.ViewportW, collector.opts.ViewportH = collector.opts.PageBoxViewport(collector.inlineSheets)
			}
		}

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

// preparseInlineStyles parses inline <style> sheets once so the caller can
// derive the stylesheet-gating viewport from inline @page geometry before any
// link or @import media query is evaluated. Invalid sheets are skipped here
// and reported once by the main walk.
func (collector *sheetCollector) preparseInlineStyles(ctx context.Context, root *html.Node) {
	if root == nil {
		return
	}

	collector.inlineByNode = make(map[*html.Node]*css.Stylesheet)

	var visits uint32

	root.Walk(func(node *html.Node) {
		if collector.err != nil || node == nil || node.Type != html.ElementNode || node.Name != "style" {
			return
		}

		visits++
		if visits&63 == 0 {
			if err := ctx.Err(); err != nil {
				collector.err = err

				return
			}
		}

		sheet, err := css.Parse(styleText(node))
		if err != nil {
			return
		}

		collector.inlineByNode[node] = sheet
		collector.inlineSheets = append(collector.inlineSheets, sheet)
	})
}

//nolint:nlreturn // collector traversal flow
func (collector *sheetCollector) collectStyle(ctx context.Context, node *html.Node) {
	if collector.err != nil {
		return
	}

	sheet := collector.inlineByNode[node]
	if sheet == nil {
		parsed, err := css.Parse(styleText(node))
		if err != nil {
			collector.warn("skipping <style>: %v", err)
			return
		}

		sheet = parsed
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

	// Fetch is bounded per request by the loader's timeout policy
	// (LoadPage.Timeout, or load.DefaultResponseTimeout when unset); ctx
	// carries the caller's overall deadline and cancellation.
	resource, err := collector.resources.Fetch(ctx, node.Attribute("href"))
	if err != nil {
		collector.warn("skipping <link href=%q>: %v", node.Attribute("href"), err)
		return
	}

	sheet, err := css.ParseBytes(resource.Body)
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
// base is the sheet's source URL. Imports resolve against it, and the sheet's
// own url() values are absolutized against it before the sheet is added, so
// font and image consumers stay base-agnostic. <base href> is deliberately
// ignored (see css.Stylesheet.ResolveURLs).
//
//nolint:wsl // collector import flow
func (collector *sheetCollector) addWithImports(ctx context.Context, sheet *css.Stylesheet, base string, depth int) {
	if sheet == nil || collector.err != nil {
		return
	}
	collector.fetchImports(ctx, sheet, base, depth)
	if collector.err != nil {
		return
	}
	sheet.ResolveURLs(base)
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

func (collector *sheetCollector) fetchOneImport(ctx context.Context, rule css.ImportRule, base string, depth int) {
	if collector.err != nil {
		return
	}

	if err := ctx.Err(); err != nil {
		collector.err = err

		return
	}

	ref := collector.prepareImportRef(rule, base)
	if ref == "" {
		return
	}

	sheet, sheetBase := collector.loadImportedSheet(ctx, base, ref)
	if sheet == nil {
		return
	}

	collector.addWithImports(ctx, sheet, sheetBase, depth+1)
}

func (collector *sheetCollector) prepareImportRef(rule css.ImportRule, base string) string {
	ref := importRef(rule.URL)
	if ref == "" {
		return ""
	}

	if rule.Media != "" && !css.MediaMatches(
		rule.Media,
		collector.opts.MediaType,
		collector.opts.ViewportW,
		collector.opts.ViewportH,
	) {
		return ""
	}

	resolved := resolvedRef(base, ref)
	if collector.seenURL(resolved) {
		return ""
	}

	collector.noteSeen(resolved)

	return ref
}

func (collector *sheetCollector) loadImportedSheet(ctx context.Context, base, ref string) (*css.Stylesheet, string) {
	resource, err := collector.fetchRef(ctx, base, ref)
	if err != nil {
		collector.warn("skipping @import %q: %v", ref, err)

		return nil, ""
	}

	if resource == nil || resource.Skip {
		collector.warn("skipping @import %q: resource skipped", ref)

		return nil, ""
	}

	collector.noteSeen(resource.URL)

	sheet, err := css.ParseBytes(resource.Body)
	if err != nil {
		collector.warn("skipping @import %q: %v", ref, err)

		return nil, ""
	}

	return sheet, resourceBase(resource)
}

// fetchRef loads ref with the same ACL as <link rel=stylesheet>. Relative
// imports resolve against the current sheet base, not the document base.
func (collector *sheetCollector) fetchRef(ctx context.Context, base, ref string) (*load.Resource, error) {
	resources := collector.resources
	if base != "" && base != resources.Base() {
		if loader := resources.Loader(); loader != nil {
			resources = loader.ForResource(&load.Resource{Base: base}, resources.PageLoad()) //nolint:exhaustruct,lll // base-only resource reference
		}
	}

	resource, err := resources.Fetch(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("fetch imported stylesheet %q: %w", ref, err)
	}

	return resource, nil
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
	spec := fontFaceSpec(face)

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
			registry.AddFamilyAliasSpec(face.Family, font, spec)
		}
	}
	return registry
}

// fontFaceSpec translates parsed @font-face descriptors into the selection
// metadata the PDF registry keeps. Zero values mean "no descriptor": the face
// keeps its file-declared weight/style and covers every code point.
func fontFaceSpec(face css.FontFace) pdf.FaceSpec {
	ranges := make([]pdf.UnicodeRange, 0, len(face.UnicodeRanges))

	for _, span := range face.UnicodeRanges {
		ranges = append(ranges, pdf.UnicodeRange{Lo: span.Lo, Hi: span.Hi})
	}

	return pdf.FaceSpec{
		Weight:   face.Weight,
		Italic:   face.Italic,
		StyleSet: face.StyleSet,
		Ranges:   ranges,
	}
}

func fetchFontFace(
	ctx context.Context,
	resources load.ResourceContext,
	uri string,
	idx int,
	log io.Writer,
) (*pdf.Font, bool) {
	if strings.HasPrefix(strings.ToLower(uri), "data:") {
		return fetchDataFontFace(ctx, resources, uri, idx, log)
	}

	if path := fontURIPath(uri); strings.HasSuffix(path, ".eot") {
		line.Emit(log, line.Warn,
			"object %d: @font-face src %q skipped (EOT unsupported; WOFF2/WOFF1/TTF/OTF only)", idx, uri)

		return nil, false
	}

	font, err := fetchAndParseFont(ctx, resources, uri)
	if err != nil {
		line.Emit(log, line.Warn, "object %d: @font-face src %q: %v", idx, uri, err)

		return nil, false
	}

	return font, true
}

// fetchDataFontFace decodes and registers a data: font src when the payload is
// a supported format. Warnings never carry the payload: scheme, media type,
// and URI length only.
func fetchDataFontFace(
	ctx context.Context,
	resources load.ResourceContext,
	uri string,
	idx int,
	log io.Writer,
) (*pdf.Font, bool) {
	font, err := fetchAndParseFont(ctx, resources, uri)
	if err != nil {
		line.Emit(log, line.Warn, "object %d: @font-face data: src skipped (%s, %d chars)",
			idx, dataURIMeta(uri), len(uri))

		return nil, false
	}

	return font, true
}

// fetchAndParseFont fetches uri and parses the body as TTF/OTF, WOFF1, or WOFF2.
func fetchAndParseFont(ctx context.Context, resources load.ResourceContext, uri string) (*pdf.Font, error) {
	resource, err := resources.Fetch(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("fetch font %q: %w", uri, err)
	}

	font, err := pdf.ParseFontBytes(resource.Body)
	if err != nil {
		return nil, fmt.Errorf("parse font %q: %w", uri, err)
	}

	return font, nil
}

// fontURIPath is the lowercased URL path of a font src. Format policy ignores
// query and fragment so `woff2?v=1` is still a WOFF2 file and `ttf?x` is not
// misclassified.
func fontURIPath(uri string) string {
	parsed, err := url.Parse(uri)
	if err != nil {
		return strings.ToLower(uri)
	}

	return strings.ToLower(parsed.Path)
}

// dataURIMeta returns a data URI's header (scheme and media type, before the
// payload comma), capped so a base64 body never reaches a log line.
func dataURIMeta(uri string) string {
	const metaCap = 64

	meta := uri
	if i := strings.IndexByte(meta, ','); i >= 0 {
		meta = meta[:i]
	}

	if len(meta) > metaCap {
		meta = meta[:metaCap] + "..."
	}

	return meta
}
