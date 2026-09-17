package layout

import (
	"math"
	"strconv"
	"strings"
)

// applyAspectRatioProps owns aspect-ratio. The ratio is width/height; 0 means
// auto/unset. Layout consumers live in aspect_ratio.go.
func applyAspectRatioProps(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	if prop != "aspect-ratio" {
		return false
	}

	ratio, ok := parseAspectRatio(value)
	if !ok {
		return true
	}

	style.AspectRatio = ratio

	return true
}

// parseAspectRatio accepts auto | <ratio> | auto && <ratio>. A bare ratio
// stores width/height; auto clears to 0. The auto+ratio form keeps the ratio
// for replaced-size hints (same storage as a bare ratio in this lite subset).
// Spaced ratios like "1 / 1" are normalized before the token parse.
func parseAspectRatio(raw string) (float64, bool) {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.Join(strings.Fields(value), " ")
	value = strings.ReplaceAll(value, " / ", "/")
	value = strings.ReplaceAll(value, "/ ", "/")
	value = strings.ReplaceAll(value, " /", "/")

	switch value {
	case "", "auto":
		return 0, true
	}

	hasAuto := false
	ratioTok := ""

	for _, tok := range strings.Fields(value) {
		if tok == "auto" {
			hasAuto = true

			continue
		}

		if ratioTok != "" {
			return 0, false
		}

		ratioTok = tok
	}

	if ratioTok == "" {
		return 0, hasAuto
	}

	ratio, ok := parseAspectRatioToken(ratioTok)
	if !ok {
		return 0, false
	}

	return ratio, true
}

func parseAspectRatioToken(tok string) (float64, bool) {
	parts := strings.Split(tok, "/")
	switch len(parts) {
	case 1:
		w, err := strconv.ParseFloat(parts[0], 64)
		if err != nil || w <= 0 || math.IsNaN(w) || math.IsInf(w, 0) {
			return 0, false
		}

		return w, true
	case 2:
		w, errW := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		h, errH := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if errW != nil || errH != nil || w <= 0 || h <= 0 ||
			math.IsNaN(w) || math.IsNaN(h) || math.IsInf(w, 0) || math.IsInf(h, 0) {
			return 0, false
		}

		return w / h, true
	default:
		return 0, false
	}
}
