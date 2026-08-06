package settings

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
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

// globalApply applies a dotted key to a PdfGlobal.
type globalApply func(g *PdfGlobal, raw string) error

// objectApply applies a dotted key to a PdfObject.
type objectApply func(o *PdfObject, raw string) error

// imageApply applies a dotted key to an ImageGlobal.
type imageApply func(g *ImageGlobal, raw string) error

var (
	globalKeyTable map[string]globalApply
	objectKeyTable map[string]objectApply
	imageKeyTable  map[string]imageApply
	keysOnce       sync.Once
)

// ignoredGlobalKeys are wkhtml global keys with no engine consumer. Set accepts
// them into PdfGlobal.Ignored (Policy A) so scripts do not fail on known stubs.
var ignoredGlobalKeys = map[string]struct{}{
	"dpi":               {},
	"resolution":        {},
	"imagedpi":          {},
	"imagequality":      {},
	"lowquality":        {},
	"usexserver":        {},
	"readargsfromstdin": {},
	"log-level":         {},
	"loglevel":          {},
	"cookiejar":         {},
	"defaultencoding":   {},
	"produceforms":      {},
	// Global load-error-handling never reached LoadPage; only load.loaderrorhandling does.
	"loaderrorhandling": {},
	// web.* stubs (also listed under object web.)
	"web.javascript":      {},
	"web.java":            {},
	"web.plugins":         {},
	"web.minimumfontsize": {},
	"web.defaultencoding": {},
	"web.userstylesheet":  {},
	"web.loadimages":      {},
}

// ignoredObjectKeys are wkhtml object/load/web keys with no engine consumer.
var ignoredObjectKeys = map[string]struct{}{
	"pagescount":   {},
	"produceforms": {},
	// load.* stubs
	"load.jsdelay":               {},
	"load.stopslowscripts":       {},
	"load.debugjavascript":       {},
	"load.windowstatus":          {},
	"load.runscript":             {},
	"load.enableplugins":         {},
	"load.defaultencoding":       {},
	"load.proxy":                 {}, // only LoadGlobal.Proxy is wired
	"load.externallinks":         {}, // PdfObject.ExternalLinks is the real gate
	"load.locallinks":            {},
	"load.repeatexternalheaders": {},
	"load.repeatexternalcookies": {},
	// web.* stubs
	"web.javascript":      {},
	"web.java":            {},
	"web.plugins":         {},
	"web.minimumfontsize": {},
	"web.defaultencoding": {},
	"web.userstylesheet":  {},
	"web.loadimages":      {},
}

func ensureKeyTables() {
	keysOnce.Do(buildKeyTables)
}

// Global.Set applies a dotted settings key ("margin.top", "load.jsdelay",
// "web.background", …) to a PdfGlobal. Known inert keys are stored in
// Ignored and succeed; truly unknown keys return an error.
func (g *PdfGlobal) Set(name, value string) error {
	ensureKeyTables()
	key := normalizeDots(name)
	if fn, ok := globalKeyTable[key]; ok {
		return fn(g, value)
	}
	if _, ok := ignoredGlobalKeys[key]; ok {
		storeIgnored(&g.Ignored, key, value)
		return nil
	}
	return fmt.Errorf("unknown global setting %q", name)
}

// Object.Set applies a dotted settings key to a PdfObject. Known inert keys
// go to Ignored; unknown keys return an error.
func (o *PdfObject) Set(name, value string) error {
	ensureKeyTables()
	key := normalizeDots(name)
	if fn, ok := objectKeyTable[key]; ok {
		return fn(o, value)
	}
	if _, ok := ignoredObjectKeys[key]; ok {
		storeIgnored(&o.Ignored, key, value)
		return nil
	}
	return fmt.Errorf("unknown object setting %q", name)
}

// ImageGlobal.Set applies an image-mode dotted settings key.
func (g *ImageGlobal) Set(name, value string) error {
	ensureKeyTables()
	key := normalizeDots(name)
	if fn, ok := imageKeyTable[key]; ok {
		return fn(g, value)
	}
	// Image web.* stubs share the global ignored web list prefix.
	if _, ok := ignoredGlobalKeys[key]; ok {
		storeIgnored(&g.Ignored, key, value)
		return nil
	}
	return fmt.Errorf("unknown image setting %q", name)
}

func storeIgnored(dst *map[string]string, key, value string) {
	if *dst == nil {
		*dst = make(map[string]string)
	}
	(*dst)[key] = value
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

// setGrayscaleFromColorMode maps colormode strings onto the Grayscale bool.
func setGrayscaleFromColorMode(g *bool) setter {
	return func(raw string) error {
		m, err := ParseColorMode(raw)
		if err != nil {
			return err
		}
		*g = m == ColorModeGrayscale
		return nil
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

func appendString(dst *[]string) setter {
	return func(raw string) error {
		*dst = append(*dst, raw)
		return nil
	}
}

func buildKeyTables() {
	globalKeyTable = buildGlobalKeyTable()
	objectKeyTable = buildObjectKeyTable()
	imageKeyTable = buildImageKeyTable()
}

func buildGlobalKeyTable() map[string]globalApply {
	s := map[string]globalApply{
		"orientation": func(g *PdfGlobal, raw string) error {
			return setOrientation(&g.Orientation)(raw)
		},
		// colormode and grayscale both write Grayscale (convert's only reader).
		"colormode": func(g *PdfGlobal, raw string) error {
			return setGrayscaleFromColorMode(&g.Grayscale)(raw)
		},
		"grayscale": func(g *PdfGlobal, raw string) error {
			return setBool(&g.Grayscale)(raw)
		},
		"pageoffset": func(g *PdfGlobal, raw string) error {
			return setInt(&g.PageOffset)(raw)
		},
		"copies": func(g *PdfGlobal, raw string) error {
			return setInt(&g.Copies)(raw)
		},
		"collate": func(g *PdfGlobal, raw string) error {
			return setBool(&g.Collate)(raw)
		},
		"outline": func(g *PdfGlobal, raw string) error {
			return setBool(&g.Outline)(raw)
		},
		"outlinedepth": func(g *PdfGlobal, raw string) error {
			return setInt(&g.OutlineDepth)(raw)
		},
		"dumpoutline": func(g *PdfGlobal, raw string) error {
			return setBool(&g.DumpOutline)(raw)
		},
		"dumpoutlinewithdefaulttocxsl": func(g *PdfGlobal, raw string) error {
			return setBool(&g.DumpDefaultTOCXSL)(raw)
		},
		"usecompression": func(g *PdfGlobal, raw string) error {
			return setBool(&g.UseCompression)(raw)
		},
		"title": func(g *PdfGlobal, raw string) error {
			return setString(&g.Title)(raw)
		},
		"smartshrinking": func(g *PdfGlobal, raw string) error {
			return setBool(&g.SmartShrinking)(raw)
		},
		"background": func(g *PdfGlobal, raw string) error {
			return setBool(&g.Background)(raw)
		},
		"enablelocalfileaccess": func(g *PdfGlobal, raw string) error {
			return setBool(&g.EnableLocalFileAccess)(raw)
		},
		"excludefromoutline": func(g *PdfGlobal, raw string) error {
			return appendString(&g.ExcludeFromOutline)(raw)
		},
		"quiet": func(g *PdfGlobal, raw string) error {
			return setBool(&g.Quiet)(raw)
		},
		"proxy": func(g *PdfGlobal, raw string) error {
			return setString(&g.Load.Proxy)(raw)
		},
		"usesystemfonts": func(g *PdfGlobal, raw string) error {
			return setBool(&g.UseSystemFonts)(raw)
		},
		"resolverelativelinks": func(g *PdfGlobal, raw string) error {
			return setBool(&g.ResolveRelativeLinks)(raw)
		},
		"fontpath": func(g *PdfGlobal, raw string) error {
			return appendString(&g.FontPaths)(raw)
		},
		"allow": func(g *PdfGlobal, raw string) error {
			return appendString(&g.Allow)(raw)
		},
	}

	for _, e := range []string{"top", "bottom", "left", "right"} {
		edge := e
		s["margin."+edge] = func(g *PdfGlobal, raw string) error {
			return marginSetter(&g.Margin, edge)(raw)
		}
	}

	// Page size: dual-write PageSize name for convert.pageGeometry; dimensions
	// live only on Size (PageWidth/PageHeight top-level fields removed).
	s["size.pagesize"] = func(g *PdfGlobal, raw string) error {
		v := strings.TrimSpace(raw)
		g.PageSize = v
		g.Size.PageSize = v
		return nil
	}
	s["size.width"] = func(g *PdfGlobal, raw string) error {
		u, err := ParseUnitReal(raw, "mm")
		if err != nil {
			return err
		}
		mm, ok := u.Mm()
		if !ok {
			return fmt.Errorf("size.width: unit %q not convertible", u.Unit)
		}
		g.Size.Width = mm
		return nil
	}
	s["size.height"] = func(g *PdfGlobal, raw string) error {
		u, err := ParseUnitReal(raw, "mm")
		if err != nil {
			return err
		}
		mm, ok := u.Mm()
		if !ok {
			return fmt.Errorf("size.height: unit %q not convertible", u.Unit)
		}
		g.Size.Height = mm
		return nil
	}

	for _, k := range []string{"fontsize", "fontname", "left", "right", "center", "line", "spacing", "htmlurl"} {
		key := k
		s["header."+key] = func(g *PdfGlobal, raw string) error {
			return hfApply(&g.Header, key, raw)
		}
		s["footer."+key] = func(g *PdfGlobal, raw string) error {
			return hfApply(&g.Footer, key, raw)
		}
	}
	for _, k := range []string{"fontscale", "indentation", "dottedlines", "captiontext", "forwardlinks", "backlinks", "xslstylesheet"} {
		key := k
		s["toc."+key] = func(g *PdfGlobal, raw string) error {
			return tocApply(&g.TOC, key, raw)
		}
	}
	for _, k := range []string{
		"background", "images", "printmediatype", "mediatype",
		"simplifydom", "simplifydomprofile", "printlinkunderline",
	} {
		key := k
		s["web."+key] = func(g *PdfGlobal, raw string) error {
			return webApply(&g.Web, key, raw)
		}
	}
	return s
}

func buildObjectKeyTable() map[string]objectApply {
	s := map[string]objectApply{
		"page": func(o *PdfObject, raw string) error {
			return setString(&o.Page)(raw)
		},
		"externallinks": func(o *PdfObject, raw string) error {
			return setBool(&o.ExternalLinks)(raw)
		},
		"locallinks": func(o *PdfObject, raw string) error {
			return setBool(&o.LocalLinks)(raw)
		},
		"includeinoutline": func(o *PdfObject, raw string) error {
			return setBool(&o.IncludeInOutline)(raw)
		},
		"useoutline": func(o *PdfObject, raw string) error {
			return setBool(&o.UseOutline)(raw)
		},
		"istableofcontent": func(o *PdfObject, raw string) error {
			return setBool(&o.IsTableOfContent)(raw)
		},
		"iscover": func(o *PdfObject, raw string) error {
			return setBool(&o.IsCover)(raw)
		},
	}

	for _, k := range []string{
		"zoomfactor", "blocklocalfileaccess", "loaderrorhandling",
		"username", "password", "mediatype", "printmediatype", "timeout",
	} {
		key := k
		s["load."+key] = func(o *PdfObject, raw string) error {
			return loadApply(&o.Load, key, raw)
		}
	}
	for _, k := range []string{
		"background", "images", "printmediatype", "mediatype",
		"simplifydom", "simplifydomprofile", "printlinkunderline",
	} {
		key := k
		s["web."+key] = func(o *PdfObject, raw string) error {
			return webApply(&o.Web, key, raw)
		}
	}
	for _, k := range []string{"fontsize", "fontname", "left", "right", "center", "line", "spacing", "htmlurl"} {
		key := k
		s["header."+key] = func(o *PdfObject, raw string) error {
			o.HeaderSet = true
			return hfApply(&o.Header, key, raw)
		}
		s["footer."+key] = func(o *PdfObject, raw string) error {
			o.FooterSet = true
			return hfApply(&o.Footer, key, raw)
		}
	}
	for _, k := range []string{"fontscale", "indentation", "dottedlines", "captiontext", "forwardlinks", "backlinks", "xslstylesheet"} {
		key := k
		s["toc."+key] = func(o *PdfObject, raw string) error {
			return tocApply(&o.TOC, key, raw)
		}
	}
	return s
}

func buildImageKeyTable() map[string]imageApply {
	s := map[string]imageApply{
		"width": func(g *ImageGlobal, raw string) error {
			return setInt(&g.Width)(raw)
		},
		"height": func(g *ImageGlobal, raw string) error {
			return setInt(&g.Height)(raw)
		},
		"quality": func(g *ImageGlobal, raw string) error {
			return setInt(&g.Quality)(raw)
		},
		"smartwidth": func(g *ImageGlobal, raw string) error {
			return setBool(&g.SmartWidth)(raw)
		},
		"transparent": func(g *ImageGlobal, raw string) error {
			return setBool(&g.Transparent)(raw)
		},
		"format": func(g *ImageGlobal, raw string) error {
			return setString(&g.Format)(raw)
		},
		"crop.left": func(g *ImageGlobal, raw string) error {
			return setInt(&g.Crop.Left)(raw)
		},
		"crop.top": func(g *ImageGlobal, raw string) error {
			return setInt(&g.Crop.Top)(raw)
		},
		"crop.width": func(g *ImageGlobal, raw string) error {
			return setInt(&g.Crop.Width)(raw)
		},
		"crop.height": func(g *ImageGlobal, raw string) error {
			return setInt(&g.Crop.Height)(raw)
		},
		"proxy": func(g *ImageGlobal, raw string) error {
			return setString(&g.Load.Proxy)(raw)
		},
	}
	for _, k := range []string{
		"background", "images", "printmediatype", "mediatype",
		"simplifydom", "simplifydomprofile", "printlinkunderline",
	} {
		key := k
		s["web."+key] = func(g *ImageGlobal, raw string) error {
			return webApply(&g.Web, key, raw)
		}
	}
	return s
}

func hfApply(hf *HeaderFooter, key, raw string) error {
	switch key {
	case "fontsize":
		return setFloat(&hf.FontSize)(raw)
	case "fontname":
		return setString(&hf.FontName)(raw)
	case "left":
		return setString(&hf.Left)(raw)
	case "right":
		return setString(&hf.Right)(raw)
	case "center":
		return setString(&hf.Center)(raw)
	case "line":
		return setBool(&hf.Line)(raw)
	case "spacing":
		return setFloat(&hf.Spacing)(raw)
	case "htmlurl":
		return setString(&hf.HTMLURL)(raw)
	}
	return fmt.Errorf("unknown header/footer key %q", key)
}

func tocApply(t *TableOfContent, key, raw string) error {
	switch key {
	case "fontscale":
		return setFloat(&t.FontScale)(raw)
	case "indentation":
		return setStringDefault(&t.Indentation)(raw)
	case "dottedlines":
		return setBool(&t.DottedLines)(raw)
	case "captiontext":
		return setString(&t.CaptionText)(raw)
	case "forwardlinks":
		return setBool(&t.ForwardLinks)(raw)
	case "backlinks":
		return setBool(&t.BackLinks)(raw)
	case "xslstylesheet":
		return setString(&t.XSLStyleSheet)(raw)
	}
	return fmt.Errorf("unknown toc key %q", key)
}

func webApply(w *Web, key, raw string) error {
	switch key {
	case "background":
		return setBool(&w.Background)(raw)
	case "images":
		return setBool(&w.Images)(raw)
	case "printmediatype":
		return setBool(&w.PrintMediaType)(raw)
	case "mediatype":
		return setMediaType(&w.MediaType)(raw)
	case "simplifydom":
		return setBool(&w.SimplifyDOM)(raw)
	case "simplifydomprofile":
		return setString(&w.SimplifyDOMProfile)(raw)
	case "printlinkunderline":
		return setBool(&w.PrintLinkUnderline)(raw)
	}
	return fmt.Errorf("unknown web key %q", key)
}

func loadApply(l *LoadPage, key, raw string) error {
	switch key {
	case "zoomfactor":
		return setFloat(&l.ZoomFactor)(raw)
	case "blocklocalfileaccess":
		return setBool(&l.BlockLocalFileAccess)(raw)
	case "loaderrorhandling":
		return setLoadErrorHandling(&l.LoadErrorHandling)(raw)
	case "username":
		return setString(&l.Username)(raw)
	case "password":
		return setString(&l.Password)(raw)
	case "mediatype":
		return setMediaType(&l.MediaType)(raw)
	case "printmediatype":
		return setBool(&l.PrintMediaType)(raw)
	case "timeout":
		return setInt(&l.Timeout)(raw)
	}
	return fmt.Errorf("unknown load key %q", key)
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
