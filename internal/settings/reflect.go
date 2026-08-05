package settings

import (
	"fmt"
	"strconv"
	"strings"
)

// errInvalid is a small error helper for enum parse failures.
type invalidError struct {
	key, value, allowed string
}

func (e invalidError) Error() string {
	return fmt.Sprintf("invalid value %q for %s (allowed: %s)", e.value, e.key, e.allowed)
}

func errInvalid(key, value, allowed string) error {
	return invalidError{key: key, value: value, allowed: allowed}
}

// setter updates a target value. It receives the raw string value.
type setter func(raw string) error

// Global.Set applies a dotted settings key ("margin.top", "load.jsdelay",
// "web.background", …) to a PdfGlobal. Unknown keys return an error.
// Numeric suffixes are parsed with implied unit mm where upstream expects mm.
func (g *PdfGlobal) Set(name, value string) error {
	setters := globalSetters(g)
	key := normalizeDots(name)
	fn, ok := setters[key]
	if !ok {
		return fmt.Errorf("unknown global setting %q", name)
	}
	return fn(value)
}

// Object.Set applies a dotted settings key to a PdfObject. Object keys
// default to the load./web./header./footer./toc. namespaces.
func (o *PdfObject) Set(name, value string) error {
	setters := objectSetters(o)
	key := normalizeDots(name)
	fn, ok := setters[key]
	if !ok {
		return fmt.Errorf("unknown object setting %q", name)
	}
	return fn(value)
}

func normalizeDots(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// value helpers

func setBool(t *bool) setter {
	return func(raw string) error {
		switch strings.ToLower(raw) {
		case "", "true", "1", "yes", "on":
			*t = true
			return nil
		case "false", "0", "no", "off":
			*t = false
			return nil
		}
		return fmt.Errorf("invalid boolean %q", raw)
	}
}

func setFloat(t *float64) setter {
	return func(raw string) error {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return fmt.Errorf("invalid number %q", raw)
		}
		*t = v
		return nil
	}
}

func setInt(t *int) setter {
	return func(raw string) error {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("invalid integer %q", raw)
		}
		*t = v
		return nil
	}
}

func setString(t *string) setter {
	return func(raw string) error {
		*t = raw
		return nil
	}
}

func setStringDefault(t *string) setter {
	return func(raw string) error {
		*t = strings.TrimSpace(raw)
		return nil
	}
}

// enum/parser-using setters

func setOrientation(o *Orientation) setter {
	return func(raw string) error {
		v, err := ParseOrientation(raw)
		if err != nil {
			return err
		}
		*o = v
		return nil
	}
}

func setColorMode(m *ColorMode) setter {
	return func(raw string) error {
		v, err := ParseColorMode(raw)
		if err != nil {
			return err
		}
		*m = v
		return nil
	}
}

func setLoadErrorHandling(h *LoadErrorHandling) setter {
	return func(raw string) error {
		v, err := ParseLoadErrorHandling(raw)
		if err != nil {
			return err
		}
		*h = v
		return nil
	}
}

func setLogLevel(l *LogLevel) setter {
	return func(raw string) error {
		v, err := ParseLogLevel(raw)
		if err != nil {
			return err
		}
		*l = v
		return nil
	}
}

func setMediaType(m *MediaType) setter {
	return func(raw string) error {
		switch normalize(raw) {
		case "", "ignore":
			*m = MediaIgnore
			return nil
		case "screen":
			*m = MediaScreen
			return nil
		case "print":
			*m = MediaPrint
			return nil
		}
		return fmt.Errorf("invalid media type %q", raw)
	}
}

// marginSetter writes one edge of a Margin, storing millimetres.
func marginSetter(m *Margin, edge string) setter {
	return func(raw string) error {
		u, err := ParseUnitReal(raw, "mm")
		if err != nil {
			return err
		}
		mm, ok := u.Mm()
		if !ok {
			return fmt.Errorf("margin %s: unit %q not convertible", edge, u.Unit)
		}
		switch edge {
		case "top":
			m.Top = mm
		case "bottom":
			m.Bottom = mm
		case "left":
			m.Left = mm
		case "right":
			m.Right = mm
		}
		return nil
	}
}

func hfSetters(hf *HeaderFooter) map[string]setter {
	return map[string]setter{
		"fontsize": setFloat(&hf.FontSize),
		"fontname": setString(&hf.FontName),
		"left":     setString(&hf.Left),
		"right":    setString(&hf.Right),
		"center":   setString(&hf.Center),
		"line":     setBool(&hf.Line),
		"spacing":  setFloat(&hf.Spacing),
		"htmlurl":  setString(&hf.HTMLURL),
	}
}

func tocSetters(t *TableOfContent) map[string]setter {
	return map[string]setter{
		"fontscale":     setFloat(&t.FontScale),
		"indentation":   setStringDefault(&t.Indentation),
		"dottedlines":   setBool(&t.DottedLines),
		"captiontext":   setString(&t.CaptionText),
		"forwardlinks":  setBool(&t.ForwardLinks),
		"backlinks":     setBool(&t.BackLinks),
		"xslstylesheet": setString(&t.XSLStyleSheet),
	}
}

func loadSetters(l *LoadPage) map[string]setter {
	return map[string]setter{
		"jsdelay":               setInt(&l.JSDelay),
		"zoomfactor":            setFloat(&l.ZoomFactor),
		"blocklocalfileaccess":  setBool(&l.BlockLocalFileAccess),
		"stopslowscripts":       setBool(&l.StopSlowScripts),
		"debugjavascript":       setBool(&l.DebugJavaScript),
		"loaderrorhandling":     setLoadErrorHandling(&l.LoadErrorHandling),
		"proxy":                 setString(&l.Proxy),
		"username":              setString(&l.Username),
		"password":              setString(&l.Password),
		"repeatexternalheaders": setBool(&l.RepeatCustomHeaders),
		"repeatexternalcookies": setBool(&l.RepeatCustomHeaders),
		"windowstatus":          setString(&l.WindowStatus),
		"runscript":             setString(&l.RunScript),
		"mediatype":             setMediaType(&l.MediaType),
		"printmediatype":        setBool(&l.PrintMediaType),
		"defaultencoding":       setString(&l.DefaultEncoding),
		"externallinks":         setBool(&l.ExternalLinks),
		"locallinks":            setBool(&l.LocalLinks),
		"enableplugins":         setBool(&l.EnablePlugins),
		"timeout":               setInt(&l.Timeout),
	}
}

func webSetters(w *Web) map[string]setter {
	return map[string]setter{
		"background":      setBool(&w.Background),
		"images":          setBool(&w.Images),
		"javascript":      setBool(&w.JavaScript),
		"java":            setBool(&w.Java),
		"plugins":         setBool(&w.Plugins),
		"minimumfontsize": setInt(&w.MinimumFontSize),
		"defaultencoding": setString(&w.DefaultEncoding),
		"userstylesheet":  setString(&w.UserStyleSheet),
		"loadimages":      setBool(&w.LoadImages),
		"printmediatype":  setBool(&w.PrintMediaType),
		"mediatype":       setMediaType(&w.MediaType),
		"simplifydom":        setBool(&w.SimplifyDOM),
		"simplifydomprofile": setString(&w.SimplifyDOMProfile),
		"printlinkunderline": setBool(&w.PrintLinkUnderline),
	}
}

func globalSetters(g *PdfGlobal) map[string]setter {
	s := map[string]setter{
		"orientation":                  setOrientation(&g.Orientation),
		"colormode":                    setColorMode(&g.ColorMode),
		"resolution":                   setInt(&g.DPI),
		"dpi":                          setInt(&g.DPI),
		"pageoffset":                   setInt(&g.PageOffset),
		"copies":                       setInt(&g.Copies),
		"collate":                      setBool(&g.Collate),
		"outline":                      setBool(&g.Outline),
		"outlinedepth":                 setInt(&g.OutlineDepth),
		"dumpoutline":                  setBool(&g.DumpOutline),
		"dumpoutlinewithdefaulttocxsl": setBool(&g.DumpDefaultTOCXSL),
		"usecompression":               setBool(&g.UseCompression),
		"title":                        setString(&g.Title),
		"imagedpi":                     setInt(&g.ImageDPI),
		"imagequality":                 setInt(&g.ImageQuality),
		"loaderrorhandling":            setLoadErrorHandling(&g.Load.LoadErrorHandling),
		"smartshrinking":               setBool(&g.SmartShrinking),
		"background":                   setBool(&g.Background),
		"enablelocalfileaccess":        setBool(&g.EnableLocalFileAccess),
		"excludefromoutline":           appendString(&g.ExcludeFromOutline),
		"produceforms":                 setBool(&g.ProduceForms),
		"grayscale":                    setBool(&g.Grayscale),
		"lowquality":                   setBool(&g.LowQuality),
		"quiet":                        setBool(&g.Quiet),
		"log-level":                    setLogLevel(&g.LogLevel),
		"loglevel":                     setLogLevel(&g.LogLevel),
		"usexserver":                   setBool(&g.UseXServer),
		"readargsfromstdin":            setBool(&g.ReadArgsFromStdin),
		"defaultencoding":              setString(&g.DefaultEncoding),
		"cookiejar":                    setString(&g.Load.CookieJar),
		"proxy":                        setString(&g.Load.Proxy),
		"usesystemfonts":               setBool(&g.UseSystemFonts),
		"resolverelativelinks":         setBool(&g.ResolveRelativeLinks),
		"fontpath":                     appendString(&g.FontPaths),
	}
	// margins
	for _, e := range []string{"top", "bottom", "left", "right"} {
		s["margin."+e] = marginSetter(&g.Margin, e)
	}
	// page size
	s["size.pagesize"] = func(raw string) error {
		v := strings.TrimSpace(raw)
		g.PageSize = v
		g.Size.PageSize = v
		return nil
	}
	s["size.width"] = func(raw string) error {
		u, err := ParseUnitReal(raw, "mm")
		if err != nil {
			return err
		}
		mm, ok := u.Mm()
		if !ok {
			return fmt.Errorf("size.width: unit %q not convertible", u.Unit)
		}
		g.Size.Width, g.PageWidth = mm, mm
		return nil
	}
	s["size.height"] = func(raw string) error {
		u, err := ParseUnitReal(raw, "mm")
		if err != nil {
			return err
		}
		mm, ok := u.Mm()
		if !ok {
			return fmt.Errorf("size.height: unit %q not convertible", u.Unit)
		}
		g.Size.Height, g.PageHeight = mm, mm
		return nil
	}
	// allow list appends
	s["allow"] = appendString(&g.Allow)
	// header/footer
	for k, v := range hfSetters(&g.Header) {
		s["header."+k] = v
	}
	for k, v := range hfSetters(&g.Footer) {
		s["footer."+k] = v
	}
	// toc
	for k, v := range tocSetters(&g.TOC) {
		s["toc."+k] = v
	}
	// web
	for k, v := range webSetters(&g.Web) {
		s["web."+k] = v
	}
	return s
}

func appendString(dst *[]string) setter {
	return func(raw string) error {
		*dst = append(*dst, raw)
		return nil
	}
}

func objectSetters(o *PdfObject) map[string]setter {
	s := map[string]setter{
		"page":             setString(&o.Page),
		"externallinks":    setBool(&o.ExternalLinks),
		"locallinks":       setBool(&o.LocalLinks),
		"includeinoutline": setBool(&o.IncludeInOutline),
		"pagescount":       setBool(&o.PagesCount),
		"produceforms":     setBool(&o.ProduceForms),
		"useoutline":       setBool(&o.UseOutline),
		"istableofcontent": setBool(&o.IsTableOfContent),
		"iscover":          setBool(&o.IsCover),
	}
	for k, v := range loadSetters(&o.Load) {
		s["load."+k] = v
	}
	for k, v := range webSetters(&o.Web) {
		s["web."+k] = v
	}
	for k, v := range hfSetters(&o.Header) {
		s["header."+k] = func(k string, v setter) setter {
			return func(raw string) error {
				o.HeaderSet = true
				return v(raw)
			}
		}(k, v)
	}
	for k, v := range hfSetters(&o.Footer) {
		s["footer."+k] = func(k string, v setter) setter {
			return func(raw string) error {
				o.FooterSet = true
				return v(raw)
			}
		}(k, v)
	}
	for k, v := range tocSetters(&o.TOC) {
		s["toc."+k] = v
	}
	return s
}

// HttpErrorCode maps an HTTP status to the wkhtmltopdf exit-code convention
// (utilities.cc): 404 → 2, 401 → 3, everything else stays 1.
func HttpErrorCode(status int) int {
	switch status {
	case 404:
		return 2
	case 401:
		return 3
	}
	return 1
}

// ImageGlobal.Set applies an image-mode dotted settings key.
func (g *ImageGlobal) Set(name, value string) error {
	setters := map[string]setter{
		"width":       setInt(&g.Width),
		"height":      setInt(&g.Height),
		"quality":     setInt(&g.Quality),
		"smartwidth":  setBool(&g.SmartWidth),
		"transparent": setBool(&g.Transparent),
		"format":      setString(&g.Format),
		"crop.left":   setInt(&g.Crop.Left),
		"crop.top":    setInt(&g.Crop.Top),
		"crop.width":  setInt(&g.Crop.Width),
		"crop.height": setInt(&g.Crop.Height),
	}
	fn, ok := setters[normalizeDots(name)]
	if !ok {
		return fmt.Errorf("unknown image setting %q", name)
	}
	return fn(value)
}
