// Package errs provides canonical domain and operational sentinel errors for gowkhtmltopdf.
package errs

import "errors"

// Shared canonical sentinel errors across all packages.
// Deprecated hub: prefer package-local sentinels (e.g., app.ErrNilCommand).
// This hub is retained for compatibility until all consumers migrate.
// See PT-GO-28.
//
// Migrated (primary definitions now in owning packages, errs not used):
//   - ErrNilLoader        -> load.ErrNilLoader
//   - ErrNilRequest       -> convert.errNilRequest / imageout.errNilRequest (private, same string)
//   - ErrImagesDisabled   -> convert.errImagesDisabled / imageout.errImagesDisabled (private, same string)
//   - ErrMissingImageOutput -> imageout.ErrMissingOutput (public, api re-exports imageout)
// Parked (still in hub, need coordinated migration):
//   - ErrNilContext (10+ consumers across app/convert/prepare/render/imageout/layout/load/api; distinct instances would break errors.Is)
//   - ErrNilCommand (primary in app; imageout/compat_test cannot import app due to app -> imageout cycle)
var (
	// ErrNilContext is returned when a cancellation-aware operation receives a nil context.
	ErrNilContext = errors.New("gowkhtmltopdf: nil context")
	// ErrNilCommand is returned when an app pipeline is executed with a nil CLI command.
	// Deprecated: use app.ErrNilCommand which is now the primary definition.
	ErrNilCommand = errors.New("gowkhtmltopdf: nil command")
)
