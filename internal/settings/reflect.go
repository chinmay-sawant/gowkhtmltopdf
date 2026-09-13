// Dotted-key settings dispatch: hand-dispatched map lookup tables and
// closures mapping wkhtmltopdf-style dotted setting names to typed struct
// fields. This file does NOT use runtime reflection (the reflect package);
// the filename is historical.
package settings

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

var errImageBackgroundNeedsGlobal = errors.New("image: background requires global settings")

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

// unknownSettingError reports a settings key with no descriptor and no
// known-ignored entry.
type unknownSettingError struct {
	kind, name string
}

func (e unknownSettingError) Error() string {
	return fmt.Sprintf("unknown %s setting %q", e.kind, e.name)
}

func errUnknownSetting(kind, name string) error {
	return unknownSettingError{kind: kind, name: name}
}

// parseError reports a raw value that a setter could not parse.
type parseError struct {
	kind string
	raw  string
}

func (e parseError) Error() string {
	return fmt.Sprintf("invalid %s %q", e.kind, e.raw)
}

func errParse(kind, raw string) error {
	return parseError{kind: kind, raw: raw}
}

// rangeError reports a numeric setting value outside its allowed range.
type rangeError struct {
	value, allowed string
}

func (e rangeError) Error() string {
	return fmt.Sprintf("value %q out of range (allowed: %s)", e.value, e.allowed)
}

func errRange(value, allowed string) error {
	return rangeError{value: value, allowed: allowed}
}

// unitError reports a measurement unit that cannot be converted to millimetres.
type unitError struct {
	ctx  string
	unit string
}

func (e unitError) Error() string {
	return fmt.Sprintf("%s: unit %q not convertible", e.ctx, e.unit)
}

func errUnitNotConvertible(ctx, unit string) error {
	return unitError{ctx: ctx, unit: unit}
}

// Compile-time checks: every settings error value satisfies the error interface.
var (
	_ error = invalidError{}        //nolint:exhaustruct // zero-value probe for interface conformance
	_ error = unknownSettingError{} //nolint:exhaustruct // zero-value probe
	_ error = parseError{}          //nolint:exhaustruct // zero-value probe
	_ error = rangeError{}          //nolint:exhaustruct // zero-value probe
	_ error = unitError{}           //nolint:exhaustruct // zero-value probe
)

// Raw string values accepted by the setter helpers.
const (
	sFalse     = "false"
	sIgnore    = "ignore"
	sScreen    = "screen"
	sPrint     = "print"
	sGrayscale = "grayscale"
	sColor     = "color"
)

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
func sub[T any, S any](pick func(*T) *S, desc field[S]) field[T] {
	return field[T]{
		apply: func(target *T, raw string) error { return desc.apply(pick(target), raw) },
		get:   func(target *T) (string, bool) { return desc.get(pick(target)) },
	}
}

// setForKey applies a dotted key through its descriptor table. Known inert
// keys are stored in the Ignored map (Policy A); truly unknown keys error.
func setForKey[T any](target *T, tables keyTable[T], known map[string]struct{},
	ignored *map[string]string, kind, name, value string,
) error {
	key := normalizeDots(name)
	if f, ok := tables[key]; ok {
		return f.apply(target, value)
	}

	if _, ok := known[key]; ok {
		storeIgnored(ignored, key, value)

		return nil
	}

	return errUnknownSetting(kind, name)
}

// getForKey reads a dotted key through its descriptor table. Accepted ignored
// keys return the last Set value; unknown keys are not found.
func getForKey[T any](target *T, tables keyTable[T], ignored *map[string]string, name string) (string, bool) {
	key := normalizeDots(name)
	if f, ok := tables[key]; ok {
		return f.get(target)
	}

	if *ignored != nil {
		if v, ok := (*ignored)[key]; ok {
			return v, true
		}
	}

	return "", false
}

// ignoredGlobalKeySet is the immutable-by-convention table of wkhtml global
// keys with no engine consumer. Keeping it at package scope avoids rebuilding
// the same map for every Set call.
//
//nolint:gochecknoglobals // static lookup map
var ignoredGlobalKeySet = map[string]struct{}{
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
	// ponytail: --default-encoding accepted then ignored; engine is UTF-8/ASCII
	// only via html.ParseDocument + load charset seam. Upgrade when multi-
	// charset decode ships.
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

// ignoredObjectKeySet is the immutable-by-convention table of wkhtml
// object/load/web keys with no engine consumer.
//
//nolint:gochecknoglobals // static lookup map
var ignoredObjectKeySet = map[string]struct{}{
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

func setBool(target *bool) setter {
	return func(raw string) error {
		switch strings.ToLower(raw) {
		case "", "true", "1", "yes", "on":
			*target = true

			return nil
		case sFalse, "0", "no", "off":
			*target = false

			return nil
		}

		return errParse("boolean", raw)
	}
}

func setFloat(target *float64) setter {
	return func(raw string) error {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return errParse("number", raw)
		}

		*target = v

		return nil
	}
}

func setInt(target *int) setter {
	return func(raw string) error {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return errParse("integer", raw)
		}

		*target = v

		return nil
	}
}

// Numeric range bounds enforced by the ranged setters. Out-of-range values
// fail at Set instead of downstream.
const (
	minCopies  = 1
	maxCopies  = 1000 // mirrors the convert engine ceiling (render.MaxCopies)
	minQuality = 0
	maxQuality = 100
	minTimeout = 0
	minZoom    = 0
)

// setIntRange parses raw as an int and rejects values outside [low, high].
func setIntRange(target *int, low, high int) setter {
	return func(raw string) error {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return errParse("integer", raw)
		}

		if value < low || value > high {
			return errRange(raw, fmt.Sprintf("%d to %d", low, high))
		}

		*target = value

		return nil
	}
}

// setFloatMin parses raw as a float and rejects non-finite values or values
// below minimum. NaN and infinities fail here so Set matches the root API
// predicates instead of storing a value layout later rejects.
func setFloatMin(target *float64, minimum float64) setter {
	return func(raw string) error {
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil || !finite(value) {
			return errParse("number", raw)
		}

		if value < minimum {
			return errRange(raw, fmt.Sprintf("at least %g", minimum))
		}

		*target = value

		return nil
	}
}

func setString(target *string) setter {
	return func(raw string) error {
		*target = raw

		return nil
	}
}

func setStringDefault(target *string) setter {
	return func(raw string) error {
		*target = strings.TrimSpace(raw)

		return nil
	}
}

func setOrientation(orient *Orientation) setter {
	return func(raw string) error {
		v, err := ParseOrientation(raw)
		if err != nil {
			return err
		}

		*orient = v

		return nil
	}
}

func setLoadErrorHandling(handling *LoadErrorHandling) setter {
	return func(raw string) error {
		v, err := ParseLoadErrorHandling(raw)
		if err != nil {
			return err
		}

		*handling = v

		return nil
	}
}

func setMediaType(media *MediaType) setter {
	return func(raw string) error {
		switch normalize(raw) {
		case "", sIgnore:
			*media = MediaUnset

			return nil
		case sScreen:
			*media = MediaScreen

			return nil
		case sPrint:
			*media = MediaPrint

			return nil
		}

		return errParse("media type", raw)
	}
}

// setGrayscaleFromColorMode maps colormode strings onto the Grayscale bool.
func setGrayscaleFromColorMode(grayscale *bool) setter {
	return func(raw string) error {
		switch normalize(raw) {
		case "", sColor:
			*grayscale = false

			return nil
		case sGrayscale:
			*grayscale = true

			return nil
		}

		return errInvalid("color-mode", raw, "color|grayscale")
	}
}

// marginEdgePtr returns the field of margin named by edge. The bool is false
// for an unrecognized edge, so a typo cannot silently target Right.
func marginEdgePtr(margin *Margin, edge string) (*float64, bool) {
	switch edge {
	case "top":
		return &margin.Top, true
	case "bottom":
		return &margin.Bottom, true
	case "left":
		return &margin.Left, true
	case "right":
		return &margin.Right, true
	default:
		return nil, false
	}
}

// setUnitMm parses raw as a real number in millimetres and stores it on target.
// ctx names the settings key for conversion errors.
func setUnitMm(target *float64, ctx string) setter {
	return func(raw string) error {
		unit, err := ParseUnitReal(raw, "mm")
		if err != nil {
			return err
		}

		mm, ok := unit.Mm()
		if !ok {
			return errUnitNotConvertible(ctx, unit.Unit)
		}

		*target = mm

		return nil
	}
}

var errUnknownMarginEdge = errors.New("unknown margin edge")

// errNegativeSideMargin reports a negative left or right margin. Top and
// bottom accept the engine's negative auto sentinel; side margins do not.
var errNegativeSideMargin = errors.New("settings: side margins must be non-negative")

// marginSetter writes one edge of a Margin, storing millimetres. An unknown
// edge is rejected rather than silently targeting the right margin, and the
// result must satisfy ValidMargins so CLI input obeys the same contract the
// root API validates.
func marginSetter(margin *Margin, edge string) setter {
	target, ok := marginEdgePtr(margin, edge)
	if !ok {
		return func(string) error {
			return fmt.Errorf("%w %q (allowed: top|bottom|left|right)", errUnknownMarginEdge, edge)
		}
	}

	return func(raw string) error {
		previous := *target

		if err := setUnitMm(target, "margin "+edge)(raw); err != nil {
			return err
		}

		if !ValidMargins(*margin) {
			value := *target
			*target = previous

			return fmt.Errorf("%w: margin %s = %g", errNegativeSideMargin, edge, value)
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

// globalKeys / objectKeys / imageKeys are the one field-descriptor tables. They
// are read as package-level identifiers by getters.go and settings_test.go, so
// they cannot be function-local.
//
//nolint:gochecknoglobals // required: shared with getters.go and settings_test.go.
var globalKeys, objectKeys, imageKeys = buildKeyTables()

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

// registerGlobalKeys wires the core page keys of the global table.
func registerGlobalKeys(keys keyTable[PdfGlobal]) {
	regGlobal := func(name string, set func(*PdfGlobal, string) error, get func(*PdfGlobal) (string, bool)) {
		keys[name] = field[PdfGlobal]{apply: set, get: get}
	}
	regGlobal("orientation",
		func(dst *PdfGlobal, raw string) error { return setOrientation(&dst.Orientation)(raw) },
		func(dst *PdfGlobal) (string, bool) { return dst.Orientation.String(), true },
	)
	// colormode and grayscale both share Grayscale.
	regGlobal("colormode",
		func(dst *PdfGlobal, raw string) error { return setGrayscaleFromColorMode(&dst.Grayscale)(raw) },
		func(dst *PdfGlobal) (string, bool) {
			if dst.Grayscale {
				return sGrayscale, true
			}

			return sColor, true
		},
	)
	regGlobal("grayscale",
		func(dst *PdfGlobal, raw string) error { return setBool(&dst.Grayscale)(raw) },
		func(dst *PdfGlobal) (string, bool) { return fmtBool(dst.Grayscale), true },
	)
	regGlobal("pageoffset",
		func(dst *PdfGlobal, raw string) error { return setInt(&dst.PageOffset)(raw) },
		func(dst *PdfGlobal) (string, bool) { return fmtInt(dst.PageOffset), true },
	)
	regGlobal("copies",
		func(dst *PdfGlobal, raw string) error { return setIntRange(&dst.Copies, minCopies, maxCopies)(raw) },
		func(dst *PdfGlobal) (string, bool) { return fmtInt(dst.Copies), true },
	)
	regGlobal("collate",
		func(dst *PdfGlobal, raw string) error { return setBool(&dst.Collate)(raw) },
		func(dst *PdfGlobal) (string, bool) { return fmtBool(dst.Collate), true },
	)
	regGlobal("outline",
		func(dst *PdfGlobal, raw string) error { return setBool(&dst.Outline)(raw) },
		func(dst *PdfGlobal) (string, bool) { return fmtBool(dst.Outline), true },
	)
	regGlobal("outlinedepth",
		func(dst *PdfGlobal, raw string) error { return setInt(&dst.OutlineDepth)(raw) },
		func(dst *PdfGlobal) (string, bool) { return fmtInt(dst.OutlineDepth), true },
	)
}

// registerGlobalDocumentKeys wires the document-level keys of the global table.
func registerGlobalDocumentKeys(keys keyTable[PdfGlobal]) {
	regGlobal := func(name string, set func(*PdfGlobal, string) error, get func(*PdfGlobal) (string, bool)) {
		keys[name] = field[PdfGlobal]{apply: set, get: get}
	}
	regGlobal("dumpoutline",
		func(dst *PdfGlobal, raw string) error { return setBool(&dst.DumpOutline)(raw) },
		func(dst *PdfGlobal) (string, bool) { return fmtBool(dst.DumpOutline), true },
	)
	regGlobal("dumpoutlinewithdefaulttocxsl",
		func(dst *PdfGlobal, raw string) error { return setBool(&dst.DumpDefaultTOCXSL)(raw) },
		func(dst *PdfGlobal) (string, bool) { return fmtBool(dst.DumpDefaultTOCXSL), true },
	)
	regGlobal("usecompression",
		func(dst *PdfGlobal, raw string) error { return setBool(&dst.UseCompression)(raw) },
		func(dst *PdfGlobal) (string, bool) { return fmtBool(dst.UseCompression), true },
	)
	regGlobal("title",
		func(dst *PdfGlobal, raw string) error { return setString(&dst.Title)(raw) },
		func(dst *PdfGlobal) (string, bool) { return dst.Title, true },
	)
	registerGlobalVersionProfileKeys(keys)
	regGlobal("smartshrinking",
		func(dst *PdfGlobal, raw string) error { return setBool(&dst.SmartShrinking)(raw) },
		func(dst *PdfGlobal) (string, bool) { return fmtBool(dst.SmartShrinking), true },
	)
	// Sole paint switch for PDF + image (no Web.Background mirror).
	paintBG := func(dst *PdfGlobal, raw string) error { return setBool(&dst.Background)(raw) }
	paintBGGet := func(dst *PdfGlobal) (string, bool) { return fmtBool(dst.Background), true }
	regGlobal("background", paintBG, paintBGGet)
	regGlobal("web.background", paintBG, paintBGGet)
}

// registerGlobalVersionProfileKeys wires pdfversion and pdfprofile keys.
func registerGlobalVersionProfileKeys(keys keyTable[PdfGlobal]) {
	regGlobal := func(name string, set func(*PdfGlobal, string) error, get func(*PdfGlobal) (string, bool)) {
		keys[name] = field[PdfGlobal]{apply: set, get: get}
	}
	regGlobal("pdfversion",
		func(dst *PdfGlobal, raw string) error {
			v, err := ParsePDFVersion(raw)
			if err != nil {
				return err
			}

			dst.PdfVersion = v

			return nil
		},
		func(dst *PdfGlobal) (string, bool) {
			if dst.PdfVersion == "" {
				return sPDFVersion14, true
			}

			return dst.PdfVersion, true
		},
	)
	regGlobal("pdfprofile",
		func(dst *PdfGlobal, raw string) error {
			p, err := ParsePDFProfile(raw)
			if err != nil {
				return err
			}

			dst.PdfProfile = p

			return nil
		},
		func(dst *PdfGlobal) (string, bool) {
			return dst.PdfProfile, true
		},
	)
}

// registerGlobalLoadKeys wires the load-level keys of the global table.
func registerGlobalLoadKeys(keys keyTable[PdfGlobal]) {
	regGlobal := func(name string, set func(*PdfGlobal, string) error, get func(*PdfGlobal) (string, bool)) {
		keys[name] = field[PdfGlobal]{apply: set, get: get}
	}
	regGlobal("enablelocalfileaccess",
		func(dst *PdfGlobal, raw string) error { return setBool(&dst.Load.EnableLocalFileAccess)(raw) },
		func(dst *PdfGlobal) (string, bool) { return fmtBool(dst.Load.EnableLocalFileAccess), true },
	)
	regGlobal("excludefromoutline",
		func(dst *PdfGlobal, raw string) error { return appendString(&dst.ExcludeFromOutline)(raw) },
		func(dst *PdfGlobal) (string, bool) { return fmtStrings(dst.ExcludeFromOutline), true },
	)
	regGlobal("quiet",
		func(dst *PdfGlobal, raw string) error { return setBool(&dst.Quiet)(raw) },
		func(dst *PdfGlobal) (string, bool) { return fmtBool(dst.Quiet), true },
	)
	regGlobal("proxy",
		func(dst *PdfGlobal, raw string) error { return setString(&dst.Load.Proxy)(raw) },
		func(dst *PdfGlobal) (string, bool) { return dst.Load.Proxy, true },
	)
	regGlobal("usesystemfonts",
		func(dst *PdfGlobal, raw string) error { return setBool(&dst.UseSystemFonts)(raw) },
		func(dst *PdfGlobal) (string, bool) { return fmtBool(dst.UseSystemFonts), true },
	)
	regGlobal("resolverelativelinks",
		func(dst *PdfGlobal, raw string) error { return setBool(&dst.ResolveRelativeLinks)(raw) },
		func(dst *PdfGlobal) (string, bool) { return fmtBool(dst.ResolveRelativeLinks), true },
	)
	regGlobal("fontpath",
		func(dst *PdfGlobal, raw string) error { return appendString(&dst.FontPaths)(raw) },
		func(dst *PdfGlobal) (string, bool) { return fmtStrings(dst.FontPaths), true },
	)
	regGlobal("allow",
		func(dst *PdfGlobal, raw string) error { return appendString(&dst.Load.Allow)(raw) },
		func(dst *PdfGlobal) (string, bool) { return fmtStrings(dst.Load.Allow), true },
	)
}

// registerGlobalGeometryKeys wires the margin and custom-size keys.
func registerGlobalGeometryKeys(keys keyTable[PdfGlobal]) {
	regGlobal := func(name string, set func(*PdfGlobal, string) error, get func(*PdfGlobal) (string, bool)) {
		keys[name] = field[PdfGlobal]{apply: set, get: get}
	}
	for _, edge := range []string{"top", "bottom", "left", "right"} {
		regGlobal("margin."+edge,
			func(dst *PdfGlobal, raw string) error { return marginSetter(&dst.Margin, edge)(raw) },
			func(dst *PdfGlobal) (string, bool) {
				ptr, ok := marginEdgePtr(&dst.Margin, edge)
				if !ok {
					return "", false
				}

				return fmtFloat(*ptr), true
			},
		)
	}

	regGlobal("size.pagesize",
		func(dst *PdfGlobal, raw string) error {
			val := strings.TrimSpace(raw)
			if _, _, err := ParsePageSize(val); err != nil {
				return err
			}

			// Store the canonical (table-key) case so Set/Get round-trips
			// are stable regardless of the caller's spelling.
			dst.PageSize = strings.ToLower(val)

			return nil
		},
		func(dst *PdfGlobal) (string, bool) {
			return dst.PageSize, true
		},
	)
	regGlobal("size.width",
		func(dst *PdfGlobal, raw string) error { return setUnitMm(&dst.Size.Width, "size.width")(raw) },
		func(dst *PdfGlobal) (string, bool) { return fmtFloat(dst.Size.Width), true },
	)
	regGlobal("size.height",
		func(dst *PdfGlobal, raw string) error { return setUnitMm(&dst.Size.Height, "size.height")(raw) },
		func(dst *PdfGlobal) (string, bool) { return fmtFloat(dst.Size.Height), true },
	)
}

// registerHeaderFooterKeys wires header.*/footer.* keys. Object keys
// additionally flag the HeaderSet/FooterSet override bit.
func registerHeaderFooterKeys(globals keyTable[PdfGlobal], objects keyTable[PdfObject]) {
	for key, entry := range subTable([]subEntry[HeaderFooter]{
		{"fontsize",
			func(h *HeaderFooter, raw string) error { return setFloat(&h.FontSize)(raw) },
			func(h *HeaderFooter) (string, bool) { return fmtFloat(h.FontSize), true },
		},
		{"fontname",
			func(h *HeaderFooter, raw string) error { return setString(&h.FontName)(raw) },
			func(h *HeaderFooter) (string, bool) { return h.FontName, true },
		},
		{"left",
			func(h *HeaderFooter, raw string) error { return setString(&h.Left)(raw) },
			func(h *HeaderFooter) (string, bool) { return h.Left, true },
		},
		{"right",
			func(h *HeaderFooter, raw string) error { return setString(&h.Right)(raw) },
			func(h *HeaderFooter) (string, bool) { return h.Right, true },
		},
		{"center",
			func(h *HeaderFooter, raw string) error { return setString(&h.Center)(raw) },
			func(h *HeaderFooter) (string, bool) { return h.Center, true },
		},
		{"line",
			func(h *HeaderFooter, raw string) error { return setBool(&h.Line)(raw) },
			func(h *HeaderFooter) (string, bool) { return fmtBool(h.Line), true },
		},
		{"spacing",
			func(h *HeaderFooter, raw string) error { return setFloat(&h.Spacing)(raw) },
			func(h *HeaderFooter) (string, bool) { return fmtFloat(h.Spacing), true },
		},
		{"htmlurl",
			func(h *HeaderFooter, raw string) error { return setString(&h.HTMLURL)(raw) },
			func(h *HeaderFooter) (string, bool) { return h.HTMLURL, true },
		},
	}) {
		globals["header."+key] = sub(func(dst *PdfGlobal) *HeaderFooter { return &dst.Header }, entry)
		globals["footer."+key] = sub(func(dst *PdfGlobal) *HeaderFooter { return &dst.Footer }, entry)
		objects["header."+key] = field[PdfObject]{
			apply: func(regObject *PdfObject, raw string) error {
				regObject.HeaderSet = true

				return entry.apply(&regObject.Header, raw)
			},
			get: func(regObject *PdfObject) (string, bool) { return entry.get(&regObject.Header) },
		}
		objects["footer."+key] = field[PdfObject]{
			apply: func(regObject *PdfObject, raw string) error {
				regObject.FooterSet = true

				return entry.apply(&regObject.Footer, raw)
			},
			get: func(regObject *PdfObject) (string, bool) { return entry.get(&regObject.Footer) },
		}
	}
}

// registerTOCKeys wires toc.* keys for global and object tables.
func registerTOCKeys(globals keyTable[PdfGlobal], objects keyTable[PdfObject]) {
	for key, entry := range subTable([]subEntry[TableOfContent]{
		{"fontscale",
			func(t *TableOfContent, raw string) error { return setFloat(&t.FontScale)(raw) },
			func(t *TableOfContent) (string, bool) { return fmtFloat(t.FontScale), true },
		},
		{"indentation",
			func(t *TableOfContent, raw string) error { return setStringDefault(&t.Indentation)(raw) },
			func(t *TableOfContent) (string, bool) { return t.Indentation, true },
		},
		{"dottedlines",
			func(t *TableOfContent, raw string) error { return setBool(&t.DottedLines)(raw) },
			func(t *TableOfContent) (string, bool) { return fmtBool(t.DottedLines), true },
		},
		{"captiontext",
			func(t *TableOfContent, raw string) error { return setString(&t.CaptionText)(raw) },
			func(t *TableOfContent) (string, bool) { return t.CaptionText, true },
		},
		{"forwardlinks",
			func(t *TableOfContent, raw string) error { return setBool(&t.ForwardLinks)(raw) },
			func(t *TableOfContent) (string, bool) { return fmtBool(t.ForwardLinks), true },
		},
		{"backlinks",
			func(t *TableOfContent, raw string) error { return setBool(&t.BackLinks)(raw) },
			func(t *TableOfContent) (string, bool) { return fmtBool(t.BackLinks), true },
		},
		{"xslstylesheet",
			func(t *TableOfContent, raw string) error { return setString(&t.XSLStyleSheet)(raw) },
			func(t *TableOfContent) (string, bool) { return t.XSLStyleSheet, true },
		},
	}) {
		globals["toc."+key] = sub(func(dst *PdfGlobal) *TableOfContent { return &dst.TOC }, entry)
		objects["toc."+key] = sub(func(regObject *PdfObject) *TableOfContent { return &regObject.TOC }, entry)
	}
}

// registerWebKeys wires web.* keys for global, object and image tables.
// web.background is not typed here — ApplyImageKey routes it to
// PdfGlobal.Background.
func registerWebKeys(globals keyTable[PdfGlobal], objects keyTable[PdfObject], images keyTable[ImageGlobal]) {
	for key, entry := range subTable([]subEntry[Web]{
		{"images",
			func(w *Web, raw string) error { return setBool(&w.Images)(raw) },
			func(w *Web) (string, bool) { return fmtBool(w.Images), true },
		},
		{"printmediatype",
			func(w *Web, raw string) error { return setBool(&w.PrintMediaType)(raw) },
			func(w *Web) (string, bool) { return fmtBool(w.PrintMediaType), true },
		},
		{"mediatype",
			func(w *Web, raw string) error { return setMediaType(&w.MediaType)(raw) },
			func(w *Web) (string, bool) { return w.MediaType.String(), true },
		},
		{"simplifydom",
			func(w *Web, raw string) error { return setBool(&w.SimplifyDOM)(raw) },
			func(w *Web) (string, bool) { return fmtBool(w.SimplifyDOM), true },
		},
		{"simplifydomprofile",
			func(w *Web, raw string) error { return setString(&w.SimplifyDOMProfile)(raw) },
			func(w *Web) (string, bool) { return w.SimplifyDOMProfile, true },
		},
		{"printlinkunderline",
			func(w *Web, raw string) error { return setBool(&w.PrintLinkUnderline)(raw) },
			func(w *Web) (string, bool) { return fmtBool(w.PrintLinkUnderline), true },
		},
	}) {
		globals["web."+key] = sub(func(dst *PdfGlobal) *Web { return &dst.Web }, entry)
		objects["web."+key] = sub(func(regObject *PdfObject) *Web { return &regObject.Web }, entry)
		images["web."+key] = sub(func(dst *ImageGlobal) *Web { return &dst.Web }, entry)
	}
}

// registerLoadPageKeys wires load.* keys for the object table.
func registerLoadPageKeys(objects keyTable[PdfObject]) {
	for key, entry := range subTable([]subEntry[LoadPage]{
		{"zoomfactor",
			func(l *LoadPage, raw string) error { return setFloatMin(&l.ZoomFactor, minZoom)(raw) },
			func(l *LoadPage) (string, bool) { return fmtFloat(l.ZoomFactor), true },
		},
		{"blocklocalfileaccess",
			func(l *LoadPage, raw string) error { return setBool(&l.BlockLocalFileAccess)(raw) },
			func(l *LoadPage) (string, bool) { return fmtBool(l.BlockLocalFileAccess), true },
		},
		{"loaderrorhandling",
			func(l *LoadPage, raw string) error { return setLoadErrorHandling(&l.LoadErrorHandling)(raw) },
			func(l *LoadPage) (string, bool) { return l.LoadErrorHandling.String(), true },
		},
		{"username",
			func(l *LoadPage, raw string) error { return setString(&l.Username)(raw) },
			func(l *LoadPage) (string, bool) { return l.Username, true },
		},
		{"password",
			func(l *LoadPage, raw string) error { return setString(&l.Password)(raw) },
			func(l *LoadPage) (string, bool) { return l.Password, true },
		},
		{"mediatype",
			func(l *LoadPage, raw string) error { return setMediaType(&l.MediaType)(raw) },
			func(l *LoadPage) (string, bool) { return l.MediaType.String(), true },
		},
		{"printmediatype",
			func(l *LoadPage, raw string) error { return setBool(&l.PrintMediaType)(raw) },
			func(l *LoadPage) (string, bool) { return fmtBool(l.PrintMediaType), true },
		},
		{"timeout",
			func(l *LoadPage, raw string) error { return setIntRange(&l.Timeout, minTimeout, math.MaxInt)(raw) },
			func(l *LoadPage) (string, bool) { return fmtInt(l.Timeout), true },
		},
	}) {
		objects["load."+key] = sub(func(regObject *PdfObject) *LoadPage { return &regObject.Load }, entry)
	}
}

// registerObjectKeys wires the object field-descriptor table.
func registerObjectKeys(keys keyTable[PdfObject]) {
	regObject := func(name string, set func(*PdfObject, string) error, get func(*PdfObject) (string, bool)) {
		keys[name] = field[PdfObject]{apply: set, get: get}
	}
	regObject("page",
		func(dst *PdfObject, raw string) error { return setString(&dst.Page)(raw) },
		func(dst *PdfObject) (string, bool) { return dst.Page, true },
	)
	regObject("externallinks",
		func(dst *PdfObject, raw string) error { return setBool(&dst.ExternalLinks)(raw) },
		func(dst *PdfObject) (string, bool) { return fmtBool(dst.ExternalLinks), true },
	)
	regObject("locallinks",
		func(dst *PdfObject, raw string) error { return setBool(&dst.LocalLinks)(raw) },
		func(dst *PdfObject) (string, bool) { return fmtBool(dst.LocalLinks), true },
	)
	regObject("includeinoutline",
		func(dst *PdfObject, raw string) error { return setBool(&dst.IncludeInOutline)(raw) },
		func(dst *PdfObject) (string, bool) { return fmtBool(dst.IncludeInOutline), true },
	)
	regObject("useoutline",
		func(dst *PdfObject, raw string) error { return setBool(&dst.UseOutline)(raw) },
		func(dst *PdfObject) (string, bool) { return fmtBool(dst.UseOutline), true },
	)
	regObject("istableofcontent",
		func(dst *PdfObject, raw string) error { return setBool(&dst.IsTableOfContent)(raw) },
		func(dst *PdfObject) (string, bool) { return fmtBool(dst.IsTableOfContent), true },
	)
	regObject("iscover",
		func(dst *PdfObject, raw string) error { return setBool(&dst.IsCover)(raw) },
		func(dst *PdfObject) (string, bool) { return fmtBool(dst.IsCover), true },
	)
}

// registerImageKeys wires the image-mode field-descriptor table.
func registerImageKeys(keys keyTable[ImageGlobal]) {
	regImage := func(name string, set func(*ImageGlobal, string) error, get func(*ImageGlobal) (string, bool)) {
		keys[name] = field[ImageGlobal]{apply: set, get: get}
	}
	regImage("width",
		func(dst *ImageGlobal, raw string) error { return setInt(&dst.Width)(raw) },
		func(dst *ImageGlobal) (string, bool) { return fmtInt(dst.Width), true },
	)
	regImage("height",
		func(dst *ImageGlobal, raw string) error { return setInt(&dst.Height)(raw) },
		func(dst *ImageGlobal) (string, bool) { return fmtInt(dst.Height), true },
	)
	regImage("padding",
		func(dst *ImageGlobal, raw string) error { return setInt(&dst.Padding)(raw) },
		func(dst *ImageGlobal) (string, bool) { return fmtInt(dst.Padding), true },
	)
	regImage("quality",
		func(dst *ImageGlobal, raw string) error {
			return setIntRange(&dst.Quality, minQuality, maxQuality)(raw)
		},
		func(dst *ImageGlobal) (string, bool) { return fmtInt(dst.Quality), true },
	)
	regImage("smartwidth",
		func(dst *ImageGlobal, raw string) error { return setBool(&dst.SmartWidth)(raw) },
		func(dst *ImageGlobal) (string, bool) { return fmtBool(dst.SmartWidth), true },
	)
	regImage("transparent",
		func(dst *ImageGlobal, raw string) error { return setBool(&dst.Transparent)(raw) },
		func(dst *ImageGlobal) (string, bool) { return fmtBool(dst.Transparent), true },
	)
	regImage("format",
		func(dst *ImageGlobal, raw string) error { return setString(&dst.Format)(raw) },
		func(dst *ImageGlobal) (string, bool) { return dst.Format, true },
	)
	regImage("crop.left",
		func(dst *ImageGlobal, raw string) error { return setInt(&dst.Crop.Left)(raw) },
		func(dst *ImageGlobal) (string, bool) { return fmtInt(dst.Crop.Left), true },
	)
	regImage("crop.top",
		func(dst *ImageGlobal, raw string) error { return setInt(&dst.Crop.Top)(raw) },
		func(dst *ImageGlobal) (string, bool) { return fmtInt(dst.Crop.Top), true },
	)
	regImage("crop.width",
		func(dst *ImageGlobal, raw string) error { return setInt(&dst.Crop.Width)(raw) },
		func(dst *ImageGlobal) (string, bool) { return fmtInt(dst.Crop.Width), true },
	)
	regImage("crop.height",
		func(dst *ImageGlobal, raw string) error { return setInt(&dst.Crop.Height)(raw) },
		func(dst *ImageGlobal) (string, bool) { return fmtInt(dst.Crop.Height), true },
	)
	regImage("proxy",
		func(dst *ImageGlobal, raw string) error { return setString(&dst.Load.Proxy)(raw) },
		func(dst *ImageGlobal) (string, bool) { return dst.Load.Proxy, true },
	)
}

// buildKeyTables wires the field-descriptor tables used by Set and Get.
func buildKeyTables() (keyTable[PdfGlobal], keyTable[PdfObject], keyTable[ImageGlobal]) {
	globals := keyTable[PdfGlobal]{}
	objects := keyTable[PdfObject]{}
	images := keyTable[ImageGlobal]{}

	// --- global keys ---
	registerGlobalKeys(globals)
	registerGlobalDocumentKeys(globals)
	registerGlobalLoadKeys(globals)
	registerGlobalGeometryKeys(globals)
	registerHeaderFooterKeys(globals, objects)
	registerTOCKeys(globals, objects)
	registerWebKeys(globals, objects, images)
	registerLoadPageKeys(objects)

	// --- object keys ---
	registerObjectKeys(objects)

	// --- image keys ---
	registerImageKeys(images)

	return globals, objects, images
}

// Global.Set applies a dotted settings key ("margin.top", "load.jsdelay",
// "web.background", …) to a PdfGlobal. Known inert keys are stored in
// Ignored and succeed; truly unknown keys return an error.
func (g *PdfGlobal) Set(name, value string) error {
	return setForKey(g, globalKeys, ignoredGlobalKeySet, &g.Ignored, "global", name, value)
}

// Object.Set applies a dotted settings key to a PdfObject. Known inert keys
// go to Ignored; unknown keys return an error.
func (o *PdfObject) Set(name, value string) error {
	return setForKey(o, objectKeys, ignoredObjectKeySet, &o.Ignored, "object", name, value)
}

// ImageGlobal.Set applies an image-mode dotted settings key.
func (g *ImageGlobal) Set(name, value string) error {
	return setForKey(g, imageKeys, ignoredGlobalKeySet, &g.Ignored, "image", name, value)
}

// ApplyImageKey routes an image-mode key: "background"/"web.background" alias
// to the shared PdfGlobal.Background paint switch (image mode has no
// Web.Background); everything else goes to ImageGlobal.Set. ImageConverter.Set
// delegates here.
func ApplyImageKey(global *PdfGlobal, img *ImageGlobal, name, value string) error {
	normalized := normalizeDots(name)

	switch normalized {
	case "background", "web.background":
		if global == nil {
			return errImageBackgroundNeedsGlobal
		}

		return global.Set("background", value)
	default:
		return img.Set(normalized, value)
	}
}
