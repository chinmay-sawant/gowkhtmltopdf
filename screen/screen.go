package screen

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/layout"
)

// Sentinel errors for caller mistakes. Engine failures are wrapped errors
// and do not match these.
var (
	ErrNilContext = errors.New("screen: nil context")
	ErrEmptyHTML  = errors.New("screen: empty html")
	ErrBadSize    = errors.New("screen: width and height must be positive")
)

// Box is one element the host can hit-test.
// It is the layout package's box, in CSS pixels.
type Box = layout.Box

// Frame is one laid-out HTML document.
type Frame struct {
	PNG    []byte
	Boxes  []Box
	Width  int
	Height int
}

// Render parses source, applies its style sheets, lays the document out,
// and returns a PNG plus the element rectangles from that layout.
// widthPx and heightPx are the viewport in CSS pixels.
// Linked style sheets and images are not fetched.
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

	doc, err := html.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("screen: parse: %w", err)
	}

	styled, err := css.Apply(ctx, doc, css.Options{
		WidthPx:  widthPx,
		HeightPx: heightPx,
		Media:    "screen",
		Extra:    nil,
		Focus:    "",
		Hover:    "",
		Active:   "",
	})
	if err != nil {
		return nil, fmt.Errorf("screen: css: %w", err)
	}

	placed, err := layout.Lay(ctx, styled)
	if err != nil {
		return nil, fmt.Errorf("screen: layout: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, placed.Image()); err != nil {
		return nil, fmt.Errorf("screen: png: %w", err)
	}

	width, height := placed.Size()

	return &Frame{
		PNG:    buf.Bytes(),
		Boxes:  placed.Boxes(),
		Width:  width,
		Height: height,
	}, nil
}
