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

// NewResourceContext creates the resource seam shared by PDF and image
// preparation. The loader owns global ACL/network policy; Load carries the
// per-document credentials, headers, cookies and error policy.
func NewResourceContext(loader *load.Loader, base string, lp settings.LoadPage) ResourceContext {
	return ResourceContext{Loader: loader, Base: base, Load: lp}
}

// Fetch resolves and loads a document-relative subresource using the bound
// base URL and load policy.
func (r ResourceContext) Fetch(ctx context.Context, ref string) (*load.Resource, error) {
	if r.Loader == nil {
		return nil, errors.New("convert: resource context has no loader")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	return r.Loader.FetchSub(ctx, r.Base, ref, r.Load)
}

// CollectSheets gathers stylesheets through this document's resource policy.
// The implementation remains CollectSheets so existing callers and the
// fix-contract's shared gatherer stay compatible.
func (r ResourceContext) CollectSheets(ctx context.Context, root *html.Node, opts SheetOptions, log io.Writer) []*css.Stylesheet {
	if r.Loader == nil {
		return nil
	}

	return CollectSheets(ctx, r.Loader, root, r.Base, r.Load, opts, log)
}

// MergeFontFaces loads @font-face resources through this document's policy.
func (r ResourceContext) MergeFontFaces(ctx context.Context, registry *pdf.Registry, sheets []*css.Stylesheet, idx int, log io.Writer) *pdf.Registry {
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
func PrepareDocument(ctx context.Context, loader *load.Loader, page string, lp settings.LoadPage, registry *pdf.Registry, opts PrepareOptions, log io.Writer) (*PreparedDocument, error) {
	if loader == nil {
		return nil, errors.New("convert: nil loader")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	res, err := loader.Load(ctx, page, lp)
	if err != nil {
		return nil, fmt.Errorf("load %q: %w", page, err)
	}

	prep := &PreparedDocument{
		Resource:  res,
		Resources: NewResourceContext(loader, res.Base, lp),
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
