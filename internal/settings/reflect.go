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

// field maps one dotted key to its apply/get pair for a settings type.
type field[T any] struct {
	apply func(*T, string) error
	get   func(*T) (string, bool)
}

// keyTable is the one field-descriptor table per settings type.
type keyTable[T any] map[string]field[T]

// sub adapts a field descriptor of a sub-struct (HeaderFooter, TableOfContent,
// Web, LoadPage) to a containing settings type.
func sub[T any, S any](pick func(*T) *S, d field[S]) field[T] {
	return field[T]{
		apply: func(t *T, raw string) error { return d.apply(pick(t), raw) },
		get:   func(t *T) (string, bool) { return d.get(pick(t)) },
	}
}

// setForKey applies a dotted key through its descriptor table. Known inert
// keys are stored in the Ignored map (Policy A); truly unknown keys error.
func setForKey[T any](t *T, tables keyTable[T], known map[string]struct{}, ignored *map[string]string, kind, name, value string) error {
	key := normalizeDots(name)
	if f, ok := tables[key]; ok {
		return f.apply(t, value)
	}
	if _, ok := known[key]; ok {
		storeIgnored(ignored, key, value)
		return nil
	}
	return fmt.Errorf("unknown %s setting %q", kind, name)
}

// getForKey reads a dotted key through its descriptor table. Accepted ignored
// keys return the last Set value; unknown keys are not found.
func getForKey[T any](t *T, tables keyTable[T], ignored *map[string]string, name string) (string, bool) {
	key := normalizeDots(name)
	if f, ok := tables[key]; ok {
		return f.get(t)
	}
	if *ignored != nil {
		if v, ok := (*ignored)[key]; ok {
			return v, true
		}
	}
	return "", false
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
	// FIX-REVIEW: P2-09 default-encoding ignored; engine decodes UTF-8/ASCII only (html/load decode seam is fix-html-load-outline's)
	"defaultencoding": {},
	"produceforms":    {},
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

// globalKeys / objectKeys / imageKeys are the one field-descriptor tables.
var (
	globalKeys = keyTable[PdfGlobal]{}
	objectKeys = keyTable[PdfObject]{}
	imageKeys  = keyTable[ImageGlobal]{}
)

// subEntry describes one key of a sub-struct table: its apply closure and its
// getter.
type subEntry[S any] struct {
	name  string
	apply func(*S, string) error
	get   func(*S) (string, bool)
}

// subTable builds the descriptor table for a sub-struct from parallel entries.
func subTable[S any](entries []subEntry[S]) map[string]field[S] {
	m := make(map[string]field[S], len(entries))
	for _, e := range entries {
		m[e.name] = field[S]{apply: e.apply, get: e.get}
	}
	return m
}

func init() {
	// --- global keys ---
	g := func(name string, set func(*PdfGlobal, string) error, get func(*PdfGlobal) (string, bool)) {
		globalKeys[name] = field[PdfGlobal]{apply: set, get: get}
	}
	g("orientation",
		func(x *PdfGlobal, raw string) error { return setOrientation(&x.Orientation)(raw) },
		func(x *PdfGlobal) (string, bool) { return x.Orientation.String(), true },
	)
	// colormode and grayscale both share Grayscale.
	g("colormode",
		func(x *PdfGlobal, raw string) error { return setGrayscaleFromColorMode(&x.Grayscale)(raw) },
		func(x *PdfGlobal) (string, bool) {
			if x.Grayscale {
				return "grayscale", true
			}
			return "color", true
		},
	)
	g("grayscale",
		func(x *PdfGlobal, raw string) error { return setBool(&x.Grayscale)(raw) },
		func(x *PdfGlobal) (string, bool) { return fmtBool(x.Grayscale), true },
	)
	g("pageoffset",
		func(x *PdfGlobal, raw string) error { return setInt(&x.PageOffset)(raw) },
		func(x *PdfGlobal) (string, bool) { return fmtInt(x.PageOffset), true },
	)
	g("copies",
		func(x *PdfGlobal, raw string) error { return setInt(&x.Copies)(raw) },
		func(x *PdfGlobal) (string, bool) { return fmtInt(x.Copies), true },
	)
	g("collate",
		func(x *PdfGlobal, raw string) error { return setBool(&x.Collate)(raw) },
		func(x *PdfGlobal) (string, bool) { return fmtBool(x.Collate), true },
	)
	g("outline",
		func(x *PdfGlobal, raw string) error { return setBool(&x.Outline)(raw) },
		func(x *PdfGlobal) (string, bool) { return fmtBool(x.Outline), true },
	)
	g("outlinedepth",
		func(x *PdfGlobal, raw string) error { return setInt(&x.OutlineDepth)(raw) },
		func(x *PdfGlobal) (string, bool) { return fmtInt(x.OutlineDepth), true },
	)
	g("dumpoutline",
		func(x *PdfGlobal, raw string) error { return setBool(&x.DumpOutline)(raw) },
		func(x *PdfGlobal) (string, bool) { return fmtBool(x.DumpOutline), true },
	)
	g("dumpoutlinewithdefaulttocxsl",
		func(x *PdfGlobal, raw string) error { return setBool(&x.DumpDefaultTOCXSL)(raw) },
		func(x *PdfGlobal) (string, bool) { return fmtBool(x.DumpDefaultTOCXSL), true },
	)
	g("usecompression",
		func(x *PdfGlobal, raw string) error { return setBool(&x.UseCompression)(raw) },
		func(x *PdfGlobal) (string, bool) { return fmtBool(x.UseCompression), true },
	)
	g("title",
		func(x *PdfGlobal, raw string) error { return setString(&x.Title)(raw) },
		func(x *PdfGlobal) (string, bool) { return x.Title, true },
	)
	g("smartshrinking",
		func(x *PdfGlobal, raw string) error { return setBool(&x.SmartShrinking)(raw) },
		func(x *PdfGlobal) (string, bool) { return fmtBool(x.SmartShrinking), true },
	)
	// Sole paint switch for PDF + image (no Web.Background mirror).
	paintBG := func(x *PdfGlobal, raw string) error { return setBool(&x.Background)(raw) }
	paintBGGet := func(x *PdfGlobal) (string, bool) { return fmtBool(x.Background), true }
	g("background", paintBG, paintBGGet)
	g("web.background", paintBG, paintBGGet)

	g("enablelocalfileaccess",
		func(x *PdfGlobal, raw string) error { return setBool(&x.Load.EnableLocalFileAccess)(raw) },
		func(x *PdfGlobal) (string, bool) { return fmtBool(x.Load.EnableLocalFileAccess), true },
	)
	g("excludefromoutline",
		func(x *PdfGlobal, raw string) error { return appendString(&x.ExcludeFromOutline)(raw) },
		func(x *PdfGlobal) (string, bool) { return fmtStrings(x.ExcludeFromOutline), true },
	)
	g("quiet",
		func(x *PdfGlobal, raw string) error { return setBool(&x.Quiet)(raw) },
		func(x *PdfGlobal) (string, bool) { return fmtBool(x.Quiet), true },
	)
	g("proxy",
		func(x *PdfGlobal, raw string) error { return setString(&x.Load.Proxy)(raw) },
		func(x *PdfGlobal) (string, bool) { return x.Load.Proxy, true },
	)
	g("usesystemfonts",
		func(x *PdfGlobal, raw string) error { return setBool(&x.UseSystemFonts)(raw) },
		func(x *PdfGlobal) (string, bool) { return fmtBool(x.UseSystemFonts), true },
	)
	g("resolverelativelinks",
		func(x *PdfGlobal, raw string) error { return setBool(&x.ResolveRelativeLinks)(raw) },
		func(x *PdfGlobal) (string, bool) { return fmtBool(x.ResolveRelativeLinks), true },
	)
	g("fontpath",
		func(x *PdfGlobal, raw string) error { return appendString(&x.FontPaths)(raw) },
		func(x *PdfGlobal) (string, bool) { return fmtStrings(x.FontPaths), true },
	)
	g("allow",
		func(x *PdfGlobal, raw string) error { return appendString(&x.Load.Allow)(raw) },
		func(x *PdfGlobal) (string, bool) { return fmtStrings(x.Load.Allow), true },
	)

	for _, edge := range []string{"top", "bottom", "left", "right"} {
		g("margin."+edge,
			func(x *PdfGlobal, raw string) error { return marginSetter(&x.Margin, edge)(raw) },
			func(x *PdfGlobal) (string, bool) {
				var v float64
				switch edge {
				case "top":
					v = x.Margin.Top
				case "bottom":
					v = x.Margin.Bottom
				case "left":
					v = x.Margin.Left
				case "right":
					v = x.Margin.Right
				}
				return fmtFloat(v), true
			},
		)
	}

	g("size.pagesize",
		func(x *PdfGlobal, raw string) error {
			v := strings.TrimSpace(raw)
			x.PageSize = v
			x.Size.PageSize = v
			return nil
		},
		func(x *PdfGlobal) (string, bool) {
			if x.PageSize != "" {
				return x.PageSize, true
			}
			return x.Size.PageSize, true
		},
	)
	g("size.width",
		func(x *PdfGlobal, raw string) error {
			u, err := ParseUnitReal(raw, "mm")
			if err != nil {
				return err
			}
			mm, ok := u.Mm()
			if !ok {
				return fmt.Errorf("size.width: unit %q not convertible", u.Unit)
			}
			x.Size.Width = mm
			return nil
		},
		func(x *PdfGlobal) (string, bool) { return fmtFloat(x.Size.Width), true },
	)
	g("size.height",
		func(x *PdfGlobal, raw string) error {
			u, err := ParseUnitReal(raw, "mm")
			if err != nil {
				return err
			}
			mm, ok := u.Mm()
			if !ok {
				return fmt.Errorf("size.height: unit %q not convertible", u.Unit)
			}
			x.Size.Height = mm
			return nil
		},
		func(x *PdfGlobal) (string, bool) { return fmtFloat(x.Size.Height), true },
	)

	// header/footer keys share one descriptor table per sub-struct. Object
	// keys additionally flag the HeaderSet/FooterSet override bit.
	for key, d := range subTable[HeaderFooter]([]subEntry[HeaderFooter]{
		{"fontsize", func(h *HeaderFooter, raw string) error { return setFloat(&h.FontSize)(raw) }, func(h *HeaderFooter) (string, bool) { return fmtFloat(h.FontSize), true }},
		{"fontname", func(h *HeaderFooter, raw string) error { return setString(&h.FontName)(raw) }, func(h *HeaderFooter) (string, bool) { return h.FontName, true }},
		{"left", func(h *HeaderFooter, raw string) error { return setString(&h.Left)(raw) }, func(h *HeaderFooter) (string, bool) { return h.Left, true }},
		{"right", func(h *HeaderFooter, raw string) error { return setString(&h.Right)(raw) }, func(h *HeaderFooter) (string, bool) { return h.Right, true }},
		{"center", func(h *HeaderFooter, raw string) error { return setString(&h.Center)(raw) }, func(h *HeaderFooter) (string, bool) { return h.Center, true }},
		{"line", func(h *HeaderFooter, raw string) error { return setBool(&h.Line)(raw) }, func(h *HeaderFooter) (string, bool) { return fmtBool(h.Line), true }},
		{"spacing", func(h *HeaderFooter, raw string) error { return setFloat(&h.Spacing)(raw) }, func(h *HeaderFooter) (string, bool) { return fmtFloat(h.Spacing), true }},
		{"htmlurl", func(h *HeaderFooter, raw string) error { return setString(&h.HTMLURL)(raw) }, func(h *HeaderFooter) (string, bool) { return h.HTMLURL, true }},
	}) {
		globalKeys["header."+key] = sub(func(x *PdfGlobal) *HeaderFooter { return &x.Header }, d)
		globalKeys["footer."+key] = sub(func(x *PdfGlobal) *HeaderFooter { return &x.Footer }, d)
		objectKeys["header."+key] = field[PdfObject]{
			apply: func(o *PdfObject, raw string) error {
				o.HeaderSet = true
				return d.apply(&o.Header, raw)
			},
			get: func(o *PdfObject) (string, bool) { return d.get(&o.Header) },
		}
		objectKeys["footer."+key] = field[PdfObject]{
			apply: func(o *PdfObject, raw string) error {
				o.FooterSet = true
				return d.apply(&o.Footer, raw)
			},
			get: func(o *PdfObject) (string, bool) { return d.get(&o.Footer) },
		}
	}

	for key, d := range subTable[TableOfContent]([]subEntry[TableOfContent]{
		{"fontscale", func(t *TableOfContent, raw string) error { return setFloat(&t.FontScale)(raw) }, func(t *TableOfContent) (string, bool) { return fmtFloat(t.FontScale), true }},
		{"indentation", func(t *TableOfContent, raw string) error { return setStringDefault(&t.Indentation)(raw) }, func(t *TableOfContent) (string, bool) { return t.Indentation, true }},
		{"dottedlines", func(t *TableOfContent, raw string) error { return setBool(&t.DottedLines)(raw) }, func(t *TableOfContent) (string, bool) { return fmtBool(t.DottedLines), true }},
		{"captiontext", func(t *TableOfContent, raw string) error { return setString(&t.CaptionText)(raw) }, func(t *TableOfContent) (string, bool) { return t.CaptionText, true }},
		{"forwardlinks", func(t *TableOfContent, raw string) error { return setBool(&t.ForwardLinks)(raw) }, func(t *TableOfContent) (string, bool) { return fmtBool(t.ForwardLinks), true }},
		{"backlinks", func(t *TableOfContent, raw string) error { return setBool(&t.BackLinks)(raw) }, func(t *TableOfContent) (string, bool) { return fmtBool(t.BackLinks), true }},
		{"xslstylesheet", func(t *TableOfContent, raw string) error { return setString(&t.XSLStyleSheet)(raw) }, func(t *TableOfContent) (string, bool) { return t.XSLStyleSheet, true }},
	}) {
		globalKeys["toc."+key] = sub(func(x *PdfGlobal) *TableOfContent { return &x.TOC }, d)
		objectKeys["toc."+key] = sub(func(o *PdfObject) *TableOfContent { return &o.TOC }, d)
	}

	// web.* except background (mapped to Global.Background above)
	for key, d := range subTable[Web]([]subEntry[Web]{
		{"images", func(w *Web, raw string) error { return setBool(&w.Images)(raw) }, func(w *Web) (string, bool) { return fmtBool(w.Images), true }},
		{"printmediatype", func(w *Web, raw string) error { return setBool(&w.PrintMediaType)(raw) }, func(w *Web) (string, bool) { return fmtBool(w.PrintMediaType), true }},
		{"mediatype", func(w *Web, raw string) error { return setMediaType(&w.MediaType)(raw) }, func(w *Web) (string, bool) { return w.MediaType.String(), true }},
		{"simplifydom", func(w *Web, raw string) error { return setBool(&w.SimplifyDOM)(raw) }, func(w *Web) (string, bool) { return fmtBool(w.SimplifyDOM), true }},
		{"simplifydomprofile", func(w *Web, raw string) error { return setString(&w.SimplifyDOMProfile)(raw) }, func(w *Web) (string, bool) { return w.SimplifyDOMProfile, true }},
		{"printlinkunderline", func(w *Web, raw string) error { return setBool(&w.PrintLinkUnderline)(raw) }, func(w *Web) (string, bool) { return fmtBool(w.PrintLinkUnderline), true }},
	}) {
		globalKeys["web."+key] = sub(func(x *PdfGlobal) *Web { return &x.Web }, d)
		objectKeys["web."+key] = sub(func(o *PdfObject) *Web { return &o.Web }, d)
		imageKeys["web."+key] = sub(func(x *ImageGlobal) *Web { return &x.Web }, d)
	}

	for key, d := range subTable[LoadPage]([]subEntry[LoadPage]{
		{"zoomfactor", func(l *LoadPage, raw string) error { return setFloat(&l.ZoomFactor)(raw) }, func(l *LoadPage) (string, bool) { return fmtFloat(l.ZoomFactor), true }},
		{"blocklocalfileaccess", func(l *LoadPage, raw string) error { return setBool(&l.BlockLocalFileAccess)(raw) }, func(l *LoadPage) (string, bool) { return fmtBool(l.BlockLocalFileAccess), true }},
		{"loaderrorhandling", func(l *LoadPage, raw string) error { return setLoadErrorHandling(&l.LoadErrorHandling)(raw) }, func(l *LoadPage) (string, bool) { return l.LoadErrorHandling.String(), true }},
		{"username", func(l *LoadPage, raw string) error { return setString(&l.Username)(raw) }, func(l *LoadPage) (string, bool) { return l.Username, true }},
		{"password", func(l *LoadPage, raw string) error { return setString(&l.Password)(raw) }, func(l *LoadPage) (string, bool) { return l.Password, true }},
		{"mediatype", func(l *LoadPage, raw string) error { return setMediaType(&l.MediaType)(raw) }, func(l *LoadPage) (string, bool) { return l.MediaType.String(), true }},
		{"printmediatype", func(l *LoadPage, raw string) error { return setBool(&l.PrintMediaType)(raw) }, func(l *LoadPage) (string, bool) { return fmtBool(l.PrintMediaType), true }},
		{"timeout", func(l *LoadPage, raw string) error { return setInt(&l.Timeout)(raw) }, func(l *LoadPage) (string, bool) { return fmtInt(l.Timeout), true }},
	}) {
		objectKeys["load."+key] = sub(func(o *PdfObject) *LoadPage { return &o.Load }, d)
	}

	// --- object keys ---
	o := func(name string, set func(*PdfObject, string) error, get func(*PdfObject) (string, bool)) {
		objectKeys[name] = field[PdfObject]{apply: set, get: get}
	}
	o("page",
		func(x *PdfObject, raw string) error { return setString(&x.Page)(raw) },
		func(x *PdfObject) (string, bool) { return x.Page, true },
	)
	o("externallinks",
		func(x *PdfObject, raw string) error { return setBool(&x.ExternalLinks)(raw) },
		func(x *PdfObject) (string, bool) { return fmtBool(x.ExternalLinks), true },
	)
	o("locallinks",
		func(x *PdfObject, raw string) error { return setBool(&x.LocalLinks)(raw) },
		func(x *PdfObject) (string, bool) { return fmtBool(x.LocalLinks), true },
	)
	o("includeinoutline",
		func(x *PdfObject, raw string) error { return setBool(&x.IncludeInOutline)(raw) },
		func(x *PdfObject) (string, bool) { return fmtBool(x.IncludeInOutline), true },
	)
	o("useoutline",
		func(x *PdfObject, raw string) error { return setBool(&x.UseOutline)(raw) },
		func(x *PdfObject) (string, bool) { return fmtBool(x.UseOutline), true },
	)
	o("istableofcontent",
		func(x *PdfObject, raw string) error { return setBool(&x.IsTableOfContent)(raw) },
		func(x *PdfObject) (string, bool) { return fmtBool(x.IsTableOfContent), true },
	)
	o("iscover",
		func(x *PdfObject, raw string) error { return setBool(&x.IsCover)(raw) },
		func(x *PdfObject) (string, bool) { return fmtBool(x.IsCover), true },
	)

	// --- image keys ---
	im := func(name string, set func(*ImageGlobal, string) error, get func(*ImageGlobal) (string, bool)) {
		imageKeys[name] = field[ImageGlobal]{apply: set, get: get}
	}
	im("width",
		func(x *ImageGlobal, raw string) error { return setInt(&x.Width)(raw) },
		func(x *ImageGlobal) (string, bool) { return fmtInt(x.Width), true },
	)
	im("height",
		func(x *ImageGlobal, raw string) error { return setInt(&x.Height)(raw) },
		func(x *ImageGlobal) (string, bool) { return fmtInt(x.Height), true },
	)
	im("quality",
		func(x *ImageGlobal, raw string) error { return setInt(&x.Quality)(raw) },
		func(x *ImageGlobal) (string, bool) { return fmtInt(x.Quality), true },
	)
	im("smartwidth",
		func(x *ImageGlobal, raw string) error { return setBool(&x.SmartWidth)(raw) },
		func(x *ImageGlobal) (string, bool) { return fmtBool(x.SmartWidth), true },
	)
	im("transparent",
		func(x *ImageGlobal, raw string) error { return setBool(&x.Transparent)(raw) },
		func(x *ImageGlobal) (string, bool) { return fmtBool(x.Transparent), true },
	)
	im("format",
		func(x *ImageGlobal, raw string) error { return setString(&x.Format)(raw) },
		func(x *ImageGlobal) (string, bool) { return x.Format, true },
	)
	im("crop.left",
		func(x *ImageGlobal, raw string) error { return setInt(&x.Crop.Left)(raw) },
		func(x *ImageGlobal) (string, bool) { return fmtInt(x.Crop.Left), true },
	)
	im("crop.top",
		func(x *ImageGlobal, raw string) error { return setInt(&x.Crop.Top)(raw) },
		func(x *ImageGlobal) (string, bool) { return fmtInt(x.Crop.Top), true },
	)
	im("crop.width",
		func(x *ImageGlobal, raw string) error { return setInt(&x.Crop.Width)(raw) },
		func(x *ImageGlobal) (string, bool) { return fmtInt(x.Crop.Width), true },
	)
	im("crop.height",
		func(x *ImageGlobal, raw string) error { return setInt(&x.Crop.Height)(raw) },
		func(x *ImageGlobal) (string, bool) { return fmtInt(x.Crop.Height), true },
	)
	im("proxy",
		func(x *ImageGlobal, raw string) error { return setString(&x.Load.Proxy)(raw) },
		func(x *ImageGlobal) (string, bool) { return x.Load.Proxy, true },
	)
	// image web.background is not typed here — ApplyImageKey routes it to
	// PdfGlobal.Background.
}

// Global.Set applies a dotted settings key ("margin.top", "load.jsdelay",
// "web.background", …) to a PdfGlobal. Known inert keys are stored in
// Ignored and succeed; truly unknown keys return an error.
func (g *PdfGlobal) Set(name, value string) error {
	return setForKey(g, globalKeys, ignoredGlobalKeys, &g.Ignored, "global", name, value)
}

// Object.Set applies a dotted settings key to a PdfObject. Known inert keys
// go to Ignored; unknown keys return an error.
func (o *PdfObject) Set(name, value string) error {
	return setForKey(o, objectKeys, ignoredObjectKeys, &o.Ignored, "object", name, value)
}

// ImageGlobal.Set applies an image-mode dotted settings key.
func (g *ImageGlobal) Set(name, value string) error {
	return setForKey(g, imageKeys, ignoredGlobalKeys, &g.Ignored, "image", name, value)
}

// ApplyImageKey routes an image-mode key: "background"/"web.background" alias
// to the shared PdfGlobal.Background paint switch (image mode has no
// Web.Background); everything else goes to ImageGlobal.Set. ImageConverter.Set
// delegates here.
func ApplyImageKey(global *PdfGlobal, img *ImageGlobal, name, value string) error {
	switch normalizeDots(name) {
	case "background", "web.background":
		return global.Set("background", value)
	default:
		return img.Set(name, value)
	}
}
