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
	return splitTopLevel(str, ',')
}

func matchOneMediaQuery(query, mediaType string, widthPt, heightPt float64) bool {
	query = strings.TrimSpace(query)
	if query == "" {
		return true
	}

	low := strings.ToLower(query)
	negated := false

	if strings.HasPrefix(low, "not ") {
		negated = true
		query = strings.TrimSpace(query[4:])
	} else if strings.HasPrefix(low, "only ") {
		query = strings.TrimSpace(query[5:])
	}

	features, rest := extractMediaFeatures(query)
	// Drop leftover "and" tokens; remainder should be a media type or empty.
	typeWord := mediaTypeWord(strings.ToLower(strings.TrimSpace(rest)))

	typeOK := true
	if typeWord != "" && typeWord != "all" {
		typeOK = typeWord == strings.ToLower(mediaType)
	}

	for _, feat := range features {
		if !mediaFeatureMatches(feat, widthPt, heightPt) {
			return negated
		}
	}

	if negated {
		return !typeOK
	}

	return typeOK
}

// mediaTypeWord returns the first non-"and" token of the media query
// remainder (the media type, or "" when there is none).
func mediaTypeWord(rest string) string {
	for _, f := range strings.Fields(rest) {
		if f == "and" {
			continue
		}

		return f
	}

	return ""
}

// extractMediaFeatures pulls parenthesized feature queries and returns the
// remainder (media type / and keywords).
func extractMediaFeatures(query string) ([]string, string) {
	var buf strings.Builder

	var features []string

	for idx := 0; idx < len(query); {
		if query[idx] == '(' {
			inner, end, ok := takeParenArg(query, idx)
			if !ok {
				buf.WriteByte(query[idx])

				idx++

				continue
			}

			features = append(features, strings.TrimSpace(inner))
			idx = end

			continue
		}

		buf.WriteByte(query[idx])

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
	case "width", featInlineSize:
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
