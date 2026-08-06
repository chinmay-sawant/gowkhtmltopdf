package cli

import (
	"fmt"
	"strconv"

	"gowkhtmltopdf/internal/settings"
)

// flagApplier receives (cmd, cur, vals) and mutates settings. Bool flags get
// one canonical "true"/"false"; value flags one token; pair flags two.
type flagApplier func(c *Command, cur *objectCtx, vals []string) error

// mode selects which binaries accept a flag.
type Mode int

const (
	ModePDF Mode = 1 << iota
	ModeImage
	ModeBoth = ModePDF | ModeImage
)

var flagTable = map[string]flagSpec{}

// shortFlags maps single-char flags to their long-form specs. Populated in
// init after flagTable so lookups resolve.
var shortFlags = map[string]flagSpec{}

// setMapEntry inserts into a nil-safe string map.
func setMapEntry(m *map[string]string, k, v string) {
	if *m == nil {
		*m = map[string]string{}
	}
	(*m)[k] = v
}

func init() {
	add := func(name string, m Mode, kind flagKind, app flagApplier) {
		flagTable[name] = flagSpec{kind: kind, mod: m, app: app}
	}

	// --- doc flags (handled by Parse before table lookup; present so
	// --help listing can include them) ---
	add("help", ModeBoth, flagBool, nopFlag)
	add("version", ModeBoth, flagBool, nopFlag)
	add("license", ModeBoth, flagBool, nopFlag)
	add("extended-help", ModeBoth, flagBool, nopFlag)

	// --- global PDF flags (engine-consumed only; Policy A) ---
	add("quiet", ModeBoth, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("quiet", vals[0])
	})
	add("collate", ModePDF, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("collate", vals[0])
	})
	add("copies", ModePDF, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("copies", vals[0])
	})
	add("orientation", ModePDF, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("orientation", vals[0])
	})
	add("page-size", ModePDF, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("size.pagesize", vals[0])
	})
	// convert reads Global.Grayscale only (ColorMode is not a stored field).
	add("grayscale", ModePDF, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("grayscale", vals[0])
	})
	add("title", ModePDF, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("title", vals[0])
	})
	add("margin-top", ModePDF, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("margin.top", vals[0])
	})
	add("margin-bottom", ModePDF, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("margin.bottom", vals[0])
	})
	add("margin-left", ModePDF, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("margin.left", vals[0])
	})
	add("margin-right", ModePDF, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("margin.right", vals[0])
	})
	add("page-width", ModePDF, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("size.width", vals[0])
	})
	add("page-height", ModePDF, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("size.height", vals[0])
	})
	add("no-pdf-compression", ModePDF, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("usecompression", negBool(vals[0]))
	})
	add("page-offset", ModePDF, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("pageoffset", vals[0])
	})
	// Smart-shrinking: enable/disable pair only (no bare --smart-shrinking).
	add("enable-smart-shrinking", ModePDF, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("smartshrinking", "true")
	})
	add("disable-smart-shrinking", ModePDF, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("smartshrinking", "false")
	})

	// --- outline flags ---
	add("outline", ModePDF, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("outline", vals[0])
	})
	add("outline-depth", ModePDF, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("outlinedepth", vals[0])
	})
	// One home: Global settings (CLI and library both write it); the engine
	// reads Global only. Negation rides the value.
	// Dump homes: Global settings only (engine reads Global; main uses
	// Global.DumpDefaultTOCXSL; convert adapter ORs legacy Command.DumpOutline).
	add("dump-outline", ModePDF, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("dumpoutline", vals[0])
	})
	add("dump-default-toc-xsl", ModePDF, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("dumpoutlinewithdefaulttocxsl", vals[0])
	})

	// --- shared load / web flags (page-scoped through the one router) ---
	add("enable-local-file-access", ModeBoth, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return cur.applyPage(c,
			func(g *settings.PdfGlobal, val string) error { return g.Set("enablelocalfileaccess", val) },
			func(o *settings.PdfObject, val string) error { return o.Set("load.blocklocalfileaccess", negBool(val)) },
			vals[0],
		)
	})
	add("disable-local-file-access", ModeBoth, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return cur.applyPage(c,
			func(g *settings.PdfGlobal, val string) error { return g.Set("enablelocalfileaccess", "false") },
			func(o *settings.PdfObject, val string) error { return o.Set("load.blocklocalfileaccess", "true") },
			vals[0],
		)
	})
	add("allow", ModePDF, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("allow", vals[0])
	})
	// PDF convert and imageout both read Global.Background (Policy A single field).
	add("background", ModeBoth, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("background", vals[0])
	})
	add("no-background", ModeBoth, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("background", "false")
	})
	// Opt-in chrome-strip for arbitrary websites (phase 21.4). Default off.
	// Distinct from --print-media-type (PDF layout always uses Media:"print").
	add("simplify-dom", ModeBoth, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return cur.applyPage(c,
			func(g *settings.PdfGlobal, val string) error { return g.Set("web.simplifydom", val) },
			func(o *settings.PdfObject, val string) error { return o.Set("web.simplifydom", val) },
			vals[0],
		)
	})
	// Extra chrome selectors when --simplify-dom is on (empty|mediawiki).
	add("simplify-dom-profile", ModeBoth, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return cur.applyPage(c,
			func(g *settings.PdfGlobal, val string) error { return g.Set("web.simplifydomprofile", val) },
			func(o *settings.PdfObject, val string) error { return o.Set("web.simplifydomprofile", val) },
			vals[0],
		)
	})
	// Opt-in: underline a[href] after cascade (CSS-faithful default is off).
	add("print-link-underline", ModeBoth, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return cur.applyPage(c,
			func(g *settings.PdfGlobal, val string) error { return g.Set("web.printlinkunderline", val) },
			func(o *settings.PdfObject, val string) error { return o.Set("web.printlinkunderline", val) },
			vals[0],
		)
	})
	// Media flags: one flag writes Global.Web.PrintMediaType plus the object
	// loader override through the router; ResolveMedia owns the resolution.
	add("print-media-type", ModeBoth, flagBool, printMediaFlag(true))
	add("no-print-media-type", ModeBoth, flagBool, printMediaFlag(false))
	add("media-type", ModeBoth, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return cur.applyPage(c,
			func(g *settings.PdfGlobal, val string) error { return g.Set("web.mediatype", val) },
			func(o *settings.PdfObject, val string) error { return o.Set("load.mediatype", val) },
			vals[0],
		)
	})
	add("zoom", ModeBoth, flagValue, pageOnlyFlag(func(o *settings.PdfObject, val string) error {
		return o.Set("load.zoomfactor", val)
	}))
	add("load-error-handling", ModeBoth, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return cur.applyPage(c,
			func(g *settings.PdfGlobal, val string) error { return g.Set("loaderrorhandling", val) },
			func(o *settings.PdfObject, val string) error { return o.Set("load.loaderrorhandling", val) },
			vals[0],
		)
	})
	add("proxy", ModeBoth, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return cur.applyPage(c,
			func(g *settings.PdfGlobal, val string) error { return g.Set("proxy", val) },
			func(o *settings.PdfObject, val string) error { return o.Set("load.proxy", val) },
			vals[0],
		)
	})
	add("username", ModeBoth, flagValue, pageOnlyFlag(func(o *settings.PdfObject, val string) error {
		return o.Set("load.username", val)
	}))
	add("password", ModeBoth, flagValue, pageOnlyFlag(func(o *settings.PdfObject, val string) error {
		return o.Set("load.password", val)
	}))
	add("timeout", ModeBoth, flagValue, pageOnlyFlag(func(o *settings.PdfObject, val string) error {
		return o.Set("load.timeout", val)
	}))
	add("external-links", ModePDF, flagBool, pageOnlyFlag(func(o *settings.PdfObject, val string) error {
		return o.Set("externallinks", val)
	}))
	add("internal-links", ModePDF, flagBool, pageOnlyFlag(func(o *settings.PdfObject, val string) error {
		return o.Set("locallinks", val)
	}))
	add("resolve-relative-links", ModePDF, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("resolverelativelinks", vals[0])
	})
	add("keep-relative-links", ModePDF, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("resolverelativelinks", "false")
	})
	add("font-path", ModeBoth, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("fontpath", vals[0])
	})
	add("use-system-fonts", ModeBoth, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Global.Set("usesystemfonts", vals[0])
	})

	// --- pair flags (two values: name value) ---
	add("cookie", ModeBoth, flagPair, func(c *Command, cur *objectCtx, vals []string) error {
		o := cur.object(c)
		setMapEntry(&o.Load.Cookies, vals[0], vals[1])
		return nil
	})
	add("custom-header", ModeBoth, flagPair, func(c *Command, cur *objectCtx, vals []string) error {
		o := cur.object(c)
		setMapEntry(&o.Load.CustomHeaders, vals[0], vals[1])
		return nil
	})
	add("post", ModeBoth, flagPair, func(c *Command, cur *objectCtx, vals []string) error {
		o := cur.object(c)
		o.Load.Post = append(o.Load.Post, settings.PostItem{Name: vals[0], Value: vals[1]})
		return nil
	})
	add("replace", ModePDF, flagPair, func(c *Command, cur *objectCtx, vals []string) error {
		o := cur.object(c)
		return c.replaceHF(o, vals[0], vals[1])
	})

	// --- header/footer flags (name encodes header|footer) ---
	for _, prefix := range []string{"header", "footer"} {
		for _, side := range []string{"left", "right", "center"} {
			prefix, side := prefix, side
			add(prefix+"-"+side, ModePDF, flagValue, hfFlag(prefix, side))
		}
		add(prefix+"-font-name", ModePDF, flagValue, hfFlag(prefix, "fontname"))
		add(prefix+"-font-size", ModePDF, flagValue, hfFlag(prefix, "fontsize"))
		add(prefix+"-spacing", ModePDF, flagValue, hfFlag(prefix, "spacing"))
		add(prefix+"-line", ModePDF, flagBool, hfFlag(prefix, "line"))
		add(prefix+"-html", ModePDF, flagValue, hfFlag(prefix, "htmlurl"))
	}

	// --- TOC flags ---
	add("xsl-style-sheet", ModePDF, flagValue, tocFlag("xslstylesheet"))
	add("toc-header-text", ModePDF, flagValue, tocFlag("captiontext"))
	add("toc-text-size-shrink", ModePDF, flagValue, tocFlag("fontscale"))
	add("disable-toc-links", ModePDF, flagBool, tocFlagBool("forwardlinks", false))
	add("disable-dotted-lines", ModePDF, flagBool, tocFlagBool("dottedlines", false))
	add("toc-level-indentation", ModePDF, flagValue, tocFlag("indentation"))
	add("toc-forward-links", ModePDF, flagBool, tocFlagBool("forwardlinks", true))
	add("toc-back-links", ModePDF, flagBool, tocFlagBool("backlinks", true))

	// --- image flags (wkhtmltoimage) ---
	add("width", ModeImage, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Image.Set("width", vals[0])
	})
	add("height", ModeImage, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Image.Set("height", vals[0])
	})
	add("crop-x", ModeImage, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Image.Set("crop.left", vals[0])
	})
	add("crop-y", ModeImage, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Image.Set("crop.top", vals[0])
	})
	add("crop-w", ModeImage, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Image.Set("crop.width", vals[0])
	})
	add("crop-h", ModeImage, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Image.Set("crop.height", vals[0])
	})
	add("format", ModeImage, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Image.Set("format", vals[0])
	})
	add("quality", ModeImage, flagValue, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Image.Set("quality", vals[0])
	})
	add("transparent", ModeImage, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Image.Set("transparent", vals[0])
	})
	add("smart-width", ModeImage, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Image.Set("smartwidth", vals[0])
	})
	add("no-smart-width", ModeImage, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		return c.Image.Set("smartwidth", "false")
	})
}

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

func nopFlag(*Command, *objectCtx, []string) error { return nil }

// printMediaFlag writes the print-media-type override to one field home —
// Global.Web.PrintMediaType — plus the object loader override through the one
// router (address remapping included). Image mode shares the global home;
// ApplyImageKey/ImageConverter.Set route "web.printmediatype" the same way.
func printMediaFlag(enable bool) flagApplier {
	return func(c *Command, cur *objectCtx, vals []string) error {
		on := enable
		if enable {
			on = vals[0] == "true"
		}
		return cur.applyPage(c,
			func(g *settings.PdfGlobal, val string) error { return g.Set("web.printmediatype", val) },
			func(o *settings.PdfObject, val string) error { return o.Set("load.printmediatype", val) },
			strconv.FormatBool(on),
		)
	}
}

// pageOnlyFlag routes a page-only flag (zoom, username, password, timeout,
// external-links, internal-links). These have no global consumer, so the
// pre-object position is rejected loudly instead of silently dropping the
// value (upstream address remapping would stamp only the first page).
func pageOnlyFlag(obj func(o *settings.PdfObject, val string) error) flagApplier {
	return func(c *Command, cur *objectCtx, vals []string) error {
		if cur.obj == nil {
			return fmt.Errorf("option must follow a page/cover/toc object")
		}
		return obj(cur.obj, vals[0])
	}
}

// hfFlag targets header.* or footer.* on the current object, falling back to
// global-only storage before any object keyword (so every object inherits the
// value via HeaderFor/FooterFor). Explicit global-only routing — pending is
// never created.
func hfFlag(prefix, field string) flagApplier {
	return func(c *Command, cur *objectCtx, vals []string) error {
		key := prefix + "." + field
		if cur.obj != nil {
			return cur.obj.Set(key, vals[0])
		}
		return c.Global.Set(key, vals[0])
	}
}

// tocFlag targets a toc.* key on the current object when it is a toc object,
// else global-only (every toc object inherits via effectiveTOC).
func tocFlag(field string) flagApplier {
	return func(c *Command, cur *objectCtx, vals []string) error {
		if cur.obj != nil && cur.obj.IsTableOfContent {
			return cur.obj.Set("toc."+field, vals[0])
		}
		return c.Global.Set("toc."+field, vals[0])
	}
}

func tocFlagBool(field string, on bool) flagApplier {
	return func(c *Command, cur *objectCtx, vals []string) error {
		v := vals[0]
		if !on {
			v = negBool(v)
		}
		if cur.obj != nil && cur.obj.IsTableOfContent {
			return cur.obj.Set("toc."+field, v)
		}
		return c.Global.Set("toc."+field, v)
	}
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

// negBool flips a canonical bool string.
func negBool(v string) string {
	if v == "true" {
		return "false"
	}
	return "true"
}
