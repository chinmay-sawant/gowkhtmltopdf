//nolint:cyclop,funlen,wsl,varnamelen,testpackage // targeted unit tests
package layout

import (
	"testing"
)

func TestWaveABookmarksAndPagedMedia(t *testing.T) {
	t.Parallel()

	var s ResolvedStyle
	applyAdvancedProps(&s, "bookmark-level", "2", 12)
	applyAdvancedProps(&s, "bookmark-label", "'Chapter 1'", 12)
	applyAdvancedProps(&s, "bookmark-state", "open", 12)
	applyAdvancedProps(&s, "footnote-display", "inline", 12)
	applyAdvancedProps(&s, "footnote-policy", "line", 12)
	applyAdvancedProps(&s, "string-set", "header-title content()", 12)

	if s.BookmarkLevel != 2 {
		t.Errorf("BookmarkLevel = %d, want 2", s.BookmarkLevel)
	}
	if s.BookmarkLabel != "Chapter 1" {
		t.Errorf("BookmarkLabel = %q, want 'Chapter 1'", s.BookmarkLabel)
	}
	if s.BookmarkState != "open" {
		t.Errorf("BookmarkState = %q, want 'open'", s.BookmarkState)
	}
	if s.FootnoteDisplay != "inline" {
		t.Errorf("FootnoteDisplay = %q, want 'inline'", s.FootnoteDisplay)
	}
	if s.FootnotePolicy != "line" {
		t.Errorf("FootnotePolicy = %q, want 'line'", s.FootnotePolicy)
	}
	if s.StringSet != "header-title content()" {
		t.Errorf("StringSet = %q, want 'header-title content()'", s.StringSet)
	}

	applyAdvancedProps(&s, "bookmark-level", "none", 12)
	if s.BookmarkLevel != 0 {
		t.Errorf("BookmarkLevel after 'none' = %d, want 0", s.BookmarkLevel)
	}
}

func TestWaveBTextTruncationAndClamping(t *testing.T) {
	t.Parallel()

	var s ResolvedStyle
	applyAdvancedProps(&s, "text-overflow", "ellipsis", 12)
	applyAdvancedProps(&s, "line-clamp", "3", 12)
	applyAdvancedProps(&s, "max-lines", "5", 12)
	applyAdvancedProps(&s, "margin-trim", "block", 12)

	if s.TextOverflow != "ellipsis" {
		t.Errorf("TextOverflow = %q, want 'ellipsis'", s.TextOverflow)
	}
	if s.LineClamp != 3 {
		t.Errorf("LineClamp = %d, want 3", s.LineClamp)
	}
	if s.MaxLines != 5 {
		t.Errorf("MaxLines = %d, want 5", s.MaxLines)
	}
	if s.MarginTrim != "block" {
		t.Errorf("MarginTrim = %q, want 'block'", s.MarginTrim)
	}
}

func TestWaveCFragmentationAndImageMetadata(t *testing.T) {
	t.Parallel()

	var s ResolvedStyle
	applyAdvancedProps(&s, "box-decoration-break", "clone", 12)
	applyAdvancedProps(&s, "image-orientation", "from-image", 12)
	applyAdvancedProps(&s, "image-resolution", "300dpi", 12)
	applyAdvancedProps(&s, "object-view-box", "inset(10px)", 12)
	applyAdvancedProps(&s, "print-color-adjust", "exact", 12)
	applyAdvancedProps(&s, "forced-color-adjust", "none", 12)
	applyAdvancedProps(&s, "color-scheme", "dark light", 12)
	applyAdvancedProps(&s, "contain", "layout paint", 12)
	applyAdvancedProps(&s, "content-visibility", "auto", 12)
	applyAdvancedProps(&s, "contain-intrinsic-width", "100px", 12)
	applyAdvancedProps(&s, "contain-intrinsic-height", "50pt", 12)

	if s.BoxDecorationBreak != "clone" {
		t.Errorf("BoxDecorationBreak = %q, want 'clone'", s.BoxDecorationBreak)
	}
	if s.ImageOrientation != "from-image" {
		t.Errorf("ImageOrientation = %q, want 'from-image'", s.ImageOrientation)
	}
	if s.ImageResolution != 300 {
		t.Errorf("ImageResolution = %v, want 300", s.ImageResolution)
	}
	if s.ObjectViewBox != "inset(10px)" {
		t.Errorf("ObjectViewBox = %q, want 'inset(10px)'", s.ObjectViewBox)
	}
	if s.PrintColorAdjust != "exact" {
		t.Errorf("PrintColorAdjust = %q, want 'exact'", s.PrintColorAdjust)
	}
	if s.ForcedColorAdjust != "none" {
		t.Errorf("ForcedColorAdjust = %q, want 'none'", s.ForcedColorAdjust)
	}
	if s.ColorScheme != "dark light" {
		t.Errorf("ColorScheme = %q, want 'dark light'", s.ColorScheme)
	}
	if s.Contain != "layout paint" {
		t.Errorf("Contain = %q, want 'layout paint'", s.Contain)
	}
	if s.ContentVisibility != "auto" {
		t.Errorf("ContentVisibility = %q, want 'auto'", s.ContentVisibility)
	}
	if s.ContainIntrinsicWidth != 75 {
		t.Errorf("ContainIntrinsicWidth = %v, want 75", s.ContainIntrinsicWidth)
	}
	if s.ContainIntrinsicHeight != 50 {
		t.Errorf("ContainIntrinsicHeight = %v, want 50", s.ContainIntrinsicHeight)
	}
}

func TestWaveDFontVariationsAndBlendModes(t *testing.T) {
	t.Parallel()

	var s ResolvedStyle
	applyAdvancedProps(&s, "font-variation-settings", "'wght' 700, 'wdth' 100", 12)
	applyAdvancedProps(&s, "font-optical-sizing", "auto", 12)
	applyAdvancedProps(&s, "font-language-override", "ENG", 12)
	applyAdvancedProps(&s, "font-palette", "dark", 12)
	applyAdvancedProps(&s, "mix-blend-mode", "multiply", 12)
	applyAdvancedProps(&s, "background-blend-mode", "screen", 12)
	applyAdvancedProps(&s, "isolation", "isolate", 12)
	applyAdvancedProps(&s, "text-combine-upright", "digits 2", 12)
	applyAdvancedProps(&s, "text-orientation", "upright", 12)
	applyAdvancedProps(&s, "unicode-bidi", "isolate", 12)
	applyAdvancedProps(&s, "text-emphasis", "dot", 12)
	applyAdvancedProps(&s, "text-emphasis-color", "#ff0000", 12)
	applyAdvancedProps(&s, "text-emphasis-position", "over right", 12)
	applyAdvancedProps(&s, "text-decoration-skip-ink", "auto", 12)
	applyAdvancedProps(&s, "overflow-clip-margin-top", "10px", 12)
	applyAdvancedProps(&s, "overflow-clip-margin-right", "20pt", 12)

	if s.FontVariationSettings != "'wght' 700, 'wdth' 100" {
		t.Errorf("FontVariationSettings = %q", s.FontVariationSettings)
	}
	if s.FontOpticalSizing != "auto" {
		t.Errorf("FontOpticalSizing = %q, want 'auto'", s.FontOpticalSizing)
	}
	if s.FontLanguageOverride != "ENG" {
		t.Errorf("FontLanguageOverride = %q, want 'ENG'", s.FontLanguageOverride)
	}
	if s.FontPalette != "dark" {
		t.Errorf("FontPalette = %q, want 'dark'", s.FontPalette)
	}
	if s.MixBlendMode != "multiply" {
		t.Errorf("MixBlendMode = %q, want 'multiply'", s.MixBlendMode)
	}
	if s.BackgroundBlendMode != "screen" {
		t.Errorf("BackgroundBlendMode = %q, want 'screen'", s.BackgroundBlendMode)
	}
	if s.Isolation != "isolate" {
		t.Errorf("Isolation = %q, want 'isolate'", s.Isolation)
	}
	if s.TextCombineUpright != "digits 2" {
		t.Errorf("TextCombineUpright = %q, want 'digits 2'", s.TextCombineUpright)
	}
	if s.TextOrientation != "upright" {
		t.Errorf("TextOrientation = %q, want 'upright'", s.TextOrientation)
	}
	if s.UnicodeBidi != "isolate" {
		t.Errorf("UnicodeBidi = %q, want 'isolate'", s.UnicodeBidi)
	}
	if s.TextEmphasis != "dot" {
		t.Errorf("TextEmphasis = %q, want 'dot'", s.TextEmphasis)
	}
	if !s.TextEmphasisColorSet || s.TextEmphasisColor[0] != 1.0 {
		t.Errorf("TextEmphasisColor = %v, want red [1, 0, 0]", s.TextEmphasisColor)
	}
	if s.TextEmphasisPosition != "over right" {
		t.Errorf("TextEmphasisPosition = %q, want 'over right'", s.TextEmphasisPosition)
	}
	if s.TextDecorationSkipInk != "auto" {
		t.Errorf("TextDecorationSkipInk = %q, want 'auto'", s.TextDecorationSkipInk)
	}
	if s.OverflowClipMarginTop != 7.5 {
		t.Errorf("OverflowClipMarginTop = %v, want 7.5", s.OverflowClipMarginTop)
	}
	if s.OverflowClipMarginRight != 20 {
		t.Errorf("OverflowClipMarginRight = %v, want 20", s.OverflowClipMarginRight)
	}
}
