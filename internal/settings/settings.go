// Package settings implements the wkhtmltopdf-compatible settings model:
// defaults, parsers, and the dotted-name Set surface used by the CLI and
// the future library API. Pure Go, no external dependencies.
package settings

// ColorMode mirrors wkhtmltopdf --color-mode.
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

// LogLevel mirrors wkhtmltopdf --log-level.
type LogLevel int

const (
	LogNone LogLevel = iota
	LogError
	LogWarn
	LogInfo
	LogDebug
)

func (l LogLevel) String() string {
	switch l {
	case LogError:
		return "error"
	case LogWarn:
		return "warn"
	case LogInfo:
		return "info"
	case LogDebug:
		return "debug"
	}
	return "none"
}

// ParseLogLevel accepts none|error|warn|info|debug.
func ParseLogLevel(s string) (LogLevel, error) {
	switch normalize(s) {
	case "", "none":
		return LogNone, nil
	case "error":
		return LogError, nil
	case "warn":
		return LogWarn, nil
	case "info":
		return LogInfo, nil
	case "debug":
		return LogDebug, nil
	}
	return LogNone, errInvalid("log-level", s, "none|error|warn|info|debug")
}

// MediaType mirrors wkhtmltopdf --print-media-type (screen|print) and the
// --media-type override.
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

// Size mirrors wkhtmltopdf page size settings.
type Size struct {
	PageSize string  // "A4", "Letter", …; empty = default
	Width    float64 // mm; 0 = unset
	Height   float64 // mm; 0 = unset
}

// Web holds web-behaviour settings (--web.* surface).
type Web struct {
	Background      bool
	Images          bool
	JavaScript      bool
	Java            bool
	Plugins         bool
	MinimumFontSize int
	DefaultEncoding string
	UserStyleSheet  string
	LoadImages      bool
	PrintMediaType  bool
	MediaType       MediaType
	// SimplifyDOM opts into chrome-strip heuristics for URL/print mode
	// (--simplify-dom). Default false so invoice/report HTML is unchanged.
	SimplifyDOM bool
}

// LoadGlobal holds settings shared by all loads.
type LoadGlobal struct {
	CookieJar         string
	Proxy             string
	LoadErrorHandling LoadErrorHandling
	CustomHeaders     map[string]string
	Cookies           map[string]string
}

// LoadPage holds per-page load settings.
type LoadPage struct {
	JSDelay              int
	ZoomFactor           float64
	BlockLocalFileAccess bool
	StopSlowScripts      bool
	DebugJavaScript      bool
	LoadErrorHandling    LoadErrorHandling
	Proxy                string
	Username             string
	Password             string
	CustomHeaders        map[string]string
	RepeatCustomHeaders  bool
	Cookies              map[string]string
	Post                 []PostItem
	WindowStatus         string
	RunScript            string
	MediaType            MediaType
	PrintMediaType       bool
	DefaultEncoding      string
	ExternalLinks        bool
	LocalLinks           bool
	EnablePlugins        bool
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
type PdfGlobal struct {
	PageSize              string
	Size                  Size
	Orientation           Orientation
	ColorMode             ColorMode
	DPI                   int
	ImageDPI              int
	ImageQuality          int
	PageOffset            int
	Copies                int
	Collate               bool
	Outline               bool
	OutlineDepth          int
	DumpOutline           bool
	DumpDefaultTOCXSL     bool
	UseCompression        bool
	Title                 string
	Margin                Margin
	SmartShrinking        bool
	Footer                HeaderFooter
	Header                HeaderFooter
	TOC                   TableOfContent
	Background            bool
	EnableLocalFileAccess bool
	Allow                 []string
	ExcludeFromOutline    []string
	ProduceForms          bool
	Grayscale             bool
	LowQuality            bool
	ReadArgsFromStdin     bool
	Quiet                 bool
	LogLevel              LogLevel
	UseXServer            bool
	PageWidth             float64 // mm; --page-width override
	PageHeight            float64 // mm; --page-height override
	Web                   Web
	Load                  LoadGlobal
	DefaultEncoding       string
	FontPaths             []string // --font-path directories (opt-in TTF discovery)
	UseSystemFonts        bool     // --use-system-fonts
	ResolveRelativeLinks  bool     // resolve relative <a href> against page URL
}

// DefaultPdfGlobal returns the pdfsettings.cc-compatible defaults.
func DefaultPdfGlobal() PdfGlobal {
	return PdfGlobal{
		PageSize:       "A4",
		Size:           Size{PageSize: "A4"},
		Orientation:    OrientationPortrait,
		ColorMode:      ColorModeColor,
		DPI:            96,
		ImageDPI:       600,
		ImageQuality:   94,
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
		LogLevel:       LogInfo,
		Web: Web{
			Background:      true,
			Images:          true,
			JavaScript:      true,
			Plugins:         false,
			DefaultEncoding: "utf-8",
		},
		ResolveRelativeLinks: true,
	}
}

// PdfObject is one page/cover/toc object's settings.
type PdfObject struct {
	ExternalLinks    bool
	LocalLinks       bool
	IncludeInOutline bool
	PagesCount       bool
	ProduceForms     bool
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

// DefaultPdfObject matches pdfsettings.cc defaults.
func DefaultPdfObject() PdfObject {
	return PdfObject{
		ExternalLinks:    true,
		LocalLinks:       true,
		IncludeInOutline: true,
		PagesCount:       true,
		UseOutline:       true,
		Load:             DefaultLoadPage(),
	}
}

// DefaultLoadPage matches loadsettings.cc defaults: jsdelay 200,
// blockLocalFileAccess true, stopSlowScripts true, load error abort.
func DefaultLoadPage() LoadPage {
	return LoadPage{
		JSDelay:              200,
		BlockLocalFileAccess: true,
		StopSlowScripts:      true,
		LoadErrorHandling:    LoadErrorAbort,
	}
}

// ImageGlobal is the image-mode global settings struct (wkhtmltoimage).
type ImageGlobal struct {
	Width       int
	Height      int
	Quality     int
	SmartWidth  bool
	Crop        CropSettings
	Format      string // "" = sniff from output; "png"|"jpg"|"jpeg"
	Transparent bool
	LogLevel    LogLevel
	Quiet       bool
	Web         Web
	Load        LoadGlobal
}

// CropSettings mirrors wkhtmltoimage crop settings.
type CropSettings struct {
	Left   int
	Top    int
	Width  int
	Height int
}

// DefaultImageGlobal matches imagesettings.cc defaults.
func DefaultImageGlobal() ImageGlobal {
	return ImageGlobal{
		Width:      1024,
		Height:     0,
		Quality:    94,
		SmartWidth: true,
		Crop:       CropSettings{Left: -1, Top: -1, Width: -1, Height: -1},
		Web: Web{
			Background:      true,
			Images:          true,
			JavaScript:      true,
			Plugins:         false,
			DefaultEncoding: "utf-8",
		},
		LogLevel: LogInfo,
	}
}

func normalize(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out = append(out, c)
	}
	return string(out)
}
