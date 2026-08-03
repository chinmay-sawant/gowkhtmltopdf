package settings

import "fmt"

// HttpStatusError is a load failure carrying the HTTP status so callers can
// map to wkhtmltopdf exit codes (utilities.cc): 404→2, 401→3, else 1.
type HttpStatusError struct {
	Status int
	URL    string
}

func (e *HttpStatusError) Error() string {
	return fmt.Sprintf("failed to load %s: HTTP %d", e.URL, e.Status)
}

// HttpErrorCode maps an HTTP status to the wkhtmltopdf exit-code convention.
func (e *HttpStatusError) HttpErrorCode() int { return HttpErrorCode(e.Status) }
