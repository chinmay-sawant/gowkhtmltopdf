package imageout_test

import (
	"bytes"
	"errors"
	"testing"

	"gowkhtmltopdf/internal/imageout"
	"gowkhtmltopdf/internal/settings"
)

func TestRequestValidateRejectsMultipleInputs(t *testing.T) {
	t.Parallel()

	first := settings.DefaultPdfObject()
	first.Page = "first.html"
	second := settings.DefaultPdfObject()
	second.Page = "second.html"

	req := imageout.NewRequest(
		settings.DefaultPdfGlobal(),
		settings.DefaultImageGlobal(),
		[]settings.PdfObject{first, second},
		&bytes.Buffer{},
	)

	if err := req.Validate(); !errors.Is(err, imageout.ErrMultipleInputs) {
		t.Fatalf("Validate() = %v, want errors.Is(..., ErrMultipleInputs)", err)
	}
}
