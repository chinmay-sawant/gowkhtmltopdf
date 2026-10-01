package screen

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"io"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/prepare"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/imageout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

const (
	// cssPxToPt matches imageout: 1 CSS pixel is 0.75 points at 96 dpi.
	cssPxToPt = 0.75
	// ptToPx converts a layout point to one PNG pixel at zoom 1.
	ptToPx = 96.0 / 72.0
)

// Sentinel errors for caller mistakes. Engine failures are wrapped errors
// and do not match these.
var (
	ErrNilContext = errors.New("screen: nil context")
	ErrEmptyHTML  = errors.New("screen: empty html")
	ErrBadSize    = errors.New("screen: width and height must be positive")

	errEmptyDocument = errors.New("screen: empty document")
)

// Box is one element the host can hit-test. X, Y, W, H are CSS pixels.
// Action is the data-action attribute. Text is the element's descendant text.
type Box struct {
	ID     string
	Tag    string
	Action string
	Text   string
	X      float64
	Y      float64
	W      float64
	H      float64
}

// Frame is one laid-out HTML document.
type Frame struct {
	PNG    []byte
	Boxes  []Box
	Width  int
	Height int
}

// Render parses source, applies its style sheets, and returns a PNG plus
// element rectangles from that same layout. widthPx and heightPx are the
// viewport in CSS pixels. The PNG is at least that size. Linked style sheets
// and images are not fetched. A style element in the document is applied.
func Render(ctx context.Context, source []byte, widthPx, heightPx int) (*Frame, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("screen: context: %w", err)
	}

	if strings.TrimSpace(string(source)) == "" {
		return nil, ErrEmptyHTML
	}

	if widthPx <= 0 || heightPx <= 0 {
		return nil, ErrBadSize
	}

	prep, err := prepareSource(ctx, source, widthPx, heightPx)
	if err != nil {
		return nil, err
	}

	return rasterFrame(ctx, prep, widthPx, heightPx)
}

func prepareSource(ctx context.Context, source []byte, widthPx, heightPx int) (
	*prepare.Prepared, error,
) {
	global := settings.DefaultPdfGlobal()
	global.Web.MediaType = settings.MediaScreen

	loader, err := load.NewLoaderWithError(global.Load)
	if err != nil {
		return nil, fmt.Errorf("screen: loader: %w", err)
	}

	//nolint:exhaustruct // inline HTML is the only load input this entry uses
	pageLoad := settings.LoadPage{
		InlineHTML: source,
		MediaType:  settings.MediaScreen,
	}

	widthPt := float64(widthPx) * cssPxToPt
	heightPt := float64(heightPx) * cssPxToPt
	opts := prepare.BuildOptions(widthPt, heightPt, "screen", 0, global.Web)

	prep, err := prepare.Document(ctx, loader, "inline", pageLoad, nil, opts, io.Discard)
	if err != nil {
		return nil, fmt.Errorf("screen: prepare: %w", err)
	}

	if prep == nil || prep.Root == nil {
		return nil, fmt.Errorf("screen: prepare: %w", errEmptyDocument)
	}

	return prep, nil
}

func rasterFrame(ctx context.Context, prep *prepare.Prepared, widthPx, heightPx int) (*Frame, error) {
	//nolint:exhaustruct // fixed viewport, screen media, backgrounds on, no crop
	opts := imageout.RenderOptions{
		Width:      widthPx,
		Height:     heightPx,
		Sheets:     prep.Sheets,
		Media:      "screen",
		Background: true,
		Registry:   prep.Registry,
	}

	img, res, err := imageout.RenderLayout(ctx, prep.Root, opts)
	if err != nil {
		return nil, fmt.Errorf("screen: render: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("screen: png: %w", err)
	}

	bounds := img.Bounds()

	return &Frame{
		PNG:    buf.Bytes(),
		Boxes:  boxesFrom(layout.PlacedElements(res)),
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
	}, nil
}

func boxesFrom(placed []layout.PlacedElement) []Box {
	boxes := make([]Box, len(placed))

	for i, item := range placed {
		boxes[i] = Box{
			ID:     item.ID,
			Tag:    item.Tag,
			Action: item.Action,
			Text:   item.Text,
			X:      item.X * ptToPx,
			Y:      item.Y * ptToPx,
			W:      item.W * ptToPx,
			H:      item.H * ptToPx,
		}
	}

	return boxes
}
