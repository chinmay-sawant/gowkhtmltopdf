// Package main builds libgowkhtmltopdf, the c-shared library exposing the
// frozen C ABI in include/gowkhtmltopdf.h. main.go stays free of build tags
// and of any import "C" so the package compiles under CGO_ENABLED=0 (stub
// exports) and CGO_ENABLED=1 (real exports) alike. The pieces both modes
// need live here: the runtime version string, the ABI/status constants, and
// the process-wide last-error slot behind its mutex.
package main

import (
	"sync"
)

// libVersion is reported by gowkhtmltopdf_version. Release builds stamp it
// via -ldflags "-X main.libVersion=$(cat VERSION)"
// (the Makefile BINDINGS_VERSION_LDFLAGS value; bindings/c is package main).
//
//nolint:gochecknoglobals // ldflags injection target shared by both build modes
var libVersion = "dev"

// abiVersionValue mirrors GOWKHTMLTOPDF_ABI_VERSION in the committed header.
const abiVersionValue = int32(1)

// Status codes returned through the C ABI. Keep in sync with the table
// documented in include/gowkhtmltopdf.h.
const (
	statusOK             = int32(0)
	statusInvalidArg     = int32(1)
	statusLoadDenied     = int32(2)
	statusRenderError    = int32(3)
	statusTimeout        = int32(4)
	statusResourceLimit  = int32(5)
	statusInternal       = int32(6)
	stubRequiredCGOError = "gowkhtmltopdf shared library requires CGO_ENABLED=1 (-buildmode=cshared)"
)

// The last-error slot records the most recent failure message from any
// exported call. It is process-wide rather than thread-local because the
// primary consumer loads the library through ctypes from Python, where the
// GIL serializes calls; the mutex still makes concurrent Go-side access safe.
type lastErrorSlot struct {
	mu    sync.Mutex
	value string
}

var lastError = lastErrorSlot{} //nolint:gochecknoglobals // single documented diagnostic slot for the whole library

// setLastError replaces the stored diagnostic message.
func setLastError(message string) {
	lastError.mu.Lock()
	defer lastError.mu.Unlock()

	lastError.value = message
}

// currentLastError returns a copy of the stored diagnostic message.
func currentLastError() string {
	lastError.mu.Lock()
	defer lastError.mu.Unlock()

	return lastError.value
}

// lastErrorLength reports the stored message length, 0 when empty.
func lastErrorLength() int32 {
	lastError.mu.Lock()
	defer lastError.mu.Unlock()

	return int32(len(lastError.value))
}

// copyLastErrorInto copies the stored message into buf, truncating and
// reserving the final byte for the NUL terminator. It returns the payload
// byte count written excluding that terminator.
func copyLastErrorInto(buf []byte) int32 {
	if len(buf) == 0 {
		return 0
	}

	limit := len(buf) - 1
	written := copy(buf[:limit], currentLastError())
	buf[written] = 0

	return int32(written)
}

func main() {}
