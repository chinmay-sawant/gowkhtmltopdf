package convert

import (
	"context"
	"errors"
	"fmt"
	"io"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/line"
	"gowkhtmltopdf/internal/load"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/settings"
)

// ResourceContext binds the policy and base URL for one loaded document.
//
// A document's CSS, images and fonts must all resolve through the same loader,
// base URL and per-page load settings. Keeping those three values together
// prevents callers from accidentally applying the primary-page policy to one
// resource kind and a different policy to another. The actual fetch state is
// delegated to load.ResourceContext; the compatibility mirrors are retained
// only for existing conversion consumers. The context is value-like; callers
// may retain it for the lifetime of a prepared document.
type ResourceContext struct {
	resource load.ResourceContext

	// These fields remain compatibility mirrors for existing internal
	// consumers. Fetch and the checked helpers use resource, the canonical
	// load seam, instead of maintaining a second fetch policy.
	Loader *load.Loader
	Base   string
	Load   settings.LoadPage
}

// errNoResourceLoader reports a ResourceContext without a bound loader.
var errNoResourceLoader = errors.New("convert: resource context has no loader")

// errNilLoader reports a nil loader at the document-prep boundary.
var errNilLoader = errors.New("convert: nil loader")

// errNilContext reports a nil context at a preparation/resource boundary.
var errNilContext = errors.New("convert: nil context")

// NewResourceContext creates the resource seam shared by PDF and image
// preparation. The loader owns global ACL/network policy; Load carries the
// per-document credentials, headers, cookies and error policy.
func NewResourceContext(loader *load.Loader, base string, lp settings.LoadPage) ResourceContext {
	//nolint:exhaustruct // intentional zero-value resource field
	resources := ResourceContext{Loader: loader, Base: base, Load: lp}
	if loader != nil {
		//nolint:exhaustruct // base-only resource reference
		resources.resource = loader.ForResource(&load.Resource{Base: base}, lp)
	}

	return resources
}

// Fetch resolves and loads a document-relative subresource using the bound
// base URL and load policy.
func (r ResourceContext) Fetch(
	ctx context.Context, ref string,
) (*load.Resource, error) {
	if r.Loader == nil {
		return nil, errNoResourceLoader
	}

	if ctx == nil {
		return nil, errNilContext
	}

	res, err := r.resource.Fetch(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("fetch %q: %w", ref, err)
	}

	return res, nil
}

// CollectSheets gathers stylesheets through this document's resource policy.
// The implementation remains CollectSheets so existing callers and the
// fix-contract's shared gatherer stay compatible.
func (r ResourceContext) CollectSheets(
	ctx context.Context, root *html.Node, opts SheetOptions, log io.Writer,
) []*css.Stylesheet {
	if r.Loader == nil {
		return nil
	}

	if ctx == nil {
		if log != nil {
			line.Emit(log, line.Warn, "stylesheet collection: %v", errNilContext)
		}

		return nil
	}

	sheets, err := collectSheets(ctx, r.resource, root, opts, log)
	if err != nil && log != nil {
		line.Emit(log, line.Warn, "stylesheet collection: %v", err)
	}

	return sheets
}

// MergeFontFaces loads @font-face resources through this document's policy.
func (r ResourceContext) MergeFontFaces(
	ctx context.Context, registry *pdf.Registry, sheets []*css.Stylesheet,
	idx int, log io.Writer,
) *pdf.Registry {
	if r.Loader == nil {
		return registry
	}

	if ctx == nil {
		if log != nil {
			line.Emit(log, line.Warn, "font-face merge: %v", errNilContext)
		}

		return registry
	}

	return mergeFontFaces(ctx, r.resource, registry, sheets, idx, log)
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
func PrepareDocument(
	ctx context.Context,
	loader *load.Loader,
	page string,
	loadPage settings.LoadPage,
	registry *pdf.Registry,
	opts PrepareOptions,
	log io.Writer,
) (*PreparedDocument, error) {
	if loader == nil {
		return nil, errNilLoader
	}

	if ctx == nil {
		return nil, errNilContext
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

	prep.Sheets, err = prep.Resources.collectSheets(ctx, root, SheetOptions{
		ViewportW:   opts.ViewportW,
		ViewportH:   opts.ViewportH,
		MediaType:   opts.MediaType,
		ObjectIndex: opts.ObjectIndex,
	}, log)
	if err != nil {
		return nil, fmt.Errorf("collect stylesheets: %w", err)
	}

	prep.Sheets = AppendSimplifySheet(prep.Sheets, opts.SimplifyDOM, opts.SimplifyProfile)
	prep.Registry = prep.Resources.mergeFontFaces(ctx, registry, prep.Sheets, opts.ObjectIndex, log)

	return prep, nil
}

func (r ResourceContext) collectSheets(
	ctx context.Context, root *html.Node, opts SheetOptions, log io.Writer,
) ([]*css.Stylesheet, error) {
	if r.Loader == nil {
		return nil, errNoResourceLoader
	}

	if ctx == nil {
		return nil, errNilContext
	}

	return collectSheets(ctx, r.resource, root, opts, log)
}

func (r ResourceContext) mergeFontFaces(
	ctx context.Context, registry *pdf.Registry, sheets []*css.Stylesheet,
	idx int, log io.Writer,
) *pdf.Registry {
	if r.Loader == nil {
		return registry
	}

	if ctx == nil {
		if log != nil {
			line.Emit(log, line.Warn, "font-face merge: %v", errNilContext)
		}

		return registry
	}

	return mergeFontFaces(ctx, r.resource, registry, sheets, idx, log)
}
