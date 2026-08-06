package settings

import (
	"strconv"
	"strings"
)

// Getters mirror Set key tables so Get/Set stay on one surface (no reflection
// walk of struct field names).

type globalGet func(g *PdfGlobal) (string, bool)
type objectGet func(o *PdfObject) (string, bool)
type imageGet func(g *ImageGlobal) (string, bool)

var (
	globalGetTable map[string]globalGet
	objectGetTable map[string]objectGet
	imageGetTable  map[string]imageGet
)

func buildGetTables() {
	globalGetTable = buildGlobalGetTable()
	objectGetTable = buildObjectGetTable()
	imageGetTable = buildImageGetTable()
}

// ensureGetTables builds Set+Get tables once.
func ensureGetTables() {
	ensureKeyTables()
}

// Get reads a dotted global key as its canonical string form. ok is false
// for unknown keys. Accepted ignored keys return the last Set value.
func (g *PdfGlobal) Get(name string) (string, bool) {
	ensureGetTables()
	key := normalizeDots(name)
	if fn, ok := globalGetTable[key]; ok {
		return fn(g)
	}
	if g.Ignored != nil {
		if v, ok := g.Ignored[key]; ok {
			return v, true
		}
	}
	return "", false
}

// Get reads a dotted object key as its canonical string form.
func (o *PdfObject) Get(name string) (string, bool) {
	ensureGetTables()
	key := normalizeDots(name)
	if fn, ok := objectGetTable[key]; ok {
		return fn(o)
	}
	if o.Ignored != nil {
		if v, ok := o.Ignored[key]; ok {
			return v, true
		}
	}
	return "", false
}

// Get reads a dotted image-mode key as its canonical string form.
func (g *ImageGlobal) Get(name string) (string, bool) {
	ensureGetTables()
	key := normalizeDots(name)
	if fn, ok := imageGetTable[key]; ok {
		return fn(g)
	}
	if g.Ignored != nil {
		if v, ok := g.Ignored[key]; ok {
			return v, true
		}
	}
	return "", false
}

func fmtBool(b bool) string { return strconv.FormatBool(b) }

func fmtFloat(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}

func fmtInt(n int) string { return strconv.Itoa(n) }

func fmtStrings(ss []string) string {
	return strings.Join(ss, "\n")
}

func buildGlobalGetTable() map[string]globalGet {
	s := map[string]globalGet{
		"orientation": func(g *PdfGlobal) (string, bool) {
			return g.Orientation.String(), true
		},
		"colormode": func(g *PdfGlobal) (string, bool) {
			if g.Grayscale {
				return "grayscale", true
			}
			return "color", true
		},
		"grayscale": func(g *PdfGlobal) (string, bool) {
			return fmtBool(g.Grayscale), true
		},
		"pageoffset":   func(g *PdfGlobal) (string, bool) { return fmtInt(g.PageOffset), true },
		"copies":       func(g *PdfGlobal) (string, bool) { return fmtInt(g.Copies), true },
		"collate":      func(g *PdfGlobal) (string, bool) { return fmtBool(g.Collate), true },
		"outline":      func(g *PdfGlobal) (string, bool) { return fmtBool(g.Outline), true },
		"outlinedepth": func(g *PdfGlobal) (string, bool) { return fmtInt(g.OutlineDepth), true },
		"dumpoutline":  func(g *PdfGlobal) (string, bool) { return fmtBool(g.DumpOutline), true },
		"dumpoutlinewithdefaulttocxsl": func(g *PdfGlobal) (string, bool) {
			return fmtBool(g.DumpDefaultTOCXSL), true
		},
		"usecompression":        func(g *PdfGlobal) (string, bool) { return fmtBool(g.UseCompression), true },
		"title":                 func(g *PdfGlobal) (string, bool) { return g.Title, true },
		"smartshrinking":        func(g *PdfGlobal) (string, bool) { return fmtBool(g.SmartShrinking), true },
		"background":            func(g *PdfGlobal) (string, bool) { return fmtBool(g.Background), true },
		"web.background":        func(g *PdfGlobal) (string, bool) { return fmtBool(g.Background), true },
		"enablelocalfileaccess": func(g *PdfGlobal) (string, bool) { return fmtBool(g.EnableLocalFileAccess), true },
		"excludefromoutline":    func(g *PdfGlobal) (string, bool) { return fmtStrings(g.ExcludeFromOutline), true },
		"quiet":                 func(g *PdfGlobal) (string, bool) { return fmtBool(g.Quiet), true },
		"proxy":                 func(g *PdfGlobal) (string, bool) { return g.Load.Proxy, true },
		"usesystemfonts":        func(g *PdfGlobal) (string, bool) { return fmtBool(g.UseSystemFonts), true },
		"resolverelativelinks":  func(g *PdfGlobal) (string, bool) { return fmtBool(g.ResolveRelativeLinks), true },
		"fontpath":              func(g *PdfGlobal) (string, bool) { return fmtStrings(g.FontPaths), true },
		"allow":                 func(g *PdfGlobal) (string, bool) { return fmtStrings(g.Allow), true },
		"size.pagesize": func(g *PdfGlobal) (string, bool) {
			if g.PageSize != "" {
				return g.PageSize, true
			}
			return g.Size.PageSize, true
		},
		"size.width":  func(g *PdfGlobal) (string, bool) { return fmtFloat(g.Size.Width), true },
		"size.height": func(g *PdfGlobal) (string, bool) { return fmtFloat(g.Size.Height), true },
	}
	for _, e := range []string{"top", "bottom", "left", "right"} {
		edge := e
		s["margin."+edge] = func(g *PdfGlobal) (string, bool) {
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
		}
	}
	for _, k := range []string{"fontsize", "fontname", "left", "right", "center", "line", "spacing", "htmlurl"} {
		key := k
		s["header."+key] = func(g *PdfGlobal) (string, bool) { return hfGet(&g.Header, key) }
		s["footer."+key] = func(g *PdfGlobal) (string, bool) { return hfGet(&g.Footer, key) }
	}
	for _, k := range []string{"fontscale", "indentation", "dottedlines", "captiontext", "forwardlinks", "backlinks", "xslstylesheet"} {
		key := k
		s["toc."+key] = func(g *PdfGlobal) (string, bool) { return tocGet(&g.TOC, key) }
	}
	for _, k := range []string{
		"images", "printmediatype", "mediatype",
		"simplifydom", "simplifydomprofile", "printlinkunderline",
	} {
		key := k
		s["web."+key] = func(g *PdfGlobal) (string, bool) { return webGet(&g.Web, key) }
	}
	return s
}

func buildObjectGetTable() map[string]objectGet {
	s := map[string]objectGet{
		"page":             func(o *PdfObject) (string, bool) { return o.Page, true },
		"externallinks":    func(o *PdfObject) (string, bool) { return fmtBool(o.ExternalLinks), true },
		"locallinks":       func(o *PdfObject) (string, bool) { return fmtBool(o.LocalLinks), true },
		"includeinoutline": func(o *PdfObject) (string, bool) { return fmtBool(o.IncludeInOutline), true },
		"useoutline":       func(o *PdfObject) (string, bool) { return fmtBool(o.UseOutline), true },
		"istableofcontent": func(o *PdfObject) (string, bool) { return fmtBool(o.IsTableOfContent), true },
		"iscover":          func(o *PdfObject) (string, bool) { return fmtBool(o.IsCover), true },
	}
	for _, k := range []string{
		"zoomfactor", "blocklocalfileaccess", "loaderrorhandling",
		"username", "password", "mediatype", "printmediatype", "timeout",
	} {
		key := k
		s["load."+key] = func(o *PdfObject) (string, bool) { return loadGet(&o.Load, key) }
	}
	for _, k := range []string{
		"background", "images", "printmediatype", "mediatype",
		"simplifydom", "simplifydomprofile", "printlinkunderline",
	} {
		key := k
		s["web."+key] = func(o *PdfObject) (string, bool) { return webGet(&o.Web, key) }
	}
	for _, k := range []string{"fontsize", "fontname", "left", "right", "center", "line", "spacing", "htmlurl"} {
		key := k
		s["header."+key] = func(o *PdfObject) (string, bool) { return hfGet(&o.Header, key) }
		s["footer."+key] = func(o *PdfObject) (string, bool) { return hfGet(&o.Footer, key) }
	}
	for _, k := range []string{"fontscale", "indentation", "dottedlines", "captiontext", "forwardlinks", "backlinks", "xslstylesheet"} {
		key := k
		s["toc."+key] = func(o *PdfObject) (string, bool) { return tocGet(&o.TOC, key) }
	}
	return s
}

func buildImageGetTable() map[string]imageGet {
	s := map[string]imageGet{
		"width":       func(g *ImageGlobal) (string, bool) { return fmtInt(g.Width), true },
		"height":      func(g *ImageGlobal) (string, bool) { return fmtInt(g.Height), true },
		"quality":     func(g *ImageGlobal) (string, bool) { return fmtInt(g.Quality), true },
		"smartwidth":  func(g *ImageGlobal) (string, bool) { return fmtBool(g.SmartWidth), true },
		"transparent": func(g *ImageGlobal) (string, bool) { return fmtBool(g.Transparent), true },
		"format":      func(g *ImageGlobal) (string, bool) { return g.Format, true },
		"crop.left":   func(g *ImageGlobal) (string, bool) { return fmtInt(g.Crop.Left), true },
		"crop.top":    func(g *ImageGlobal) (string, bool) { return fmtInt(g.Crop.Top), true },
		"crop.width":  func(g *ImageGlobal) (string, bool) { return fmtInt(g.Crop.Width), true },
		"crop.height": func(g *ImageGlobal) (string, bool) { return fmtInt(g.Crop.Height), true },
		"proxy":       func(g *ImageGlobal) (string, bool) { return g.Load.Proxy, true },
	}
	for _, k := range []string{
		"background", "images", "printmediatype", "mediatype",
		"simplifydom", "simplifydomprofile", "printlinkunderline",
	} {
		key := k
		s["web."+key] = func(g *ImageGlobal) (string, bool) { return webGet(&g.Web, key) }
	}
	return s
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

func webGet(w *Web, key string) (string, bool) {
	switch key {
	case "background":
		return fmtBool(w.Background), true
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
