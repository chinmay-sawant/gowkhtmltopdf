package imageout

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// ErrMultipleInputs reports an image request that carries more than one
// object. Image conversion has one output canvas and therefore owns exactly
// one input object; callers must choose the source before running the job.
var ErrMultipleInputs = errors.New("imageout: exactly one input object is required")

// ErrMissingOutput reports an image request with a nil output writer.
// Primary definition (migrated from internal/errs, PT-GO-28).
var ErrMissingOutput = errors.New("imageout: nil Output writer")

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

	// Dimensions, padding, and crop are validated here so the app preflight
	// rejects them before cmd.OpenOutput truncates an existing file. The
	// same helper backs RenderOptions.Validate; CropSettings' -1 "unset"
	// sentinel is the accepted crop floor for this layer.
	crop := r.Image.Crop
	if err := validateCanvasGeometry(r.Image.Width, r.Image.Height, r.Image.Padding,
		[4]int{crop.Left, crop.Top, crop.Width, crop.Height}, -1); err != nil {
		return fmt.Errorf("imageout: %w", err)
	}

	if err := settings.ValidateRenderableObjects(r.Objects); err != nil {
		return fmt.Errorf("imageout: %w", err)
	}

	return nil
}
