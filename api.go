// Library API: wkhtmltopdf-compatible HTML-to-PDF and HTML-to-image
// conversion, exposing the internal convert/imageout pipelines through an
// idiomatic pure-Go surface with an allowlisted module graph.
package gowkhtmltopdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"gowkhtmltopdf/internal/convert"
	"gowkhtmltopdf/internal/errs"
	"gowkhtmltopdf/internal/imageout"
	"gowkhtmltopdf/internal/line"
	"gowkhtmltopdf/internal/load"
	"gowkhtmltopdf/internal/settings"
)

// LibraryVersion is the upstream wkhtmltopdf compatibility identifier
// (0.12.x settings surface). It is not the project release number; that
// lives in the VERSION file and is stamped into cli.Version at build time.
const LibraryVersion = "0.12.7-dev"

// Static conversion errors; callers can match with errors.Is.
var (
	// ErrNoPageObjectsAdded reports a PDF conversion without any page objects.
	ErrNoPageObjectsAdded = errors.New("gowkhtmltopdf: no page objects added")
	// ErrNoRenderablePDFObjects reports a typed or legacy PDF request that has
	// no body object with a page source or inline HTML. TOC-only requests do not
	// satisfy this invariant.
	ErrNoRenderablePDFObjects = ErrNoPageObjectsAdded
	// ErrEmptyHTML reports an empty document passed to ConvertHTML.
	ErrEmptyHTML = errors.New("gowkhtmltopdf: empty HTML")
	// ErrNoInputPageAdded reports an image conversion without an input page.
	ErrNoInputPageAdded = errors.New("gowkhtmltopdf: no input page added")
	// ErrNilImageConverter reports a conversion attempted through a nil
	// ImageConverter receiver.
	ErrNilImageConverter = errors.New("gowkhtmltopdf: nil image converter")
	// ErrNilConverter reports a conversion attempted through a nil Converter.
	ErrNilConverter = errors.New("gowkhtmltopdf: nil converter")
	// ErrNilGlobalSettings reports a method call on nil global settings.
	ErrNilGlobalSettings = errors.New("gowkhtmltopdf: nil global settings")
	// ErrInvalidPageSize reports a named page size that is not in the canonical
	// page-size table.
	ErrInvalidPageSize = errors.New("gowkhtmltopdf: invalid page size")
	// ErrNilObjectSettings reports a method call on nil object settings.
	ErrNilObjectSettings = errors.New("gowkhtmltopdf: nil object settings")
	// ErrNilImageSettings reports a method call on nil image settings.
	ErrNilImageSettings = errors.New("gowkhtmltopdf: nil image settings")
	// ErrNilPDFRequest reports a conversion attempted through a nil typed PDF
	// request. It is distinct from ErrNilConverter so callers can tell a
	// missing PDFRequest from a nil Converter receiver.
	ErrNilPDFRequest = errors.New("gowkhtmltopdf: nil pdf request")
	// ErrNilImageRequest reports a conversion attempted through a nil typed
	// image request. It is distinct from ErrNilImageConverter so callers can
	// tell a missing ImageRequest from a nil ImageConverter receiver.
	ErrNilImageRequest = errors.New("gowkhtmltopdf: nil image request")
	// ErrMissingPDFOutput reports a typed PDF request without a document sink.
	// It aliases the engine sentinel so errors.Is remains stable across the
	// public preflight and direct internal request seams.
	ErrMissingPDFOutput = convert.ErrMissingOutput
	// ErrInvalidPDFCopies reports a typed PDF request with a non-positive copy count.
	ErrInvalidPDFCopies = convert.ErrInvalidCopies
	// ErrMissingPDFOutlineOutput reports a dump-outline request without a
	// dedicated metadata sink. It aliases the engine sentinel.
	ErrMissingPDFOutlineOutput = convert.ErrMissingOutlineOutput
	// ErrMissingImageOutput reports a typed image request without an output
	// sink.
	ErrMissingImageOutput = errs.ErrMissingImageOutput
	// ErrNilContext reports a cancellation-aware operation without a context.
	ErrNilContext = errs.ErrNilContext
	// ErrInvalidPDFVersion reports an invalid or unsupported PDF version string.
	ErrInvalidPDFVersion = settings.ErrInvalidPDFVersion
	// ErrPDF20Unsupported reports PDF 2.0 requested before support is implemented.
	ErrPDF20Unsupported = settings.ErrPDF20Unsupported
	// ErrInvalidPDFProfile reports an invalid or unsupported PDF profile string.
	ErrInvalidPDFProfile = settings.ErrInvalidPDFProfile
	// ErrProfilePDF20Unsupported reports PDF 2.0 conformance profiles (PDF/A-4, PDF/UA-2) are unsupported.
	ErrProfilePDF20Unsupported = settings.ErrProfilePDF20Unsupported
	// ErrProfilePDFA1Unsupported reports PDF/A-1 is unsupported.
	ErrProfilePDFA1Unsupported = settings.ErrProfilePDFA1Unsupported
	// ErrConformanceRequiresPDF17 indicates a conformance profile was requested without PDF 1.7.
	ErrConformanceRequiresPDF17 = convert.ErrProfileRequiresPDF17
	// ErrProfileRequiresPDF17 is an alias for ErrConformanceRequiresPDF17.
	ErrProfileRequiresPDF17 = convert.ErrProfileRequiresPDF17
)

// Version returns the library version banner.
func Version() string {
	return LibraryVersion + " (gowkhtmltopdf pure-go)"
}

// ---------------------------------------------------------------------------
// Settings
// ---------------------------------------------------------------------------

// GlobalSettings wraps the wkhtmltopdf global settings: page geometry,
// margins, headers/footers, load and web behaviour. Create with
// NewGlobalSettings, which starts from the wkhtmltopdf defaults.
type GlobalSettings struct {
	g settings.PdfGlobal
}

// NetworkPolicy controls HTTP(S) document and subresource loading. An empty
// AllowedHosts list permits any host allowed by the scheme policy; a non-empty
// list is an exact or wildcard host allowlist. Explicitly allowlisted hosts
// may be private, which is useful for trusted internal services and tests.
type NetworkPolicy = load.NetworkPolicy

// CompatibleNetworkPolicy preserves the historical permissive URL behavior.
func CompatibleNetworkPolicy() NetworkPolicy {
	return load.CompatibleNetworkPolicy()
}

// RestrictedNetworkPolicy is intended for untrusted HTML in an isolated
// service: private destinations are blocked and redirects stay same-origin.
func RestrictedNetworkPolicy() NetworkPolicy {
	return load.RestrictedNetworkPolicy()
}

// PdfGlobalOptions is the typed builder for common PDF global settings. It is
// an alternative to GlobalSettings.Set for library callers; CLI-compatible
// string settings remain available through GlobalSettings. Existing fluent
// methods panic on a nil receiver because a nil builder is a programmer error;
// WithSetting is the error-returning escape hatch for validated input.
type PdfGlobalOptions struct {
	options settings.PdfGlobalOptions
}

// NewPdfGlobalOptions returns a typed builder initialized with PDF defaults.
func NewPdfGlobalOptions() *PdfGlobalOptions {
	return &PdfGlobalOptions{options: settings.NewPdfGlobalOptions()}
}

func (o *PdfGlobalOptions) require() *PdfGlobalOptions {
	if o == nil {
		panic("gowkhtmltopdf: nil PdfGlobalOptions")
	}

	return o
}

// WithPageSize validates a named page size through the canonical page-size
// table before storing it. Invalid input panics; use WithSetting when an
// WithPageSize sets the named page size ("A4", "Letter", "Legal", etc.).
// Invalid values fail during conversion / validation with ErrInvalidPageSize.
func (o *PdfGlobalOptions) WithPageSize(pageSize string) *PdfGlobalOptions {
	o = o.require()
	o.options = o.options.WithPageSize(pageSize)

	return o
}

// WithMargins sets top, right, bottom, and left margins in millimetres.
func (o *PdfGlobalOptions) WithMargins(top, right, bottom, left float64) *PdfGlobalOptions {
	o = o.require()
	o.options = o.options.WithMargins(top, right, bottom, left)

	return o
}

// WithTitle sets the document title metadata.
func (o *PdfGlobalOptions) WithTitle(title string) *PdfGlobalOptions {
	o = o.require()
	o.options = o.options.WithTitle(title)

	return o
}

// WithCopies sets the number of printed copies and collation. Non-positive
// copy counts fail during conversion / validation with ErrInvalidPDFCopies.
func (o *PdfGlobalOptions) WithCopies(copies int, collate bool) *PdfGlobalOptions {
	o = o.require()
	o.options = o.options.WithCopies(copies, collate)

	return o
}

// WithOutline enables or disables PDF outline generation and sets outline depth.
func (o *PdfGlobalOptions) WithOutline(enabled bool, depth int) *PdfGlobalOptions {
	o = o.require()
	o.options = o.options.WithOutline(enabled, depth)

	return o
}

// WithSmartShrinking enables or disables the smart shrinking algorithm.
func (o *PdfGlobalOptions) WithSmartShrinking(enabled bool) *PdfGlobalOptions {
	o = o.require()
	o.options = o.options.WithSmartShrinking(enabled)

	return o
}

// WithBackground enables or disables printing of CSS background images and colors.
func (o *PdfGlobalOptions) WithBackground(enabled bool) *PdfGlobalOptions {
	o = o.require()
	o.options = o.options.WithBackground(enabled)

	return o
}

// WithCompression enables or disables PDF stream compression.
func (o *PdfGlobalOptions) WithCompression(enabled bool) *PdfGlobalOptions {
	o = o.require()
	o.options = o.options.WithCompression(enabled)

	return o
}

// WithResolveRelativeLinks enables or disables resolving relative link targets.
func (o *PdfGlobalOptions) WithResolveRelativeLinks(enabled bool) *PdfGlobalOptions {
	o = o.require()
	o.options = o.options.WithResolveRelativeLinks(enabled)

	return o
}

// WithPDFVersion sets the target PDF version ("1.4" or "1.7").
// Invalid values fail during conversion / validation with ErrInvalidPDFVersion
// or ErrPDF20Unsupported.
func (o *PdfGlobalOptions) WithPDFVersion(version string) *PdfGlobalOptions {
	o = o.require()
	o.options = o.options.WithPDFVersion(version)

	return o
}

// WithPDFProfile sets the target PDF conformance profile ("a3a-ua1", "PDF/A-3a+PDF/UA-1", "a3a", "ua1", etc.).
// Invalid values fail during conversion / validation with ErrInvalidPDFProfile or ErrProfilePDF20Unsupported.
func (o *PdfGlobalOptions) WithPDFProfile(profile string) *PdfGlobalOptions {
	o = o.require()
	o.options = o.options.WithPDFProfile(profile)

	return o
}

// WithSetting applies any supported dotted key and returns an error for an
// unknown or invalid value. Prefer the typed With* methods for common options.
func (o *PdfGlobalOptions) WithSetting(name, value string) (*PdfGlobalOptions, error) {
	o = o.require()
	key := strings.ToLower(strings.TrimSpace(name))

	if key == "size.pagesize" {
		normalized, err := normalizePageSize(value)
		if err != nil {
			return o, err
		}

		value = normalized
	}

	if key == "copies" {
		copies, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil && copies < 1 {
			return o, fmt.Errorf("%w: got %d", ErrInvalidPDFCopies, copies)
		}
	}

	options, err := o.options.WithSetting(name, value)
	if err != nil {
		return o, fmt.Errorf("settings builder: %w", err)
	}

	o.options = options

	return o, nil
}

// Build returns an independent public settings snapshot. Pass it as
// PDFRequest.Global, to ConvertHTML, or to Converter.WithGlobal. Those
// conversion entry points copy the settings at their ownership boundary, so
// the builder and the built settings can be reused independently. Nil-builder
// and invalid-input behavior follows the fluent policy documented above.
func (o *PdfGlobalOptions) Build() *GlobalSettings {
	o = o.require()

	return &GlobalSettings{g: o.options.Build()}
}

func normalizePageSize(pageSize string) (string, error) {
	pageSize = strings.TrimSpace(pageSize)
	if pageSize == "" {
		pageSize = "A4"
	}

	if _, _, err := settings.ParsePageSize(pageSize); err != nil {
		return "", fmt.Errorf("%w: %q", ErrInvalidPageSize, pageSize)
	}

	return pageSize, nil
}

// NewGlobalSettings returns the wkhtmltopdf-compatible default global
// settings (A4 portrait, 10 mm margins, background on, …).
func NewGlobalSettings() *GlobalSettings {
	return &GlobalSettings{g: settings.DefaultPdfGlobal()}
}

// SetNetworkPolicy installs an explicit URL-loading policy on these global
// settings. Existing callers that do not call this method retain compatible
// loader behavior.
func (s *GlobalSettings) SetNetworkPolicy(policy NetworkPolicy) error {
	if s == nil {
		return ErrNilGlobalSettings
	}

	load.ApplyNetworkPolicy(&s.g.Load, policy)

	return nil
}

// EnableLocalFileAccess configures global settings to permit local file reads
// and unblocks local file access on the loader ACL.
func (s *GlobalSettings) EnableLocalFileAccess() *GlobalSettings {
	if s == nil {
		return nil
	}

	s.g.Load.EnableLocalFileAccess = true

	return s
}

// Set applies a dotted settings key ("size.pagesize", "margin.top",
// "orientation", "web.background", "load.timeout", …). Named page sizes and
// copy counts are validated at this public settings seam; non-positive copies
// are rejected. Negative margins remain valid for header/footer layout.
// Unknown names return an error.
func (s *GlobalSettings) Set(name, value string) error {
	if s == nil {
		return ErrNilGlobalSettings
	}

	key := strings.ToLower(strings.TrimSpace(name))
	if key == "size.pagesize" {
		pageSize, err := normalizePageSize(value)
		if err != nil {
			return fmt.Errorf("global set %q: %w", name, err)
		}

		value = pageSize
	}

	if key == "copies" {
		copies, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil && copies < 1 {
			return fmt.Errorf("global set %q: %w: got %d", name, ErrInvalidPDFCopies, copies)
		}
	}

	if err := s.g.Set(name, value); err != nil {
		return fmt.Errorf("global set %q: %w", name, err)
	}

	return nil
}

// Get reads a dotted settings key back as its canonical string form via the
// same key table as Set (plus Policy A Ignored values). ok is false when the
// name is unknown. Booleans read as "true"/"false", enums as their canonical
// name ("Portrait", "print", "abort", …), floats with the shortest round-trip
// representation.
func (s *GlobalSettings) Get(name string) (string, bool) {
	if s == nil {
		return "", false
	}

	return s.g.Get(name)
}

// ObjectSettings wraps the per-page wkhtmltopdf settings: the input page
// plus its load/web/header/footer overrides. Create with NewObjectSettings,
// which starts from the wkhtmltopdf defaults.
type ObjectSettings struct {
	o settings.PdfObject
}

// NewObjectSettings returns the wkhtmltopdf-compatible default object
// settings. Note that, like the CLI, local file access is blocked by
// default; combine GlobalSettings "enablelocalfileaccess" with
// "load.blocklocalfileaccess" = "false" to allow local inputs.
func NewObjectSettings() *ObjectSettings {
	return &ObjectSettings{o: settings.DefaultPdfObject()}
}

// Set applies a dotted object settings key ("page", "load.jsdelay",
// "web.images", "header.left", "footer.right", …). Unknown names return an
// error.
func (s *ObjectSettings) Set(name, value string) error {
	if s == nil {
		return ErrNilObjectSettings
	}

	if err := s.o.Set(name, value); err != nil {
		return fmt.Errorf("object set %q: %w", name, err)
	}

	return nil
}

// Get reads a dotted object settings key via the same key table as Set.
func (s *ObjectSettings) Get(name string) (string, bool) {
	if s == nil {
		return "", false
	}

	return s.o.Get(name)
}

// EnableLocalFileAccess unblocks local file access for this page object.
func (s *ObjectSettings) EnableLocalFileAccess() *ObjectSettings {
	if s == nil {
		return nil
	}

	s.o.Load.BlockLocalFileAccess = false

	return s
}

// SetPage sets the input page (a path, URL or "inline:…" / "data:…" source)
// and returns s so calls can be chained.
func (s *ObjectSettings) SetPage(page string) *ObjectSettings {
	if s == nil {
		return nil
	}

	s.o.Page = page

	return s
}

// SetBody sets an in-memory HTML document as the input page and returns s
// so calls can be chained. base resolves relative subresources (<link>,
// <img>, <a>); an empty base leaves them unresolvable. No URL guessing is
// involved: the bytes are always treated as a document.
func (s *ObjectSettings) SetBody(html []byte, base string) *ObjectSettings {
	if s == nil {
		return nil
	}

	s.o.Page = ""
	s.o.Load.InlineHTML = cloneBytes(html)
	s.o.Load.InlineBase = base

	return s
}

// NewTOCObject returns object settings with the registered dotted key
// istableofcontents=true, matching a CLI `toc` object. The keys stay
// registered on the settings table; this constructor is the documented
// library way to build a TOC object.
func NewTOCObject() *ObjectSettings {
	obj := NewObjectSettings()
	if err := obj.Set("istableofcontent", "true"); err != nil {
		// Registered key; a failure here is a settings-table bug.
		panic(err)
	}

	obj.o.UseOutline = false

	return obj
}

// NewCoverObject returns object settings with the registered dotted key
// iscover=true, matching a CLI `cover` object (no header/footer, excluded
// from the outline).
func NewCoverObject() *ObjectSettings {
	obj := NewObjectSettings()
	if err := obj.Set("iscover", "true"); err != nil {
		panic(err)
	}

	obj.o.IncludeInOutline = false
	obj.o.HeaderSet, obj.o.FooterSet = true, true
	obj.o.Header, obj.o.Footer = settings.HeaderFooter{}, settings.HeaderFooter{} //nolint:exhaustruct

	return obj
}

// ---------------------------------------------------------------------------
// PDF conversion
// ---------------------------------------------------------------------------

// Converter drives one HTML-to-PDF conversion. Configure it with
// Global()/AddObject, then Convert produces PDF bytes readable via Output().
// A Converter is not safe for concurrent Convert calls; create one per
// conversion.
type Converter struct {
	global  *GlobalSettings
	objects []*ObjectSettings
	output  []byte

	// OnInfo, OnWarn and OnError receive the conversion's log lines,
	// classified by severity marker ("warning:"/"warn:" lines go to
	// OnWarn, "error:"/"err:" lines to OnError, everything else -
	// including phase lines - to OnInfo). They may be nil.
	OnInfo  func(string)
	OnWarn  func(string)
	OnError func(string)

	// OnPhase receives human-readable phase names ("Loading pages (1/1)",
	// "Printing pages (1/1)", "Done") and OnProgress receives the 0-100
	// percentage, as the conversion advances. They may be nil.
	OnPhase    func(phase string)
	OnProgress func(percent int)
}

// NewConverter returns a Converter with the wkhtmltopdf default global
// settings and no page objects.
func NewConverter() *Converter {
	return &Converter{global: NewGlobalSettings()} //nolint:exhaustruct // intentional zero/partial fields
}

// Global returns the converter's global settings, for Set/Get.
func (c *Converter) Global() *GlobalSettings {
	if c == nil {
		return nil
	}

	if c.global == nil {
		c.global = NewGlobalSettings()
	}

	return c.global
}

// WithGlobal replaces the converter's global settings with an owned copy of
// s and returns c for chaining. A nil receiver or nil settings pointer is
// returned unchanged.
func (c *Converter) WithGlobal(s *GlobalSettings) *Converter {
	if c == nil || s == nil {
		return c
	}

	c.global = &GlobalSettings{g: settings.ClonePdfGlobal(s.g)}

	return c
}

// AddObject appends a page object and returns c for chaining. The object's
// settings are copied, so later mutations of s do not affect the converter.
// A converter needs at least one object to convert. A nil settings pointer is
// ignored so an optional object cannot panic the conversion setup.
func (c *Converter) AddObject(s *ObjectSettings) *Converter {
	if c == nil || s == nil {
		return c
	}

	cp := &ObjectSettings{o: settings.ClonePdfObject(s.o)}
	c.objects = append(c.objects, cp)

	return c
}

// clonePdfObject / clonePdfGlobal / cloneImageGlobal are thin wrappers
// around the settings package clones so the public API has a single
// ownership-boundary helper name.
func clonePdfObject(src settings.PdfObject) settings.PdfObject {
	return settings.ClonePdfObject(src)
}

func clonePdfGlobal(src settings.PdfGlobal) settings.PdfGlobal {
	return settings.ClonePdfGlobal(src)
}

func cloneImageGlobal(src settings.ImageGlobal) settings.ImageGlobal {
	return settings.CloneImageGlobal(src)
}

func cloneBytes(src []byte) []byte {
	if src == nil {
		return nil
	}

	dst := make([]byte, len(src))
	copy(dst, src)

	return dst
}

// EnableLocalFileAccess sets enablelocalfileaccess on global settings and
// unblocks local file access across all attached objects.
func (c *Converter) EnableLocalFileAccess() *Converter {
	if c == nil {
		return nil
	}

	c.Global().EnableLocalFileAccess()

	for _, obj := range c.objects {
		if obj != nil {
			obj.EnableLocalFileAccess()
		}
	}

	return c
}

// AddHTML appends an in-memory HTML document as a page object and returns c
// for chaining. baseURL resolves relative subresources; an empty baseURL
// leaves them unresolvable. Unlike SetPage, the bytes are always treated as
// a document - no URL guessing is applied.
func (c *Converter) AddHTML(page []byte, baseURL string) *Converter {
	if c == nil {
		return nil
	}

	c.AddObject(NewObjectSettings().SetBody(page, baseURL))

	return c
}

// Convert runs the conversion. The produced bytes replace the previous
// Output. ctx is threaded into every load; cancel it to abort. Engine and
// preflight validation errors are also reported to OnError when set. Output
// is captured via an in-memory writer (no temp file). Prefer ConvertTo when
// the caller already has an io.Writer; RunPDF is the canonical writer-first
// typed path.
func (c *Converter) Convert(ctx context.Context) error {
	if c == nil {
		return ErrNilConverter
	}

	var buf bytes.Buffer
	if err := c.ConvertTo(ctx, &buf); err != nil {
		return err
	}

	c.output = buf.Bytes()

	return nil
}

// ConvertTo writes the PDF directly to w. It snapshots global and object
// settings so later mutations of Global() / objects cannot change the
// in-flight request. Callers do not need to call Output(); Convert() still
// buffers for Output() compatibility.
//
//nolint:cyclop // sequential validation and setup steps
func (c *Converter) ConvertTo(ctx context.Context, writer io.Writer) error {
	if c == nil {
		return ErrNilConverter
	}

	if writer == nil {
		return reportPreflight(c.OnError, ErrMissingPDFOutput)
	}

	if c.global == nil {
		c.global = NewGlobalSettings()
	}

	global := settings.ClonePdfGlobal(c.global.g)
	objects := make([]settings.PdfObject, len(c.objects))

	for i, o := range c.objects {
		objects[i] = settings.ClonePdfObject(o.o)
	}

	if global.PageSize != "" {
		if _, _, err := settings.ParsePageSize(global.PageSize); err != nil {
			return reportPreflight(c.OnError, fmt.Errorf("%w: %q", ErrInvalidPageSize, global.PageSize))
		}
	}

	if global.PdfProfile != "" {
		if _, err := settings.ParsePDFProfile(global.PdfProfile); err != nil {
			return reportPreflight(c.OnError, err)
		}
	}

	if global.PdfVersion != "" {
		if _, err := settings.ParsePDFVersion(global.PdfVersion); err != nil {
			return reportPreflight(c.OnError, err)
		}
	}

	if _, err := convert.PolicyForGlobal(global); err != nil {
		return reportPreflight(c.OnError, err)
	}

	if global.Copies < 1 {
		return reportPreflight(c.OnError, ErrInvalidPDFCopies)
	}

	if err := validatePDFObjects(objects); err != nil {
		return reportPreflight(c.OnError, err)
	}

	req := convert.NewPDFRequest(global, objects, writer, nil)
	h := convertHooks{
		OnInfo: c.OnInfo, OnWarn: c.OnWarn, OnError: c.OnError,
		OnPhase: c.OnPhase, OnProgress: c.OnProgress,
	}

	return h.executePDFTo(ctx, req)
}

// Output returns the PDF bytes produced by the last successful Convert, or
// nil if none ran yet. The returned slice is a copy; it stays valid across
// later conversions.
func (c *Converter) Output() []byte {
	if c == nil {
		return nil
	}

	return append([]byte(nil), c.output...)
}

// ConvertHTML is a one-shot helper: convert an in-memory HTML document to
// PDF bytes without creating a temp input file. global may be nil (defaults
// apply). Local file / subresource ACL is unchanged — linked local assets
// still need enablelocalfileaccess + load.blocklocalfileaccess=false.
//

func ConvertHTML(ctx context.Context, html []byte, global *GlobalSettings) ([]byte, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}

	if len(html) == 0 {
		return nil, ErrEmptyHTML
	}

	conv := NewConverter()
	conv.WithGlobal(global)

	conv.AddObject(NewObjectSettings().SetBody(html, ""))

	if err := conv.Convert(ctx); err != nil {
		return nil, err
	}

	return conv.Output(), nil
}

// ---------------------------------------------------------------------------
// Image conversion
// ---------------------------------------------------------------------------

// ImageConverter drives one HTML-to-image conversion (the wkhtmltoimage
// counterpart): it renders the current page and encodes it as PNG or JPEG.
// Configure with Set/AddObject, then Convert produces the encoded bytes via
// Output(). Not safe for concurrent Convert calls.
type ImageConverter struct {
	global      *GlobalSettings
	image       settings.ImageGlobal
	object      *ObjectSettings
	output      []byte
	initialized bool

	// OnInfo, OnWarn and OnError receive the conversion's log lines,
	// classified like Converter's. They may be nil.
	OnInfo  func(string)
	OnWarn  func(string)
	OnError func(string)
}

// NewImageConverter returns an ImageConverter with the wkhtmltoimage default
// settings: 1024 px smart-width viewport, PNG output. Object() is valid
// immediately; its page is empty until AddObject.
func NewImageConverter() *ImageConverter {
	return &ImageConverter{ //nolint:exhaustruct // intentional zero/partial fields
		global:      NewGlobalSettings(),
		image:       settings.DefaultImageGlobal(),
		object:      NewObjectSettings(),
		initialized: true,
	}
}

// Set applies an image-mode settings key: "width", "height", "quality",
// "smartwidth", "transparent", "format" ("png"|"jpg") and the crop keys
// "crop.left", "crop.top", "crop.width", "crop.height". Unknown names return
// an error. "web.background" also updates the shared Global.Background paint
// switch so image and PDF share one background field.
func (c *ImageConverter) Set(name, value string) error {
	if c == nil {
		return ErrNilImageConverter
	}

	c.ensureDefaults()

	if err := settings.ApplyImageKey(&c.global.g, &c.image, name, value); err != nil {
		return fmt.Errorf("image set %q: %w", name, err)
	}

	return nil
}

// Global returns the shared global settings (merged with image load settings
// via load.ResolveEffectiveLoadGlobal for ACL, proxy, and network restrictions).
func (c *ImageConverter) Global() *GlobalSettings {
	if c == nil {
		return nil
	}

	c.ensureDefaults()

	return c.global
}

// AddObject replaces the current page to convert (a path, URL, or
// "inline:"/"data:" source) and returns c for chaining. Image conversion
// renders the most recently added page. The page's load settings can be
// adjusted through Object.
func (c *ImageConverter) AddObject(page string) *ImageConverter {
	if c == nil {
		return nil
	}

	c.ensureDefaults()

	o := NewObjectSettings()
	o.SetPage(page)
	c.object = o

	return c
}

// Object returns the settings of the page to convert (the one passed to the
// most recent AddObject), so per-page load options can be set - for example
// "load.blocklocalfileaccess" = "false" to allow local inputs. Object is
// always valid; its page is empty until AddObject is called.
func (c *ImageConverter) Object() *ObjectSettings {
	if c == nil {
		return nil
	}

	c.ensureDefaults()

	return c.object
}

func (c *ImageConverter) ensureDefaults() {
	if !c.initialized {
		c.image = settings.DefaultImageGlobal()
		c.initialized = true
	}

	if c.global == nil {
		c.global = NewGlobalSettings()
	}

	if c.object == nil {
		c.object = NewObjectSettings()
	}
}

// Convert runs the conversion, replacing the previous Output. ctx is threaded
// into the load; cancel it to abort. Engine and preflight validation errors
// are also reported to OnError when set.
// Output is captured via an in-memory writer (no temp file). Prefer ConvertTo
// when the caller already has an io.Writer.
//
// The page may be a path/URL via AddObject/SetPage, or in-memory HTML via
// Object().SetBody (P2-04 InlineHTML source kind).
func (c *ImageConverter) Convert(ctx context.Context) error {
	if c == nil {
		return ErrNilImageConverter
	}

	var buf bytes.Buffer
	if err := c.ConvertTo(ctx, &buf); err != nil {
		return err
	}

	c.output = buf.Bytes()

	return nil
}

// ConvertTo writes the encoded image directly to w. It snapshots global,
// image, and object settings so later mutations cannot change the in-flight
// request. Callers do not need to call Output().
func (c *ImageConverter) ConvertTo(ctx context.Context, writer io.Writer) error {
	if c == nil {
		return ErrNilImageConverter
	}

	if writer == nil {
		return reportPreflight(c.OnError, ErrMissingImageOutput)
	}

	c.ensureDefaults()

	global := settings.ClonePdfGlobal(c.global.g)
	img := settings.CloneImageGlobal(c.image)
	obj := settings.ClonePdfObject(c.object.o)

	if err := validateImageObjects([]settings.PdfObject{obj}); err != nil {
		return reportPreflight(c.OnError, err)
	}

	req := imageout.NewRequest(global, img, []settings.PdfObject{obj}, writer)
	h := convertHooks{ //nolint:exhaustruct // intentional zero/partial fields
		OnInfo: c.OnInfo, OnWarn: c.OnWarn, OnError: c.OnError,
	}

	return h.executeImageTo(ctx, req)
}

// Output returns the encoded image bytes (PNG, or JPEG when format was set
// to "jpg") from the last successful Convert, or nil if none ran yet. The
// returned slice is a copy.
func (c *ImageConverter) Output() []byte {
	if c == nil {
		return nil
	}

	return append([]byte(nil), c.output...)
}

// ---------------------------------------------------------------------------
// Shared executor (P1-8)
// ---------------------------------------------------------------------------

// reportPreflight dispatches err to onError if non-nil and returns err.
func reportPreflight(onError func(string), err error) error {
	if err != nil && onError != nil {
		onError(err.Error())
	}

	return err
}

// convertHooks holds the optional log/progress callbacks shared by PDF and
// image Convert drivers. Both Converter and ImageConverter reduce to building
// a convert.Request and calling executePDF / executeImage.
type convertHooks struct {
	OnInfo, OnWarn, OnError func(string)
	OnPhase                 func(string)
	OnProgress              func(int)
}

func (h convertHooks) lineLog() *lineLog {
	return &lineLog{ //nolint:exhaustruct // intentional zero/partial fields
		onInfo:  h.OnInfo,
		onWarn:  h.OnWarn,
		onError: h.OnError,
	}
}

func (h convertHooks) progress() func(string, int) {
	if h.OnPhase == nil && h.OnProgress == nil {
		return nil
	}

	return func(phase string, percent int) {
		if h.OnPhase != nil {
			h.OnPhase(phase)
		}

		if h.OnProgress != nil {
			h.OnProgress(percent)
		}
	}
}

// executePDFTo runs the PDF pipeline into req.Output and reports failures
// to OnError. The caller owns the writer; this path does not buffer a copy
// for Output().
func (h convertHooks) executePDFTo(ctx context.Context, req *convert.Request) error {
	if ctx == nil {
		return reportPreflight(h.OnError, ErrNilContext)
	}

	if err := convert.Run(ctx, req, h.lineLog(), h.progress()); err != nil {
		if h.OnError != nil {
			h.OnError(err.Error())
		}

		return fmt.Errorf("convert: %w", err)
	}

	return nil
}

// executeImageTo runs the image pipeline into req.Output and reports
// failures to OnError.
func (h convertHooks) executeImageTo(ctx context.Context, req *imageout.Request) error {
	if ctx == nil {
		return reportPreflight(h.OnError, ErrNilContext)
	}

	if err := imageout.RunRequest(ctx, req, h.lineLog()); err != nil {
		if h.OnError != nil {
			h.OnError(err.Error())
		}

		return fmt.Errorf("image convert: %w", err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Typed request API (Phase 2)
// ---------------------------------------------------------------------------

// PDFRequest is the type-safe public API for PDF-only conversions. Public
// wrapper types keep internal/settings out of the library interface while the
// conversion engine retains its internal request representation.
type PDFRequest struct {
	Global        *GlobalSettings
	Objects       []*ObjectSettings
	Now           func() time.Time
	Output        io.Writer
	OutlineOutput io.Writer
}

// ImageSettings is the type-safe image-mode settings surface for ImageRequest.
type ImageSettings struct {
	i             settings.ImageGlobal
	background    bool
	backgroundSet bool
}

// NewImageSettings returns the default wkhtmltoimage settings.
func NewImageSettings() *ImageSettings {
	return &ImageSettings{
		i:             settings.DefaultImageGlobal(),
		background:    true,
		backgroundSet: false,
	}
}

// Set applies an image-mode settings key such as "width", "quality", or
// "format".
func (s *ImageSettings) Set(name, value string) error {
	if s == nil {
		return ErrNilImageSettings
	}

	key := normalizeImageSettingKey(name)

	var global settings.PdfGlobal

	if err := settings.ApplyImageKeyNormalized(&global, &s.i, key, value); err != nil {
		return fmt.Errorf("image set %q: %w", name, err)
	}

	if key == "background" || key == "web.background" {
		s.background = global.Background
		s.backgroundSet = true
	}

	return nil
}

// Get reads an image-mode setting by its dotted key.
func (s *ImageSettings) Get(name string) (string, bool) {
	if s == nil {
		return "", false
	}

	key := normalizeImageSettingKey(name)
	if key == "background" || key == "web.background" {
		return strconv.FormatBool(s.effectiveBackground()), true
	}

	return s.i.Get(key)
}

// effectiveBackground returns the shared body-paint switch represented by
// either accepted image setting alias. ImageGlobal intentionally does not own
// this PDF/image shared setting, so ImageSettings keeps it explicitly.
func (s *ImageSettings) effectiveBackground() bool {
	if s == nil || !s.backgroundSet {
		return true
	}

	return s.background
}

func normalizeImageSettingKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// ImageRequest is the type-safe public API for image-only conversions. It
// accepts one object because image mode renders one input document.
type ImageRequest struct {
	Global *GlobalSettings
	Image  *ImageSettings
	Object *ObjectSettings
	Now    func() time.Time
	Output io.Writer
}

// EnableLocalFileAccess enables global local file access and unblocks local
// access across all attached objects in the PDF request.
func (r *PDFRequest) EnableLocalFileAccess() *PDFRequest {
	if r == nil {
		return nil
	}

	if r.Global == nil {
		r.Global = NewGlobalSettings()
	}

	r.Global.EnableLocalFileAccess()

	for _, obj := range r.Objects {
		if obj != nil {
			obj.EnableLocalFileAccess()
		}
	}

	return r
}

// EnableLocalFileAccess enables global local file access and unblocks local
// access on the attached image object.
func (r *ImageRequest) EnableLocalFileAccess() *ImageRequest {
	if r == nil {
		return nil
	}

	if r.Global == nil {
		r.Global = NewGlobalSettings()
	}

	r.Global.EnableLocalFileAccess()

	if r.Object != nil {
		r.Object.EnableLocalFileAccess()
	}

	return r
}

// ValidatePDF checks the public typed PDF request contract without starting
// the renderer. It is safe to call before output files or other resources are
// opened. The request must have a document sink, copies must be positive, and
// at least one body object must be present; TOC-only and empty objects are
// rejected.
//
//nolint:cyclop // sequential validation checks
func (r *PDFRequest) ValidatePDF() error {
	if r == nil {
		return ErrNilPDFRequest
	}

	if r.Output == nil {
		return ErrMissingPDFOutput
	}

	global := settings.DefaultPdfGlobal()
	if r.Global != nil {
		global = r.Global.g
	}

	if global.DumpOutline && r.OutlineOutput == nil {
		return ErrMissingPDFOutlineOutput
	}

	if global.PageSize != "" {
		if _, _, err := settings.ParsePageSize(global.PageSize); err != nil {
			return fmt.Errorf("%w: %q", ErrInvalidPageSize, global.PageSize)
		}
	}

	if global.PdfProfile != "" {
		if _, err := settings.ParsePDFProfile(global.PdfProfile); err != nil {
			return err //nolint:wrapcheck // sentinel error from settings package
		}
	}

	if global.PdfVersion != "" {
		if _, err := settings.ParsePDFVersion(global.PdfVersion); err != nil {
			return err //nolint:wrapcheck // sentinel error from settings package
		}
	}

	if _, err := convert.PolicyForGlobal(global); err != nil {
		return err //nolint:wrapcheck // validation error from convert package
	}

	if global.Copies < 1 {
		return ErrInvalidPDFCopies
	}

	objects := r.toRequest().Objects

	return validatePDFObjects(objects)
}

// ValidateImage checks the public typed image request contract without
// starting the renderer. ImageSettings and Global are optional and receive
// defaults, but an image request needs one renderable body object and a sink.
func (r *ImageRequest) ValidateImage() error {
	if r == nil {
		return ErrNilImageRequest
	}

	if r.Output == nil {
		return ErrMissingImageOutput
	}

	object := settings.PdfObject{} //nolint:exhaustruct // zero object is the invalid-input sentinel
	if r.Object != nil {
		object = r.Object.o
	}

	return validateImageObjects([]settings.PdfObject{object})
}

func (r *PDFRequest) toRequest() *convert.PDFRequest {
	if r == nil {
		return nil
	}

	global := settings.DefaultPdfGlobal()
	if r.Global != nil {
		global = clonePdfGlobal(r.Global.g)
	}

	objects := make([]settings.PdfObject, len(r.Objects))

	for idx, object := range r.Objects {
		if object != nil {
			objects[idx] = clonePdfObject(object.o)
		}
	}

	return &convert.PDFRequest{
		Global:        global,
		Objects:       objects,
		Now:           r.Now,
		Output:        r.Output,
		OutlineOutput: r.OutlineOutput,
	}
}

func (r *ImageRequest) toRequest() *imageout.Request {
	if r == nil {
		return nil
	}

	global := settings.DefaultPdfGlobal()
	if r.Global != nil {
		global = clonePdfGlobal(r.Global.g)
	}

	imageSettings := settings.DefaultImageGlobal()

	if r.Image != nil {
		imageSettings = cloneImageGlobal(r.Image.i)

		if r.Image.backgroundSet {
			global.Background = r.Image.background
		}
	}

	var object settings.PdfObject
	if r.Object != nil {
		object = clonePdfObject(r.Object.o)
	}

	return &imageout.Request{
		Global:  global,
		Image:   imageSettings,
		Objects: []settings.PdfObject{object},
		Now:     r.Now,
		Output:  r.Output,
	}
}

// RunPDF is a one-shot typed PDF conversion. It accepts a PDFRequest that
// cannot carry image-mode settings, providing compile-time safety. The
// conversion runs synchronously; cancel ctx to abort. Output bytes are
// written to req.Output.
func RunPDF(ctx context.Context, req *PDFRequest) error {
	if ctx == nil {
		return ErrNilContext
	}

	if req == nil {
		return ErrNilPDFRequest
	}

	if err := req.ValidatePDF(); err != nil {
		return err
	}

	if err := convert.RunTypedPDF(ctx, req.toRequest(), nil, nil); err != nil {
		return fmt.Errorf("pdf: %w", err)
	}

	return nil
}

// RunImage is a one-shot typed image conversion. It accepts an ImageRequest
// that cannot carry PDF-specific multi-object or outline settings, providing
// compile-time safety. The conversion runs synchronously; cancel ctx to abort.
// Output bytes (PNG or JPEG) are written to req.Output.
func RunImage(ctx context.Context, req *ImageRequest) error {
	if ctx == nil {
		return ErrNilContext
	}

	if req == nil {
		return ErrNilImageRequest
	}

	if err := req.ValidateImage(); err != nil {
		return err
	}

	if err := imageout.RunRequest(ctx, req.toRequest(), nil); err != nil {
		return fmt.Errorf("image: %w", err)
	}

	return nil
}

// validatePDFObjects adapts the engine's shared object invariant to the
// root-package error vocabulary. The engine remains independent of this
// package, while every public entry point exposes errors.Is-compatible errors.
func validatePDFObjects(objects []settings.PdfObject) error {
	if err := convert.ValidateRenderableObjects(objects); err != nil {
		return ErrNoRenderablePDFObjects
	}

	return nil
}

func validateImageObjects(objects []settings.PdfObject) error {
	if err := convert.ValidateRenderableObjects(objects); err != nil {
		return ErrNoInputPageAdded
	}

	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// lineLog is an io.Writer that classifies newline-terminated log lines and
// forwards them to the converter callbacks. Severity is decided by
// line.SeverityOf (the marker-prefix grammar owned by internal/line); the
// callback mapping stays here.
type lineLog struct {
	buf     bytes.Buffer
	onInfo  func(string)
	onWarn  func(string)
	onError func(string)
}

func (w *lineLog) Write(payload []byte) (int, error) {
	w.buf.Write(payload)

	for {
		raw := w.buf.Bytes()

		nlIdx := bytes.IndexByte(raw, '\n')
		if nlIdx < 0 {
			break
		}

		logLine := strings.TrimSpace(string(raw[:nlIdx]))
		w.buf.Next(nlIdx + 1)

		if logLine == "" {
			continue
		}

		switch line.SeverityOf(logLine) {
		case line.Warn:
			if w.onWarn != nil {
				w.onWarn(logLine)
			}
		case line.Error:
			if w.onError != nil {
				w.onError(logLine)
			}
		case line.Info:
			if w.onInfo != nil {
				w.onInfo(logLine)
			}
		}
	}

	return len(payload), nil
}
