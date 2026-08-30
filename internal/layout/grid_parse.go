package layout

import (
	"strconv"
	"strings"
)

// --- Track parsing (minmax / fr / intrinsic) --------------------------------

type trackSizeKind int

const (
	trackFixed trackSizeKind = iota
	trackFr
	trackAuto
	trackMinContent
	trackMaxContent
)

// gridTrackSize is one side of a track (min or max).
type gridTrackSize struct {
	kind trackSizeKind
	val  float64 // pt for fixed (pre-scale raw pt), or fr coefficient
}

// gridTrackDef is minmax(min, max); a bare size is stored as minmax(size, size)
// except fr -> minmax(auto, fr) per CSS Grid.
type gridTrackDef struct {
	min, max gridTrackSize
}

func flexibleTrack(frac float64) gridTrackDef {
	if frac <= 0 {
		frac = 1
	}

	return gridTrackDef{
		min: gridTrackSize{kind: trackAuto}, //nolint:exhaustruct // intentional zero fields
		max: gridTrackSize{kind: trackFr, val: frac},
	}
}

// parseGridTrackFixedMins returns fixed (non-fr) track sizes as minimums for
// auto-height grids. fr / unknown / intrinsic tracks yield 0.
func parseGridTrackFixedMins(raw string, eng *engine) []float64 {
	defs := parseGridTrackDefs(raw)
	if len(defs) == 0 {
		return nil
	}

	out := make([]float64, len(defs))

	for i, d := range defs {
		if d.min.kind == trackFixed {
			out[i] = eng.scalePt(d.min.val)
		}
	}

	return out
}

// parseGridTracks parses grid-template-columns/rows into resolved lengths.
// columnGap is subtracted from contentW before distributing fr tracks so
// (n tracks + n-1 gaps) fit the content box. Supports minmax(), fr, lengths,
// %, auto, min-content, max-content (intrinsics default to 0 without measure).
func parseGridTracks(raw string, contentW, columnGap float64, eng *engine) []float64 {
	defs := parseGridTrackDefs(raw)
	if len(defs) == 0 {
		return nil
	}

	return resolveGridTrackSizes(defs, contentW, columnGap, eng, nil)
}

// parseGridTrackDefs tokenizes and expands repeat()/minmax() into track defs.
func parseGridTrackDefs(raw string) []gridTrackDef {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, cssDisplayNone) || strings.EqualFold(raw, "masonry") {
		return nil
	}

	raw = expandRepeatFunctions(raw)

	toks := tokenizeGridTracks(raw)
	if len(toks) == 0 {
		return nil
	}

	out := make([]gridTrackDef, 0, len(toks))
	for _, t := range toks {
		out = append(out, parseOneTrackDef(t))
	}

	return out
}

// expandRepeatFunctions replaces repeat(N, <track-list>) with N copies.
func expandRepeatFunctions(raw string) string {
	lower := strings.ToLower(raw)

	for {
		idx := strings.Index(lower, "repeat(")
		if idx < 0 {
			return raw
		}

		start := idx + len("repeat(")

		end := findMatchingParen(raw, start-1)
		if end < 0 {
			return raw
		}

		inner := raw[start:end]

		parts := splitTopLevelComma(inner)
		if len(parts) != 2 {
			return raw
		}

		node, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || node <= 0 || node >= 64 {
			return raw
		}

		track := strings.TrimSpace(parts[1])

		var boxNode strings.Builder

		boxNode.WriteString(raw[:idx])

		for i := range node {
			if i > 0 {
				boxNode.WriteByte(' ')
			}

			boxNode.WriteString(track)
		}

		boxNode.WriteString(raw[end+1:])
		raw = boxNode.String()
		lower = strings.ToLower(raw)
	}
}

func findMatchingParen(s string, openIdx int) int {
	depth := 0

	for idx := openIdx; idx < len(s); idx++ {
		switch s[idx] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return idx
			}
		}
	}

	return -1
}

func splitTopLevelComma(cssSheet string) []string {
	return splitParenArgs(cssSheet, ',')
}

// isTrackWhitespace reports CSS whitespace between grid track tokens.
func isTrackWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// tokenizeGridTracks splits on whitespace but keeps function calls intact.
func tokenizeGridTracks(raw string) []string {
	var toks []string

	var boxNode strings.Builder

	depth := 0
	flush := func() {
		if boxNode.Len() == 0 {
			return
		}

		toks = append(toks, boxNode.String())
		boxNode.Reset()
	}

	for i := range len(raw) {
		child := raw[i]

		switch {
		case child == '(':
			depth++

			boxNode.WriteByte(child)
		case child == ')':
			if depth > 0 {
				depth--
			}

			boxNode.WriteByte(child)
		case isTrackWhitespace(child) && depth == 0:
			flush()
		default:
			boxNode.WriteByte(child)
		}
	}

	flush()

	return toks
}

func parseOneTrackDef(tok string) gridTrackDef {
	tok = strings.TrimSpace(tok)
	lower := strings.ToLower(tok)

	if strings.HasPrefix(lower, "minmax(") && strings.HasSuffix(tok, ")") {
		inner := tok[len("minmax(") : len(tok)-1]

		parts := splitTopLevelComma(inner)
		if len(parts) == 2 {
			minS := parseTrackSize(strings.TrimSpace(parts[0]))
			maxS := parseTrackSize(strings.TrimSpace(parts[1]))
			// Spec: if max < min for fixed/fixed, use min for both (lite).
			if minS.kind == trackFixed && maxS.kind == trackFixed && maxS.val < minS.val {
				maxS = minS
			}

			return gridTrackDef{min: minS, max: maxS}
		}
	}

	size := parseTrackSize(tok)
	if size.kind == trackFr {
		return gridTrackDef{
			min: gridTrackSize{kind: trackAuto}, //nolint:exhaustruct // intentional zero fields
			max: size,
		}
	}

	return gridTrackDef{min: size, max: size}
}

// parseFrSize parses a "Nfr" track size; ok=false when tok is not fr.
func parseFrSize(tok string) (gridTrackSize, bool) {
	lower := strings.ToLower(tok)
	if !strings.HasSuffix(lower, "fr") {
		return gridTrackSize{}, false //nolint:exhaustruct // intentional zero fields
	}

	v, err := strconv.ParseFloat(strings.TrimSuffix(lower, "fr"), 64)
	if err != nil || v <= 0 {
		v = 1
	}

	return gridTrackSize{kind: trackFr, val: v}, true
}

// parseTrackPct parses a percentage track size into the negative fixed
// sentinel used by resolveTrackSide; ok=false when tok is not a valid %.
func parseTrackPct(tok string) (gridTrackSize, bool) {
	if !strings.HasSuffix(tok, "%") {
		return gridTrackSize{}, false //nolint:exhaustruct // intentional zero fields
	}

	pct, err := strconv.ParseFloat(strings.TrimSuffix(tok, "%"), 64)
	if err != nil {
		return gridTrackSize{}, false //nolint:exhaustruct // intentional zero fields
	}

	return gridTrackSize{kind: trackFixed, val: -pct}, true
}

func parseTrackSize(tok string) gridTrackSize {
	tok = strings.TrimSpace(tok)

	lower := strings.ToLower(tok)
	switch lower {
	case overflowAuto:
		return gridTrackSize{kind: trackAuto} //nolint:exhaustruct // intentional zero fields
	case "min-content":
		return gridTrackSize{kind: trackMinContent} //nolint:exhaustruct // intentional zero fields
	case "max-content":
		return gridTrackSize{kind: trackMaxContent} //nolint:exhaustruct // intentional zero fields
	}

	if size, ok := parseFrSize(lower); ok {
		return size
	}

	if val, ok := lengthBox(tok, 12, 0, overflowAuto); ok && val >= 0 {
		// Percentages are re-resolved in resolveGridTrackSizes against the
		// definite container; store raw % as a sentinel via kind+val.
		if pctSize, ok := parseTrackPct(tok); ok {
			return pctSize
		}

		return gridTrackSize{kind: trackFixed, val: val}
	}

	return gridTrackSize{kind: trackAuto} //nolint:exhaustruct // intentional zero fields
}

// --- Areas + auto-flow placement (kept separate from track parsing) ---------

// gridAreaRect is a 0-based rectangle covering a named template area.
type gridAreaRect struct {
	row, col, rowSpan, colSpan int
}

// gridTemplateAreasMap holds the parsed grid-template-areas name -> rect map.
type gridTemplateAreasMap struct {
	names      map[string]gridAreaRect
	rows, cols int
}

// parseGridTemplateAreas parses quoted area rows into a name map.
// Tokens "none", ".", and empty cells are holes (no name).
func parseGridTemplateAreas(raw string) gridTemplateAreasMap {
	out := gridTemplateAreasMap{names: map[string]gridAreaRect{}} //nolint:exhaustruct // intentional zero fields
	raw = strings.TrimSpace(raw)

	if raw == "" || strings.EqualFold(raw, cssDisplayNone) {
		return out
	}

	// Collect quoted strings: "a b" "c d" or 'a b'
	rows := collectGridAreaRows(raw)
	if len(rows) == 0 {
		return out
	}

	out.rows = len(rows)
	for _, r := range rows {
		if len(r) > out.cols {
			out.cols = len(r)
		}
	}

	// Pad short rows with "." so indexing is safe.
	for i := range rows {
		for len(rows[i]) < out.cols {
			rows[i] = append(rows[i], ".")
		}
	}

	out.names = accumulateGridAreaBounds(rows)

	return out
}

// collectGridAreaRows extracts the quoted template-area rows from raw text.
func collectGridAreaRows(raw string) [][]string {
	var rows [][]string

	for idx := 0; idx < len(raw); {
		for idx < len(raw) && isTrackWhitespace(raw[idx]) {
			idx++
		}

		if idx >= len(raw) {
			break
		}

		quote := raw[idx]
		if quote != '"' && quote != '\'' {
			// Unquoted token - skip (invalid lite)
			idx = skipGridUnquotedToken(raw, idx)

			continue
		}

		cell, nextIdx := scanGridAreaCell(raw, idx, quote)
		idx = nextIdx

		toks := strings.Fields(cell)
		if len(toks) == 0 {
			continue
		}

		rows = append(rows, toks)
	}

	return rows
}

// skipGridUnquotedToken advances past an unquoted template-area token.
func skipGridUnquotedToken(raw string, idx int) int {
	for idx < len(raw) && raw[idx] != ' ' && raw[idx] != '\t' && raw[idx] != '"' && raw[idx] != '\'' {
		idx++
	}

	return idx
}

// scanGridAreaCell advances to the closing quote and returns the cell text
// plus the index just past the closing quote.
func scanGridAreaCell(raw string, idx int, quote byte) (string, int) {
	idx++
	start := idx

	for idx < len(raw) && raw[idx] != quote {
		idx++
	}

	cell := raw[start:idx]

	if idx < len(raw) {
		idx++ // closing quote
	}

	return cell, idx
}

// gridAreaBounds tracks the running bounds of one named template area.
type gridAreaBounds struct {
	r0, c0, r1, c1 int
	seen           bool
}

// accumulateGridAreaBounds folds each area name onto its bounding rectangle.
func accumulateGridAreaBounds(rows [][]string) map[string]gridAreaRect {
	acc := map[string]*gridAreaBounds{}

	for runic, row := range rows {
		for child, name := range row {
			if name == "." || strings.EqualFold(name, cssDisplayNone) {
				continue
			}

			cur := acc[name]
			if cur == nil {
				acc[name] = &gridAreaBounds{r0: runic, c0: child, r1: runic, c1: child, seen: true}

				continue
			}

			extendGridAreaBounds(cur, runic, child)
		}
	}

	out := make(map[string]gridAreaRect, len(acc))
	for name, b := range acc {
		out[name] = gridAreaRect{
			row:     b.r0,
			col:     b.c0,
			rowSpan: b.r1 - b.r0 + 1,
			colSpan: b.c1 - b.c0 + 1,
		}
	}

	return out
}

// extendGridAreaBounds grows a bounds rect to include (row, col).
func extendGridAreaBounds(cur *gridAreaBounds, runic, child int) {
	if runic < cur.r0 {
		cur.r0 = runic
	}

	if runic > cur.r1 {
		cur.r1 = runic
	}

	if child < cur.c0 {
		cur.c0 = child
	}

	if child > cur.c1 {
		cur.c1 = child
	}
}

// resolveNamedGridArea looks up a custom-ident in the areas map.
func resolveNamedGridArea(areas gridTemplateAreasMap, name string) (gridAreaRect, bool) {
	if areas.names == nil {
		return gridAreaRect{}, false //nolint:exhaustruct // intentional zero fields
	}

	rect, ok := areas.names[name]

	return rect, ok
}
