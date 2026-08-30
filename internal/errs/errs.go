// Package errs provides canonical domain and operational sentinel errors for gowkhtmltopdf.
package errs

import "errors"

// Shared canonical sentinel errors across all packages.
// Deprecated hub: prefer package-local sentinels (e.g., app.ErrNilCommand).
// This hub is retained for compatibility until all consumers migrate.
// See PT-GO-28.
var (
	// ErrNilContext is returned when a cancellation-aware operation receives a nil context.
	ErrNilContext = errors.New("gowkhtmltopdf: nil context")
	// ErrNilLoader is returned when a load operation is attempted with a nil Loader.
	ErrNilLoader = errors.New("gowkhtmltopdf: nil loader")
	// ErrNilCommand is returned when an app pipeline is executed with a nil CLI command.
	// Deprecated: use app.ErrNilCommand which is now the primary definition.
	ErrNilCommand = errors.New("gowkhtmltopdf: nil command")
	// ErrNilRequest is returned when a conversion operation is executed with a nil request.
	ErrNilRequest = errors.New("gowkhtmltopdf: nil request")
	// ErrImagesDisabled is returned when an image load or rasterization is skipped because images are disabled.
	ErrImagesDisabled = errors.New("gowkhtmltopdf: images disabled")
	// ErrMissingImageOutput is returned when an image conversion request has no output sink.
	ErrMissingImageOutput = errors.New("imageout: nil Output writer")
)
