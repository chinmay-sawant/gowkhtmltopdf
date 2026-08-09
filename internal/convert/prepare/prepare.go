// Package prepare owns the load, parse, stylesheet, and font-resource phase
// shared by the PDF and image conversion pipelines.
package prepare

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

var (
	errNoResourceLoader = errors.New("convert: resource context has no loader")
	errNilLoader        = errors.New("convert: nil loader")
	errNilContext       = errors.New("convert: nil context")
)

// ResourceContext binds one loader, base URL, and per-page policy for a
// prepared document. All subresources use this same resource seam.
type ResourceContext struct {
	resource load.ResourceContext
	Loader   *load.Loader
	Base     string
	Load     settings.LoadPage
}

// NewResourceContext creates the resource seam shared by PDF and image
// preparation.
//
//nolint:wsl // extracted preparation flow
func NewResourceContext(loader *load.Loader, base string, lp settings.LoadPage) ResourceContext {
	resources := ResourceContext{Loader: loader, Base: base, Load: lp} //nolint:exhaustruct
	if loader != nil {
		resource := &load.Resource{Base: base} //nolint:exhaustruct // base-only resource reference

		resources.resource = loader.ForResource(resource, lp)
	}

	return resources
}

// Fetch resolves and loads a document-relative subresource.
//
//nolint:wsl // resource validation flow
func (r ResourceContext) Fetch(ctx context.Context, ref string) (*load.Resource, error) {
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
//
//nolint:wsl,lll // resource validation flow
func (r ResourceContext) CollectSheets(ctx context.Context, root *html.Node, opts SheetOptions, log io.Writer) []*css.Stylesheet {
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
//
//nolint:wsl,lll // resource validation flow
func (r ResourceContext) MergeFontFaces(ctx context.Context, registry *pdf.Registry, sheets []*css.Stylesheet, idx int, log io.Writer) *pdf.Registry {
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

// Options controls the shared document preparation phase.
type Options struct {
	ViewportW       float64
	ViewportH       float64
	MediaType       string
	ObjectIndex     int
	SimplifyDOM     bool
	SimplifyProfile string
}

// Prepared is the output of the shared load/parse/resource phase.
type Prepared struct {
	Resource  *load.Resource
	Root      *html.Node
	Resources ResourceContext
	Sheets    []*css.Stylesheet
	Registry  *pdf.Registry
}

// Document loads and parses one document, gathers stylesheets, and
// merges its @font-face resources. A skipped resource returns a nil Root.
//
//nolint:wsl,lll // document preparation flow
func Document(ctx context.Context, loader *load.Loader, page string, loadPage settings.LoadPage, registry *pdf.Registry, opts Options, log io.Writer) (*Prepared, error) {
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

	prep := &Prepared{ //nolint:exhaustruct // Root and Sheets are populated below
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
		ViewportW: opts.ViewportW, ViewportH: opts.ViewportH,
		MediaType: opts.MediaType, ObjectIndex: opts.ObjectIndex,
	}, log)
	if err != nil {
		return nil, fmt.Errorf("collect stylesheets: %w", err)
	}

	prep.Sheets = AppendSimplifySheet(prep.Sheets, opts.SimplifyDOM, opts.SimplifyProfile)
	prep.Registry = prep.Resources.mergeFontFaces(ctx, registry, prep.Sheets, opts.ObjectIndex, log)

	return prep, nil
}

//nolint:wsl,lll // resource validation flow
func (r ResourceContext) collectSheets(ctx context.Context, root *html.Node, opts SheetOptions, log io.Writer) ([]*css.Stylesheet, error) {
	if r.Loader == nil {
		return nil, errNoResourceLoader
	}
	if ctx == nil {
		return nil, errNilContext
	}

	return collectSheets(ctx, r.resource, root, opts, log)
}

//nolint:wsl,lll // resource validation flow
func (r ResourceContext) mergeFontFaces(ctx context.Context, registry *pdf.Registry, sheets []*css.Stylesheet, idx int, log io.Writer) *pdf.Registry {
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
