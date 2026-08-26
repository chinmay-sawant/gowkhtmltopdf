/*
 * gowkhtmltopdf C ABI contract.
 *
 * This header is the authoritative interface of libgowkhtmltopdf, the
 * c-shared build (-buildmode=c-shared, CGO_ENABLED=1) produced from the
 * bindings/c package. It is committed and curated: a regenerated header
 * emitted next to the shared library is a build artifact, never a source
 * of truth.
 *
 * All functions are one-shot: they take an HTML buffer plus one options
 * struct and return either encoded bytes or an error message. There is no
 * document handle and no global renderer state to release.
 *
 * Status codes returned by gowkhtmltopdf_html_to_pdf and
 * gowkhtmltopdf_html_to_image:
 *
 *   0  OK            conversion succeeded, out_data/out_len are valid
 *   1  INVALID_ARG   nil or empty HTML, bad abi_version/struct_size, or
 *                    an option value rejected by validation (page size,
 *                    orientation, PDF version/profile, copies range)
 *   2  LOAD_DENIED   a local-file ACL or network policy rule denied a
 *                    resource needed by the document
 *   3  RENDER_ERROR  layout, paint, pagination, or encoding failed
 *   4  TIMEOUT       the caller timeout elapsed or the operation was
 *                    cancelled before completion
 *   5  RESOURCE_LIMIT  an engine ceiling was exceeded (for example the
 *                    copies limit enforced inside the converter)
 *   6  INTERNAL      unexpected internal failure; also returned by every
 *                    function when the library was built without cgo
 *
 * Memory ownership rules:
 *
 *   - html and every string/array field inside the options struct are
 *     borrowed for the duration of the call only. The library copies what
 *     it needs before returning; the caller may free them immediately
 *     after the call returns.
 *   - On success (*out_err == NULL): *out_data points to a single heap
 *     allocation holding *out_len bytes. The caller must release it with
 *     gowkhtmltopdf_free. The library always sets *out_err to NULL on
 *     success so callers can rely on that sentinel.
 *   - On failure (*out_data == NULL, *out_len == 0): *out_err points to a
 *     NUL-terminated message allocated with the same allocator used by
 *     gowkhtmltopdf_free_string. The caller must release it with
 *     gowkhtmltopdf_free_string.
 *   - The string returned by gowkhtmltopdf_version is heap allocated and
 *     must be released with gowkhtmltopdf_free_string.
 *   - Passing NULL to gowkhtmltopdf_free or gowkhtmltopdf_free_string is
 *     allowed and does nothing.
 *
 * Options structs:
 *
 *   - Every struct starts with abi_version followed by struct_size. Callers
 *     must set abi_version to GOWKHTMLTOPDF_ABI_VERSION and struct_size to
 *     sizeof of the struct they compiled against. A zero struct_size skips
 *     the size gate; any other mismatched size is rejected with
 *     INVALID_ARG so layout drift cannot corrupt reads.
 *   - Passing opts == NULL selects defaults for every field.
 *   - String fields left NULL behave like empty strings and select engine
 *     defaults. The allow array is an array of pointers to NUL-terminated
 *     path prefixes; allow_len entries are read only during the call.
 *   - network_policy: 0 keeps the compatible default policy (HTTP(S) to any
 *     host, cross-host redirects allowed); 1 installs the restricted policy
 *     (private and link-local destinations blocked unless allowlisted,
 *     redirects stay on the original host).
 *   - enable_local_file_access: 0 keeps local file access denied (default),
 *     1 allows the document to reference local files.
 *   - timeout_ms: values <= 0 disable the deadline; positive values cancel
 *     the conversion after that many milliseconds (status TIMEOUT).
 *   - copies: 0 selects the engine default; 1 through 1000 is accepted;
 *     anything else is INVALID_ARG.
 *   - Image options: quality 0 selects the engine default (94); width and
 *     height 0 keep auto sizing; smart_width -1 means unset (engine
 *     default true), 0 disables, 1 enables; crop_left, crop_top,
 *     crop_width, crop_height use -1 for "no crop on this axis"; zoom 0
 *     keeps the engine default scale of 1.
 *
 * Error retrieval helpers:
 *
 *   gowkhtmltopdf_last_error_length reports the byte length of the most
 *   recent error message recorded by this process (0 when none).
 *   gowkhtmltopdf_last_error copies that message into buf, truncating and
 *   NUL-terminating if buf_len is too small, and returns the number of
 *   payload bytes written excluding the terminator. The slot is
 *   process-wide: concurrent callers race over which message is current,
 *   so prefer the out_err string when available.
 *
 * Version macros:
 *
 *   GOWKHTMLTOPDF_ABI_VERSION   ABI contract revision (struct layouts,
 *                               signatures, status codes)
 *   GOWKHTMLTOPDF_VERSION       project release this header shipped with
 *   GOWKHTMLTOPDF_LIBRARY_VERSION  upstream settings-surface identifier
 */
#ifndef GOWKHTMLTOPDF_H
#define GOWKHTMLTOPDF_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

#define GOWKHTMLTOPDF_ABI_VERSION 1
#define GOWKHTMLTOPDF_VERSION "0.2.4"
#define GOWKHTMLTOPDF_LIBRARY_VERSION "0.12.7-dev"

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

/* Returns the ABI revision this library implements. Always 1 today. */
int32_t gowkhtmltopdf_abi_version(void);

/* Returns the runtime library version string (allocated, see above). */
const char* gowkhtmltopdf_version(void);

/*
 * Converts html_len bytes of HTML into a PDF document.
 *
 * opts may be NULL for defaults. On success stores 0 in the return value,
 * a heap allocation in *out_data (release with gowkhtmltopdf_free), its
 * byte length in *out_len, and NULL in *out_err. On failure stores a
 * non-zero status code, NULL in *out_data, 0 in *out_len, and an allocated
 * diagnostic in *out_err (release with gowkhtmltopdf_free_string).
 */
int gowkhtmltopdf_html_to_pdf(const char* html, size_t html_len, const GwkPdfOptions* opts, unsigned char** out_data, size_t* out_len, char** out_err);

/*
 * Converts html_len bytes of HTML into an encoded image (PNG by default,
 * JPEG when format says so).
 *
 * Ownership and return conventions match gowkhtmltopdf_html_to_pdf.
 */
int gowkhtmltopdf_html_to_image(const char* html, size_t html_len, const GwkImageOptions* opts, unsigned char** out_data, size_t* out_len, char** out_err);

/* Releases a *out_data allocation. NULL is ignored. */
void gowkhtmltopdf_free(void* p);

/* Releases a *out_err or gowkhtmltopdf_version string. NULL is ignored. */
void gowkhtmltopdf_free_string(char* s);

/* Byte length of the most recent error message, 0 when there is none. */
int32_t gowkhtmltopdf_last_error_length(void);

/*
 * Copies the most recent error message into buf (buf_len includes room
 * for the NUL terminator). Returns payload bytes written excluding the
 * terminator, 0 for NULL buf, non-positive buf_len, or no stored message.
 */
int32_t gowkhtmltopdf_last_error(char* buf, int32_t buf_len);

#ifdef __cplusplus
}
#endif

#endif /* GOWKHTMLTOPDF_H */
