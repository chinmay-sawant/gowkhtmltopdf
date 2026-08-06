package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"gowkhtmltopdf/internal/settings"
)

// ErrHelp / ErrVersion / ErrLicense / ErrExtHelp are returned for doc flags
// so the caller can print and exit 0.
var (
	ErrHelp    = fmt.Errorf("help requested")
	ErrVersion = fmt.Errorf("version requested")
	ErrLicense = fmt.Errorf("license requested")
	ErrExtHelp = fmt.Errorf("extended help requested")
)

// Exit codes (utilities.cc): 0 ok, 1 error, 2 HTTP 404, 3 HTTP 401.
const (
	ExitOK    = 0
	ExitError = 1
)

// Command is the result of parsing argv.
type Command struct {
	Global  settings.PdfGlobal
	Image   settings.ImageGlobal
	Objects []settings.PdfObject
	// Output is a path or "-" (stdout). Ignored when OutputWriter is set.
	Output string
	// OutputWriter, when non-nil, receives PDF/image bytes directly (library
	// path). Takes precedence over Output so embedders need no temp files.
	OutputWriter io.Writer

	DumpDefaultTOCXSL bool
	DumpOutline       bool
}

// OpenOutput returns the writer for this command: OutputWriter (library bytes
// sink) takes precedence over Output path; empty/"-" use stdout.
// The closer must be called when done (no-op for writer/stdout).
func (cmd *Command) OpenOutput() (io.Writer, func() error, error) {
	if cmd.OutputWriter != nil {
		return cmd.OutputWriter, func() error { return nil }, nil
	}
	if cmd.Output != "" && cmd.Output != "-" {
		f, err := os.Create(cmd.Output)
		if err != nil {
			return nil, nil, fmt.Errorf("output %q: %w", cmd.Output, err)
		}
		return f, f.Close, nil
	}
	return os.Stdout, func() error { return nil }, nil
}

// flagSpec describes one accepted flag.
type flagSpec struct {
	kind string // "bool" | "value"
	mod  Mode   // which binaries accept it
	app  func(c *Command, cur *objectCtx, val string) error
}

type objectCtx struct {
	obj *settings.PdfObject
	// pending holds page-scoped settings seen before any object keyword
	// (upstream "address remapping"). It is promoted into a real object when
	// the first page/cover is created or when free positionals resolve. TOC
	// objects do not consume pending, so `--enable-local-file-access toc page
	// in.html out.pdf` does not leave an empty ghost page.
	pending *settings.PdfObject
}

// object returns the current object, creating one if needed (promoting
// pending page-scoped settings when present).
func (ctx *objectCtx) object(c *Command) *settings.PdfObject {
	if ctx.obj == nil {
		return ctx.newObject(c)
	}
	return ctx.obj
}

// newObject appends a page/cover object and makes it current. If page-scoped
// flags were collected before any object keyword, they seed this object.
func (ctx *objectCtx) newObject(c *Command) *settings.PdfObject {
	if ctx.pending != nil {
		c.Objects = append(c.Objects, *ctx.pending)
		ctx.pending = nil
	} else {
		c.Objects = append(c.Objects, settings.DefaultPdfObject())
	}
	ctx.obj = &c.Objects[len(c.Objects)-1]
	return ctx.obj
}

// newFreshObject appends a new object without consuming pending page-scoped
// settings. Used for toc so pre-object page flags apply to the first real
// page that follows, not to the TOC entry itself.
func (ctx *objectCtx) newFreshObject(c *Command) *settings.PdfObject {
	c.Objects = append(c.Objects, settings.DefaultPdfObject())
	ctx.obj = &c.Objects[len(c.Objects)-1]
	return ctx.obj
}

// Parse parses wkhtmltopdf-style arguments.
func Parse(argv []string) (*Command, error) {
	cmd := &Command{Global: settings.DefaultPdfGlobal(), Image: settings.DefaultImageGlobal()}
	cur := &objectCtx{}
	var free []string

	i := 0
	for i < len(argv) {
		arg := argv[i]
		i++
		switch {
		case arg == "-h" || arg == "--help":
			return cmd, ErrHelp
		case arg == "-V" || arg == "--version":
			return cmd, ErrVersion
		case arg == "-L" || arg == "--license":
			return cmd, ErrLicense
		case arg == "-E" || arg == "--extended-help":
			return cmd, ErrExtHelp
		case arg == "--":
			// end of options; remaining args are positional
			for ; i < len(argv); i++ {
				free = append(free, argv[i])
			}
			i = len(argv)
			continue
		case strings.HasPrefix(arg, "--"):
			name, val, hasVal := splitFlag(arg[2:])
			name = strings.ToLower(name)
			spec, negated, ok := lookupFlag(name)
			if !ok {
				return nil, fmt.Errorf("unknown option --%s", name)
			}
			if err := apply(cmd, cur, name, spec, negated, val, hasVal, argv, &i); err != nil {
				return nil, err
			}
			continue
		case strings.HasPrefix(arg, "-") && len(arg) == 2:
			name := arg[1:]
			spec, ok := shortFlags[name]
			if !ok {
				return nil, fmt.Errorf("unknown option -%s", name)
			}
			if err := apply(cmd, cur, name, spec, false, "", false, argv, &i); err != nil {
				return nil, err
			}
			continue
		}
		if err := cmd.positional(arg, cur, &free); err != nil {
			return nil, err
		}
	}
	if err := cmd.resolveFree(cur, free); err != nil {
		return nil, err
	}
	return cmd, nil
}

// positional handles object keywords and queues bare positionals.
func (c *Command) positional(arg string, cur *objectCtx, free *[]string) error {
	switch arg {
	case "page":
		cur.newObject(c)
		return nil
	case "cover":
		obj := cur.newObject(c)
		obj.IsCover = true
		obj.IncludeInOutline = false
		obj.HeaderSet, obj.FooterSet = true, true
		obj.Header, obj.Footer = settings.HeaderFooter{}, settings.HeaderFooter{}
		return nil
	case "toc":
		obj := cur.newFreshObject(c)
		obj.IsTableOfContent = true
		obj.UseOutline = false
		return nil
	}
	// Fill an explicit empty page object first; else queue for resolution
	// (last free arg is the output, the rest become implicit pages).
	if cur.obj != nil && cur.obj.Page == "" && !cur.obj.IsTableOfContent {
		cur.obj.Page = arg
		return nil
	}
	*free = append(*free, arg)
	return nil
}

// resolveFree assigns queued positionals: last → output, others → implicit
// page objects. Pending page-scoped settings (from flags before any object
// keyword) are applied to the first free page URL.
func (c *Command) resolveFree(cur *objectCtx, free []string) error {
	if len(free) == 0 {
		return c.validate()
	}
	c.Output = free[len(free)-1]
	for _, u := range free[:len(free)-1] {
		// Prefer filling an already-opened empty page/cover object.
		if cur.obj != nil && cur.obj.Page == "" && !cur.obj.IsTableOfContent {
			cur.obj.Page = u
			continue
		}
		// Next: promote pending pre-object page settings into this page.
		if cur.pending != nil {
			o := *cur.pending
			o.Page = u
			c.Objects = append(c.Objects, o)
			cur.pending = nil
			cur.obj = &c.Objects[len(c.Objects)-1]
			continue
		}
		cur.newObject(c).Page = u
	}
	return c.validate()
}

func (c *Command) validate() error {
	// At least one page-like object with a URL must exist.
	hasInput := false
	for _, o := range c.Objects {
		if !o.IsTableOfContent && o.Page != "" {
			hasInput = true
		}
	}
	if !hasInput {
		return fmt.Errorf("you need to specify at least one input file")
	}
	return nil
}

// apply runs a flag with value extraction (next-arg or =value). Pair flags
// consume two values joined by pairSep.
func apply(c *Command, cur *objectCtx, name string, spec flagSpec, negated bool, inlineVal string, hasInline bool, argv []string, i *int) error {
	if spec.kind == "bool" {
		val := "true"
		if negated {
			val = "false"
		}
		if hasInline {
			switch strings.ToLower(inlineVal) {
			case "true", "1", "yes", "on":
				val = "true"
			case "false", "0", "no", "off":
				val = "false"
			default:
				return fmt.Errorf("invalid boolean value %q", inlineVal)
			}
		}
		return spec.app(c, cur, val)
	}
	// value flag
	if hasInline {
		return spec.app(c, cur, inlineVal)
	}
	if *i >= len(argv) {
		return fmt.Errorf("option requires a value")
	}
	val := argv[*i]
	*i++
	if isPairFlag(name) {
		if *i >= len(argv) {
			return fmt.Errorf("option requires two values (name value)")
		}
		val += pairSep + argv[*i]
		*i++
	}
	return spec.app(c, cur, val)
}

func splitFlag(name string) (string, string, bool) {
	if eq := strings.IndexByte(name, '='); eq >= 0 {
		return name[:eq], name[eq+1:], true
	}
	return name, "", false
}

// lookupFlag resolves a long flag name with its --no- negation.
func lookupFlag(name string) (flagSpec, bool, bool) {
	if spec, ok := flagTable[name]; ok {
		return spec, false, true
	}
	if strings.HasPrefix(name, "no-") {
		if spec, ok := flagTable[name[3:]]; ok && spec.kind == "bool" {
			return spec, true, true
		}
	}
	return flagSpec{}, false, false
}
