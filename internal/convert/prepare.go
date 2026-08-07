package convert

import (
	"context"
	"errors"
	"fmt"
	"io"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/load"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/settings"
)

// ResourceContext binds the policy and base URL for one loaded document.
//
// A document's CSS, images and fonts must all resolve through the same loader,
// base URL and per-page load settings. Keeping those three values together
// prevents callers from accidentally applying the primary-page policy to one
// resource kind and a different policy to another. The context is small and
// value-like; callers may retain it for the lifetime of a prepared document.
type ResourceContext struct {
	Loader *load.Loader
	Base   string
	Load   settings.LoadPage
}

// errNoResourceLoader reports a ResourceContext without a bound loader.
var errNoResourceLoader = errors.New("convert: resource context has no loader")

// errNilLoader reports a nil loader at the document-prep boundary.
var errNilLoader = errors.New("convert: nil loader")

// NewResourceContext creates the resource seam shared by PDF and image
// preparation. The loader owns global ACL/network policy; Load carries the
// per-document credentials, headers, cookies and error policy.
func NewResourceContext(loader *load.Loader, base string, lp settings.LoadPage) ResourceContext {
	return ResourceContext{Loader: loader, Base: base, Load: lp}
}

// Fetch resolves and loads a document-relative subresource using the bound
// base URL and load policy.
func (r ResourceContext) Fetch(ctx context.Context, ref string) (*load.Resource, error) { //nolint:contextcheck,lll // nil-ctx guard falls back to a fresh context
	if r.Loader == nil {
		return nil, errNoResourceLoader
	}

	if ctx == nil {
		ctx = context.Background()
	}

	res, err := r.Loader.FetchSub(ctx, r.Base, ref, r.Load)
	if err != nil {
		return nil, fmt.Errorf("fetch %q: %w", ref, err)
	}

	return res, nil
}

// CollectSheets gathers stylesheets through this document's resource policy.
// The implementation remains CollectSheets so existing callers and the
// fix-contract's shared gatherer stay compatible.
func (r ResourceContext) CollectSheets(ctx context.Context, root *html.Node, opts SheetOptions, log io.Writer) []*css.Stylesheet { //nolint:lll // resource seam signature
	if r.Loader == nil {
		return nil
	}

	return CollectSheets(ctx, r.Loader, root, r.Base, r.Load, opts, log)
}

// MergeFontFaces loads @font-face resources through this document's policy.
func (r ResourceContext) MergeFontFaces(ctx context.Context, registry *pdf.Registry, sheets []*css.Stylesheet, idx int, log io.Writer) *pdf.Registry { //nolint:lll // resource seam signature
	if r.Loader == nil {
		return registry
	}

	return MergeFontFaces(ctx, r.Loader, registry, sheets, r.Base, r.Load, idx, log)
}

// PrepareOptions controls the document preparation shared by PDF and image
// pipelines. Viewport values are points because CSS media feature matching is
// performed in the layout coordinate system.
type PrepareOptions struct {
	ViewportW       float64
	ViewportH       float64
	MediaType       string
	ObjectIndex     int
	SimplifyDOM     bool
	SimplifyProfile string
}

// PreparedDocument is the output of the shared load/parse/resource phase.
// It deliberately contains no layout or paint state, so both output modes can
// consume the same source document and resource decisions.
type PreparedDocument struct {
	Resource  *load.Resource
	Root      *html.Node
	Resources ResourceContext
	Sheets    []*css.Stylesheet
	Registry  *pdf.Registry
}

// PrepareDocument loads and parses one document, gathers its stylesheets and
// merges its @font-face resources. A skipped resource is returned with a nil
// Root and no error so the caller can apply its mode-specific skip policy.
// All subresources are bound to one ResourceContext before any CSS or font
// work begins.
func PrepareDocument(ctx context.Context, loader *load.Loader, page string, loadPage settings.LoadPage, registry *pdf.Registry, opts PrepareOptions, log io.Writer) (*PreparedDocument, error) { //nolint:contextcheck,lll // nil-ctx guard falls back to a fresh context
	if loader == nil {
		return nil, errNilLoader
	}

	if ctx == nil {
		ctx = context.Background()
	}

	res, err := loader.Load(ctx, page, loadPage)
	if err != nil {
		return nil, fmt.Errorf("load %q: %w", page, err)
	}

	prep := &PreparedDocument{ //nolint:exhaustruct // intentional zero-value fields
		Resource:  res,
		Resources: NewResourceContext(loader, res.Base, loadPage),
		Registry:  registry,
	}
	if res.Skip {
		return prep, nil
	}

	root, err := html.ParseDocument(res.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	prep.Root = root
	prep.Sheets = prep.Resources.CollectSheets(ctx, root, SheetOptions{
		ViewportW:   opts.ViewportW,
		ViewportH:   opts.ViewportH,
		MediaType:   opts.MediaType,
		ObjectIndex: opts.ObjectIndex,
	}, log)
	prep.Sheets = AppendSimplifySheet(prep.Sheets, opts.SimplifyDOM, opts.SimplifyProfile)
	prep.Registry = prep.Resources.MergeFontFaces(ctx, registry, prep.Sheets, opts.ObjectIndex, log)

	return prep, nil
}
