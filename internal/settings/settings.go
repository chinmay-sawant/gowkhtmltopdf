package settings

import (
	"errors"
	"strings"
)

const (
	defaultMarginMM       = 10
	defaultHeaderFontSize = 12
	defaultTOCFontScale   = 0.8
	defaultOutlineDepth   = 4
	defaultImageWidth     = 1024
	defaultImageQuality   = 94
	sAbort                = "abort"
)

// ColorMode mirrors wkhtmltopdf --color-mode. Kept as a parse helper for
// Set("colormode"); the engine stores only PdfGlobal.Grayscale.
//
// ponytail: ColorMode is not a stored field — convert reads Grayscale only.
type ColorMode int

const (
	ColorModeColor ColorMode = iota
	ColorModeGrayscale
)

func (m ColorMode) String() string {
	if m == ColorModeGrayscale {
		return "grayscale"
	}

	return "color"
}

// ParseColorMode accepts "color" (default) or "grayscale".
func ParseColorMode(value string) (ColorMode, error) {
	switch value {
	case "", "color":
		return ColorModeColor, nil
	case "grayscale":
		return ColorModeGrayscale, nil
	}

	return ColorModeColor, errInvalid("color-mode", value, "color|grayscale")
}

// Orientation mirrors wkhtmltopdf --orientation.
type Orientation int

const (
	OrientationPortrait Orientation = iota
	OrientationLandscape
)

func (o Orientation) String() string {
	if o == OrientationLandscape {
		return "Landscape"
	}

	return "Portrait"
}

// ParseOrientation accepts "portrait" or "landscape" (case-insensitive).
func ParseOrientation(value string) (Orientation, error) {
	switch normalize(value) {
	case "", "portrait":
		return OrientationPortrait, nil
	case "landscape":
		return OrientationLandscape, nil
	}

	return OrientationPortrait, errInvalid("orientation", value, "portrait|landscape")
}

// LoadErrorHandling mirrors wkhtmltopdf --load-error-handling.
type LoadErrorHandling int

const (
	LoadErrorAbort LoadErrorHandling = iota
	LoadErrorSkip
	LoadErrorIgnore
)

func (h LoadErrorHandling) String() string {
	switch h {
	case LoadErrorAbort:
		return sAbort
	case LoadErrorSkip:
		return "skip"
	case LoadErrorIgnore:
		return sIgnore
	}

	return sAbort
}

// ParseLoadErrorHandling accepts abort|skip|ignore.
func ParseLoadErrorHandling(value string) (LoadErrorHandling, error) {
	switch normalize(value) {
	case "", sAbort:
		return LoadErrorAbort, nil
	case "skip":
		return LoadErrorSkip, nil
	case "ignore":
		return LoadErrorIgnore, nil
	}

	return LoadErrorAbort, errInvalid("load-error-handling", value, "abort|skip|ignore")
}

// MediaType mirrors wkhtmltopdf --print-media-type (screen|print) and the
// --media-type override. Consumed by image mode (imageout.mediaFor).
type MediaType int

const (
	MediaIgnore MediaType = iota
	MediaScreen
	MediaPrint
)

func (m MediaType) String() string {
	switch m {
	case MediaIgnore:
		return sIgnore
	case MediaScreen:
		return sScreen
	case MediaPrint:
		return sPrint
	}

	return sIgnore
}

// ResolveMedia computes the effective CSS media type: the print-media-type
// override (either home) wins, then the object media-type, then the global
// media-type, falling back to base (the mode default: "print" for PDF,
// "screen" for image). obj may be nil.
func ResolveMedia(base string, global Web, obj *Web) string {
	if global.PrintMediaType || obj != nil && obj.PrintMediaType {
		return sPrint
	}

	media := global.MediaType

	if obj != nil && obj.MediaType != MediaIgnore {
		media = obj.MediaType
	}

	switch media {
	case MediaPrint:
		return sPrint
	case MediaScreen:
		return sScreen
	case MediaIgnore:
	}

	return base
}

// ResolvePDFMedia resolves layout CSS media for PDF mode via ResolveMedia.
// PDF default is "print".
func ResolvePDFMedia(glob PdfGlobal, obj *PdfObject) string {
	var objWeb *Web

	if obj != nil {
		objView := Web{ //nolint:exhaustruct // intentional zero-value fields
			PrintMediaType: obj.Load.PrintMediaType || obj.Web.PrintMediaType,
			MediaType:      obj.Load.MediaType,
		}
		if obj.Web.MediaType != MediaIgnore {
			objView.MediaType = obj.Web.MediaType
		}

		objWeb = &objView
	}

	return ResolveMedia(sPrint, glob.Web, objWeb)
}

// ResolveImageMedia resolves layout CSS media for Image mode via ResolveMedia.
// Image default is "screen".
func ResolveImageMedia(global PdfGlobal, image ImageGlobal, obj *PdfObject) string {
	web := image.Web
	if global.Web.PrintMediaType {
		web.PrintMediaType = true
	}

	if web.MediaType == MediaIgnore {
		web.MediaType = global.Web.MediaType
	}

	var objWeb *Web

	if obj != nil {
		objView := Web{ //nolint:exhaustruct // intentional zero/partial fields
			PrintMediaType: obj.Load.PrintMediaType || obj.Web.PrintMediaType,
			MediaType:      obj.Load.MediaType,
		}
		if obj.Web.MediaType != MediaIgnore {
			objView.MediaType = obj.Web.MediaType
		}

		objWeb = &objView
	}

	return ResolveMedia(sScreen, web, objWeb)
}

// Margin holds the four page margins in millimetres.
type Margin struct {
	Top    float64
	Bottom float64
	Left   float64
	Right  float64
}

// DefaultMargins match pdfsettings.cc: 10 mm on all sides.
func DefaultMargins() Margin {
	return Margin{Top: defaultMarginMM, Bottom: defaultMarginMM, Left: defaultMarginMM, Right: defaultMarginMM}
}

// Size holds optional custom page dimensions in millimetres (0 = unset).
type Size struct {
	Width  float64 // mm; 0 = unset
	Height float64 // mm; 0 = unset
}

// Web holds web-behaviour settings that the engine actually consults.
// Body paint background is PdfGlobal.Background only (not a Web field).
// Inert wkhtml keys (javascript, plugins, user-style-sheet, …) are accepted
// via Set into Ignored maps — not typed fields (Policy A).
type Web struct {
	Images bool
	// PrintMediaType / MediaType: image mode media selection (imageout.mediaFor).
	// PDF convert uses mediaFor with object/global load+web fields.
	PrintMediaType bool
	MediaType      MediaType
	// SimplifyDOM opts into chrome-strip heuristics for URL/print mode
	// (--simplify-dom). Default false so invoice/report HTML is unchanged.
	SimplifyDOM bool
	// SimplifyDOMProfile selects extra chrome-strip selectors when SimplifyDOM
	// is on. Empty = landmarks only; "mediawiki" adds #mw-navigation / .mw-jump-link.
	SimplifyDOMProfile string
	// PrintLinkUnderline opts into underlining a[href] after cascade
	// (--print-link-underline). Default false — author text-decoration wins.
	PrintLinkUnderline bool
}

// LoadGlobal holds load settings shared by all page loads. NewLoader applies
// the full policy (proxy, allow prefixes, local-access flag) in one place.
type LoadGlobal struct {
	Proxy                 string
	Allow                 []string // local ACL prefixes (--allow)
	EnableLocalFileAccess bool
	// NetworkPolicySet distinguishes the explicit network policy from the
	// compatibility default used by existing CLI and library callers.
	NetworkPolicySet      bool
	NetworkAllowedSchemes []string
	NetworkAllowedHosts   []string
	NetworkBlockPrivate   bool
	NetworkBlockCrossHost bool
}

// LoadPage holds per-page load settings with engine consumers in load/convert.
// JS/plugin/encoding stubs are not typed; Set routes them to Ignored.
type LoadPage struct {
	ZoomFactor           float64
	BlockLocalFileAccess bool
	LoadErrorHandling    LoadErrorHandling
	Username             string
	Password             string
	CustomHeaders        map[string]string
	Cookies              map[string]string
	Post                 []PostItem
	MediaType            MediaType
	PrintMediaType       bool
	Timeout              int // seconds; 0 = default
	// InlineHTML is an in-memory HTML document source (SetBody); when set it
	// replaces Page as the input and skips URL guessing entirely. InlineBase
	// resolves relative subresources (load.Load).
	InlineHTML []byte
	InlineBase string
}

// PostItem is one urlencoded form field for POST loads.
type PostItem struct {
	Name  string
	Value string
}

// HeaderFooter mirrors the text/HTML header & footer settings.
type HeaderFooter struct {
	FontSize float64
	FontName string
	Left     string
	Right    string
	Center   string
	Line     bool
	Spacing  float64
	HTMLURL  string
	Replace  map[string]string
}

// DefaultHeaderFooter matches pdfsettings.cc defaults (Arial 12, no spacing).
func DefaultHeaderFooter() HeaderFooter {
	return HeaderFooter{ //nolint:exhaustruct // intentional zero/partial fields
		FontSize: defaultHeaderFontSize,
		FontName: "Arial",
		Spacing:  0,
	}
}

// TableOfContent mirrors TOC object settings.
type TableOfContent struct {
	FontScale     float64
	Indentation   string
	DottedLines   bool
	CaptionText   string
	ForwardLinks  bool
	BackLinks     bool
	XSLStyleSheet string
}

// DefaultTableOfContent matches pdfsettings.cc defaults.
func DefaultTableOfContent() TableOfContent {
	return TableOfContent{ //nolint:exhaustruct // intentional zero/partial fields
		FontScale:   defaultTOCFontScale,
		Indentation: "1em",
		DottedLines: true,
		CaptionText: "Table of Contents",
	}
}

// PdfGlobal is the PDF-mode global settings struct.
//
// Policy A: only fields with convert/load/imageout consumers (or CLI homes that
// convert still reads) are typed. Inert wkhtml keys may land in Ignored.
type PdfGlobal struct {
	// Page geometry: named size plus optional custom Size width/height (mm).
	// Custom Size.Width/Height overrides the named size when both are > 0.
	PageSize    string
	Size        Size
	Orientation Orientation
	// Grayscale is the sole color control convert reads (doc.SetGrayscale).
	// Set("colormode") / Set("grayscale") both write this field.
	Grayscale    bool
	PageOffset   int
	Copies       int
	Collate      bool
	Outline      bool
	OutlineDepth int
	// DumpOutline / DumpDefaultTOCXSL: one home is Global settings (CLI and
	// library both write it); the engine reads it only.
	DumpOutline        bool
	DumpDefaultTOCXSL  bool
	UseCompression     bool
	Title              string
	Margin             Margin
	SmartShrinking     bool
	Footer             HeaderFooter
	Header             HeaderFooter
	TOC                TableOfContent
	Background         bool // sole paint switch for PDF + image body backgrounds
	ExcludeFromOutline []string
	Quiet              bool
	Web                Web
	// Load carries the shared load policy: Proxy, Allow (ACL prefixes) and
	// EnableLocalFileAccess live on LoadGlobal, applied by load.NewLoader.
	Load                 LoadGlobal
	FontPaths            []string // --font-path directories (opt-in TTF discovery)
	UseSystemFonts       bool     // --use-system-fonts
	ResolveRelativeLinks bool     // resolve relative <a href> against page URL
	// Ignored holds accepted-but-inert wkhtml keys (dpi, javascript, …).
	// ponytail: Policy A sink — do not re-add typed stubs without engine consumers.
	Ignored map[string]string
}

// DefaultPdfGlobal returns the pdfsettings.cc-compatible defaults for fields
// the engine actually uses.
func DefaultPdfGlobal() PdfGlobal {
	return PdfGlobal{ //nolint:exhaustruct // intentional zero/partial fields
		PageSize:       "A4",
		Orientation:    OrientationPortrait,
		Copies:         1,
		Collate:        true,
		Outline:        true,
		OutlineDepth:   defaultOutlineDepth,
		UseCompression: true,
		Margin:         DefaultMargins(),
		SmartShrinking: true,
		Footer:         DefaultHeaderFooter(),
		Header:         DefaultHeaderFooter(),
		TOC:            DefaultTableOfContent(),
		Background:     true,
		Web: Web{ //nolint:exhaustruct // intentional zero/partial fields
			Images: true,
		},
		ResolveRelativeLinks: true,
	}
}

// PdfObject is one page/cover/toc object's settings.
type PdfObject struct {
	ExternalLinks    bool
	LocalLinks       bool
	IncludeInOutline bool
	Page             string // URL or path or "-"
	IsTableOfContent bool
	IsCover          bool
	Header           HeaderFooter
	Footer           HeaderFooter
	HeaderSet        bool // true when object-level header overrides exist
	FooterSet        bool // true when object-level footer overrides exist
	TOC              TableOfContent
	Load             LoadPage
	Web              Web
	UseOutline       bool
	// Ignored holds accepted-but-inert object/load/web keys (Policy A).
	Ignored map[string]string
}

// HeaderFor returns the effective header: object override or global.
func (o *PdfObject) HeaderFor(g PdfGlobal) HeaderFooter {
	if o.HeaderSet {
		return o.Header
	}

	return g.Header
}

// FooterFor returns the effective footer: object override or global.
func (o *PdfObject) FooterFor(g PdfGlobal) HeaderFooter {
	if o.FooterSet {
		return o.Footer
	}

	return g.Footer
}

// ErrNoRenderableObjects reports a conversion job whose object list has no
// renderable page source (all empty pages or only TOC objects).
var ErrNoRenderableObjects = errors.New("settings: no renderable page objects")

// ValidateRenderableObjects checks that objects contains at least one non-TOC
// object with a non-empty page source or inline HTML.
func ValidateRenderableObjects(objects []PdfObject) error {
	for _, object := range objects {
		if object.IsTableOfContent {
			continue
		}

		if strings.TrimSpace(object.Page) != "" || len(object.Load.InlineHTML) > 0 {
			return nil
		}
	}

	return ErrNoRenderableObjects
}

// DefaultPdfObject matches pdfsettings.cc defaults for engine-consumed fields.
func DefaultPdfObject() PdfObject {
	return PdfObject{ //nolint:exhaustruct // intentional zero/partial fields
		ExternalLinks:    true,
		LocalLinks:       true,
		IncludeInOutline: true,
		UseOutline:       true,
		Load:             DefaultLoadPage(),
	}
}

// DefaultLoadPage matches loadsettings.cc defaults for engine-consumed fields:
// blockLocalFileAccess true, load error abort. JS-delay and similar stubs are
// not typed (Policy A).
func DefaultLoadPage() LoadPage {
	return LoadPage{ //nolint:exhaustruct // intentional zero/partial fields
		BlockLocalFileAccess: true,
		LoadErrorHandling:    LoadErrorAbort,
	}
}

// ImageGlobal is the image-mode global settings struct (wkhtmltoimage).
// Quiet lives on PdfGlobal (Command.Global.Quiet); imageout uses that bit.
type ImageGlobal struct {
	Width       int
	Height      int
	Quality     int
	SmartWidth  bool
	Crop        CropSettings
	Format      string // "" = sniff from output; "png"|"jpg"|"jpeg"
	Transparent bool
	Web         Web
	Load        LoadGlobal
	// Ignored holds accepted-but-inert image keys (Policy A).
	Ignored map[string]string
}

// CropSettings mirrors wkhtmltoimage crop settings.
type CropSettings struct {
	Left   int
	Top    int
	Width  int
	Height int
}

// DefaultImageGlobal matches imagesettings.cc defaults for engine fields.
func DefaultImageGlobal() ImageGlobal {
	return ImageGlobal{ //nolint:exhaustruct // intentional zero/partial fields
		Width:      defaultImageWidth,
		Height:     0,
		Quality:    defaultImageQuality,
		SmartWidth: true,
		Crop:       CropSettings{Left: -1, Top: -1, Width: -1, Height: -1},
		Web: Web{ //nolint:exhaustruct // intentional zero/partial fields
			Images: true,
		},
	}
}

func normalize(s string) string { return strings.ToLower(s) }
