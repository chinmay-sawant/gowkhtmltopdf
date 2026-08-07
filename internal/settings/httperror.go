package settings

import "fmt"

// HTTP status codes that map to distinct wkhtmltopdf exit codes (utilities.cc).
const (
	httpStatusNotFound       = 404
	httpStatusUnauthorized   = 401
	exitCodeNotFound         = 2
	exitCodeUnauthorized     = 3
	exitCodeGenericLoadError = 1
)

// HttpStatusError is a load failure carrying the HTTP status so callers can
// map to wkhtmltopdf exit codes (utilities.cc): 404→2, 401→3, else 1.
type HttpStatusError struct { //nolint:revive,stylecheck // API name; cli/load construct it by this exact name
	Status int
	URL    string
}

func (e *HttpStatusError) Error() string {
	return fmt.Sprintf("failed to load %s: HTTP %d", e.URL, e.Status)
}

// HttpErrorCode maps an HTTP status to the wkhtmltopdf exit-code convention
// (utilities.cc): 404 → 2, 401 → 3, everything else stays 1.
func HttpErrorCode(status int) int { //nolint:revive,stylecheck // API name; settings_test exercises it directly
	switch status {
	case httpStatusNotFound:
		return exitCodeNotFound
	case httpStatusUnauthorized:
		return exitCodeUnauthorized
	}

	return exitCodeGenericLoadError
}

// HttpErrorCode reports the exit code this load failure maps to.
func (e *HttpStatusError) HttpErrorCode() int { //nolint:revive,stylecheck // matched by cli.ExitCode interface check
	return HttpErrorCode(e.Status)
}
