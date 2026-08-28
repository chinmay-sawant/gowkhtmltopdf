//nolint:cyclop,wsl,nlreturn,goconst // font property helpers
package layout

import (
	"strconv"
	"strings"
)

// applyFontPropsWave4 resolves font-variant, font-synthesis, font-feature-settings,
// font-kerning, font-stretch/width, and font-size-adjust.
func applyFontPropsWave4(style *ResolvedStyle, raw map[string]string) {
	if val, ok := raw["font-feature-settings"]; ok {
		style.FontFeatureSettings = strings.TrimSpace(val)
	}
	if val, ok := raw["font-kerning"]; ok {
		style.FontKerning = strings.ToLower(strings.TrimSpace(val))
	}
	if val, ok := raw["font-variant"]; ok {
		style.FontVariant = strings.ToLower(strings.TrimSpace(val))
	}
	if val, ok := raw["font-variant-caps"]; ok {
		style.FontVariantCaps = strings.ToLower(strings.TrimSpace(val))
	}
	if val, ok := raw["font-variant-ligatures"]; ok {
		style.FontVariantLigatures = strings.ToLower(strings.TrimSpace(val))
	}
	if val, ok := raw["font-variant-numeric"]; ok {
		style.FontVariantNumeric = strings.ToLower(strings.TrimSpace(val))
	}
	if val, ok := raw["font-variant-position"]; ok {
		style.FontVariantPosition = strings.ToLower(strings.TrimSpace(val))
	}
	if val, ok := raw["font-variant-east-asian"]; ok {
		style.FontVariantEastAsian = strings.ToLower(strings.TrimSpace(val))
	}
	if val, ok := raw["font-variant-emoji"]; ok {
		style.FontVariantEmoji = strings.ToLower(strings.TrimSpace(val))
	}
	if val, ok := raw["font-variant-alternates"]; ok {
		style.FontVariantAlternates = strings.ToLower(strings.TrimSpace(val))
	}
	if val, ok := raw["font-stretch"]; ok {
		style.FontStretch = strings.ToLower(strings.TrimSpace(val))
	}
	if val, ok := raw["font-width"]; ok {
		style.FontStretch = strings.ToLower(strings.TrimSpace(val))
	}
	if val, ok := raw["font-synthesis"]; ok {
		applyFontSynthesis(style, val)
	}
	if val, ok := raw["font-synthesis-weight"]; ok {
		style.FontSynthesisWeight = strings.ToLower(strings.TrimSpace(val)) != "none"
	}
	if val, ok := raw["font-synthesis-style"]; ok {
		style.FontSynthesisStyle = strings.ToLower(strings.TrimSpace(val)) != "none"
	}
	if val, ok := raw["font-synthesis-small-caps"]; ok {
		style.FontSynthesisSmallCaps = strings.ToLower(strings.TrimSpace(val)) != "none"
	}
	if val, ok := raw["font-synthesis-position"]; ok {
		style.FontSynthesisPosition = strings.ToLower(strings.TrimSpace(val)) != "none"
	}
	if val, ok := raw["font-size-adjust"]; ok {
		applyFontSizeAdjust(style, val)
	}
}

func applyFontSynthesis(style *ResolvedStyle, value string) {
	val := strings.ToLower(strings.TrimSpace(value))
	if val == "none" {
		style.FontSynthesisWeight = false
		style.FontSynthesisStyle = false
		style.FontSynthesisSmallCaps = false
		style.FontSynthesisPosition = false
		return
	}
	if val == "auto" {
		style.FontSynthesisWeight = true
		style.FontSynthesisStyle = true
		style.FontSynthesisSmallCaps = true
		style.FontSynthesisPosition = true
		return
	}
	style.FontSynthesisWeight = strings.Contains(val, "weight")
	style.FontSynthesisStyle = strings.Contains(val, "style")
	style.FontSynthesisSmallCaps = strings.Contains(val, "small-caps")
	style.FontSynthesisPosition = strings.Contains(val, "position")
}

func applyFontSizeAdjust(style *ResolvedStyle, value string) {
	val := strings.TrimSpace(value)
	if val == "none" {
		style.FontSizeAdjust = 0
		return
	}
	if n, err := strconv.ParseFloat(val, 64); err == nil && n > 0 {
		style.FontSizeAdjust = n
	}
}
