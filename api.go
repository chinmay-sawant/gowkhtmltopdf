// Library API: wkhtmltopdf-compatible HTML-to-PDF and HTML-to-image
// conversion, exposing the internal convert/imageout pipelines through an
// idiomatic, stdlib-only Go surface.
package gowkhtmltopdf

import (
	"bytes"
	"context"
	"errors"
	"strings"

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/convert"
	"gowkhtmltopdf/internal/imageout"
	"gowkhtmltopdf/internal/settings"
)

// LibraryVersion is the wkhtmltopdf release this library tracks; upstream
// 0.12.x compatibility surface.
const LibraryVersion = "0.12.7-dev"

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
	return s.g.Set(name, value)
}

// Get reads a dotted settings key back as its canonical string form via the
// same key table as Set (plus Policy A Ignored values). ok is false when the
// name is unknown. Booleans read as "true"/"false", enums as their canonical
// name ("Portrait", "print", "abort", …), floats with the shortest round-trip
// representation.
func (s *GlobalSettings) Get(name string) (string, bool) {
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
	return s.o.Set(name, value)
}

// Get reads a dotted object settings key via the same key table as Set.
func (s *ObjectSettings) Get(name string) (string, bool) {
	return s.o.Get(name)
}

// SetPage sets the input page (a path, URL or "inline:…" / "data:…" source)
// and returns s so calls can be chained.
func (s *ObjectSettings) SetPage(page string) *ObjectSettings {
	s.o.Page = page
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
	// classified by content ("warning:" prefixed lines go to OnWarn,
	// error lines to OnError, everything else - including phase lines - to
	// OnInfo). They may be nil.
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
	return &Converter{global: NewGlobalSettings()}
}

// Global returns the converter's global settings, for Set/Get.
func (c *Converter) Global() *GlobalSettings {
	return c.global
}

// AddObject appends a page object and returns c for chaining. The object's
// settings are copied, so later mutations of s do not affect the converter.
// A converter needs at least one object to convert.
func (c *Converter) AddObject(s *ObjectSettings) *Converter {
	cp := *s
	c.objects = append(c.objects, &cp)
	return c
}

// Convert runs the conversion. The produced bytes replace the previous
// Output. ctx is threaded into every load; cancel it to abort. Errors are
// also reported to OnError when set. Output is captured via an in-memory
// writer (no temp file).
func (c *Converter) Convert(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(c.objects) == 0 {
		return errors.New("gowkhtmltopdf: no page objects added")
	}
	var buf bytes.Buffer
	cmd := &cli.Command{Global: c.global.g, OutputWriter: &buf}
	for _, o := range c.objects {
		cmd.Objects = append(cmd.Objects, o.o)
	}

	flush := &lineLog{
		onInfo:  c.OnInfo,
		onWarn:  c.OnWarn,
		onError: c.OnError,
	}
	var progress func(phase string, percent int)
	if c.OnPhase != nil || c.OnProgress != nil {
		progress = func(phase string, percent int) {
			if c.OnPhase != nil {
				c.OnPhase(phase)
			}
			if c.OnProgress != nil {
				c.OnProgress(percent)
			}
		}
	}
	if err := convert.RunPDFContext(ctx, cmd, flush, progress); err != nil {
		if c.OnError != nil {
			c.OnError(err.Error())
		}
		return err
	}
	c.output = buf.Bytes()
	return nil
}

// Output returns the PDF bytes produced by the last successful Convert, or
// nil if none ran yet. The returned slice is a copy; it stays valid across
// later conversions.
func (c *Converter) Output() []byte {
	return append([]byte(nil), c.output...)
}

// ConvertHTML is a one-shot helper: convert an in-memory HTML document to
// PDF bytes without creating a temp input file. global may be nil (defaults
// apply). Local file / subresource ACL is unchanged — linked local assets
// still need enablelocalfileaccess + load.blocklocalfileaccess=false.
func ConvertHTML(ctx context.Context, html []byte, global *GlobalSettings) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(html) == 0 {
		return nil, errors.New("gowkhtmltopdf: empty HTML")
	}
	c := NewConverter()
	if global != nil {
		c.global = global
	}
	obj := NewObjectSettings().SetPage(string(html))
	c.AddObject(obj)
	if err := c.Convert(ctx); err != nil {
		return nil, err
	}
	return c.Output(), nil
}

// ---------------------------------------------------------------------------
// Image conversion
// ---------------------------------------------------------------------------

// ImageConverter drives one HTML-to-image conversion (the wkhtmltoimage
// counterpart): it renders the first added page and encodes it as PNG or
// JPEG. Configure with Set/AddObject, then Convert produces the encoded
// bytes via Output(). Not safe for concurrent Convert calls.
type ImageConverter struct {
	global *GlobalSettings
	image  settings.ImageGlobal
	object *ObjectSettings
	output []byte

	// OnInfo, OnWarn and OnError receive the conversion's log lines,
	// classified like Converter's. They may be nil.
	OnInfo  func(string)
	OnWarn  func(string)
	OnError func(string)
}

// NewImageConverter returns an ImageConverter with the wkhtmltoimage default
// settings: 1024 px smart-width viewport, PNG output.
func NewImageConverter() *ImageConverter {
	return &ImageConverter{
		global: NewGlobalSettings(),
		image:  settings.DefaultImageGlobal(),
	}
}

// Set applies an image-mode settings key: "width", "height", "quality",
// "smartwidth", "transparent", "format" ("png"|"jpg") and the crop keys
// "crop.left", "crop.top", "crop.width", "crop.height". Unknown names return
// an error. "web.background" also updates the shared Global.Background paint
// switch so image and PDF share one background field.
func (c *ImageConverter) Set(name, value string) error {
	key := strings.ToLower(strings.TrimSpace(name))
	// Sole paint field is Global.Background (image has no Web.Background).
	if key == "web.background" || key == "background" {
		return c.global.g.Set("background", value)
	}
	return c.image.Set(name, value)
}

// Global returns the shared global settings (only "enablelocalfileaccess"
// and "allow" influence image conversion, via the loader ACL).
func (c *ImageConverter) Global() *GlobalSettings {
	return c.global
}

// AddObject sets the page to convert (a path, URL, or "inline:"/"data:"
// source) and returns c for chaining. Image conversion renders the first
// added page only. The page's load settings can be adjusted through Object.
func (c *ImageConverter) AddObject(page string) *ImageConverter {
	o := NewObjectSettings()
	o.SetPage(page)
	c.object = o
	return c
}

// Object returns the settings of the page to convert (the one passed to the
// most recent AddObject), so per-page load options can be set - for example
// "load.blocklocalfileaccess" = "false" to allow local inputs.
func (c *ImageConverter) Object() *ObjectSettings {
	return c.object
}

// Convert runs the conversion, replacing the previous Output. ctx is threaded
// into the load; cancel it to abort. Errors are also reported to OnError.
// Output is captured via an in-memory writer (no temp file).
func (c *ImageConverter) Convert(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if c.object == nil || strings.TrimSpace(c.object.o.Page) == "" {
		return errors.New("gowkhtmltopdf: no input page added")
	}
	var buf bytes.Buffer
	cmd := &cli.Command{
		Global:       c.global.g,
		Image:        c.image,
		Objects:      []settings.PdfObject{c.object.o},
		OutputWriter: &buf,
	}
	flush := &lineLog{
		onInfo:  c.OnInfo,
		onWarn:  c.OnWarn,
		onError: c.OnError,
	}
	if err := imageout.Run(ctx, cmd, flush); err != nil {
		if c.OnError != nil {
			c.OnError(err.Error())
		}
		return err
	}
	c.output = buf.Bytes()
	return nil
}

// Output returns the encoded image bytes (PNG, or JPEG when format was set
// to "jpg") from the last successful Convert, or nil if none ran yet. The
// returned slice is a copy.
func (c *ImageConverter) Output() []byte {
	return append([]byte(nil), c.output...)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// lineLog is an io.Writer that classifies newline-terminated log lines and
// forwards them to the converter callbacks: "warning:" lines to onWarn,
// error lines to onError, everything else to onInfo.
type lineLog struct {
	buf     bytes.Buffer
	onInfo  func(string)
	onWarn  func(string)
	onError func(string)
}

func (w *lineLog) Write(p []byte) (int, error) {
	w.buf.Write(p)
	for {
		raw := w.buf.Bytes()
		i := bytes.IndexByte(raw, '\n')
		if i < 0 {
			break
		}
		line := strings.TrimSpace(string(raw[:i]))
		w.buf.Next(i + 1)
		if line == "" {
			continue
		}
		switch lower := strings.ToLower(line); {
		case strings.Contains(lower, "warning:"):
			if w.onWarn != nil {
				w.onWarn(line)
			}
		case strings.Contains(lower, "error"):
			if w.onError != nil {
				w.onError(line)
			}
		default:
			if w.onInfo != nil {
				w.onInfo(line)
			}
		}
	}
	return len(p), nil
}
