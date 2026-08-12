package imageout

import (
	"fmt"
	"io"
	"time"

	"gowkhtmltopdf/internal/convert"
	"gowkhtmltopdf/internal/settings"
)

// Request is the image-mode job. It does not share convert.Request so the
// PDF union never carries image settings.
type Request struct {
	Global  settings.PdfGlobal
	Image   settings.ImageGlobal
	Objects []settings.PdfObject
	Now     func() time.Time
	Output  io.Writer
}

// NewRequest builds an image conversion request. Objects must contain one
// renderable page source; Output is required.
func NewRequest(
	global settings.PdfGlobal,
	image settings.ImageGlobal,
	objects []settings.PdfObject,
	output io.Writer,
) *Request {
	return &Request{ //nolint:exhaustruct // Now is optional
		Global:  global,
		Image:   image,
		Objects: objects,
		Output:  output,
	}
}

// FromConvertImage adapts the typed convert.ImageRequest builder.
func FromConvertImage(req *convert.ImageRequest) *Request {
	if req == nil {
		return nil
	}

	return &Request{
		Global:  req.Global,
		Image:   req.Image,
		Objects: []settings.PdfObject{req.Object},
		Now:     req.Now,
		Output:  req.Output,
	}
}

func (r *Request) Validate() error {
	if r == nil {
		return errNilRequest
	}

	if r.Output == nil {
		return errNilOutput
	}

	if err := convert.ValidateRenderableObjects(r.Objects); err != nil {
		return fmt.Errorf("imageout: %w", err)
	}

	return nil
}
