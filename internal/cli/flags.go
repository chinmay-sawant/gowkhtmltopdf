package cli

import (
	"strconv"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// flagApplier receives (cmd, cur, vals) and mutates settings. Bool flags get
// one canonical "true"/"false"; value flags one token; pair flags two.
type flagApplier func(c *Command, cur *objectCtx, vals []string) error

// flagAdder registers one flag spec into the table being built.
type flagAdder func(name string, m Mode, kind flagKind, app flagApplier)

// mode selects which binaries accept a flag.
type Mode int

const (
	ModePDF Mode = 1 << iota
	ModeImage
	ModeBoth = ModePDF | ModeImage
)

// flagTable maps long flag names to their specs. It is a static lookup table
// (built once at package init, never mutated afterwards).
var flagTable = buildFlagTable() //nolint:gochecknoglobals // static flag lookup table

// shortFlags maps single-char flags to their long-form specs. It must be
// declared after flagTable so initialization dependencies resolve in order.
var shortFlags = map[string]flagSpec{ //nolint:gochecknoglobals // static flag lookup table
	"o": flagTable["output"],
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

// Canonical boolean strings stored in settings; negBool flips between them.
const (
	canonicalTrue  = "true"
	canonicalFalse = "false"
)

// flagTableSize is the estimated long-flag count (capacity hint only).
const flagTableSize = 120

// setMapEntry inserts into a nil-safe string map.
func setMapEntry(m *map[string]string, k, v string) {
	if *m == nil {
		*m = map[string]string{}
	}

	(*m)[k] = v
}

// buildFlagTable assembles the long-flag table, grouped by policy so each
// group stays small and reviewable.
func buildFlagTable() map[string]flagSpec {
	table := make(map[string]flagSpec, flagTableSize)
	add := func(name string, m Mode, kind flagKind, app flagApplier) {
		table[name] = flagSpec{kind: kind, mod: m, app: app}
	}

	addDocFlags(add)
	addDocumentFlags(add)
	addGlobalFlags(add)
	addOutlineFlags(add)
	addLocalAccessFlags(add)
	addWebPageFlags(add)
	addLoadPageFlags(add)
	addFontFlags(add)
	addPairFlags(add)
	addHeaderFooterFlags(add)
	addTOCFlags(add)
	addImageFlags(add)

	return table
}

// addDocFlags registers doc flags (handled by Parse before table lookup;
// present so the --help listing can include them).
func addDocFlags(add flagAdder) {
	add("help", ModeBoth, flagBool, nopFlag)
	add("version", ModeBoth, flagBool, nopFlag)
	add("license", ModeBoth, flagBool, nopFlag)
	add("extended-help", ModeBoth, flagBool, nopFlag)
}

// addGlobalFlags registers global PDF flags (engine-consumed only; Policy A).
func addGlobalFlags(add flagAdder) {
	add("quiet", ModeBoth, flagBool, func(command *Command, _ *objectCtx, vals []string) error {
		return command.Global.Set("quiet", vals[0])
	})
	add("collate", ModePDF, flagBool, func(command *Command, _ *objectCtx, vals []string) error {
		return command.Global.Set("collate", vals[0])
	})
	add("copies", ModePDF, flagValue, func(command *Command, _ *objectCtx, vals []string) error {
		return command.Global.Set("copies", vals[0])
	})
	add("orientation", ModePDF, flagValue, func(command *Command, _ *objectCtx, vals []string) error {
		return command.Global.Set("orientation", vals[0])
	})
	add("page-size", ModePDF, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("size.pagesize", vals[0])
	})
	// convert reads Global.Grayscale only (ColorMode is not a stored field).
	add("grayscale", ModePDF, flagBool, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("grayscale", vals[0])
	})
	add("title", ModePDF, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("title", vals[0])
	})
	add("margin-top", ModePDF, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("margin.top", vals[0])
	})
	add("margin-bottom", ModePDF, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("margin.bottom", vals[0])
	})
	add("margin-left", ModePDF, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("margin.left", vals[0])
	})
	add("margin-right", ModePDF, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("margin.right", vals[0])
	})
	add("page-width", ModePDF, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("size.width", vals[0])
	})
	add("page-height", ModePDF, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("size.height", vals[0])
	})
	add("no-pdf-compression", ModePDF, flagBool, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("usecompression", negBool(vals[0]))
	})
	add("page-offset", ModePDF, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("pageoffset", vals[0])
	})
	add("pdf-version", ModePDF, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("pdfversion", vals[0])
	})
	add("pdf-profile", ModePDF, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("pdfprofile", vals[0])
	})
	// Smart-shrinking: enable/disable pair only (no bare --smart-shrinking).
	add("enable-smart-shrinking", ModePDF, flagBool, func(c *Command, _ *objectCtx, _ []string) error {
		return c.Global.Set("smartshrinking", "true")
	})
	add("disable-smart-shrinking", ModePDF, flagBool, func(c *Command, _ *objectCtx, _ []string) error {
		return c.Global.Set("smartshrinking", "false")
	})
}

// addDocumentFlags registers the document-shaped source and output grammar.
// These flags are deliberately separate from the private settings mapping so
// the parser exposes a stable Document-like boundary to both binaries.
func addDocumentFlags(add flagAdder) {
	add("output", ModeBoth, flagValue, func(command *Command, _ *objectCtx, vals []string) error {
		if command.outputSet {
			return ErrDuplicateOutput
		}

		command.Output = vals[0]
		command.outputSet = true

		return nil
	})
	add("html", ModeBoth, flagValue, func(command *Command, _ *objectCtx, vals []string) error {
		if command.htmlSet {
			return errDuplicateSource
		}

		command.htmlSource = vals[0]
		command.htmlSet = true

		return nil
	})
	add("url", ModeBoth, flagValue, func(command *Command, _ *objectCtx, vals []string) error {
		if command.urlSet {
			return errDuplicateSource
		}

		command.urlSource = vals[0]
		command.urlSet = true

		return nil
	})
	add("cover", ModePDF, flagValue, func(command *Command, _ *objectCtx, vals []string) error {
		if command.coverSet {
			return errDuplicateSource
		}

		command.coverSource = vals[0]
		command.coverSet = true

		return nil
	})
	add("toc", ModePDF, flagBool, func(c *Command, _ *objectCtx, vals []string) error {
		c.tocRequested = vals[0] == canonicalTrue

		return nil
	})
}

// addOutlineFlags registers outline flags.
func addOutlineFlags(add flagAdder) {
	add("outline", ModePDF, flagBool, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("outline", vals[0])
	})
	add("outline-depth", ModePDF, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("outlinedepth", vals[0])
	})
	// One home: Global settings (CLI and library both write it); the engine
	// reads Global only. Negation rides the value.
	add("exclude-from-outline", ModePDF, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("excludefromoutline", vals[0])
	})
	add("dump-outline", ModePDF, flagBool, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("dumpoutline", vals[0])
	})
	add("dump-default-toc-xsl", ModePDF, flagBool, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("dumpoutlinewithdefaulttocxsl", vals[0])
	})
}

// addLocalAccessFlags registers the local-file-access pair and the shared
// background paint switch (page-scoped through the one router).
func addLocalAccessFlags(add flagAdder) {
	add("allow-local-files", ModeBoth, flagBool, func(c *Command, cur *objectCtx, vals []string) error {
		allowed := vals[0] == canonicalTrue

		return cur.applyPage(c,
			func(g *settings.PdfGlobal, _ string) error {
				return g.Set("enablelocalfileaccess", strconv.FormatBool(allowed))
			},
			func(o *settings.PdfObject, _ string) error {
				return o.Set("load.blocklocalfileaccess", strconv.FormatBool(!allowed))
			},
			vals[0],
		)
	})
	add("allow", ModeBoth, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("allow", vals[0])
	})
	// Restricted network policy: private destinations and cross-host
	// redirects are denied; HTTP(S) remain the only allowed schemes.
	add("restrict-network", ModeBoth, flagBool, func(cmd *Command, _ *objectCtx, vals []string) error {
		if vals[0] != canonicalTrue {
			return nil
		}

		cmd.Global.Load.NetworkPolicySet = true
		cmd.Global.Load.NetworkBlockPrivate = true
		cmd.Global.Load.NetworkBlockCrossHost = true
		cmd.Global.Load.NetworkAllowedSchemes = []string{"http", "https"}

		return nil
	})
	// Exact or wildcard host allowlist entry. Sets NetworkPolicySet so
	// NewLoader does not fall back to CompatibleNetworkPolicy.
	add("allow-host", ModeBoth, flagValue, func(cmd *Command, _ *objectCtx, vals []string) error {
		cmd.Global.Load.NetworkPolicySet = true
		cmd.Global.Load.NetworkAllowedHosts = append(cmd.Global.Load.NetworkAllowedHosts, vals[0])

		return nil
	})
	// PDF convert and imageout both read Global.Background (Policy A single field).
	add("background", ModeBoth, flagBool, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("background", vals[0])
	})
	add("no-background", ModeBoth, flagBool, func(c *Command, _ *objectCtx, _ []string) error {
		return c.Global.Set("background", "false")
	})
}

// addWebPageFlags registers the web engine flags (simplify-dom, link
// underline, print media type) routed global+object.
func addWebPageFlags(add flagAdder) {
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
}

// addLoadPageFlags registers loader/page flags: page-only flags (zoom, auth,
// timeout, links) plus the load-error-handling and proxy routers.
func addLoadPageFlags(add flagAdder) {
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
	add("resolve-relative-links", ModePDF, flagBool, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("resolverelativelinks", vals[0])
	})
	add("keep-relative-links", ModePDF, flagBool, func(c *Command, _ *objectCtx, _ []string) error {
		return c.Global.Set("resolverelativelinks", "false")
	})
}

// addFontFlags registers the font-path flags.
func addFontFlags(add flagAdder) {
	add("font-path", ModeBoth, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("fontpath", vals[0])
	})
	add("use-system-fonts", ModeBoth, flagBool, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Global.Set("usesystemfonts", vals[0])
	})
}

// addPairFlags registers pair flags (two values: name value).
func addPairFlags(add flagAdder) {
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
}

// addHeaderFooterFlags registers header/footer flags (name encodes
// header|footer).
func addHeaderFooterFlags(add flagAdder) {
	for _, prefix := range []string{"header", "footer"} {
		for _, side := range []string{"left", "right", "center"} {
			add(prefix+"-"+side, ModePDF, flagValue, hfFlag(prefix, side))
		}

		add(prefix+"-font-name", ModePDF, flagValue, hfFlag(prefix, "fontname"))
		add(prefix+"-font-size", ModePDF, flagValue, hfFlag(prefix, "fontsize"))
		add(prefix+"-spacing", ModePDF, flagValue, hfFlag(prefix, "spacing"))
		add(prefix+"-line", ModePDF, flagBool, hfFlag(prefix, "line"))
		add(prefix+"-html", ModePDF, flagValue, hfFlag(prefix, "htmlurl"))
	}
}

// addTOCFlags registers TOC flags.
func addTOCFlags(add flagAdder) {
	add("xsl-style-sheet", ModePDF, flagValue, tocFlag("xslstylesheet"))
	add("toc-header-text", ModePDF, flagValue, tocFlag("captiontext"))
	add("toc-text-size-shrink", ModePDF, flagValue, tocFlag("fontscale"))
	add("disable-toc-links", ModePDF, flagBool, tocFlagBool("forwardlinks", false))
	add("disable-dotted-lines", ModePDF, flagBool, tocFlagBool("dottedlines", false))
	add("toc-level-indentation", ModePDF, flagValue, tocFlag("indentation"))
	add("toc-forward-links", ModePDF, flagBool, tocFlagBool("forwardlinks", true))
	add("toc-back-links", ModePDF, flagBool, tocFlagBool("backlinks", true))
}

// addImageFlags registers image flags (wkhtmltoimage).
func addImageFlags(add flagAdder) {
	add("width", ModeImage, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Image.Set("width", vals[0])
	})
	add("height", ModeImage, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Image.Set("height", vals[0])
	})
	add("crop-x", ModeImage, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Image.Set("crop.left", vals[0])
	})
	add("crop-y", ModeImage, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Image.Set("crop.top", vals[0])
	})
	add("crop-w", ModeImage, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Image.Set("crop.width", vals[0])
	})
	add("crop-h", ModeImage, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Image.Set("crop.height", vals[0])
	})
	add("format", ModeImage, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Image.Set("format", vals[0])
	})
	add("quality", ModeImage, flagValue, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Image.Set("quality", vals[0])
	})
	add("transparent", ModeImage, flagBool, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Image.Set("transparent", vals[0])
	})
	add("smart-width", ModeImage, flagBool, func(c *Command, _ *objectCtx, vals []string) error {
		return c.Image.Set("smartwidth", vals[0])
	})
	add("no-smart-width", ModeImage, flagBool, func(c *Command, _ *objectCtx, _ []string) error {
		return c.Image.Set("smartwidth", "false")
	})
}

func nopFlag(*Command, *objectCtx, []string) error { return nil }

// printMediaFlag writes the print-media-type override to one field home —
// Global.Web.PrintMediaType — plus the object loader override through the one
// router (address remapping included). Image mode shares the global home;
// ApplyImageKey/ImageConverter.Set route "web.printmediatype" the same way.
func printMediaFlag(enable bool) flagApplier {
	return func(cmd *Command, cur *objectCtx, vals []string) error {
		enabled := enable
		if enable {
			enabled = vals[0] == "true"
		}

		return cur.applyPage(cmd,
			func(g *settings.PdfGlobal, val string) error { return g.Set("web.printmediatype", val) },
			func(o *settings.PdfObject, val string) error { return o.Set("load.printmediatype", val) },
			strconv.FormatBool(enabled),
		)
	}
}

// pageOnlyFlag routes a page-only flag (zoom, username, password, timeout,
// external-links, internal-links). There is no Global consumer; flags apply
// to the current object, or accumulate as pending first-page settings when
// they appear before any page/cover/toc keyword (upstream address remapping:
// `--zoom 0.67 url out.pdf` stamps zoom on the first page). TOC objects do
// not consume pending (see newFreshObject), so pre-object zoom still lands
// on the first real page after a leading toc.
func pageOnlyFlag(obj func(o *settings.PdfObject, val string) error) flagApplier {
	return func(_ *Command, cur *objectCtx, vals []string) error {
		return obj(cur.object(nil), vals[0])
	}
}

// hfFlag targets header.* or footer.* on the current object, falling back to
// global-only storage before any object keyword (so every object inherits the
// value via HeaderFor/FooterFor). Explicit global-only routing — pending is
// never created.
func hfFlag(prefix, field string) flagApplier {
	return func(cmd *Command, cur *objectCtx, vals []string) error {
		key := prefix + "." + field
		if cur.obj != nil {
			return cur.obj.Set(key, vals[0])
		}

		return cmd.Global.Set(key, vals[0])
	}
}

// tocFlag targets a toc.* key on the current object when it is a toc object,
// else global-only (every toc object inherits via effectiveTOC).
func tocFlag(field string) flagApplier {
	return func(cmd *Command, cur *objectCtx, vals []string) error {
		if cur.obj != nil && cur.obj.IsTableOfContent {
			return cur.obj.Set("toc."+field, vals[0])
		}

		return cmd.Global.Set("toc."+field, vals[0])
	}
}

func tocFlagBool(field string, on bool) flagApplier {
	return func(cmd *Command, cur *objectCtx, vals []string) error {
		val := vals[0]
		if !on {
			val = negBool(val)
		}

		if cur.obj != nil && cur.obj.IsTableOfContent {
			return cur.obj.Set("toc."+field, val)
		}

		return cmd.Global.Set("toc."+field, val)
	}
}

func (c *Command) replaceHF(obj *settings.PdfObject, key, val string) error {
	if obj.HeaderSet {
		if obj.Header.Replace == nil {
			obj.Header.Replace = map[string]string{}
		}

		obj.Header.Replace[key] = val

		return nil
	}

	if c.Global.Header.Replace == nil {
		c.Global.Header.Replace = map[string]string{}
	}

	c.Global.Header.Replace[key] = val

	return nil
}

// negBool flips a canonical bool string.
func negBool(v string) string {
	if v == canonicalTrue {
		return canonicalFalse
	}

	return canonicalTrue
}
