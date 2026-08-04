package layout

import (
	"strings"
	"unicode"

	"gowkhtmltopdf/internal/html"
)

// buildVerticalBlock lays out writing-mode: vertical-rl|vertical-lr.
// Ideographic / Hangul / kana glyphs are rotated 90° (sideways); Latin and
// other scripts stay upright in a stacked column.
func (e *engine) buildVerticalBlock(n *html.Node, st ResolvedStyle, availW, x, y float64) *box {
	ml := e.scalePt(st.MarginLeft)
	b := &box{node: n, style: st, kind: "block", x: x + ml, y: y}
	if st.WidthPercent >= 0 {
		b.w = availW * st.WidthPercent / 100
	} else if st.Width >= 0 {
		b.w = e.scalePt(st.Width)
	} else {
		b.w = e.scalePt(st.FontSize) * 1.6
		b.w += e.scalePt(st.PaddingLeft) + e.scalePt(st.PaddingRight) +
			e.scalePt(st.BorderLeft.Width) + e.scalePt(st.BorderRight.Width)
	}
	contentStart := len(e.ops)
	padT := e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)
	padB := e.scalePt(st.PaddingBottom)
	padL := e.scalePt(st.PaddingLeft) + e.scalePt(st.BorderLeft.Width)
	contentX := b.x + padL
	fs := e.scalePt(st.FontSize)
	lh := lineHeightOf(&st)
	if lh <= 0 {
		lh = fs * 1.2
	}
	text := strings.TrimSpace(n.TextContent())
	runes := []rune(text)
	cy := padT
	face := e.faceFor(st)
	for _, r := range runes {
		if r == '\n' || r == '\r' {
			continue
		}
		if r == ' ' {
			cy += lh * 0.4
			continue
		}
		rot := 0.0
		if isVerticalSidewaysRune(r) {
			rot = 90
		}
		e.add(Op{
			Kind: OpText, X: contentX, Y: y + cy + e.fontAscent(fs)*0.85,
			Text: string(r), Font: face, Size: fs, Bold: st.FontWeight >= 700,
			R: st.Color[0], G: st.Color[1], B: st.Color[2],
			RotateDeg: rot,
		})
		cy += lh
	}
	cy += padB
	if st.MinHeight > 0 && cy < e.scalePt(st.MinHeight) {
		cy = e.scalePt(st.MinHeight)
	}
	b.h = cy
	e.prependChrome(contentStart, st, b.x, y, b.w, b.h)
	return b
}

func isVerticalSidewaysRune(r rune) bool {
	switch {
	case r >= 0x2E80 && r <= 0x9FFF: // CJK radicals + ideographs
		return true
	case r >= 0xF900 && r <= 0xFAFF:
		return true
	case r >= 0x3040 && r <= 0x30FF: // hiragana/katakana
		return true
	case r >= 0xAC00 && r <= 0xD7AF: // Hangul syllables
		return true
	case unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hangul, r):
		return true
	}
	return false
}
