// Library API: wkhtmltopdf-compatible HTML-to-PDF and HTML-to-image
// conversion, exposing the internal convert/imageout pipelines through an
// idiomatic, stdlib-only Go surface.
package gowkhtmltopdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"gowkhtmltopdf/internal/convert"
	"gowkhtmltopdf/internal/imageout"
	"gowkhtmltopdf/internal/line"
	"gowkhtmltopdf/internal/settings"
)

// LibraryVersion is the wkhtmltopdf release this library tracks; upstream
// 0.12.x compatibility surface.
const LibraryVersion = "0.12.7-dev"

// Static conversion errors; callers can match with errors.Is.
var (
	// ErrNoPageObjectsAdded reports a PDF conversion without any page objects.
	ErrNoPageObjectsAdded = errors.New("gowkhtmltopdf: no page objects added")
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
	// ErrNilObjectSettings reports a method call on nil object settings.
	ErrNilObjectSettings = errors.New("gowkhtmltopdf: nil object settings")
	// ErrNilContext reports a cancellation-aware operation without a context.
	ErrNilContext = errors.New("gowkhtmltopdf: nil context")
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

// NewGlobalSettings returns the wkhtmltopdf-compatible default global
// settings (A4 portrait, 10 mm margins, background on, …).
func NewGlobalSettings() *GlobalSettings {
	return &GlobalSettings{g: settings.DefaultPdfGlobal()}
}

// Set applies a dotted settings key ("size.pagesize", "margin.top",
// "orientation", "web.background", "load.timeout", …). Unknown names return
// an error.
func (s *GlobalSettings) Set(name, value string) error {
	if s == nil {
		return ErrNilGlobalSettings
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

// AddObject appends a page object and returns c for chaining. The object's
// settings are copied, so later mutations of s do not affect the converter.
// A converter needs at least one object to convert. A nil settings pointer is
// ignored so an optional object cannot panic the conversion setup.
func (c *Converter) AddObject(s *ObjectSettings) *Converter {
	if c == nil || s == nil {
		return c
	}

	cp := &ObjectSettings{o: clonePdfObject(s.o)}
	c.objects = append(c.objects, cp)

	return c
}

// clonePdfObject creates an ownership boundary at the public API. PdfObject
// contains maps and slices in load, header/footer, ignored-setting, and
// inline-document fields; a struct assignment alone would let callers mutate
// a converter after AddObject returned.
func clonePdfObject(src settings.PdfObject) settings.PdfObject {
	dst := src
	dst.Header = cloneHeaderFooter(src.Header)
	dst.Footer = cloneHeaderFooter(src.Footer)
	dst.Load.CustomHeaders = cloneStringMap(src.Load.CustomHeaders)
	dst.Load.Cookies = cloneStringMap(src.Load.Cookies)
	dst.Load.Post = clonePostItems(src.Load.Post)
	dst.Load.InlineHTML = cloneBytes(src.Load.InlineHTML)
	dst.Ignored = cloneStringMap(src.Ignored)

	return dst
}

func clonePdfGlobal(src settings.PdfGlobal) settings.PdfGlobal {
	dst := src
	dst.Header = cloneHeaderFooter(src.Header)
	dst.Footer = cloneHeaderFooter(src.Footer)
	dst.Load.Allow = cloneStrings(src.Load.Allow)
	dst.ExcludeFromOutline = cloneStrings(src.ExcludeFromOutline)
	dst.FontPaths = cloneStrings(src.FontPaths)
	dst.Ignored = cloneStringMap(src.Ignored)

	return dst
}

func cloneHeaderFooter(src settings.HeaderFooter) settings.HeaderFooter {
	dst := src
	dst.Replace = cloneStringMap(src.Replace)

	return dst
}

func clonePostItems(src []settings.PostItem) []settings.PostItem {
	if src == nil {
		return nil
	}

	dst := make([]settings.PostItem, len(src))
	copy(dst, src)

	return dst
}

func cloneStrings(src []string) []string {
	if src == nil {
		return nil
	}

	dst := make([]string, len(src))
	copy(dst, src)

	return dst
}

func cloneStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}

	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}

	return dst
}

func cloneBytes(src []byte) []byte {
	if src == nil {
		return nil
	}

	dst := make([]byte, len(src))
	copy(dst, src)

	return dst
}

// AddHTML appends an in-memory HTML document as a page object and returns c
// for chaining. baseURL resolves relative subresources; an empty baseURL
// leaves them unresolvable. Unlike SetPage, the bytes are always treated as
// a document - no URL guessing is applied.
func (c *Converter) AddHTML(page []byte, baseURL string) *Converter {
	c.AddObject(NewObjectSettings().SetBody(page, baseURL))

	return c
}

// Convert runs the conversion. The produced bytes replace the previous
// Output. ctx is threaded into every load; cancel it to abort. Errors are
// also reported to OnError when set. Output is captured via an in-memory
// writer (no temp file).
func (c *Converter) Convert(ctx context.Context) error {
	if c == nil {
		return ErrNilConverter
	}

	if c.global == nil {
		c.global = NewGlobalSettings()
	}

	if len(c.objects) == 0 {
		return ErrNoPageObjectsAdded
	}

	objects := make([]settings.PdfObject, len(c.objects))
	for i, o := range c.objects {
		objects[i] = o.o
	}

	req := convert.NewPDFRequest(c.global.g, objects, nil, nil)
	h := convertHooks{
		OnInfo: c.OnInfo, OnWarn: c.OnWarn, OnError: c.OnError,
		OnPhase: c.OnPhase, OnProgress: c.OnProgress,
	}

	out, err := h.executePDF(ctx, req)
	if err != nil {
		return err
	}

	c.output = out

	return nil
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
//nolint:contextcheck // defensive nil-context contract
func ConvertHTML(ctx context.Context, html []byte, global *GlobalSettings) ([]byte, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}

	if len(html) == 0 {
		return nil, ErrEmptyHTML
	}

	conv := NewConverter()
	if global != nil {
		conv.global = &GlobalSettings{g: clonePdfGlobal(global.g)}
	}

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
// counterpart): it renders the first added page and encodes it as PNG or
// JPEG. Configure with Set/AddObject, then Convert produces the encoded
// bytes via Output(). Not safe for concurrent Convert calls.
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

// Global returns the shared global settings (only "enablelocalfileaccess"
// and "allow" influence image conversion, via the loader ACL).
func (c *ImageConverter) Global() *GlobalSettings {
	if c == nil {
		return nil
	}

	c.ensureDefaults()

	return c.global
}

// AddObject sets the page to convert (a path, URL, or "inline:"/"data:"
// source) and returns c for chaining. Image conversion renders the first
// added page only. The page's load settings can be adjusted through Object.
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
// into the load; cancel it to abort. Errors are also reported to OnError.
// Output is captured via an in-memory writer (no temp file).
//
// The page may be a path/URL via AddObject/SetPage, or in-memory HTML via
// Object().SetBody (P2-04 InlineHTML source kind).
func (c *ImageConverter) Convert(ctx context.Context) error {
	if c == nil {
		return ErrNilImageConverter
	}

	c.ensureDefaults()

	if strings.TrimSpace(c.object.o.Page) == "" && len(c.object.o.Load.InlineHTML) == 0 {
		return ErrNoInputPageAdded
	}

	img := c.image
	obj := clonePdfObject(c.object.o)
	req := convert.NewImageRequest(c.global.g, img, []settings.PdfObject{obj}, nil)
	h := convertHooks{ //nolint:exhaustruct // intentional zero/partial fields
		OnInfo: c.OnInfo, OnWarn: c.OnWarn, OnError: c.OnError,
	}

	out, err := h.executeImage(ctx, req)
	if err != nil {
		return err
	}

	c.output = out

	return nil
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

// executePDF runs the PDF pipeline into an in-memory buffer and reports
// failures to OnError.
//
//nolint:contextcheck // defensive nil-context contract
func (h convertHooks) executePDF(ctx context.Context, req *convert.Request) ([]byte, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}

	var buf bytes.Buffer

	req.Output = &buf
	if err := convert.Run(ctx, req, h.lineLog(), h.progress()); err != nil {
		if h.OnError != nil {
			h.OnError(err.Error())
		}

		return nil, fmt.Errorf("convert: %w", err)
	}

	return buf.Bytes(), nil
}

// executeImage runs the image pipeline into an in-memory buffer and reports
// failures to OnError.
//
//nolint:contextcheck // defensive nil-context contract
func (h convertHooks) executeImage(ctx context.Context, req *convert.Request) ([]byte, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}

	var buf bytes.Buffer

	req.Output = &buf
	if err := imageout.RunRequest(ctx, req, h.lineLog()); err != nil {
		if h.OnError != nil {
			h.OnError(err.Error())
		}

		return nil, fmt.Errorf("image convert: %w", err)
	}

	return buf.Bytes(), nil
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
