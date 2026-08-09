// Package errs provides canonical domain and operational sentinel errors for gowkhtmltopdf.
package errs

import "errors"

// Shared canonical sentinel errors across all packages.
var (
	// ErrNilContext is returned when a cancellation-aware operation receives a nil context.
	ErrNilContext = errors.New("gowkhtmltopdf: nil context")
	// ErrNilLoader is returned when a load operation is attempted with a nil Loader.
	ErrNilLoader = errors.New("gowkhtmltopdf: nil loader")
	// ErrNilCommand is returned when an app pipeline is executed with a nil CLI command.
	ErrNilCommand = errors.New("gowkhtmltopdf: nil command")
	// ErrNilRequest is returned when a conversion operation is executed with a nil request.
	ErrNilRequest = errors.New("gowkhtmltopdf: nil request")
)
