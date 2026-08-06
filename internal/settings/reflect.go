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

// globalGet reads a dotted key from a PdfGlobal.
type globalGet func(g *PdfGlobal) (string, bool)

// objectApply applies a dotted key to a PdfObject.
type objectApply func(o *PdfObject, raw string) error

// objectGet reads a dotted key from a PdfObject.
type objectGet func(o *PdfObject) (string, bool)

// imageApply applies a dotted key to an ImageGlobal.
type imageApply func(g *ImageGlobal, raw string) error

// imageGet reads a dotted key from an ImageGlobal.
type imageGet func(g *ImageGlobal) (string, bool)

var (
	globalKeyTable map[string]globalApply
	globalGetTable map[string]globalGet
	objectKeyTable map[string]objectApply
	objectGetTable map[string]objectGet
	imageKeyTable  map[string]imageApply
	imageGetTable  map[string]imageGet
	keysOnce       sync.Once
)

// gReg / oReg / iReg register one key once for both Set and Get.
func gReg(set map[string]globalApply, get map[string]globalGet, key string, s globalApply, g globalGet) {
	set[key] = s
	get[key] = g
}

func oReg(set map[string]objectApply, get map[string]objectGet, key string, s objectApply, g objectGet) {
	set[key] = s
	get[key] = g
}

func iReg(set map[string]imageApply, get map[string]imageGet, key string, s imageApply, g imageGet) {
	set[key] = s
	get[key] = g
}

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

// Note: global web.background is a real key (→ PdfGlobal.Background), not ignored.

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
	// web.* stubs — paint background is Global only
	"web.background":      {},
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
	globalKeyTable, globalGetTable = buildGlobalKeys()
	objectKeyTable, objectGetTable = buildObjectKeys()
	imageKeyTable, imageGetTable = buildImageKeys()
}

func buildGlobalKeys() (map[string]globalApply, map[string]globalGet) {
	set := make(map[string]globalApply)
	get := make(map[string]globalGet)

	gReg(set, get, "orientation",
		func(g *PdfGlobal, raw string) error { return setOrientation(&g.Orientation)(raw) },
		func(g *PdfGlobal) (string, bool) { return g.Orientation.String(), true },
	)
	// colormode and grayscale both share Grayscale.
	gReg(set, get, "colormode",
		func(g *PdfGlobal, raw string) error { return setGrayscaleFromColorMode(&g.Grayscale)(raw) },
		func(g *PdfGlobal) (string, bool) {
			if g.Grayscale {
				return "grayscale", true
			}
			return "color", true
		},
	)
	gReg(set, get, "grayscale",
		func(g *PdfGlobal, raw string) error { return setBool(&g.Grayscale)(raw) },
		func(g *PdfGlobal) (string, bool) { return fmtBool(g.Grayscale), true },
	)
	gReg(set, get, "pageoffset",
		func(g *PdfGlobal, raw string) error { return setInt(&g.PageOffset)(raw) },
		func(g *PdfGlobal) (string, bool) { return fmtInt(g.PageOffset), true },
	)
	gReg(set, get, "copies",
		func(g *PdfGlobal, raw string) error { return setInt(&g.Copies)(raw) },
		func(g *PdfGlobal) (string, bool) { return fmtInt(g.Copies), true },
	)
	gReg(set, get, "collate",
		func(g *PdfGlobal, raw string) error { return setBool(&g.Collate)(raw) },
		func(g *PdfGlobal) (string, bool) { return fmtBool(g.Collate), true },
	)
	gReg(set, get, "outline",
		func(g *PdfGlobal, raw string) error { return setBool(&g.Outline)(raw) },
		func(g *PdfGlobal) (string, bool) { return fmtBool(g.Outline), true },
	)
	gReg(set, get, "outlinedepth",
		func(g *PdfGlobal, raw string) error { return setInt(&g.OutlineDepth)(raw) },
		func(g *PdfGlobal) (string, bool) { return fmtInt(g.OutlineDepth), true },
	)
	gReg(set, get, "dumpoutline",
		func(g *PdfGlobal, raw string) error { return setBool(&g.DumpOutline)(raw) },
		func(g *PdfGlobal) (string, bool) { return fmtBool(g.DumpOutline), true },
	)
	gReg(set, get, "dumpoutlinewithdefaulttocxsl",
		func(g *PdfGlobal, raw string) error { return setBool(&g.DumpDefaultTOCXSL)(raw) },
		func(g *PdfGlobal) (string, bool) { return fmtBool(g.DumpDefaultTOCXSL), true },
	)
	gReg(set, get, "usecompression",
		func(g *PdfGlobal, raw string) error { return setBool(&g.UseCompression)(raw) },
		func(g *PdfGlobal) (string, bool) { return fmtBool(g.UseCompression), true },
	)
	gReg(set, get, "title",
		func(g *PdfGlobal, raw string) error { return setString(&g.Title)(raw) },
		func(g *PdfGlobal) (string, bool) { return g.Title, true },
	)
	gReg(set, get, "smartshrinking",
		func(g *PdfGlobal, raw string) error { return setBool(&g.SmartShrinking)(raw) },
		func(g *PdfGlobal) (string, bool) { return fmtBool(g.SmartShrinking), true },
	)
	// Sole paint switch for PDF + image (no Web.Background mirror).
	paintBG := func(g *PdfGlobal, raw string) error { return setBool(&g.Background)(raw) }
	paintBGGet := func(g *PdfGlobal) (string, bool) { return fmtBool(g.Background), true }
	gReg(set, get, "background", paintBG, paintBGGet)
	gReg(set, get, "web.background", paintBG, paintBGGet)

	gReg(set, get, "enablelocalfileaccess",
		func(g *PdfGlobal, raw string) error { return setBool(&g.EnableLocalFileAccess)(raw) },
		func(g *PdfGlobal) (string, bool) { return fmtBool(g.EnableLocalFileAccess), true },
	)
	gReg(set, get, "excludefromoutline",
		func(g *PdfGlobal, raw string) error { return appendString(&g.ExcludeFromOutline)(raw) },
		func(g *PdfGlobal) (string, bool) { return fmtStrings(g.ExcludeFromOutline), true },
	)
	gReg(set, get, "quiet",
		func(g *PdfGlobal, raw string) error { return setBool(&g.Quiet)(raw) },
		func(g *PdfGlobal) (string, bool) { return fmtBool(g.Quiet), true },
	)
	gReg(set, get, "proxy",
		func(g *PdfGlobal, raw string) error { return setString(&g.Load.Proxy)(raw) },
		func(g *PdfGlobal) (string, bool) { return g.Load.Proxy, true },
	)
	gReg(set, get, "usesystemfonts",
		func(g *PdfGlobal, raw string) error { return setBool(&g.UseSystemFonts)(raw) },
		func(g *PdfGlobal) (string, bool) { return fmtBool(g.UseSystemFonts), true },
	)
	gReg(set, get, "resolverelativelinks",
		func(g *PdfGlobal, raw string) error { return setBool(&g.ResolveRelativeLinks)(raw) },
		func(g *PdfGlobal) (string, bool) { return fmtBool(g.ResolveRelativeLinks), true },
	)
	gReg(set, get, "fontpath",
		func(g *PdfGlobal, raw string) error { return appendString(&g.FontPaths)(raw) },
		func(g *PdfGlobal) (string, bool) { return fmtStrings(g.FontPaths), true },
	)
	gReg(set, get, "allow",
		func(g *PdfGlobal, raw string) error { return appendString(&g.Allow)(raw) },
		func(g *PdfGlobal) (string, bool) { return fmtStrings(g.Allow), true },
	)

	for _, e := range []string{"top", "bottom", "left", "right"} {
		edge := e
		gReg(set, get, "margin."+edge,
			func(g *PdfGlobal, raw string) error { return marginSetter(&g.Margin, edge)(raw) },
			func(g *PdfGlobal) (string, bool) {
				var v float64
				switch edge {
				case "top":
					v = g.Margin.Top
				case "bottom":
					v = g.Margin.Bottom
				case "left":
					v = g.Margin.Left
				case "right":
					v = g.Margin.Right
				}
				return fmtFloat(v), true
			},
		)
	}

	gReg(set, get, "size.pagesize",
		func(g *PdfGlobal, raw string) error {
			v := strings.TrimSpace(raw)
			g.PageSize = v
			g.Size.PageSize = v
			return nil
		},
		func(g *PdfGlobal) (string, bool) {
			if g.PageSize != "" {
				return g.PageSize, true
			}
			return g.Size.PageSize, true
		},
	)
	gReg(set, get, "size.width",
		func(g *PdfGlobal, raw string) error {
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
		},
		func(g *PdfGlobal) (string, bool) { return fmtFloat(g.Size.Width), true },
	)
	gReg(set, get, "size.height",
		func(g *PdfGlobal, raw string) error {
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
		},
		func(g *PdfGlobal) (string, bool) { return fmtFloat(g.Size.Height), true },
	)

	for _, k := range []string{"fontsize", "fontname", "left", "right", "center", "line", "spacing", "htmlurl"} {
		key := k
		gReg(set, get, "header."+key,
			func(g *PdfGlobal, raw string) error { return hfApply(&g.Header, key, raw) },
			func(g *PdfGlobal) (string, bool) { return hfGet(&g.Header, key) },
		)
		gReg(set, get, "footer."+key,
			func(g *PdfGlobal, raw string) error { return hfApply(&g.Footer, key, raw) },
			func(g *PdfGlobal) (string, bool) { return hfGet(&g.Footer, key) },
		)
	}
	for _, k := range []string{"fontscale", "indentation", "dottedlines", "captiontext", "forwardlinks", "backlinks", "xslstylesheet"} {
		key := k
		gReg(set, get, "toc."+key,
			func(g *PdfGlobal, raw string) error { return tocApply(&g.TOC, key, raw) },
			func(g *PdfGlobal) (string, bool) { return tocGet(&g.TOC, key) },
		)
	}
	// web.* except background (mapped to Global.Background above)
	for _, k := range []string{
		"images", "printmediatype", "mediatype",
		"simplifydom", "simplifydomprofile", "printlinkunderline",
	} {
		key := k
		gReg(set, get, "web."+key,
			func(g *PdfGlobal, raw string) error { return webApply(&g.Web, key, raw) },
			func(g *PdfGlobal) (string, bool) { return webGet(&g.Web, key) },
		)
	}
	return set, get
}

func buildObjectKeys() (map[string]objectApply, map[string]objectGet) {
	set := make(map[string]objectApply)
	get := make(map[string]objectGet)

	oReg(set, get, "page",
		func(o *PdfObject, raw string) error { return setString(&o.Page)(raw) },
		func(o *PdfObject) (string, bool) { return o.Page, true },
	)
	oReg(set, get, "externallinks",
		func(o *PdfObject, raw string) error { return setBool(&o.ExternalLinks)(raw) },
		func(o *PdfObject) (string, bool) { return fmtBool(o.ExternalLinks), true },
	)
	oReg(set, get, "locallinks",
		func(o *PdfObject, raw string) error { return setBool(&o.LocalLinks)(raw) },
		func(o *PdfObject) (string, bool) { return fmtBool(o.LocalLinks), true },
	)
	oReg(set, get, "includeinoutline",
		func(o *PdfObject, raw string) error { return setBool(&o.IncludeInOutline)(raw) },
		func(o *PdfObject) (string, bool) { return fmtBool(o.IncludeInOutline), true },
	)
	oReg(set, get, "useoutline",
		func(o *PdfObject, raw string) error { return setBool(&o.UseOutline)(raw) },
		func(o *PdfObject) (string, bool) { return fmtBool(o.UseOutline), true },
	)
	oReg(set, get, "istableofcontent",
		func(o *PdfObject, raw string) error { return setBool(&o.IsTableOfContent)(raw) },
		func(o *PdfObject) (string, bool) { return fmtBool(o.IsTableOfContent), true },
	)
	oReg(set, get, "iscover",
		func(o *PdfObject, raw string) error { return setBool(&o.IsCover)(raw) },
		func(o *PdfObject) (string, bool) { return fmtBool(o.IsCover), true },
	)

	for _, k := range []string{
		"zoomfactor", "blocklocalfileaccess", "loaderrorhandling",
		"username", "password", "mediatype", "printmediatype", "timeout",
	} {
		key := k
		oReg(set, get, "load."+key,
			func(o *PdfObject, raw string) error { return loadApply(&o.Load, key, raw) },
			func(o *PdfObject) (string, bool) { return loadGet(&o.Load, key) },
		)
	}
	// object web.background is inert (paint uses Global.Background only) — Policy A ignored
	for _, k := range []string{
		"images", "printmediatype", "mediatype",
		"simplifydom", "simplifydomprofile", "printlinkunderline",
	} {
		key := k
		oReg(set, get, "web."+key,
			func(o *PdfObject, raw string) error { return webApply(&o.Web, key, raw) },
			func(o *PdfObject) (string, bool) { return webGet(&o.Web, key) },
		)
	}
	for _, k := range []string{"fontsize", "fontname", "left", "right", "center", "line", "spacing", "htmlurl"} {
		key := k
		oReg(set, get, "header."+key,
			func(o *PdfObject, raw string) error {
				o.HeaderSet = true
				return hfApply(&o.Header, key, raw)
			},
			func(o *PdfObject) (string, bool) { return hfGet(&o.Header, key) },
		)
		oReg(set, get, "footer."+key,
			func(o *PdfObject, raw string) error {
				o.FooterSet = true
				return hfApply(&o.Footer, key, raw)
			},
			func(o *PdfObject) (string, bool) { return hfGet(&o.Footer, key) },
		)
	}
	for _, k := range []string{"fontscale", "indentation", "dottedlines", "captiontext", "forwardlinks", "backlinks", "xslstylesheet"} {
		key := k
		oReg(set, get, "toc."+key,
			func(o *PdfObject, raw string) error { return tocApply(&o.TOC, key, raw) },
			func(o *PdfObject) (string, bool) { return tocGet(&o.TOC, key) },
		)
	}
	return set, get
}

func buildImageKeys() (map[string]imageApply, map[string]imageGet) {
	set := make(map[string]imageApply)
	get := make(map[string]imageGet)

	iReg(set, get, "width",
		func(g *ImageGlobal, raw string) error { return setInt(&g.Width)(raw) },
		func(g *ImageGlobal) (string, bool) { return fmtInt(g.Width), true },
	)
	iReg(set, get, "height",
		func(g *ImageGlobal, raw string) error { return setInt(&g.Height)(raw) },
		func(g *ImageGlobal) (string, bool) { return fmtInt(g.Height), true },
	)
	iReg(set, get, "quality",
		func(g *ImageGlobal, raw string) error { return setInt(&g.Quality)(raw) },
		func(g *ImageGlobal) (string, bool) { return fmtInt(g.Quality), true },
	)
	iReg(set, get, "smartwidth",
		func(g *ImageGlobal, raw string) error { return setBool(&g.SmartWidth)(raw) },
		func(g *ImageGlobal) (string, bool) { return fmtBool(g.SmartWidth), true },
	)
	iReg(set, get, "transparent",
		func(g *ImageGlobal, raw string) error { return setBool(&g.Transparent)(raw) },
		func(g *ImageGlobal) (string, bool) { return fmtBool(g.Transparent), true },
	)
	iReg(set, get, "format",
		func(g *ImageGlobal, raw string) error { return setString(&g.Format)(raw) },
		func(g *ImageGlobal) (string, bool) { return g.Format, true },
	)
	iReg(set, get, "crop.left",
		func(g *ImageGlobal, raw string) error { return setInt(&g.Crop.Left)(raw) },
		func(g *ImageGlobal) (string, bool) { return fmtInt(g.Crop.Left), true },
	)
	iReg(set, get, "crop.top",
		func(g *ImageGlobal, raw string) error { return setInt(&g.Crop.Top)(raw) },
		func(g *ImageGlobal) (string, bool) { return fmtInt(g.Crop.Top), true },
	)
	iReg(set, get, "crop.width",
		func(g *ImageGlobal, raw string) error { return setInt(&g.Crop.Width)(raw) },
		func(g *ImageGlobal) (string, bool) { return fmtInt(g.Crop.Width), true },
	)
	iReg(set, get, "crop.height",
		func(g *ImageGlobal, raw string) error { return setInt(&g.Crop.Height)(raw) },
		func(g *ImageGlobal) (string, bool) { return fmtInt(g.Crop.Height), true },
	)
	iReg(set, get, "proxy",
		func(g *ImageGlobal, raw string) error { return setString(&g.Load.Proxy)(raw) },
		func(g *ImageGlobal) (string, bool) { return g.Load.Proxy, true },
	)
	// image web.background is not typed here — ImageConverter.Set routes it to Global.Background
	for _, k := range []string{
		"images", "printmediatype", "mediatype",
		"simplifydom", "simplifydomprofile", "printlinkunderline",
	} {
		key := k
		iReg(set, get, "web."+key,
			func(g *ImageGlobal, raw string) error { return webApply(&g.Web, key, raw) },
			func(g *ImageGlobal) (string, bool) { return webGet(&g.Web, key) },
		)
	}
	return set, get
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

func hfGet(hf *HeaderFooter, key string) (string, bool) {
	switch key {
	case "fontsize":
		return fmtFloat(hf.FontSize), true
	case "fontname":
		return hf.FontName, true
	case "left":
		return hf.Left, true
	case "right":
		return hf.Right, true
	case "center":
		return hf.Center, true
	case "line":
		return fmtBool(hf.Line), true
	case "spacing":
		return fmtFloat(hf.Spacing), true
	case "htmlurl":
		return hf.HTMLURL, true
	}
	return "", false
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

func tocGet(t *TableOfContent, key string) (string, bool) {
	switch key {
	case "fontscale":
		return fmtFloat(t.FontScale), true
	case "indentation":
		return t.Indentation, true
	case "dottedlines":
		return fmtBool(t.DottedLines), true
	case "captiontext":
		return t.CaptionText, true
	case "forwardlinks":
		return fmtBool(t.ForwardLinks), true
	case "backlinks":
		return fmtBool(t.BackLinks), true
	case "xslstylesheet":
		return t.XSLStyleSheet, true
	}
	return "", false
}

func webApply(w *Web, key, raw string) error {
	switch key {
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

func webGet(w *Web, key string) (string, bool) {
	switch key {
	case "images":
		return fmtBool(w.Images), true
	case "printmediatype":
		return fmtBool(w.PrintMediaType), true
	case "mediatype":
		return w.MediaType.String(), true
	case "simplifydom":
		return fmtBool(w.SimplifyDOM), true
	case "simplifydomprofile":
		return w.SimplifyDOMProfile, true
	case "printlinkunderline":
		return fmtBool(w.PrintLinkUnderline), true
	}
	return "", false
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

func loadGet(l *LoadPage, key string) (string, bool) {
	switch key {
	case "zoomfactor":
		return fmtFloat(l.ZoomFactor), true
	case "blocklocalfileaccess":
		return fmtBool(l.BlockLocalFileAccess), true
	case "loaderrorhandling":
		return l.LoadErrorHandling.String(), true
	case "username":
		return l.Username, true
	case "password":
		return l.Password, true
	case "mediatype":
		return l.MediaType.String(), true
	case "printmediatype":
		return fmtBool(l.PrintMediaType), true
	case "timeout":
		return fmtInt(l.Timeout), true
	}
	return "", false
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
