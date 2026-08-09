package convert

import (
	"context"
	"fmt"
	"io"

	"gowkhtmltopdf/internal/convert/prepare"
	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/load"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/settings"
)

// These aliases preserve the existing convert package seam while the
// implementation lives in the focused prepare module.
type ResourceContext = prepare.ResourceContext
type SheetOptions = prepare.SheetOptions
type PrepareOptions = prepare.Options
type PreparedDocument = prepare.Prepared

func NewResourceContext(loader *load.Loader, base string, lp settings.LoadPage) ResourceContext {
	return prepare.NewResourceContext(loader, base, lp)
}

func PrepareDocument(
	ctx context.Context, loader *load.Loader, page string, loadPage settings.LoadPage,
	registry *pdf.Registry, opts PrepareOptions, log io.Writer,
) (*PreparedDocument, error) {
	document, err := prepare.Document(ctx, loader, page, loadPage, registry, opts, log)
	if err != nil {
		return nil, fmt.Errorf("prepare document: %w", err)
	}

	return document, nil
}

func CollectSheets(
	ctx context.Context, loader *load.Loader, root *html.Node, base string,
	loadPage settings.LoadPage, opts SheetOptions, log io.Writer,
) []*css.Stylesheet {
	return prepare.CollectSheets(ctx, loader, root, base, loadPage, opts, log)
}

func MergeFontFaces(
	ctx context.Context, loader *load.Loader, registry *pdf.Registry, sheets []*css.Stylesheet,
	base string, loadPage settings.LoadPage, idx int, log io.Writer,
) *pdf.Registry {
	return prepare.MergeFontFaces(ctx, loader, registry, sheets, base, loadPage, idx, log)
}

func linkStylesheet(node *html.Node, viewportW, viewportH float64, mediaType string) bool { //nolint:unparam
	_ = viewportW

	return prepare.LinkStylesheet(node, viewportW, viewportH, mediaType)
}
