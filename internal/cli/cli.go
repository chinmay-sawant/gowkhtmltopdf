package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// ErrHelp / ErrVersion / ErrLicense / ErrExtHelp are returned for doc flags
// so the caller can print and exit 0.
var (
	ErrHelp    = errors.New("help requested")
	ErrVersion = errors.New("version requested")
	ErrLicense = errors.New("license requested")
	ErrExtHelp = errors.New("extended help requested")
	// ErrMissingOutput reports a conversion invocation without -o/--output.
	ErrMissingOutput = errors.New("output is required; use -o or --output")
	// ErrConflictingInputs reports mutually exclusive document source flags.
	ErrConflictingInputs = errors.New("document inputs are mutually exclusive")
	// ErrDuplicateOutput reports more than one explicit output flag.
	ErrDuplicateOutput = errors.New("output may be specified only once")
	// ErrLegacyObjectSyntax reports the removed wkhtmltopdf object grammar.
	ErrLegacyObjectSyntax = errors.New("legacy page/cover/toc object syntax is not supported; use document flags")
	// ErrTerminalConflict reports conversion arguments combined with a
	// terminal-only action such as --dump-default-toc-xsl.
	ErrTerminalConflict = errors.New("cli: terminal action conflicts with conversion arguments")
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
	errDuplicateSource     = errors.New("document source may be specified only once")
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
	// Output is a path or "-" (stdout). It is populated only by -o/--output
	// for parsed commands and is ignored when OutputWriter is set.
	Output string
	// OutputWriter, when non-nil, receives PDF/image bytes directly (library
	// path). Takes precedence over Output so embedders need no temp files.
	OutputWriter io.Writer

	// The fields below are the private CLI document seam. They are resolved to
	// settings.PdfObject values once parsing is complete; the public Document
	// model is intentionally not coupled to this package.
	htmlSource   string
	htmlSet      bool
	urlSource    string
	urlSet       bool
	coverSource  string
	coverSet     bool
	tocRequested bool
	outputSet    bool
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

// object returns the current page template, creating one if needed. The
// template is materialized into Document.Pages during final resolution.
func (ctx *objectCtx) object(_ *Command) *settings.PdfObject {
	if ctx.obj == nil {
		obj := settings.DefaultPdfObject()
		ctx.obj = &obj
	}

	return ctx.obj
}

// Parse parses the 0.2.4 document-oriented command grammar.
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
	case strings.HasPrefix(arg, "-") && arg != "-":
		return fmt.Errorf("%w %s", errUnknownOption, arg)
	default:
		if isLegacyObjectToken(arg) {
			return fmt.Errorf("%w: %s", ErrLegacyObjectSyntax, arg)
		}

		s.free = append(s.free, arg)

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

// isLegacyObjectToken identifies the removed multi-object grammar. A bare
// path named "page", "cover", or "toc" is intentionally reserved so the
// new CLI cannot silently interpret old invocations as different documents.
func isLegacyObjectToken(arg string) bool {
	switch arg {
	case "page", "cover", "toc":
		return true
	default:
		return false
	}
}

// resolveFree turns the document source flags and positional page files into
// ordered private engine objects: cover, TOC, then body pages.
//
//nolint:cyclop,funlen,wsl // source validation and ordered assembly are one parser boundary.
func (c *Command) resolveFree(cur *objectCtx, free []string) error {
	if c.Global.DumpDefaultTOCXSL {
		if len(free) != 0 || len(c.Objects) != 0 {
			return fmt.Errorf("%w: --dump-default-toc-xsl cannot be combined with input/output arguments", ErrTerminalConflict)
		}

		if c.Global.DumpOutline {
			return fmt.Errorf("%w: --dump-default-toc-xsl cannot be combined with --dump-outline", ErrTerminalConflict)
		}

		return nil
	}

	if !c.outputSet || strings.TrimSpace(c.Output) == "" {
		return ErrMissingOutput
	}

	sources := 0
	if c.htmlSet {
		sources++
	}
	if c.urlSet {
		sources++
	}
	if len(free) > 0 {
		sources++
	}
	if sources > 1 {
		return fmt.Errorf("%w: use exactly one of --html, --url, or positional page files", ErrConflictingInputs)
	}
	if sources == 0 {
		return errNeedInputFile
	}

	template := settings.DefaultPdfObject()
	if cur.obj != nil {
		template = settings.ClonePdfObject(*cur.obj)
	}

	c.Objects = nil
	if c.coverSet {
		cover := settings.ClonePdfObject(template)
		cover.Page = c.coverSource
		settings.StampCover(&cover)
		c.Objects = append(c.Objects, cover)
	}
	if c.tocRequested {
		toc := settings.DefaultPdfObject()
		settings.StampTOC(&toc)
		c.Objects = append(c.Objects, toc)
	}

	body := func(source string, inline bool) {
		obj := settings.ClonePdfObject(template)
		if inline {
			obj.Load.InlineHTML = []byte(source)
		} else {
			obj.Page = source
		}
		c.Objects = append(c.Objects, obj)
	}

	switch {
	case c.htmlSet:
		body(c.htmlSource, true)
	case c.urlSet:
		body(c.urlSource, false)
	default:
		for _, page := range free {
			body(page, false)
		}
	}

	return c.validate()
}

func (c *Command) validate() error {
	if err := settings.ValidateRenderableObjects(c.Objects); err != nil {
		return errNeedInputFile
	}

	return nil
}

// applyPage routes a page-ish flag to both global and current-page settings.
func (ctx *objectCtx) applyPage(command *Command, glob func(g *settings.PdfGlobal, val string) error,
	obj func(o *settings.PdfObject, val string) error, val string,
) error {
	if err := glob(&command.Global, val); err != nil {
		return err
	}

	return obj(ctx.object(command), val)
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
		base := name[3:]
		if isDocFlagName(base) {
			return flagSpec{}, false, false //nolint:exhaustruct // terminal flags are not negatable
		}

		if spec, ok := flagTable[base]; ok && spec.kind == flagBool {
			return spec, true, true
		}
	}

	return flagSpec{}, false, false //nolint:exhaustruct // intentional zero/partial fields
}

func isDocFlagName(name string) bool {
	switch name {
	case "help", "version", "license", "extended-help":
		return true
	default:
		return false
	}
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
