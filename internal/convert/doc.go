// Package convert orchestrates the load → layout → paginate → print
// pipeline (mirrors PdfConverterPrivate): one pdf.Document for the whole
// job, one page object laid out and painted into it per input.
package convert

import (
	"fmt"

	"gowkhtmltopdf/internal/settings"
)

// HttpError is a load failure carrying the upstream exit-code mapping.
type HttpError struct {
	Status int
	URL    string
}

func (e *HttpError) Error() string {
	return fmt.Sprintf("failed to load %s: HTTP %d", e.URL, e.Status)
}

// HttpErrorCode implements the exit-code mapping (404→2, 401→3).
func (e *HttpError) HttpErrorCode() int { return settings.HttpErrorCode(e.Status) }
