package css

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/prepare"
	icss "github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	ihtml "github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pubstate"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// cssPxToPt matches the screen path: 1 CSS pixel is 0.75 points at 96 dpi.
const cssPxToPt = 0.75

func registerStyled() {
	pubstate.RegisterStyled(func(doc any) (pubstate.Styled, bool) {
		styled, ok := doc.(*Document)
		if !ok || styled == nil || styled.root == nil {
			return pubstate.Styled{}, false //nolint:exhaustruct // the document is not styled
		}

		return pubstate.Styled{
			Root:     styled.root,
			Sheets:   styled.sheets,
			Registry: styled.registry,
			Media:    styled.media,
			WidthPx:  styled.widthPx,
			HeightPx: styled.heightPx,
			State:    styled.state,
		}, true
	})
}

// Sheet is one parsed stylesheet.
type Sheet struct {
	sheet *icss.Stylesheet
}

// Document is one HTML tree plus the stylesheets that apply to it.
type Document struct {
	root     *ihtml.Node
	sheets   []*icss.Stylesheet
	registry *pdf.Registry
	media    string
	widthPx  int
	heightPx int
	state    icss.MatchState
}

// Options selects the viewport and any stylesheets beyond the document.
// WidthPx and HeightPx are CSS pixels. Media is "screen" or "print".
// An empty Media means screen. Extra sheets are applied after the
// document's own style elements.
type Options struct {
	WidthPx  int
	HeightPx int
	Media    string
	Extra    []*Sheet
	// Focus, Hover, and Active carry the ids of the focused, hovered, and
	// pressed elements for the stateful pseudo-classes. Empty means none.
	Focus  string
	Hover  string
	Active string
}

// Parse parses one stylesheet.
func Parse(source string) (*Sheet, error) {
	sheet, err := icss.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("css: parse: %w", err)
	}

	if sheet == nil {
		return nil, errEmptySheet
	}

	return &Sheet{sheet: sheet}, nil
}

// Apply collects the document's style elements and the extra sheets.
// The returned document is what layout.Lay places. The HTML tree is the
// one html.Parse produced.
func Apply(ctx context.Context, doc *html.Document, opts Options) (*Document, error) {
	registerStyled()

	if ctx == nil {
		return nil, ErrNilContext
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("css: context: %w", err)
	}

	if doc == nil {
		return nil, ErrNilDocument
	}

	if opts.WidthPx <= 0 || opts.HeightPx <= 0 {
		return nil, ErrBadSize
	}

	media, kind, err := mediaType(opts.Media)
	if err != nil {
		return nil, err
	}

	root := pubstate.Root(doc)
	if root == nil {
		return nil, errUnreadable
	}

	sheets, registry, err := collect(ctx, root, opts, kind, media)
	if err != nil {
		return nil, err
	}

	return &Document{
		root:     root,
		sheets:   sheets,
		registry: registry,
		media:    kind,
		widthPx:  opts.WidthPx,
		heightPx: opts.HeightPx,
		state:    icss.MatchState{Focus: opts.Focus, Hover: opts.Hover, Active: opts.Active},
	}, nil
}

func mediaType(media string) (settings.MediaType, string, error) {
	switch strings.ToLower(strings.TrimSpace(media)) {
	case "", "screen":
		return settings.MediaScreen, "screen", nil
	case "print":
		return settings.MediaPrint, "print", nil
	default:
		return 0, "", fmt.Errorf("%w: %q", ErrBadMedia, media)
	}
}

func collect(
	ctx context.Context,
	root *ihtml.Node,
	opts Options,
	kind string,
	media settings.MediaType,
) ([]*icss.Stylesheet, *pdf.Registry, error) {
	global := settings.DefaultPdfGlobal()
	global.Web.MediaType = media

	loader, err := load.NewLoaderWithError(global.Load)
	if err != nil {
		return nil, nil, fmt.Errorf("css: loader: %w", err)
	}

	//nolint:exhaustruct // media is the only load input sheet collection uses
	page := settings.LoadPage{MediaType: media}
	resources := prepare.NewResourceContext(loader, "inline", page)
	sheetOpts := prepare.SheetOptions{
		ViewportW:       float64(opts.WidthPx) * cssPxToPt,
		ViewportH:       float64(opts.HeightPx) * cssPxToPt,
		MediaType:       kind,
		ObjectIndex:     0,
		PageBoxViewport: nil,
	}

	sheets, err := prepare.CollectTreeSheets(ctx, resources, root, sheetOpts, io.Discard)
	if err != nil {
		return nil, nil, fmt.Errorf("css: sheets: %w", err)
	}

	sheets, err = appendExtra(sheets, opts.Extra)
	if err != nil {
		return nil, nil, err
	}

	registry := resources.MergeFontFaces(ctx, nil, sheets, 0, io.Discard)

	return sheets, registry, nil
}

func appendExtra(sheets []*icss.Stylesheet, extra []*Sheet) ([]*icss.Stylesheet, error) {
	if len(extra) == 0 {
		return sheets, nil
	}

	out := make([]*icss.Stylesheet, 0, len(sheets)+len(extra))
	out = append(out, sheets...)

	for _, sheet := range extra {
		if sheet == nil || sheet.sheet == nil {
			return nil, errNilSheet
		}

		out = append(out, sheet.sheet)
	}

	return out, nil
}
