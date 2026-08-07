package css

import (
	"strings"
)

// defaultMediaFontPt is the em base for media-query lengths when no element
// font-size applies (16px CSS → 12pt).
const defaultMediaFontPt = 12

// MediaMatches reports whether a CSS media-query list applies for the given
// conversion media type ("print" or "screen") and viewport size in points.
//
// Policy (print/PDF pipeline):
//   - empty query and "all" match
//   - media types print/screen must equal mediaType (unless type is all/omitted)
//   - size features (min-/max-/width/height/inline-size/block-size) evaluate
//     against the viewport
//   - orientation: portrait|landscape is supported
//   - unknown features evaluate to false (rules do not apply)
//   - comma-separated queries are ORed; "not"/"only" follow CSS grammar subset
//
// When mediaType is empty, every query matches (legacy Options.Media == "").
func MediaMatches(query, mediaType string, widthPt, heightPt float64) bool {
	if mediaType == "" {
		return true
	}

	query = strings.TrimSpace(query)
	if len(query) >= 6 && strings.EqualFold(query[:6], "@media") {
		query = strings.TrimSpace(query[6:])
	}

	if query == "" || strings.EqualFold(query, "all") {
		return true
	}

	for _, part := range splitMediaList(query) {
		if matchOneMediaQuery(part, mediaType, widthPt, heightPt) {
			return true
		}
	}

	return false
}

func splitMediaList(str string) []string {
	var parts []string

	depth := 0
	start := 0

	for idx := range len(str) {
		switch str[idx] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(str[start:idx]))
				start = idx + 1
			}
		}
	}

	parts = append(parts, strings.TrimSpace(str[start:]))

	return parts
}

func matchOneMediaQuery(q, mediaType string, widthPt, heightPt float64) bool {
	q = strings.TrimSpace(q)
	if q == "" {
		return true
	}

	low := strings.ToLower(q)
	negated := false

	if strings.HasPrefix(low, "not ") {
		negated = true
		q = strings.TrimSpace(q[4:])
	} else if strings.HasPrefix(low, "only ") {
		q = strings.TrimSpace(q[5:])
	}

	features, rest := extractMediaFeatures(q)
	rest = strings.ToLower(strings.TrimSpace(rest))
	// Drop leftover "and" tokens; remainder should be a media type or empty.
	fields := strings.Fields(rest)

	var typeWord string

	for _, f := range fields {
		if f == "and" {
			continue
		}

		typeWord = f

		break
	}

	typeOK := true
	if typeWord != "" && typeWord != "all" {
		typeOK = typeWord == strings.ToLower(mediaType)
	}

	featOK := true

	for _, feat := range features {
		if !mediaFeatureMatches(feat, widthPt, heightPt) {
			featOK = false

			break
		}
	}

	ok := typeOK && featOK
	if negated {
		return !ok
	}

	return ok
}

// extractMediaFeatures pulls parenthesized feature queries and returns the
// remainder (media type / and keywords).
func extractMediaFeatures(q string) (features []string, rest string) {
	var buf strings.Builder

	for idx := 0; idx < len(q); {
		if q[idx] == '(' {
			inner, end, ok := takeParenArg(q, idx)
			if !ok {
				buf.WriteByte(q[idx])

				idx++

				continue
			}

			features = append(features, strings.TrimSpace(inner))
			idx = end

			continue
		}

		buf.WriteByte(q[idx])

		idx++
	}

	return features, buf.String()
}

func mediaFeatureMatches(inner string, widthPt, heightPt float64) bool {
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return false
	}

	low := strings.ToLower(inner)
	if strings.HasPrefix(low, "orientation") {
		return matchOrientation(low, widthPt, heightPt)
	}

	feat, ok := parseSizeFeature(inner)
	if !ok {
		return false
	}

	switch feat.Name {
	case "width", "inline-size":
		return feat.matchesAxis(widthPt, defaultMediaFontPt)
	case "height", "block-size":
		return feat.matchesAxis(heightPt, defaultMediaFontPt)
	default:
		// Unknown feature → false (documented policy).
		return false
	}
}

func matchOrientation(low string, widthPt, heightPt float64) bool {
	// orientation: portrait | landscape
	const kvParts = 2

	parts := strings.SplitN(low, ":", kvParts)
	if len(parts) != kvParts {
		return false
	}

	val := strings.TrimSpace(parts[1])
	portrait := heightPt >= widthPt

	switch val {
	case "portrait":
		return portrait
	case "landscape":
		return !portrait
	default:
		return false
	}
}

// matchesAxis compares a size feature against one axis length in points.
func (f SizeFeature) matchesAxis(sizePt, fontSizePt float64) bool {
	thresh, ok := LengthToPt(f.Value, f.Unit, fontSizePt)
	if !ok {
		return false
	}

	switch f.Op {
	case "<":
		return sizePt < thresh
	case ">":
		return sizePt > thresh
	case "<=":
		return sizePt <= thresh
	case ">=":
		return sizePt >= thresh
	case "=":
		const eps = 0.01

		return sizePt > thresh-eps && sizePt < thresh+eps
	default:
		return false
	}
}
