//nolint:cyclop,funlen,gocognit,gocyclo,maintidx // CSS Fonts feature/variant parsers
package layout

import (
	"strconv"
	"strings"
)

const (
	fontKerningAuto   = "auto"
	fontKerningNormal = "normal"
	fontKerningNone   = "none"
)

// applyFontFeatureProps owns font-feature-settings, font-kerning, font-variant
// shorthand, and the font-variant-* longhands that map to OpenType tags.
func applyFontFeatureProps(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext,
	_ *ResolvedStyle, _ bool,
) bool {
	switch prop {
	case "font-feature-settings":
		if settings, ok := parseFontFeatureSettingsValue(value); ok {
			style.FontFeatureSettings = settings
		}
	case "font-kerning":
		switch strings.ToLower(strings.TrimSpace(value)) {
		case fontKerningAuto:
			style.FontKerning = fontKerningAuto
		case fontKerningNormal:
			style.FontKerning = fontKerningNormal
		case fontKerningNone:
			style.FontKerning = fontKerningNone
		}
	case "font-variant":
		applyFontVariantShorthand(style, value)
	case "font-variant-caps":
		if v, ok := normalizeFontVariantCaps(value); ok {
			style.FontVariantCaps = v
		}
	case "font-variant-ligatures":
		if v, ok := normalizeFontVariantLigatures(value); ok {
			style.FontVariantLigatures = v
		}
	case "font-variant-numeric":
		if v, ok := normalizeFontVariantNumeric(value); ok {
			style.FontVariantNumeric = v
		}
	case "font-variant-position":
		if v, ok := normalizeFontVariantPosition(value); ok {
			style.FontVariantPosition = v
		}
	case "font-variant-east-asian":
		if v, ok := normalizeFontVariantEastAsian(value); ok {
			style.FontVariantEastAsian = v
		}
	case "font-variant-alternates":
		if v, ok := normalizeFontVariantAlternates(value); ok {
			style.FontVariantAlternates = v
		}
	case "font-variant-emoji":
		if v, ok := normalizeFontVariantEmoji(value); ok {
			style.FontVariantEmoji = v
		}
	default:
		return false
	}

	return true
}

// copyInheritedFontShapingProps copies the coalesced CSS Fonts inherit cluster.
func copyInheritedFontShapingProps(dst, src *ResolvedStyle) {
	dst.FontLanguageOverride = src.FontLanguageOverride
	dst.FontOpticalSizing = src.FontOpticalSizing
	dst.FontPalette = src.FontPalette
	dst.FontVariationSettings = src.FontVariationSettings
	dst.FontFeatureSettings = src.FontFeatureSettings
	dst.FontKerning = src.FontKerning
	dst.FontSizeAdjust = src.FontSizeAdjust
	dst.FontSizeAdjustSet = src.FontSizeAdjustSet
	dst.FontWidth = src.FontWidth
	dst.FontSynthesisWeight = src.FontSynthesisWeight
	dst.FontSynthesisStyle = src.FontSynthesisStyle
	dst.FontSynthesisSmallCaps = src.FontSynthesisSmallCaps
	dst.FontSynthesisPosition = src.FontSynthesisPosition
	dst.FontVariantCaps = src.FontVariantCaps
	dst.FontVariantLigatures = src.FontVariantLigatures
	dst.FontVariantNumeric = src.FontVariantNumeric
	dst.FontVariantPosition = src.FontVariantPosition
	dst.FontVariantEastAsian = src.FontVariantEastAsian
	dst.FontVariantAlternates = src.FontVariantAlternates
	dst.FontVariantEmoji = src.FontVariantEmoji
}

func parseFontFeatureSettingsValue(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if strings.EqualFold(value, fontVariantNormal) {
		return fontVariantNormal, true
	}

	feats := splitFontVariationList(value)
	if len(feats) == 0 {
		return "", false
	}

	var out strings.Builder

	for i, part := range feats {
		tag, count, ok := parseFeatureSettingsPart(part)
		if !ok {
			return "", false
		}

		if i > 0 {
			out.WriteString(", ")
		}

		out.WriteString(`"`)
		out.WriteString(tag)
		out.WriteString(`" `)
		out.WriteString(strconv.FormatUint(uint64(count), 10))
	}

	return out.String(), true
}

func parseFeatureSettingsPart(part string) (string, uint32, bool) {
	part = strings.TrimSpace(part)
	if part == "" {
		return "", 0, false
	}

	if len(part) < 2 || (part[0] != '"' && part[0] != '\'') {
		return "", 0, false
	}

	quote := part[0]
	end := strings.IndexByte(part[1:], quote)
	if end < 0 {
		return "", 0, false
	}

	tag := part[1 : 1+end]
	if len(tag) != fontAxisTagLength {
		return "", 0, false
	}

	for i := range len(tag) {
		if !isASCIIAlnum(tag[i]) {
			return "", 0, false
		}
	}

	rest := strings.TrimSpace(part[2+end:])
	val := uint32(1)

	if rest != "" {
		switch strings.ToLower(rest) {
		case "on":
			val = 1
		case "off":
			val = 0
		default:
			n, err := strconv.ParseUint(rest, 10, 16)
			if err != nil {
				return "", 0, false
			}

			val = uint32(n)
		}
	}

	return tag, val, true
}

func applyFontVariantShorthand(style *ResolvedStyle, raw string) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return
	}

	if value == fontVariantNormal {
		style.FontVariantCaps = fontVariantNormal
		style.FontVariantLigatures = fontVariantNormal
		style.FontVariantNumeric = fontVariantNormal
		style.FontVariantPosition = fontVariantNormal
		style.FontVariantEastAsian = fontVariantNormal
		style.FontVariantAlternates = fontVariantNormal
		style.FontVariantEmoji = fontVariantNormal

		return
	}

	if value == fontKerningNone {
		style.FontVariantLigatures = fontKerningNone

		return
	}

	// CSS2 small-caps alone is still the common print form.
	if value == "small-caps" {
		style.FontVariantCaps = "small-caps"

		return
	}

	tokens := strings.Fields(value)
	caps, lig, num, pos, east := fontVariantNormal, fontVariantNormal, fontVariantNormal, fontVariantNormal, fontVariantNormal
	ligParts, numParts, eastParts := []string{}, []string{}, []string{}

	for _, tok := range tokens {
		switch {
		case isFontVariantCapsKeyword(tok):
			caps = tok
		case isFontVariantPositionKeyword(tok):
			pos = tok
		case isFontVariantLigatureKeyword(tok):
			ligParts = append(ligParts, tok)
		case isFontVariantNumericKeyword(tok):
			numParts = append(numParts, tok)
		case isFontVariantEastAsianKeyword(tok):
			eastParts = append(eastParts, tok)
		default:
			return // drop invalid shorthand entirely
		}
	}

	if len(ligParts) > 0 {
		lig = strings.Join(ligParts, " ")
	}

	if len(numParts) > 0 {
		num = strings.Join(numParts, " ")
	}

	if len(eastParts) > 0 {
		east = strings.Join(eastParts, " ")
	}

	style.FontVariantCaps = caps
	style.FontVariantLigatures = lig
	style.FontVariantNumeric = num
	style.FontVariantPosition = pos
	style.FontVariantEastAsian = east
}

func normalizeFontVariantCaps(raw string) (string, bool) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == fontVariantNormal || isFontVariantCapsKeyword(v) {
		return v, true
	}

	return "", false
}

func normalizeFontVariantLigatures(raw string) (string, bool) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == fontVariantNormal || v == fontKerningNone {
		return v, true
	}

	parts := strings.Fields(v)
	if len(parts) == 0 {
		return "", false
	}

	for _, p := range parts {
		if !isFontVariantLigatureKeyword(p) {
			return "", false
		}
	}

	return strings.Join(parts, " "), true
}

func normalizeFontVariantNumeric(raw string) (string, bool) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == fontVariantNormal {
		return v, true
	}

	parts := strings.Fields(v)
	if len(parts) == 0 {
		return "", false
	}

	for _, p := range parts {
		if !isFontVariantNumericKeyword(p) {
			return "", false
		}
	}

	return strings.Join(parts, " "), true
}

func normalizeFontVariantPosition(raw string) (string, bool) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == fontVariantNormal || isFontVariantPositionKeyword(v) {
		return v, true
	}

	return "", false
}

func normalizeFontVariantEastAsian(raw string) (string, bool) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == fontVariantNormal {
		return v, true
	}

	parts := strings.Fields(v)
	if len(parts) == 0 {
		return "", false
	}

	for _, p := range parts {
		if !isFontVariantEastAsianKeyword(p) {
			return "", false
		}
	}

	return strings.Join(parts, " "), true
}

func normalizeFontVariantAlternates(raw string) (string, bool) {
	v := strings.TrimSpace(raw)
	low := strings.ToLower(v)
	if low == fontVariantNormal || low == "historical-forms" {
		return low, true
	}
	// Named alternates need @font-feature-values; store the raw token list for
	// honesty as Partial rather than inventing OT tags.
	if v != "" {
		return low, true
	}

	return "", false
}

func normalizeFontVariantEmoji(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case fontVariantNormal, "text", "emoji", "unicode":
		return strings.ToLower(strings.TrimSpace(raw)), true
	default:
		return "", false
	}
}

func isFontVariantCapsKeyword(v string) bool {
	switch v {
	case "small-caps", "all-small-caps", "petite-caps", "all-petite-caps", "unicase", "titling-caps":
		return true
	default:
		return false
	}
}

func isFontVariantPositionKeyword(v string) bool {
	return v == "sub" || v == "super"
}

func isFontVariantLigatureKeyword(v string) bool {
	switch v {
	case "common-ligatures", "no-common-ligatures",
		"discretionary-ligatures", "no-discretionary-ligatures",
		"historical-ligatures", "no-historical-ligatures",
		"contextual", "no-contextual":
		return true
	default:
		return false
	}
}

func isFontVariantNumericKeyword(v string) bool {
	switch v {
	case "lining-nums", "oldstyle-nums", "proportional-nums", "tabular-nums",
		"diagonal-fractions", "stacked-fractions", "ordinal", "slashed-zero":
		return true
	default:
		return false
	}
}

func isFontVariantEastAsianKeyword(v string) bool {
	switch v {
	case "jis78", "jis83", "jis90", "jis04", "simplified", "traditional",
		"full-width", "proportional-width", "ruby":
		return true
	default:
		return false
	}
}

// fontShapingFeatureSettings builds the OpenType feature list string passed to
// pdf.ParseFontFeatureSettings. Low-level font-feature-settings wins on tag
// collisions. Empty means "no CSS features" (shaper keeps CJK defaults).
func fontShapingFeatureSettings(sty *ResolvedStyle) string {
	if sty == nil {
		return ""
	}

	order := make([]string, 0, 8)
	tags := make(map[string]uint32, 8)

	put := func(tag string, val uint32) {
		if _, seen := tags[tag]; !seen {
			order = append(order, tag)
		}

		tags[tag] = val
	}

	if sty.FontKerning == fontKerningNone {
		put("kern", 0)
	}

	appendVariantFeatureTags(sty, put)

	if sty.FontFeatureSettings != "" && sty.FontFeatureSettings != fontVariantNormal {
		for _, part := range splitFontVariationList(sty.FontFeatureSettings) {
			tag, val, ok := parseFeatureSettingsPart(part)
			if !ok {
				continue
			}

			put(tag, val)
		}
	}

	if len(order) == 0 {
		return ""
	}

	var out strings.Builder

	for i, tag := range order {
		if i > 0 {
			out.WriteString(", ")
		}

		out.WriteString(`"`)
		out.WriteString(tag)
		out.WriteString(`" `)
		out.WriteString(strconv.FormatUint(uint64(tags[tag]), 10))
	}

	return out.String()
}

func appendVariantFeatureTags(sty *ResolvedStyle, put func(string, uint32)) {
	switch sty.FontVariantCaps {
	case "small-caps":
		put("smcp", 1)
	case "all-small-caps":
		put("c2sc", 1)
		put("smcp", 1)
	case "petite-caps":
		put("pcap", 1)
	case "all-petite-caps":
		put("c2pc", 1)
		put("pcap", 1)
	case "unicase":
		put("unic", 1)
	case "titling-caps":
		put("titl", 1)
	}

	for _, tok := range strings.Fields(sty.FontVariantLigatures) {
		switch tok {
		case fontKerningNone:
			put("liga", 0)
			put("clig", 0)
			put("dlig", 0)
			put("hlig", 0)
			put("calt", 0)
		case "common-ligatures":
			put("liga", 1)
			put("clig", 1)
		case "no-common-ligatures":
			put("liga", 0)
			put("clig", 0)
		case "discretionary-ligatures":
			put("dlig", 1)
		case "no-discretionary-ligatures":
			put("dlig", 0)
		case "historical-ligatures":
			put("hlig", 1)
		case "no-historical-ligatures":
			put("hlig", 0)
		case "contextual":
			put("calt", 1)
		case "no-contextual":
			put("calt", 0)
		}
	}

	for _, tok := range strings.Fields(sty.FontVariantNumeric) {
		switch tok {
		case "lining-nums":
			put("lnum", 1)
		case "oldstyle-nums":
			put("onum", 1)
		case "proportional-nums":
			put("pnum", 1)
		case "tabular-nums":
			put("tnum", 1)
		case "diagonal-fractions":
			put("frac", 1)
		case "stacked-fractions":
			put("afrc", 1)
		case "ordinal":
			put("ordn", 1)
		case "slashed-zero":
			put("zero", 1)
		}
	}

	switch sty.FontVariantPosition {
	case "sub":
		put("subs", 1)
	case "super":
		put("sups", 1)
	}

	switch sty.FontVariantAlternates {
	case "historical-forms":
		put("hist", 1)
	}

	for _, tok := range strings.Fields(sty.FontVariantEastAsian) {
		switch tok {
		case "jis78":
			put("jp78", 1)
		case "jis83":
			put("jp83", 1)
		case "jis90":
			put("jp90", 1)
		case "jis04":
			put("jp04", 1)
		case "simplified":
			put("smpl", 1)
		case "traditional":
			put("trad", 1)
		case "full-width":
			put("fwid", 1)
		case "proportional-width":
			put("pwid", 1)
		case "ruby":
			put("ruby", 1)
		}
	}
}
