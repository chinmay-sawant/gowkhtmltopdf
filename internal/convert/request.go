package convert

import (
	"context"
	"io"
	"time"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// PDFRequest is the type-safe API for PDF-only conversions. Unlike the shared
// Request union, it cannot carry image-mode settings and guarantees at compile
// time that only PDF-valid fields are populated.
type PDFRequest struct {
	Global        settings.PdfGlobal
	Objects       []settings.PdfObject
	Now           func() time.Time
	Output        io.Writer
	OutlineOutput io.Writer
}

// ToRequest converts a PDFRequest to the internal shared Request for pipeline
// consumption. This keeps the conversion engine unchanged.
func (pr *PDFRequest) ToRequest() *Request {
	if pr == nil {
		return nil
	}

	return &Request{ //nolint:exhaustruct // intentional zero-value fields
		Global:        pr.Global,
		Objects:       pr.Objects,
		Now:           pr.Now,
		Output:        pr.Output,
		OutlineOutput: pr.OutlineOutput,
	}
}

// RunTypedPDF executes the PDF pipeline from a type-safe PDFRequest.
// It converts to the internal Request and delegates to Run.
func RunTypedPDF(ctx context.Context, req *PDFRequest, log io.Writer, progress func(phase string, percent int)) error {
	if req == nil {
		return errNilRequest
	}

	return Run(ctx, req.ToRequest(), log, progress)
}
