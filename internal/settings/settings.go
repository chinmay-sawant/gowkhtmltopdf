package settings

import "strings"

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
func ParseColorMode(s string) (ColorMode, error) {
	switch s {
	case "", "color":
		return ColorModeColor, nil
	case "grayscale":
		return ColorModeGrayscale, nil
	}
	return ColorModeColor, errInvalid("color-mode", s, "color|grayscale")
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
func ParseOrientation(s string) (Orientation, error) {
	switch normalize(s) {
	case "", "portrait":
		return OrientationPortrait, nil
	case "landscape":
		return OrientationLandscape, nil
	}
	return OrientationPortrait, errInvalid("orientation", s, "portrait|landscape")
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
	case LoadErrorSkip:
		return "skip"
	case LoadErrorIgnore:
		return "ignore"
	}
	return "abort"
}

// ParseLoadErrorHandling accepts abort|skip|ignore.
func ParseLoadErrorHandling(s string) (LoadErrorHandling, error) {
	switch normalize(s) {
	case "", "abort":
		return LoadErrorAbort, nil
	case "skip":
		return LoadErrorSkip, nil
	case "ignore":
		return LoadErrorIgnore, nil
	}
	return LoadErrorAbort, errInvalid("load-error-handling", s, "abort|skip|ignore")
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
	case MediaScreen:
		return "screen"
	case MediaPrint:
		return "print"
	}
	return "ignore"
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
	return Margin{Top: 10, Bottom: 10, Left: 10, Right: 10}
}

// Size holds optional custom page dimensions in millimetres (0 = unset).
// Named sizes live on PdfGlobal.PageSize; Size.PageSize is dual-written by
// Set for convert.pageGeometry until that path collapses to one field.
//
// ponytail: PageSize name is mirrored on PdfGlobal.PageSize and Size.PageSize.
type Size struct {
	PageSize string  // "A4", "Letter", …; empty = default
	Width    float64 // mm; 0 = unset
	Height   float64 // mm; 0 = unset
}

// Web holds web-behaviour settings that the engine actually consults.
// Inert wkhtml keys (javascript, plugins, user-style-sheet, …) are accepted
// via Set into Ignored maps — not typed fields (Policy A).
type Web struct {
	Background bool
	Images     bool
	// PrintMediaType / MediaType: image mode media selection (imageout.mediaFor).
	// PDF convert currently hardcodes print media — see convert package.
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

// LoadGlobal holds load settings shared by all page loads. Only Proxy is
// consumed by internal/load today.
type LoadGlobal struct {
	Proxy string
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
	return HeaderFooter{
		FontSize: 12,
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
	return TableOfContent{
		FontScale:   0.8,
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
	// Page geometry: named size + optional custom Size width/height (mm).
	// pageGeometry prefers PageSize, then Size.PageSize; custom Size.Width/Height
	// override the named size when both are > 0.
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
	// DumpOutline / DumpDefaultTOCXSL: settings home is Global (not Command).
	// convert still ORs Command.DumpOutline for CLI dual-write until cli collapses.
	DumpOutline           bool
	DumpDefaultTOCXSL     bool
	UseCompression        bool
	Title                 string
	Margin                Margin
	SmartShrinking        bool
	Footer                HeaderFooter
	Header                HeaderFooter
	TOC                   TableOfContent
	Background            bool // sole paint switch for PDF + image body backgrounds
	EnableLocalFileAccess bool
	Allow                 []string
	ExcludeFromOutline    []string
	Quiet                 bool
	Web                   Web
	Load                  LoadGlobal
	FontPaths             []string // --font-path directories (opt-in TTF discovery)
	UseSystemFonts        bool     // --use-system-fonts
	ResolveRelativeLinks  bool     // resolve relative <a href> against page URL
	// Ignored holds accepted-but-inert wkhtml keys (dpi, javascript, …).
	// ponytail: Policy A sink — do not re-add typed stubs without engine consumers.
	Ignored map[string]string
}

// DefaultPdfGlobal returns the pdfsettings.cc-compatible defaults for fields
// the engine actually uses.
func DefaultPdfGlobal() PdfGlobal {
	return PdfGlobal{
		PageSize:       "A4",
		Size:           Size{PageSize: "A4"},
		Orientation:    OrientationPortrait,
		Copies:         1,
		Collate:        true,
		Outline:        true,
		OutlineDepth:   4,
		UseCompression: true,
		Margin:         DefaultMargins(),
		SmartShrinking: true,
		Footer:         DefaultHeaderFooter(),
		Header:         DefaultHeaderFooter(),
		TOC:            DefaultTableOfContent(),
		Background:     true,
		Web: Web{
			Background: true,
			Images:     true,
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

// DefaultPdfObject matches pdfsettings.cc defaults for engine-consumed fields.
func DefaultPdfObject() PdfObject {
	return PdfObject{
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
	return LoadPage{
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
	return ImageGlobal{
		Width:      1024,
		Height:     0,
		Quality:    94,
		SmartWidth: true,
		Crop:       CropSettings{Left: -1, Top: -1, Width: -1, Height: -1},
		Web: Web{
			Background: true,
			Images:     true,
		},
	}
}

func normalize(s string) string { return strings.ToLower(s) }
