//nolint:wsl // PDF token emission groups buffer writes with font recording.
package pdf

import "unicode"

// TextShowLanguageFeaturesWithAutospace emits text with an additional point
// gap before each ideograph/Latin boundary. The gap is encoded as a PDF TJ
// adjustment so the visible glyph positions, not only the layout bounds,
// include text-autospace.
func (c *Content) TextShowLanguageFeaturesWithAutospace(
	text, lang, featureSettings string, gap float64,
) {
	if gap <= 0 || c.curSize <= 0 {
		c.TextShowLanguageFeatures(text, lang, featureSettings)

		return
	}

	feats := ParseFontFeatureSettings(featureSettings)
	shaped := ShapeTextFontWithFeaturesLanguage(text, c.fontFiles[c.curFont], feats, lang)
	if shaped == "" {
		return
	}

	fnt := c.fontFiles[c.curFont]
	if fnt == nil || !c.textNeedsType0(shaped) {
		c.emitTextRunsWithAutospace([]textRun{{s: shaped, type0: false}}, gap)

		return
	}

	c.emitTextRunsWithAutospace(splitType0Runs(shaped, fnt), gap)
}

func (c *Content) emitTextRunsWithAutospace(runs []textRun, gap float64) {
	base := trimUnicodeFontSuffix(c.curFont)
	var prev rune

	for _, runVal := range runs {
		if runVal.type0 {
			if c.curFont != base && c.curFont != base+"_u" {
				c.SetFont(base, c.curSize)
			}

			prev = c.textShowType0WithAutospace(runVal.s, gap, prev)

			continue
		}

		name := c.runFallbackFont(base, runVal.s)
		if c.curFont != name {
			c.SetFont(name, c.curSize)
		}

		prev = c.textShowSimpleWithAutospace(runVal.s, gap, prev)
	}
}

func (c *Content) textShowSimpleWithAutospace(str string, gap float64, prev rune) rune {
	if str == "" {
		return prev
	}

	out := c.buf.AvailableBuffer()
	out = append(out, '[', ' ')
	for _, rVal := range str {
		if ideographAlphaPairPDF(prev, rVal) || ideographAlphaPairPDF(rVal, prev) {
			out = appendPDFNum(out, -gap*1000/c.curSize)
			out = append(out, ' ')
		}

		out = append(out, '(')
		encoded := rVal
		if encoded > maxLatin1Code {
			encoded = winAnsiFold(encoded)
		}
		if encoded > maxLatin1Code {
			encoded = '?'
		}
		c.recordFontRune(c.curFont, encoded)
		out = appendPDFLiteralByte(out, byte(encoded))
		out = append(out, ')', ' ')
		prev = rVal
	}
	out = append(out, ']', ' ', 'T', 'J', '\n')
	_, _ = c.buf.Write(out)

	return prev
}

func (c *Content) textShowType0WithAutospace(str string, gap float64, prev rune) rune {
	if str == "" {
		return prev
	}

	base := trimUnicodeFontSuffix(c.curFont)
	uname := base + "_u"
	if f := c.fontFiles[base]; f != nil {
		c.UseEmbeddedFont(uname, f)
	} else if f := c.fontFiles[c.curFont]; f != nil && c.curFont == uname {
		c.UseEmbeddedFont(uname, f)
	}
	if c.curFont != uname {
		c.SetFont(uname, c.curSize)
	}

	out := c.buf.AvailableBuffer()
	out = append(out, '[', ' ')
	for _, rVal := range str {
		if ideographAlphaPairPDF(prev, rVal) || ideographAlphaPairPDF(rVal, prev) {
			out = appendPDFNum(out, -gap*1000/c.curSize)
			out = append(out, ' ')
		}

		if rVal > maxBMPCode {
			rVal = '?'
		}
		c.recordFontRune(uname, rVal)
		out = append(out, '<')
		out = appendHex4(out, rVal)
		out = append(out, '>', ' ')
		prev = rVal
	}
	out = append(out, ']', ' ', 'T', 'J', '\n')
	_, _ = c.buf.Write(out)

	return prev
}

func trimUnicodeFontSuffix(name string) string {
	if len(name) >= 2 && name[len(name)-2:] == "_u" {
		return name[:len(name)-2]
	}

	return name
}

func ideographAlphaPairPDF(a, b rune) bool {
	return isIdeographicRunePDF(a) && isLatinAlphaRunePDF(b)
}

func isIdeographicRunePDF(runeValue rune) bool {
	switch {
	case runeValue >= 0x3040 && runeValue <= 0x30FF:
		return true
	case runeValue >= 0x3400 && runeValue <= 0x9FFF:
		return true
	case runeValue >= 0xF900 && runeValue <= 0xFAFF:
		return true
	case runeValue >= 0xFF66 && runeValue <= 0xFF9D:
		return true
	default:
		return unicode.In(runeValue, unicode.Han, unicode.Hiragana, unicode.Katakana)
	}
}

func isLatinAlphaRunePDF(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}
