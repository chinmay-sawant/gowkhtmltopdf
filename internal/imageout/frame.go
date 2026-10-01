package imageout

import (
	"context"
	"fmt"
	"image"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// RenderLayout lays out root and rasterizes it, and also returns the layout
// result. RenderContext stays the image-only entry. This second entry exists
// so a caller can read element boxes from the same layout that produced the
// pixels. The returned image is the cropped final canvas, not the supersample
// buffer.
func RenderLayout(ctx context.Context, root *html.Node, opts RenderOptions) (image.Image, *layout.Result, error) {
	if root == nil {
		return nil, nil, errNilRoot
	}

	if err := opts.Validate(); err != nil {
		return nil, nil, err
	}

	if ctx == nil {
		return nil, nil, errNilContext
	}

	if err := ctx.Err(); err != nil {
		return nil, nil, fmt.Errorf("imageout: context: %w", err)
	}

	font := opts.Font
	if font == nil {
		var err error

		font, err = pdf.DefaultFont()
		if err != nil {
			return nil, nil, fmt.Errorf("imageout: default font: %w", err)
		}
	}

	res, err := layoutResult(ctx, root, opts, font)
	if err != nil {
		return nil, nil, err
	}

	img, err := rasterizeContext(ctx, res, maxHeight(res, opts), opts.Transparent, opts.Padding, opts.Zoom)
	if err != nil {
		return nil, nil, err
	}

	out, err := applyCrop(img, opts.Crop)
	if err != nil {
		return nil, nil, err
	}

	return out, res, nil
}
