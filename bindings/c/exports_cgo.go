//go:build cgo

// cgo facade for libgowkhtmltopdf. This file owns every C pointer
// interaction: it validates the ABI gate, copies borrowed buffers into Go
// memory, drives the shared run hooks, and hands results back under the
// ownership rules of include/gowkhtmltopdf.h.
package main

/*
// The struct layouts below mirror include/gowkhtmltopdf.h. Including the
// committed header directly is not possible here: cgo regenerates
// prototypes for exported functions without const qualifiers, which
// conflicts with the header's const declarations. The runtime abi_version
// and struct_size gate keeps any layout drift between this preamble and
// the authoritative header detectable at call time.

#include <stdint.h>
#include <stdlib.h>

typedef struct {
    int32_t abi_version;
    int32_t struct_size;
    const char* page_size;
    const char* orientation;
    const char* title;
    const char* pdf_version;
    const char* pdf_profile;
    const char* base_url;
    const char* const* allow;
    size_t allow_len;
    double width_mm;
    double height_mm;
    double margin_top;
    double margin_right;
    double margin_bottom;
    double margin_left;
    int32_t copies;
    int32_t grayscale;
    int32_t enable_local_file_access;
    int32_t network_policy;
    int32_t timeout_ms;
} GwkPdfOptions;

typedef struct {
    int32_t abi_version;
    int32_t struct_size;
    const char* format;
    const char* base_url;
    const char* const* allow;
    size_t allow_len;
    int32_t width;
    int32_t height;
    int32_t quality;
    int32_t smart_width;
    int32_t transparent;
    int32_t crop_left;
    int32_t crop_top;
    int32_t crop_width;
    int32_t crop_height;
    double zoom;
    int32_t enable_local_file_access;
    int32_t network_policy;
    int32_t timeout_ms;
} GwkImageOptions;
*/
import "C"

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"time"
	"unsafe"
)

// networkPolicyRestricted selects gowkhtmltopdf.RestrictedNetworkPolicy when
// set in either options struct; 0 keeps the compatible default policy.
const networkPolicyRestricted = 1

//export gowkhtmltopdf_abi_version
func gowkhtmltopdf_abi_version() C.int32_t {
	return C.int32_t(abiVersionValue)
}

//export gowkhtmltopdf_version
func gowkhtmltopdf_version() *C.char {
	return C.CString(libVersion)
}

//export gowkhtmltopdf_html_to_pdf
func gowkhtmltopdf_html_to_pdf(
	cHTML *C.char,
	cLen C.size_t,
	cOpts *C.GwkPdfOptions,
	cOutData **C.uchar,
	cOutLen *C.size_t,
	cErr **C.char,
) C.int {
	html, opts, rejected := parsePDFRequest(cHTML, cLen, cOpts, cErr)
	if rejected {
		return C.int(statusInvalidArg)
	}

	ctx, cancel := requestContext(opts.timeoutMS)
	defer cancel()

	status, data, message := runPDFWithContext(ctx, html, opts)

	return finishResult(status, data, message, cOutData, cOutLen, cErr)
}

//export gowkhtmltopdf_html_to_image
func gowkhtmltopdf_html_to_image(
	cHTML *C.char,
	cLen C.size_t,
	cOpts *C.GwkImageOptions,
	cOutData **C.uchar,
	cOutLen *C.size_t,
	cErr **C.char,
) C.int {
	html, opts, rejected := parseImageRequest(cHTML, cLen, cOpts, cErr)
	if rejected {
		return C.int(statusInvalidArg)
	}

	ctx, cancel := requestContext(opts.timeoutMS)
	defer cancel()

	status, data, message := runImageWithContext(ctx, html, opts)

	return finishResult(status, data, message, cOutData, cOutLen, cErr)
}

//export gowkhtmltopdf_free
func gowkhtmltopdf_free(p unsafe.Pointer) {
	C.free(p)
}

//export gowkhtmltopdf_free_string
func gowkhtmltopdf_free_string(s *C.char) {
	C.free(unsafe.Pointer(s))
}

//export gowkhtmltopdf_last_error_length
func gowkhtmltopdf_last_error_length() C.int32_t {
	return C.int32_t(lastErrorLength())
}

//export gowkhtmltopdf_last_error
func gowkhtmltopdf_last_error(buf *C.char, bufLen C.int32_t) C.int32_t {
	if buf == nil || bufLen <= 0 {
		return 0
	}

	sink := unsafe.Slice((*byte)(unsafe.Pointer(buf)), int(bufLen))

	return C.int32_t(copyLastErrorInto(sink))
}

// runPDFWithContext renders one inline HTML page to PDF bytes and returns
// the ABI status, the payload bytes, and a diagnostic message that is empty
// on success. It exists as a hook so tests can drive the exact export path
// with their own contexts.
func runPDFWithContext(ctx context.Context, html []byte, opts pdfOptions) (int32, []byte, string) {
	if message, ok := validatePDFRange(opts); !ok {
		return statusInvalidArg, nil, message
	}

	var output bytes.Buffer
	if err := buildPDFDocument(html, opts).WritePDF(ctx, &output); err != nil {
		return classifyError(err, ctx), nil, err.Error()
	}

	return statusOK, output.Bytes(), ""
}

// runImageWithContext renders one inline HTML page to encoded image bytes
// using the same conventions as runPDFWithContext.
func runImageWithContext(ctx context.Context, html []byte, opts imageOptions) (int32, []byte, string) {
	var output bytes.Buffer
	if err := buildImageDocument(html, opts).WriteImage(ctx, &output); err != nil {
		return classifyError(err, ctx), nil, err.Error()
	}

	return statusOK, output.Bytes(), ""
}

// requestContext builds the conversion context. Positive timeout values
// install the documented millisecond deadline; every other value disables it.
func requestContext(timeoutMS int64) (context.Context, context.CancelFunc) {
	if timeoutMS <= 0 {
		return context.Background(), func() {}
	}

	return context.WithTimeout(context.Background(), time.Duration(timeoutMS)*time.Millisecond)
}

// parsePDFRequest copies the borrowed HTML buffer and converts the options
// struct. A true third result means the request was rejected and both the
// out_err parameter and the last-error slot already carry the diagnostic.
func parsePDFRequest(
	cHTML *C.char,
	cLen C.size_t,
	cOpts *C.GwkPdfOptions,
	cErr **C.char,
) ([]byte, pdfOptions, bool) {
	var empty pdfOptions

	if cHTML == nil || cLen == 0 {
		rejectRequest("html buffer is nil or empty", cErr)

		return nil, empty, true
	}
	if int64(cLen) > int64(math.MaxInt32) {
		message := fmt.Sprintf("html length %d exceeds the %d byte limit",
			int64(cLen), int64(math.MaxInt32))
		rejectRequest(message, cErr)

		return nil, empty, true
	}

	opts, ok := convertPDFOptions(cOpts, cErr)
	if !ok {
		return nil, empty, true
	}

	return C.GoBytes(unsafe.Pointer(cHTML), C.int(cLen)), opts, false
}

// parseImageRequest mirrors parsePDFRequest for GwkImageOptions.
func parseImageRequest(
	cHTML *C.char,
	cLen C.size_t,
	cOpts *C.GwkImageOptions,
	cErr **C.char,
) ([]byte, imageOptions, bool) {
	var empty imageOptions

	if cHTML == nil || cLen == 0 {
		rejectRequest("html buffer is nil or empty", cErr)

		return nil, empty, true
	}
	if int64(cLen) > int64(math.MaxInt32) {
		message := fmt.Sprintf("html length %d exceeds the %d byte limit",
			int64(cLen), int64(math.MaxInt32))
		rejectRequest(message, cErr)

		return nil, empty, true
	}

	opts, ok := convertImageOptions(cOpts, cErr)
	if !ok {
		return nil, empty, true
	}

	return C.GoBytes(unsafe.Pointer(cHTML), C.int(cLen)), opts, false
}

// convertPDFOptions maps a GwkPdfOptions pointer onto pdfOptions. A nil
// pointer selects defaults for every field. Unknown abi_version values and
// mismatched struct sizes are rejected so layout drift cannot corrupt reads.
func convertPDFOptions(cOpts *C.GwkPdfOptions, cErr **C.char) (pdfOptions, bool) {
	var opts pdfOptions
	if cOpts == nil {
		return opts, true
	}

	if int32(cOpts.abi_version) != abiVersionValue {
		rejectRequest(abiGateMessage(int64(cOpts.abi_version)), cErr)

		return opts, false
	}
	if cOpts.struct_size != 0 && int64(cOpts.struct_size) != int64(C.sizeof_GwkPdfOptions) {
		rejectRequest(structSizeMessage("GwkPdfOptions",
			int64(cOpts.struct_size), int64(C.sizeof_GwkPdfOptions)), cErr)

		return opts, false
	}

	convertPDFStrings(cOpts, &opts)
	convertPDFNumbers(cOpts, &opts)

	return opts, true
}

// convertPDFStrings copies the borrowed string and array fields of a PDF
// options struct. Nil strings behave like empty ones per the header contract.
func convertPDFStrings(cOpts *C.GwkPdfOptions, opts *pdfOptions) {
	if cOpts.page_size != nil {
		opts.pageSize = C.GoString(cOpts.page_size)
	}
	if cOpts.orientation != nil {
		opts.orientation = C.GoString(cOpts.orientation)
	}
	if cOpts.title != nil {
		opts.title = C.GoString(cOpts.title)
	}
	if cOpts.pdf_version != nil {
		opts.pdfVersion = C.GoString(cOpts.pdf_version)
	}
	if cOpts.pdf_profile != nil {
		opts.pdfProfile = C.GoString(cOpts.pdf_profile)
	}
	if cOpts.base_url != nil {
		opts.baseURL = C.GoString(cOpts.base_url)
	}

	opts.allow = convertAllowList(cOpts.allow, cOpts.allow_len)
}

// convertPDFNumbers copies the scalar fields of a PDF options struct.
func convertPDFNumbers(cOpts *C.GwkPdfOptions, opts *pdfOptions) {
	opts.widthMM = float64(cOpts.width_mm)
	opts.heightMM = float64(cOpts.height_mm)
	opts.marginTop = float64(cOpts.margin_top)
	opts.marginRight = float64(cOpts.margin_right)
	opts.marginBottom = float64(cOpts.margin_bottom)
	opts.marginLeft = float64(cOpts.margin_left)
	opts.copies = int(cOpts.copies)
	opts.grayscale = cOpts.grayscale != 0
	opts.localFiles = cOpts.enable_local_file_access != 0
	opts.restricted = int(cOpts.network_policy) == networkPolicyRestricted
	opts.timeoutMS = int64(cOpts.timeout_ms)
}

// convertImageOptions maps a GwkImageOptions pointer onto imageOptions using
// the same gates as convertPDFOptions.
func convertImageOptions(cOpts *C.GwkImageOptions, cErr **C.char) (imageOptions, bool) {
	var opts imageOptions
	if cOpts == nil {
		return opts, true
	}

	if int32(cOpts.abi_version) != abiVersionValue {
		rejectRequest(abiGateMessage(int64(cOpts.abi_version)), cErr)

		return opts, false
	}
	if cOpts.struct_size != 0 && int64(cOpts.struct_size) != int64(C.sizeof_GwkImageOptions) {
		rejectRequest(structSizeMessage("GwkImageOptions",
			int64(cOpts.struct_size), int64(C.sizeof_GwkImageOptions)), cErr)

		return opts, false
	}

	if cOpts.format != nil {
		opts.format = C.GoString(cOpts.format)
	}
	if cOpts.base_url != nil {
		opts.baseURL = C.GoString(cOpts.base_url)
	}
	opts.allow = convertAllowList(cOpts.allow, cOpts.allow_len)

	opts.width = int(cOpts.width)
	opts.height = int(cOpts.height)
	opts.quality = int(cOpts.quality)
	opts.smartWidth = int(cOpts.smart_width)
	opts.transparent = cOpts.transparent != 0
	opts.cropLeft = int(cOpts.crop_left)
	opts.cropTop = int(cOpts.crop_top)
	opts.cropWidth = int(cOpts.crop_width)
	opts.cropHeight = int(cOpts.crop_height)
	opts.zoom = float64(cOpts.zoom)
	opts.localFiles = cOpts.enable_local_file_access != 0
	opts.restricted = int(cOpts.network_policy) == networkPolicyRestricted
	opts.timeoutMS = int64(cOpts.timeout_ms)

	return opts, true
}

// convertAllowList copies allow_len entries from the borrowed allow array.
// Nil entries are skipped because the header lets callers leave slots empty.
func convertAllowList(allow **C.char, allowLen C.size_t) []string {
	if allow == nil || allowLen == 0 {
		return nil
	}

	entries := unsafe.Slice(allow, int(allowLen))
	copied := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry != nil {
			copied = append(copied, C.GoString(entry))
		}
	}

	if len(copied) == 0 {
		return nil
	}

	return copied
}

// finishResult writes the success payload or the failure diagnostic into the
// caller-owned out parameters and records diagnostics in the last-error
// slot. Memory conventions follow include/gowkhtmltopdf.h: success stores an
// allocation for gowkhtmltopdf_free with NULL out_err; failure stores NULL
// data, zero length, and an allocation for gowkhtmltopdf_free_string.
func finishResult(
	status int32,
	data []byte,
	message string,
	cOutData **C.uchar,
	cOutLen *C.size_t,
	cErr **C.char,
) C.int {
	if status == statusOK && len(data) > 0 {
		*cOutData = (*C.uchar)(C.CBytes(data))
		*cOutLen = C.size_t(len(data))
		*cErr = nil

		return C.int(statusOK)
	}

	if status == statusOK {
		status = statusRenderError
		message = "conversion produced no output"
	}
	if message == "" {
		message = "conversion failed"
	}

	*cOutData = nil
	*cOutLen = 0
	*cErr = C.CString(message)
	setLastError(message)

	return C.int(status)
}

// rejectRequest records a validation failure in both the out_err parameter
// and the process-wide last-error slot.
func rejectRequest(message string, cErr **C.char) {
	setLastError(message)
	if cErr != nil {
		*cErr = C.CString(message)
	}
}

// abiGateMessage formats the rejection text for an unsupported abi_version.
func abiGateMessage(got int64) string {
	return fmt.Sprintf("unsupported abi_version %d; expected %d", got, int64(abiVersionValue))
}

// structSizeMessage formats the rejection text for a mismatched struct_size.
func structSizeMessage(kind string, got, want int64) string {
	return fmt.Sprintf("%s struct_size %d does not match expected %d", kind, got, want)
}

// exportedABI returns the result of gowkhtmltopdf_abi_version. It and the
// helpers below let Go-side tests drive the real export signatures; go vet
// rejects cgo inside test files, so every C pointer interaction lives here.
func exportedABI() int32 {
	return int32(gowkhtmltopdf_abi_version())
}

// exportedVersion returns the version string allocated by
// gowkhtmltopdf_version, releasing it through gowkhtmltopdf_free_string.
func exportedVersion() string {
	version := gowkhtmltopdf_version()
	defer gowkhtmltopdf_free_string(version)

	return C.GoString(version)
}

// invokePDFExport calls gowkhtmltopdf_html_to_pdf with the supplied options
// pointer, converts the out parameters back to Go values, and releases both
// allocations through the documented free functions.
func invokePDFExport(html []byte, opts *C.GwkPdfOptions) (int32, []byte, string) {
	cHTML := C.CString(string(html))
	defer C.free(unsafe.Pointer(cHTML))

	var out *C.uchar
	var outLen C.size_t
	var cErr *C.char

	status := gowkhtmltopdf_html_to_pdf(
		cHTML, C.size_t(len(html)), opts, &out, &outLen, &cErr)
	if cErr != nil {
		message := C.GoString(cErr)
		gowkhtmltopdf_free_string(cErr)

		return int32(status), nil, message
	}
	if out == nil {
		return int32(status), nil, ""
	}

	data := C.GoBytes(unsafe.Pointer(out), C.int(outLen))
	gowkhtmltopdf_free(unsafe.Pointer(out))

	return int32(status), data, ""
}

// probePDFExport drives the export with a hand-built options struct carrying
// an explicit abiVersion value and structSize stamp (a negative structSize
// skips the stamp). It exists to exercise ABI gate rejections from tests.
func probePDFExport(html []byte, abiVersion int32, structSize int64) (int32, []byte, string) {
	var opts C.GwkPdfOptions
	opts.abi_version = C.int32_t(abiVersion)

	if structSize >= 0 {
		opts.struct_size = C.int32_t(structSize)
	}

	return invokePDFExport(html, &opts)
}

// probeStampedPDFExport drives the export with a correctly stamped options
// struct: supported abi_version plus the real sizeof(GwkPdfOptions).
func probeStampedPDFExport(html []byte) (int32, []byte, string) {
	var opts C.GwkPdfOptions
	opts.abi_version = C.int32_t(abiVersionValue)
	opts.struct_size = C.int32_t(C.sizeof_GwkPdfOptions)

	return invokePDFExport(html, &opts)
}

// smokeFreePairingLoop runs the PDF export through the real C ABI n times,
// releasing each payload with gowkhtmltopdf_free, then frees NULL. The
// returned string describes the first failure and is empty on success.
func smokeFreePairingLoop(html []byte, iterations int) string {
	for range iterations {
		status, data, message := invokePDFExport(html, nil)
		if status != statusOK || len(data) == 0 || message != "" {
			return message
		}
	}

	gowkhtmltopdf_free(nil)

	return ""
}

// readLastErrorViaExport copies the process-wide diagnostic through
// gowkhtmltopdf_last_error into C memory and back as a Go string.
func readLastErrorViaExport() string {
	length := gowkhtmltopdf_last_error_length()
	if length <= 0 {
		return ""
	}

	sink := (*C.char)(C.malloc(C.size_t(length) + 1))
	if sink == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(sink))

	written := gowkhtmltopdf_last_error(sink, length+1)
	if written <= 0 {
		return ""
	}

	return C.GoStringN(sink, C.int(written))
}
