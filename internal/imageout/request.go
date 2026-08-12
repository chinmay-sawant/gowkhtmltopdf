package imageout

import (
	"errors"
	"fmt"
	"io"
	"time"

	"gowkhtmltopdf/internal/convert"
	"gowkhtmltopdf/internal/settings"
)

// ErrMultipleInputs reports an image request that carries more than one
// object. Image conversion has one output canvas and therefore owns exactly
// one input object; callers must choose the source before running the job.
var ErrMultipleInputs = errors.New("imageout: exactly one input object is required")

// Request is the image-mode job. It does not share convert.Request so the
// PDF union never carries image settings.
type Request struct {
	Global  settings.PdfGlobal
	Image   settings.ImageGlobal
	Objects []settings.PdfObject
	Now     func() time.Time
	Output  io.Writer
}

// NewRequest builds an image conversion request. Objects must contain exactly
// one renderable page source; Output is required. The slice is retained at
// this compatibility boundary, but validation rejects zero or multiple
// objects before rendering starts.
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

	if len(r.Objects) > 1 {
		return fmt.Errorf("%w: got %d", ErrMultipleInputs, len(r.Objects))
	}

	if err := convert.ValidateRenderableObjects(r.Objects); err != nil {
		return fmt.Errorf("imageout: %w", err)
	}

	return nil
}
