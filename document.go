package gowkhtmltopdf

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/imageout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// Content identifies exactly one document source. HTML is an in-memory
// document and Base resolves its relative resources; File and URL are loaded
// by the engine without public inline/data prefixes.
type Content struct {
	HTML []byte
	Base string
	File string
	URL  string
}

// HTML returns an owned in-memory HTML source. An optional base URL resolves
// relative resources referenced by the document.
func HTML(html []byte, base ...string) Content {
	owned := make([]byte, len(html))
	copy(owned, html)

	var baseURL string
	if len(base) > 0 {
		baseURL = base[0]
	}

	return Content{HTML: owned, Base: baseURL, File: "", URL: ""}
}

// File returns a local filesystem document source.
func File(path string) Content {
	return Content{HTML: nil, Base: "", File: path, URL: ""}
}

// URL returns an HTTP(S) document source.
func URL(rawURL string) Content {
	return Content{HTML: nil, Base: "", File: "", URL: rawURL}
}

// Margin holds page margins in millimetres.
type Margin struct {
	Top    float64
	Right  float64
	Bottom float64
	Left   float64
}

// HeaderFooter holds text or HTML header/footer settings.
type HeaderFooter struct {
	Left     string
	Center   string
	Right    string
	FontSize float64
	FontName string
	Line     bool
	Spacing  float64
	HTMLURL  string
}

// Page is one cover or body page in a Document.
type Page struct {
	Source           Content
	Header           *HeaderFooter
	Footer           *HeaderFooter
	IncludeInOutline *bool
	ExternalLinks    *bool
	LocalLinks       *bool
}

// TOC configures the generated table-of-contents object.
type TOC struct {
	Caption      string
	DottedLines  *bool
	FontScale    float64
	Indentation  string
	ForwardLinks *bool
	BackLinks    *bool
}

// Crop identifies an image crop rectangle in pixels.
type Crop struct {
	Left   int
	Top    int
	Width  int
	Height int
}

// Document is the preferred HTML-to-PDF API. Its zero-valued options retain
// the engine defaults; a document must contain a valid Cover or body Page.
type Document struct {
	Cover *Page
	TOC   *TOC
	Pages []Page

	PageSize    string
	WidthMM     float64
	HeightMM    float64
	Orientation string
	Margin      Margin
	Title       string
	PDFVersion  string
	PDFProfile  string

	Copies          int
	Collate         bool
	Outline         *bool
	OutlineDepth    int
	Background      *bool
	SmartShrinking  *bool
	Compression     *bool
	ResolveRelLinks *bool
	Header          *HeaderFooter
	Footer          *HeaderFooter

	AllowLocalFiles      bool
	FontPaths            []string
	UseSystemFonts       bool
	UseMetricFontAliases bool
	Network              *NetworkPolicy

	Now        func() time.Time
	OnInfo     func(string)
	OnWarn     func(string)
	OnError    func(string)
	OnPhase    func(string)
	OnProgress func(int)
}

// ImageDocument is the preferred HTML-to-image API.
type ImageDocument struct {
	Source Content

	Width       int
	Height      int
	Format      string
	Quality     int
	SmartWidth  *bool
	Transparent bool
	Crop        *Crop

	AllowLocalFiles      bool
	Background           *bool
	FontPaths            []string
	UseSystemFonts       bool
	UseMetricFontAliases bool
	Network              *NetworkPolicy

	Now        func() time.Time
	OnInfo     func(string)
	OnWarn     func(string)
	OnError    func(string)
	OnPhase    func(string)
	OnProgress func(int)
}

const documentObjectCapacity = 2

// NewDocument returns a document containing the supplied body pages. Page
// source bytes are copied so the helper does not retain caller-owned HTML.
//
//nolint:exhaustruct // zero-valued options intentionally inherit engine defaults.
func NewDocument(pages ...Page) *Document {
	owned := make([]Page, len(pages))
	for index, page := range pages {
		owned[index] = clonePage(page)
	}

	return &Document{
		Cover: nil,
		TOC:   nil,
		Pages: owned,
	}
}

// WritePDF validates d, maps it to the PDF engine request, and writes a PDF
// to w. The public and internal settings are cloned at this boundary.
func (d *Document) WritePDF(ctx context.Context, w io.Writer) error {
	return d.writePDF(ctx, w, nil, false)
}

// WritePDFOutline writes the PDF and its outline XML to separate sinks.
func (d *Document) WritePDFOutline(ctx context.Context, pdfWriter, outlineWriter io.Writer) error {
	if d == nil {
		return ErrNilDocument
	}

	if outlineWriter == nil {
		return reportPreflight(d.OnError, ErrMissingPDFOutlineOutput)
	}

	return d.writePDF(ctx, pdfWriter, outlineWriter, true)
}

func (d *Document) writePDF(
	ctx context.Context,
	pdfWriter io.Writer,
	outlineWriter io.Writer,
	dumpOutline bool,
) error {
	if d == nil {
		return ErrNilDocument
	}

	if err := d.Validate(); err != nil {
		return reportPreflight(d.OnError, err)
	}

	if pdfWriter == nil {
		return reportPreflight(d.OnError, ErrMissingPDFOutput)
	}

	req := d.toPDFRequest(pdfWriter, outlineWriter, dumpOutline)
	hooks := convertHooks{
		OnInfo:     d.OnInfo,
		OnWarn:     d.OnWarn,
		OnError:    d.OnError,
		OnPhase:    d.OnPhase,
		OnProgress: d.OnProgress,
	}

	return hooks.executePDFTo(ctx, req.ToRequest())
}

// PDF returns the PDF bytes produced by the document.
func (d *Document) PDF(ctx context.Context) ([]byte, error) {
	if d == nil {
		return nil, ErrNilDocument
	}

	var output bytes.Buffer
	if err := d.WritePDF(ctx, &output); err != nil {
		return nil, err
	}

	return append([]byte(nil), output.Bytes()...), nil
}

// WriteImage validates d, maps it to the image engine request, and writes
// encoded image bytes to w.
//
//nolint:wsl,mnd // the image lifecycle reports its documented 0-to-100 range.
func (d *ImageDocument) WriteImage(ctx context.Context, writer io.Writer) error {
	if d == nil {
		return ErrNilImageDocument
	}

	if err := d.Validate(); err != nil {
		return reportPreflight(d.OnError, err)
	}

	if writer == nil {
		return reportPreflight(d.OnError, ErrMissingImageOutput)
	}

	req := d.toImageRequest(writer)
	hooks := convertHooks{
		OnInfo:     d.OnInfo,
		OnWarn:     d.OnWarn,
		OnError:    d.OnError,
		OnPhase:    d.OnPhase,
		OnProgress: d.OnProgress,
	}

	if d.OnPhase != nil {
		d.OnPhase("Rendering image")
	}
	if d.OnProgress != nil {
		d.OnProgress(0)
	}

	if err := hooks.executeImageTo(ctx, req); err != nil {
		return err
	}

	if d.OnProgress != nil {
		d.OnProgress(100)
	}
	if d.OnPhase != nil {
		d.OnPhase("Done")
	}

	return nil
}

// Image returns encoded PNG or JPEG bytes produced by the image document.
func (d *ImageDocument) Image(ctx context.Context) ([]byte, error) {
	if d == nil {
		return nil, ErrNilImageDocument
	}

	var output bytes.Buffer
	if err := d.WriteImage(ctx, &output); err != nil {
		return nil, err
	}

	return append([]byte(nil), output.Bytes()...), nil
}

func (d *Document) toPDFRequest(output, outline io.Writer, dumpOutline bool) *convert.PDFRequest {
	global := d.pdfGlobal(dumpOutline)
	objects := make([]settings.PdfObject, 0, len(d.Pages)+documentObjectCapacity)

	if d.Cover != nil {
		objects = append(objects, d.mapPage(*d.Cover, true))
	}

	if d.TOC != nil {
		objects = append(objects, d.mapTOC(*d.TOC))
	}

	for _, page := range d.Pages {
		objects = append(objects, d.mapPage(page, false))
	}

	return &convert.PDFRequest{
		Global:        global,
		Objects:       objects,
		Now:           d.Now,
		Output:        output,
		OutlineOutput: outline,
	}
}

//nolint:cyclop,funlen,wsl // one adapter mirrors the documented global options.
func (d *Document) pdfGlobal(dumpOutline bool) settings.PdfGlobal {
	global := settings.DefaultPdfGlobal()

	if d.PageSize != "" {
		global.PageSize = d.PageSize
	}
	if d.WidthMM != 0 || d.HeightMM != 0 {
		global.Size = settings.Size{Width: d.WidthMM, Height: d.HeightMM}
	}
	if d.Orientation != "" {
		orientation, _ := settings.ParseOrientation(d.Orientation)
		global.Orientation = orientation
	}
	if d.Margin != (Margin{Top: 0, Right: 0, Bottom: 0, Left: 0}) {
		global.Margin = settings.Margin{
			Top:    d.Margin.Top,
			Right:  d.Margin.Right,
			Bottom: d.Margin.Bottom,
			Left:   d.Margin.Left,
		}
	}
	if d.Title != "" {
		global.Title = d.Title
	}
	if d.PDFVersion != "" {
		global.PdfVersion, _ = settings.ParsePDFVersion(d.PDFVersion)
	}
	if d.PDFProfile != "" {
		global.PdfProfile, _ = settings.ParsePDFProfile(d.PDFProfile)
	}
	if d.Copies != 0 {
		global.Copies = d.Copies
		global.Collate = d.Collate
	}
	if d.Outline != nil {
		global.Outline = *d.Outline
	}
	if d.OutlineDepth != 0 {
		global.OutlineDepth = d.OutlineDepth
	}
	if d.Background != nil {
		global.Background = *d.Background
	}
	if d.SmartShrinking != nil {
		global.SmartShrinking = *d.SmartShrinking
	}
	if d.Compression != nil {
		global.UseCompression = *d.Compression
	}
	if d.ResolveRelLinks != nil {
		global.ResolveRelativeLinks = *d.ResolveRelLinks
	}
	if d.Header != nil {
		global.Header = mapHeaderFooter(*d.Header)
	}
	if d.Footer != nil {
		global.Footer = mapHeaderFooter(*d.Footer)
	}
	if d.AllowLocalFiles {
		global.Load.EnableLocalFileAccess = true
	}
	global.FontPaths = documentCloneStrings(d.FontPaths)
	global.UseSystemFonts = d.UseSystemFonts
	global.UseMetricFontAliases = d.UseMetricFontAliases
	if d.Network != nil {
		load.ApplyNetworkPolicy(&global.Load, *d.Network)
	}
	global.DumpOutline = dumpOutline

	if d.TOC != nil {
		global.TOC = mapTOCSettings(*d.TOC)
	}

	return global
}

//nolint:wsl // page mapping follows the public option groups in order.
func (d *Document) mapPage(page Page, cover bool) settings.PdfObject {
	object := settings.DefaultPdfObject()
	mapContent(&object, page.Source)
	object.ExternalLinks = boolValue(page.ExternalLinks, object.ExternalLinks)
	object.LocalLinks = boolValue(page.LocalLinks, object.LocalLinks)
	object.IncludeInOutline = boolValue(page.IncludeInOutline, object.IncludeInOutline)
	object.IsCover = cover

	if cover && page.IncludeInOutline == nil {
		object.IncludeInOutline = false
	}
	if page.Header != nil {
		object.Header = mapHeaderFooter(*page.Header)
		object.HeaderSet = true
	}
	if page.Footer != nil {
		object.Footer = mapHeaderFooter(*page.Footer)
		object.FooterSet = true
	}
	if d.AllowLocalFiles {
		object.Load.BlockLocalFileAccess = false
	}

	return object
}

//nolint:wsl // TOC mapping follows the public option groups in order.
func (d *Document) mapTOC(toc TOC) settings.PdfObject {
	object := settings.DefaultPdfObject()
	object.Page = ""
	object.IsTableOfContent = true
	object.UseOutline = false
	object.TOC = mapTOCSettings(toc)
	object.IncludeInOutline = false
	if d.AllowLocalFiles {
		object.Load.BlockLocalFileAccess = false
	}

	return object
}

//nolint:cyclop,wsl // image mapping mirrors the public image option groups.
func (d *ImageDocument) toImageRequest(output io.Writer) *imageout.Request {
	global := settings.DefaultPdfGlobal()
	image := settings.DefaultImageGlobal()

	if d.Background != nil {
		global.Background = *d.Background
	}
	if d.AllowLocalFiles {
		global.Load.EnableLocalFileAccess = true
	}
	global.FontPaths = documentCloneStrings(d.FontPaths)
	global.UseSystemFonts = d.UseSystemFonts
	global.UseMetricFontAliases = d.UseMetricFontAliases
	if d.Network != nil {
		load.ApplyNetworkPolicy(&global.Load, *d.Network)
	}

	if d.Width != 0 {
		image.Width = d.Width
	}
	if d.Height != 0 {
		image.Height = d.Height
	}
	if d.Quality != 0 {
		image.Quality = d.Quality
	}
	if d.Format != "" {
		image.Format = d.Format
	}
	if d.SmartWidth != nil {
		image.SmartWidth = *d.SmartWidth
	}
	image.Transparent = d.Transparent
	if d.Crop != nil {
		image.Crop = settings.CropSettings{
			Left:   d.Crop.Left,
			Top:    d.Crop.Top,
			Width:  d.Crop.Width,
			Height: d.Crop.Height,
		}
	}

	object := settings.DefaultPdfObject()
	mapContent(&object, d.Source)
	if d.AllowLocalFiles {
		object.Load.BlockLocalFileAccess = false
	}

	req := imageout.NewRequest(global, image, []settings.PdfObject{object}, output)
	req.Now = d.Now

	return req
}

func mapContent(object *settings.PdfObject, content Content) {
	switch {
	case content.HTML != nil:
		object.Page = ""
		object.Load.InlineHTML = cloneBytes(content.HTML)
		object.Load.InlineBase = content.Base
	case content.File != "":
		object.Page = content.File
	case content.URL != "":
		object.Page = content.URL
	}
}

//nolint:wsl // defaults are resolved before the complete internal value is built.
func mapHeaderFooter(header HeaderFooter) settings.HeaderFooter {
	defaults := settings.DefaultHeaderFooter()
	fontName := header.FontName
	if fontName == "" {
		fontName = defaults.FontName
	}
	fontSize := header.FontSize
	if fontSize == 0 {
		fontSize = defaults.FontSize
	}

	return settings.HeaderFooter{
		FontSize: fontSize,
		FontName: fontName,
		Left:     header.Left,
		Right:    header.Right,
		Center:   header.Center,
		Line:     header.Line,
		Spacing:  header.Spacing,
		HTMLURL:  header.HTMLURL,
		Replace:  nil,
	}
}

//nolint:wsl // optional public fields selectively override engine defaults.
func mapTOCSettings(toc TOC) settings.TableOfContent {
	defaults := settings.DefaultTableOfContent()
	result := defaults
	if toc.Caption != "" {
		result.CaptionText = toc.Caption
	}
	if toc.DottedLines != nil {
		result.DottedLines = *toc.DottedLines
	}
	if toc.FontScale != 0 {
		result.FontScale = toc.FontScale
	}
	if toc.Indentation != "" {
		result.Indentation = toc.Indentation
	}
	if toc.ForwardLinks != nil {
		result.ForwardLinks = *toc.ForwardLinks
	}
	if toc.BackLinks != nil {
		result.BackLinks = *toc.BackLinks
	}

	return result
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}

	return *value
}

func clonePage(page Page) Page {
	page.Source.HTML = cloneBytes(page.Source.HTML)
	page.Header = cloneHeaderFooter(page.Header)
	page.Footer = cloneHeaderFooter(page.Footer)

	return page
}

func cloneHeaderFooter(header *HeaderFooter) *HeaderFooter {
	if header == nil {
		return nil
	}

	clone := *header

	return &clone
}

func documentCloneStrings(src []string) []string {
	if src == nil {
		return nil
	}

	dst := make([]string, len(src))
	copy(dst, src)

	return dst
}
