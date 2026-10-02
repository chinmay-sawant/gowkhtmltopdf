package layout

import (
	"context"
	"fmt"
	"image"

	"github.com/chinmay-sawant/gowkhtmltopdf/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/imageout"
	ilayout "github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pubstate"
)

// ptToPx converts a layout point to one CSS pixel at zoom 1.
const ptToPx = 96.0 / 72.0

// Box is one element border box in CSS pixels.
// Action is the data-action attribute. Text is the descendant text.
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

// Result is one placement and the picture painted from it.
type Result struct {
	img    image.Image
	boxes  []Box
	width  int
	height int
}

// Lay places doc and paints the picture from that same placement.
func Lay(ctx context.Context, doc *css.Document) (*Result, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("layout: context: %w", err)
	}

	styled, ok := pubstate.StyledOf(doc)
	if !ok {
		return nil, ErrNilDocument
	}

	//nolint:exhaustruct // fixed viewport, backgrounds on, no crop, no smart width
	opts := imageout.RenderOptions{
		Width:      styled.WidthPx,
		Height:     styled.HeightPx,
		Sheets:     styled.Sheets,
		Media:      styled.Media,
		Background: true,
		Registry:   styled.Registry,
		State:      styled.State,
	}

	img, res, err := imageout.RenderLayout(ctx, styled.Root, opts)
	if err != nil {
		return nil, fmt.Errorf("layout: place: %w", err)
	}

	if img == nil {
		return nil, errNoImage
	}

	bounds := img.Bounds()

	return &Result{
		img:    img,
		boxes:  boxesFrom(ilayout.PlacedElements(res)),
		width:  bounds.Dx(),
		height: bounds.Dy(),
	}, nil
}

// Image returns the painted picture.
func (r *Result) Image() image.Image {
	if r == nil {
		return nil
	}

	return r.img
}

// Boxes returns the element boxes. The slice is the one Lay built.
func (r *Result) Boxes() []Box {
	if r == nil {
		return nil
	}

	return r.boxes
}

// Size returns the picture size in pixels.
func (r *Result) Size() (int, int) {
	if r == nil {
		return 0, 0
	}

	return r.width, r.height
}

func boxesFrom(placed []ilayout.PlacedElement) []Box {
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
