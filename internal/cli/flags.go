package cli

import (
	"gowkhtmltopdf/internal/settings"
)

// flagApplier receives (cmd, cur, value) and mutates settings. For pair
// flags the value is two tokens joined by \x00.
type flagApplier func(c *Command, cur *objectCtx, val string) error

// mode selects which binaries accept a flag.
type Mode int

const (
	ModePDF Mode = 1 << iota
	ModeImage
	ModeBoth = ModePDF | ModeImage
)

const pairSep = "\x00"

var flagTable = map[string]flagSpec{}

// shortFlags maps single-char flags to their long-form specs. Populated in
// init after flagTable so lookups resolve.
var shortFlags = map[string]flagSpec{}

// isPairFlag reports whether the flag consumes two values.
func isPairFlag(name string) bool {
	switch name {
	case "cookie", "custom-header", "post", "replace":
		return true
	}
	return false
}

func init() {
	add := func(name string, m Mode, kind string, app flagApplier) {
		flagTable[name] = flagSpec{kind: kind, mod: m, app: app}
	}

	// --- doc flags (handled by Parse before table lookup; present so
	// --help listing can include them) ---
	add("help", ModeBoth, "bool", nopFlag)
	add("version", ModeBoth, "bool", nopFlag)
	add("license", ModeBoth, "bool", nopFlag)
	add("extended-help", ModeBoth, "bool", nopFlag)
	add("man", ModeBoth, "bool", func(c *Command, cur *objectCtx, val string) error {
		c.Man = true
		return nil
	})
	add("html", ModeBoth, "bool", func(c *Command, cur *objectCtx, val string) error {
		c.HTMLHelp = true
		return nil
	})

	// --- global PDF flags ---
	add("quiet", ModeBoth, "bool", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("quiet", val)
	})
	add("log-level", ModeBoth, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("log-level", val)
	})
	add("collate", ModePDF, "bool", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("collate", val)
	})
	add("copies", ModePDF, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("copies", val)
	})
	add("orientation", ModePDF, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("orientation", val)
	})
	add("page-size", ModePDF, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("size.pagesize", val)
	})
	add("grayscale", ModePDF, "bool", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("colormode", boolVal(val, "grayscale", "color"))
	})
	add("lowquality", ModePDF, "bool", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("lowquality", val)
	})
	add("title", ModePDF, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("title", val)
	})
	add("margin-top", ModePDF, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("margin.top", val)
	})
	add("margin-bottom", ModePDF, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("margin.bottom", val)
	})
	add("margin-left", ModePDF, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("margin.left", val)
	})
	add("margin-right", ModePDF, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("margin.right", val)
	})
	add("dpi", ModePDF, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("dpi", val)
	})
	add("page-width", ModePDF, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("size.width", val)
	})
	add("page-height", ModePDF, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("size.height", val)
	})
	add("image-quality", ModeBoth, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("imagequality", val)
	})
	add("image-dpi", ModePDF, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("imagedpi", val)
	})
	add("no-pdf-compression", ModePDF, "bool", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("usecompression", negBool(val))
	})
	add("use-xserver", ModeBoth, "bool", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("usexserver", val)
	})
	add("cookie-jar", ModeBoth, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("cookiejar", val)
	})
	add("read-args-from-stdin", ModeBoth, "bool", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("readargsfromstdin", val)
	})
	add("page-offset", ModePDF, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("pageoffset", val)
	})
	add("smart-shrinking", ModePDF, "bool", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("smartshrinking", val)
	})
	add("enable-smart-shrinking", ModePDF, "bool", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("smartshrinking", "true")
	})
	add("disable-smart-shrinking", ModePDF, "bool", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("smartshrinking", "false")
	})

	// --- outline flags ---
	add("outline", ModePDF, "bool", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("outline", val)
	})
	add("outline-depth", ModePDF, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("outlinedepth", val)
	})
	add("dump-outline", ModePDF, "bool", func(c *Command, cur *objectCtx, val string) error {
		c.DumpOutline = true
		return nil
	})
	add("dump-default-toc-xsl", ModePDF, "bool", func(c *Command, cur *objectCtx, val string) error {
		c.DumpDefaultTOCXSL = true
		return nil
	})

	// --- shared web/load flags (page-scoped: current object else global) ---
	add("enable-javascript", ModeBoth, "bool", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return g.Set("web.javascript", val) },
		func(o *settings.PdfObject, val string) error { return o.Set("web.javascript", val) },
	))
	add("disable-javascript", ModeBoth, "bool", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return g.Set("web.javascript", "false") },
		func(o *settings.PdfObject, val string) error { return o.Set("web.javascript", "false") },
	))
	add("enable-local-file-access", ModeBoth, "bool", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return g.Set("enablelocalfileaccess", val) },
		func(o *settings.PdfObject, val string) error { return o.Set("load.blocklocalfileaccess", negBool(val)) },
	))
	add("disable-local-file-access", ModeBoth, "bool", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return g.Set("enablelocalfileaccess", "false") },
		func(o *settings.PdfObject, val string) error { return o.Set("load.blocklocalfileaccess", "true") },
	))
	add("allow", ModePDF, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("allow", val)
	})
	add("background", ModeBoth, "bool", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return g.Set("background", val) },
		func(o *settings.PdfObject, val string) error { return o.Set("web.background", val) },
	))
	add("no-background", ModeBoth, "bool", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return g.Set("background", "false") },
		func(o *settings.PdfObject, val string) error { return o.Set("web.background", "false") },
	))
	add("enable-plugins", ModeBoth, "bool", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return g.Set("web.plugins", val) },
		func(o *settings.PdfObject, val string) error { return o.Set("web.plugins", val) },
	))
	add("disable-plugins", ModeBoth, "bool", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return g.Set("web.plugins", "false") },
		func(o *settings.PdfObject, val string) error { return o.Set("web.plugins", "false") },
	))
	add("default-encoding", ModeBoth, "value", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return g.Set("web.defaultencoding", val) },
		func(o *settings.PdfObject, val string) error { return o.Set("web.defaultencoding", val) },
	))
	add("minimum-font-size", ModeBoth, "value", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return g.Set("web.minimumfontsize", val) },
		func(o *settings.PdfObject, val string) error { return o.Set("web.minimumfontsize", val) },
	))
	add("user-style-sheet", ModeBoth, "value", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return g.Set("web.userstylesheet", val) },
		func(o *settings.PdfObject, val string) error { return o.Set("web.userstylesheet", val) },
	))
	add("print-media-type", ModeBoth, "bool", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return g.Set("web.printmediatype", val) },
		func(o *settings.PdfObject, val string) error { return o.Set("load.printmediatype", val) },
	))
	add("no-print-media-type", ModeBoth, "bool", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return g.Set("web.printmediatype", "false") },
		func(o *settings.PdfObject, val string) error { return o.Set("load.printmediatype", "false") },
	))
	add("media-type", ModeBoth, "value", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return nil },
		func(o *settings.PdfObject, val string) error { return o.Set("load.mediatype", val) },
	))
	add("javascript-delay", ModeBoth, "value", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return nil },
		func(o *settings.PdfObject, val string) error { return o.Set("load.jsdelay", val) },
	))
	add("window-status", ModeBoth, "value", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return nil },
		func(o *settings.PdfObject, val string) error { return o.Set("load.windowstatus", val) },
	))
	add("run-script", ModeBoth, "value", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return nil },
		func(o *settings.PdfObject, val string) error { return o.Set("load.runscript", val) },
	))
	add("zoom", ModeBoth, "value", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return nil },
		func(o *settings.PdfObject, val string) error { return o.Set("load.zoomfactor", val) },
	))
	add("stop-slow-scripts", ModeBoth, "bool", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return nil },
		func(o *settings.PdfObject, val string) error { return o.Set("load.stopslowscripts", val) },
	))
	add("no-stop-slow-scripts", ModeBoth, "bool", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return nil },
		func(o *settings.PdfObject, val string) error { return o.Set("load.stopslowscripts", "false") },
	))
	add("debug-javascript", ModeBoth, "bool", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return nil },
		func(o *settings.PdfObject, val string) error { return o.Set("load.debugjavascript", val) },
	))
	add("no-debug-javascript", ModeBoth, "bool", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return nil },
		func(o *settings.PdfObject, val string) error { return o.Set("load.debugjavascript", "false") },
	))
	add("load-error-handling", ModeBoth, "value", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return g.Set("loaderrorhandling", val) },
		func(o *settings.PdfObject, val string) error { return o.Set("load.loaderrorhandling", val) },
	))
	add("load-media-error-handling", ModeBoth, "value", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return nil },
		func(o *settings.PdfObject, val string) error { return nil },
	))
	add("proxy", ModeBoth, "value", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return g.Set("proxy", val) },
		func(o *settings.PdfObject, val string) error { return o.Set("load.proxy", val) },
	))
	add("username", ModeBoth, "value", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return nil },
		func(o *settings.PdfObject, val string) error { return o.Set("load.username", val) },
	))
	add("password", ModeBoth, "value", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return nil },
		func(o *settings.PdfObject, val string) error { return o.Set("load.password", val) },
	))
	add("custom-header-propagation", ModeBoth, "bool", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return nil },
		func(o *settings.PdfObject, val string) error { return o.Set("load.repeatexternalheaders", val) },
	))
	add("no-custom-header-propagation", ModeBoth, "bool", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return nil },
		func(o *settings.PdfObject, val string) error { return o.Set("load.repeatexternalheaders", "false") },
	))
	add("timeout", ModeBoth, "value", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return nil },
		func(o *settings.PdfObject, val string) error { return o.Set("load.timeout", val) },
	))
	add("external-links", ModePDF, "bool", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return nil },
		func(o *settings.PdfObject, val string) error { return o.Set("externallinks", val) },
	))
	add("internal-links", ModePDF, "bool", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return nil },
		func(o *settings.PdfObject, val string) error { return o.Set("locallinks", val) },
	))
	add("resolve-relative-links", ModePDF, "bool", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("resolverelativelinks", val)
	})
	add("keep-relative-links", ModePDF, "bool", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("resolverelativelinks", "false")
	})
	add("font-path", ModeBoth, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("fontpath", val)
	})
	add("use-system-fonts", ModeBoth, "bool", func(c *Command, cur *objectCtx, val string) error {
		return c.Global.Set("usesystemfonts", val)
	})
	add("produce-forms", ModePDF, "bool", pageScoped(
		func(g *settings.PdfGlobal, val string) error { return g.Set("produceforms", val) },
		func(o *settings.PdfObject, val string) error { return o.Set("produceforms", val) },
	))

	// --- pair flags (two values: name value) ---
	add("cookie", ModeBoth, "value", func(c *Command, cur *objectCtx, val string) error {
		k, v := splitPair(val)
		o := cur.object(c)
		if o.Load.Cookies == nil {
			o.Load.Cookies = map[string]string{}
		}
		o.Load.Cookies[k] = v
		return nil
	})
	add("custom-header", ModeBoth, "value", func(c *Command, cur *objectCtx, val string) error {
		k, v := splitPair(val)
		o := cur.object(c)
		if o.Load.CustomHeaders == nil {
			o.Load.CustomHeaders = map[string]string{}
		}
		o.Load.CustomHeaders[k] = v
		return nil
	})
	add("post", ModeBoth, "value", func(c *Command, cur *objectCtx, val string) error {
		k, v := splitPair(val)
		o := cur.object(c)
		o.Load.Post = append(o.Load.Post, settings.PostItem{Name: k, Value: v})
		return nil
	})
	add("replace", ModePDF, "value", func(c *Command, cur *objectCtx, val string) error {
		k, v := splitPair(val)
		o := cur.object(c)
		return c.replaceHF(o, k, v)
	})

	// --- header/footer flags (name encodes header|footer) ---
	for _, prefix := range []string{"header", "footer"} {
		for _, side := range []string{"left", "right", "center"} {
			prefix, side := prefix, side
			add(prefix+"-"+side, ModePDF, "value", hfFlag(prefix, side, "text"))
		}
		add(prefix+"-font-name", ModePDF, "value", hfFlag(prefix, "fontname", "text"))
		add(prefix+"-font-size", ModePDF, "value", hfFlag(prefix, "fontsize", "text"))
		add(prefix+"-spacing", ModePDF, "value", hfFlag(prefix, "spacing", "text"))
		add(prefix+"-line", ModePDF, "bool", hfFlag(prefix, "line", "bool"))
		add(prefix+"-html", ModePDF, "value", hfFlag(prefix, "htmlurl", "text"))
	}

	// --- TOC flags ---
	add("xsl-style-sheet", ModePDF, "value", tocFlag("xslstylesheet"))
	add("toc-header-text", ModePDF, "value", tocFlag("captiontext"))
	add("toc-text-size-shrink", ModePDF, "value", tocFlag("fontscale"))
	add("disable-toc-links", ModePDF, "bool", tocFlagBool("forwardlinks", false))
	add("disable-dotted-lines", ModePDF, "bool", tocFlagBool("dottedlines", false))
	add("toc-level-indentation", ModePDF, "value", tocFlag("indentation"))
	add("toc-forward-links", ModePDF, "bool", tocFlagBool("forwardlinks", true))
	add("toc-back-links", ModePDF, "bool", tocFlagBool("backlinks", true))

	// --- image flags (wkhtmltoimage) ---
	add("width", ModeImage, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Image.Set("width", val)
	})
	add("height", ModeImage, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Image.Set("height", val)
	})
	add("crop-x", ModeImage, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Image.Set("crop.left", val)
	})
	add("crop-y", ModeImage, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Image.Set("crop.top", val)
	})
	add("crop-w", ModeImage, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Image.Set("crop.width", val)
	})
	add("crop-h", ModeImage, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Image.Set("crop.height", val)
	})
	add("format", ModeImage, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Image.Set("format", val)
	})
	add("quality", ModeImage, "value", func(c *Command, cur *objectCtx, val string) error {
		return c.Image.Set("quality", val)
	})
	add("transparent", ModeImage, "bool", func(c *Command, cur *objectCtx, val string) error {
		return c.Image.Set("transparent", val)
	})
	add("smart-width", ModeImage, "bool", func(c *Command, cur *objectCtx, val string) error {
		return c.Image.Set("smartwidth", val)
	})
	add("no-smart-width", ModeImage, "bool", func(c *Command, cur *objectCtx, val string) error {
		return c.Image.Set("smartwidth", "false")
	})
}

func nopFlag(*Command, *objectCtx, string) error { return nil }

func init() {
	shortFlags = map[string]flagSpec{
		"q": flagTable["quiet"],
		"g": flagTable["grayscale"],
		"O": flagTable["orientation"],
		"s": flagTable["page-size"],
		"T": flagTable["margin-top"],
		"B": flagTable["margin-bottom"],
		"L": flagTable["margin-left"],
		"R": flagTable["margin-right"],
		"c": flagTable["copies"],
		"t": flagTable["title"],
	}
}

// pageScoped routes a flag to the current object when one exists, else
// accumulates it as pending first-page settings (upstream address remapping:
// page settings before any object keyword apply to the first page). Pending
// settings are not inserted into Objects until a real page/cover is created,
// so a leading --enable-local-file-access does not leave an empty ghost page
// when the next token is toc/cover/page.
func pageScoped(glob func(g *settings.PdfGlobal, val string) error, obj func(o *settings.PdfObject, val string) error) flagApplier {
	return func(c *Command, cur *objectCtx, val string) error {
		if err := glob(&c.Global, val); err != nil {
			return err
		}
		if cur.obj != nil {
			return obj(cur.obj, val)
		}
		if cur.pending == nil {
			cur.pending = &settings.PdfObject{}
		}
		return obj(cur.pending, val)
	}
}

// hfFlag targets header.* or footer.* on the current object or global.
func hfFlag(prefix, field, kind string) flagApplier {
	return func(c *Command, cur *objectCtx, val string) error {
		key := prefix + "." + field
		if kind == "bool" {
			val = boolVal(val, "true", "false")
		}
		if cur.obj != nil {
			return cur.obj.Set(key, val)
		}
		return c.Global.Set(key, val)
	}
}

func tocFlag(field string) flagApplier {
	return func(c *Command, cur *objectCtx, val string) error {
		if cur.obj != nil && cur.obj.IsTableOfContent {
			return cur.obj.Set("toc."+field, val)
		}
		return c.Global.Set("toc."+field, val)
	}
}

func tocFlagBool(field string, on bool) flagApplier {
	return func(c *Command, cur *objectCtx, val string) error {
		v := val
		if !on {
			v = negBool(val)
		}
		if cur.obj != nil && cur.obj.IsTableOfContent {
			return cur.obj.Set("toc."+field, v)
		}
		return c.Global.Set("toc."+field, v)
	}
}

func splitPair(val string) (string, string) {
	for i := 0; i < len(val); i++ {
		if val[i] == pairSep[0] {
			return val[:i], val[i+1:]
		}
	}
	return val, ""
}

func (c *Command) replaceHF(o *settings.PdfObject, k, v string) error {
	if o.HeaderSet {
		if o.Header.Replace == nil {
			o.Header.Replace = map[string]string{}
		}
		o.Header.Replace[k] = v
		return nil
	}
	if c.Global.Header.Replace == nil {
		c.Global.Header.Replace = map[string]string{}
	}
	c.Global.Header.Replace[k] = v
	return nil
}

func boolVal(val, ifTrue, ifFalse string) string {
	switch val {
	case "true", "1", "yes", "on":
		return ifTrue
	case "false", "0", "no", "off":
		return ifFalse
	}
	return ifFalse
}

func negBool(val string) string { return boolVal(val, "false", "true") }
