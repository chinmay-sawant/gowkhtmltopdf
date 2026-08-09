package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"gowkhtmltopdf/internal/settings"
)

// ErrHelp / ErrVersion / ErrLicense / ErrExtHelp are returned for doc flags
// so the caller can print and exit 0.
var (
	ErrHelp    = errors.New("help requested")
	ErrVersion = errors.New("version requested")
	ErrLicense = errors.New("license requested")
	ErrExtHelp = errors.New("extended help requested")
)

// Static parse-stage errors, wrapped with the offending token so callers can
// match with errors.Is.
var (
	errUnknownOption       = errors.New("unknown option")
	errTooManyModes        = errors.New("cli: Parse accepts at most one mode")
	errInvalidMode         = errors.New("cli: invalid mode")
	errOptionNotSupported  = errors.New("not supported in")
	errNeedInputFile       = errors.New("you need to specify at least one input file")
	errOptionRequiresValue = errors.New("option requires a value")
	errOptionRequiresPair  = errors.New("option requires two values (name value)")
	errUnknownFlagKind     = errors.New("internal: unknown flag kind for")
	errInvalidBoolValue    = errors.New("invalid boolean value")
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
	// OutlineWriter receives dump-outline XML when the internal request adapter
	// is used. A nil writer means discard; application adapters may inject
	// stdout explicitly without making the parser depend on process globals.
	OutlineWriter io.Writer

	DumpDefaultTOCXSL bool
	DumpOutline       bool
}

// OpenOutput returns the writer for this command: OutputWriter (library bytes
// sink) takes precedence over Output path; empty/"-" use stdout.
// The closer must be called when done (no-op for writer/stdout).
func (c *Command) OpenOutput() (io.Writer, func() error, error) {
	if c.OutputWriter != nil {
		return c.OutputWriter, func() error { return nil }, nil
	}

	if c.Output != "" && c.Output != "-" {
		f, err := os.Create(c.Output)
		if err != nil {
			return nil, nil, fmt.Errorf("output %q: %w", c.Output, err)
		}

		return f, f.Close, nil
	}

	return os.Stdout, func() error { return nil }, nil
}

// flagKind distinguishes how a flag's value is extracted and delivered.
type flagKind uint8

const (
	flagKindUnknown flagKind = iota
	flagBool                 // app receives one canonical "true"/"false"
	flagValue                // app receives one value token
	flagPair                 // app receives exactly two tokens (name value)
)

// flagSpec describes one accepted flag.
type flagSpec struct {
	kind flagKind
	mod  Mode // which binaries accept it
	app  func(c *Command, ctx *objectCtx, vals []string) error
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

// parseState groups the mutable parser lifecycle so flag helpers do not expose
// argv/index/mode/free-argument plumbing at every call site.
type parseState struct {
	cmd  *Command
	cur  *objectCtx
	argv []string
	idx  int
	mode Mode
	free []string
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
func (ctx *objectCtx) newObject(cmd *Command) *settings.PdfObject {
	if ctx.pending != nil {
		cmd.Objects = append(cmd.Objects, *ctx.pending)
		ctx.pending = nil
	} else {
		cmd.Objects = append(cmd.Objects, settings.DefaultPdfObject())
	}

	ctx.obj = &cmd.Objects[len(cmd.Objects)-1]

	return ctx.obj
}

// newFreshObject appends a new object without consuming pending page-scoped
// settings. Used for toc so pre-object page flags apply to the first real
// page that follows, not to the TOC entry itself.
func (ctx *objectCtx) newFreshObject(cmd *Command) *settings.PdfObject {
	cmd.Objects = append(cmd.Objects, settings.DefaultPdfObject())
	ctx.obj = &cmd.Objects[len(cmd.Objects)-1]

	return ctx.obj
}

// Parse parses wkhtmltopdf-style arguments.
//
// The optional mode restricts the flags accepted by the parser. Omitting it
// preserves the historical library behaviour and accepts the union of PDF
// and image flags; command binaries should pass their concrete mode. A
// variadic parameter keeps existing callers source-compatible while making
// the mode contract explicit for new callers.
func Parse(argv []string, modes ...Mode) (*Command, error) {
	mode, err := parseMode(modes)
	if err != nil {
		return nil, err
	}

	cmd := &Command{ //nolint:exhaustruct // intentional zero/partial fields
		Global: settings.DefaultPdfGlobal(),
		Image:  settings.DefaultImageGlobal(),
	}
	state := &parseState{ //nolint:exhaustruct // intentional zero/partial fields
		cmd:  cmd,
		cur:  &objectCtx{}, //nolint:exhaustruct // empty initial object context
		argv: argv,
		mode: mode,
	}

	for state.idx < len(state.argv) {
		arg := state.argv[state.idx]
		state.idx++

		if err := state.step(arg); err != nil {
			return cmd, err
		}
	}

	if err := cmd.resolveFree(state.cur, state.free); err != nil {
		return nil, err
	}

	return cmd, nil
}

// step processes one argv token. Doc flags return their sentinel error; the
// caller returns cmd alongside so --help/-h etc. exit with a parsed command.
func (s *parseState) step(arg string) error {
	if ok, err := docFlagErr(arg); ok {
		return err
	}

	switch {
	case arg == "--":
		// end of options; remaining args are positional
		s.free = append(s.free, s.argv[s.idx:]...)
		s.idx = len(s.argv)
	case strings.HasPrefix(arg, "--"):
		return s.parseLongFlag(arg)
	case isShortFlag(arg):
		return s.parseShortFlag(arg)
	default:
		s.cmd.positional(arg, s.cur, &s.free)

		return nil
	}

	return nil
}

// docFlagErr maps the doc flags to their sentinel errors.
func docFlagErr(arg string) (bool, error) {
	switch arg {
	case "-h", "--help":
		return true, ErrHelp
	case "-V", "--version":
		return true, ErrVersion
	case "-L", "--license":
		return true, ErrLicense
	case "-E", "--extended-help":
		return true, ErrExtHelp
	}

	return false, nil
}

// isShortFlag reports whether arg is a single-char flag token ("-x").
func isShortFlag(arg string) bool {
	return strings.HasPrefix(arg, "-") && len(arg) == 2
}

// parseLongFlag handles one "--name" token: lookup, mode check and value
// application. negated is true for --no-<bool> forms.
func (s *parseState) parseLongFlag(arg string) error {
	name, val, hasVal := splitFlag(arg[2:])
	name = strings.ToLower(name)
	spec, negated, ok := lookupFlag(name)

	if !ok {
		return fmt.Errorf("%w --%s", errUnknownOption, name)
	}

	if err := checkMode(name, spec, s.mode); err != nil {
		return err
	}

	return s.apply(name, spec, negated, val, hasVal)
}

// parseShortFlag handles one "-x" token: lookup, mode check and value
// application. Short flags never carry inline values or negation.
func (s *parseState) parseShortFlag(arg string) error {
	name := arg[1:]
	spec, ok := shortFlags[name]

	if !ok {
		return fmt.Errorf("%w -%s", errUnknownOption, name)
	}

	if err := checkMode(name, spec, s.mode); err != nil {
		return err
	}

	return s.apply(name, spec, false, "", false)
}

// ParseMode is the explicit form of Parse for callers that know which
// executable mode they are implementing.
func ParseMode(argv []string, mode Mode) (*Command, error) {
	return Parse(argv, mode)
}

func parseMode(modes []Mode) (Mode, error) {
	if len(modes) > 1 {
		return 0, errTooManyModes
	}

	if len(modes) == 0 {
		return ModeBoth, nil
	}

	mode := modes[0]
	if mode == 0 || mode&^ModeBoth != 0 {
		return 0, fmt.Errorf("%w %d", errInvalidMode, mode)
	}

	return mode, nil
}

func checkMode(name string, spec flagSpec, mode Mode) error {
	if spec.mod&mode != 0 {
		return nil
	}

	modeName := "requested mode"
	if mode == ModePDF {
		modeName = "pdf mode"
	} else if mode == ModeImage {
		modeName = "image mode"
	}

	return fmt.Errorf("option --%s is %w %s", name, errOptionNotSupported, modeName)
}

// positional handles object keywords and queues bare positionals.
func (c *Command) positional(arg string, cur *objectCtx, free *[]string) {
	switch arg {
	case "page":
		cur.newObject(c)

		return
	case "cover":
		obj := cur.newObject(c)
		obj.IsCover = true
		obj.IncludeInOutline = false
		obj.HeaderSet, obj.FooterSet = true, true
		//nolint:exhaustruct // intentional zero/partial fields
		obj.Header, obj.Footer = settings.HeaderFooter{}, settings.HeaderFooter{}

		return
	case "toc":
		obj := cur.newFreshObject(c)
		obj.IsTableOfContent = true
		obj.UseOutline = false

		return
	}
	// Fill an explicit empty page object first; else queue for resolution
	// (last free arg is the output, the rest become implicit pages).
	if cur.obj != nil && cur.obj.Page == "" && !cur.obj.IsTableOfContent {
		cur.obj.Page = arg

		return
	}

	*free = append(*free, arg)
}

// resolveFree assigns queued positionals: last → output, others → implicit
// page objects. Pending page-scoped settings (from flags before any object
// keyword) are applied to the first free page URL.
func (c *Command) resolveFree(cur *objectCtx, free []string) error {
	if len(free) == 0 {
		return c.validate()
	}

	c.Output = free[len(free)-1]

	for _, pageURL := range free[:len(free)-1] {
		// Prefer filling an already-opened empty page/cover object.
		if cur.obj != nil && cur.obj.Page == "" && !cur.obj.IsTableOfContent {
			cur.obj.Page = pageURL

			continue
		}
		// Next: promote pending pre-object page settings into this page.
		if cur.pending != nil {
			o := *cur.pending
			o.Page = pageURL
			c.Objects = append(c.Objects, o)
			cur.pending = nil
			cur.obj = &c.Objects[len(c.Objects)-1]

			continue
		}

		cur.newObject(c).Page = pageURL
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
		return errNeedInputFile
	}

	return nil
}

// applyPage routes a page-ish flag: the global half first, then the current
// object, else accumulates it as pending first-page settings (upstream
// address remapping: page settings before any object keyword apply to the
// first page).
func (ctx *objectCtx) applyPage(c *Command, glob func(g *settings.PdfGlobal, val string) error,
	obj func(o *settings.PdfObject, val string) error, val string,
) error {
	if err := glob(&c.Global, val); err != nil {
		return err
	}

	if ctx.obj != nil {
		return obj(ctx.obj, val)
	}

	if ctx.pending == nil {
		o := settings.DefaultPdfObject()
		ctx.pending = &o
	}

	return obj(ctx.pending, val)
}

// apply runs a flag with value extraction (next-arg or =value). Bool flags
// arrive pre-parsed as canonical "true"/"false"; pair flags arrive as two
// separate tokens.
//
//nolint:cyclop // flag application dispatch
func (s *parseState) apply(name string, spec flagSpec, negated bool, inlineVal string, hasInline bool) error {
	switch spec.kind {
	case flagBool:
		b, err := parseBool(inlineVal, negated, hasInline)
		if err != nil {
			return err
		}

		return spec.app(s.cmd, s.cur, []string{strconv.FormatBool(b)})
	case flagValue:
		vals := []string{inlineVal}

		if !hasInline {
			if s.idx >= len(s.argv) {
				return errOptionRequiresValue
			}

			vals[0] = s.argv[s.idx]
			s.idx++
		}

		return spec.app(s.cmd, s.cur, vals)
	case flagPair:
		vals := [2]string{}
		if hasInline {
			vals[0] = inlineVal
		} else {
			if s.idx >= len(s.argv) {
				return errOptionRequiresPair
			}

			vals[0] = s.argv[s.idx]
			s.idx++
		}

		if s.idx >= len(s.argv) {
			return errOptionRequiresPair
		}

		vals[1] = s.argv[s.idx]
		s.idx++

		return spec.app(s.cmd, s.cur, vals[:])
	case flagKindUnknown:
		return fmt.Errorf("%w --%s", errUnknownFlagKind, name)
	}

	return fmt.Errorf("%w --%s", errUnknownFlagKind, name)
}

// parseBool turns the bool-flag contract into a real bool: an inline value
// (--flag=x) wins, otherwise --no-flag negates. Unknown inline values error.
func parseBool(inlineVal string, negated, hasInline bool) (bool, error) {
	if hasInline {
		switch strings.ToLower(inlineVal) {
		case "true", "1", "yes", "on":
			return true, nil
		case "false", "0", "no", "off":
			return false, nil
		}

		return false, fmt.Errorf("%w %q", errInvalidBoolValue, inlineVal)
	}

	return !negated, nil
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
		if spec, ok := flagTable[name[3:]]; ok && spec.kind == flagBool {
			return spec, true, true
		}
	}

	return flagSpec{}, false, false //nolint:exhaustruct // intentional zero/partial fields
}

// ExitCode converts an error into a process exit code: errors carrying an
// HttpErrorCode() int method (e.g. settings.HttpStatusError) report that
// code; everything else exits 1.
func ExitCode(err error) int {
	var hc interface{ HttpErrorCode() int }
	if errors.As(err, &hc) {
		return hc.HttpErrorCode()
	}

	return ExitError
}
