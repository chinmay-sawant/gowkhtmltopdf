package css

import (
	"math"
	"strconv"
	"strings"
)

// MathEnv supplies the bases that relative units resolve against. All values
// are in CSS pixels. A zero base resolves that unit to zero rather than
// failing, matching how callers treat an unknown containing block today.
type MathEnv struct {
	FontSizePx float64 // em / ex / ch base
	PercentPx  float64 // 1% of the containing block
	ViewportW  float64 // vw / vmin / vmax base
	ViewportH  float64 // vh / dvh / svh / lvh / vmin / vmax base
}

const (
	mathPxPerInch = 96.0
	mathPxPerPica = 16.0
)

// EvalMath evaluates a CSS math function over lengths and percentages:
// calc(), min(), max(), or clamp(). It supports + - * /, parentheses,
// comma-separated arguments, nesting, and mixed units. The result is in CSS
// pixels. Values that are not a math function, or that fail to parse, return
// false so the caller can keep its fallback.
func EvalMath(value string, env MathEnv) (float64, bool) {
	src := strings.TrimSpace(value)
	if src == "" || !isMathLetter(src[0]) {
		return 0, false
	}

	parser := mathParser{src: src, pos: 0, env: env}

	result, ok := parser.parseExpr()
	if !ok || parser.pos != len(parser.src) {
		return 0, false
	}

	return result, true
}

type mathParser struct {
	src string
	pos int
	env MathEnv
}

func (p *mathParser) parseExpr() (float64, bool) {
	left, ok := p.parseTerm()
	if !ok {
		return 0, false
	}

	for {
		p.skipSpace()

		if p.pos >= len(p.src) {
			return left, true
		}

		operator := p.src[p.pos]
		if operator != '+' && operator != '-' {
			return left, true
		}

		p.pos++

		right, ok := p.parseTerm()
		if !ok {
			return 0, false
		}

		if operator == '+' {
			left += right
		} else {
			left -= right
		}
	}
}

func (p *mathParser) parseTerm() (float64, bool) {
	left, ok := p.parseUnary()
	if !ok {
		return 0, false
	}

	for {
		p.skipSpace()

		if p.pos >= len(p.src) {
			return left, true
		}

		operator := p.src[p.pos]
		if operator != '*' && operator != '/' {
			return left, true
		}

		p.pos++

		right, ok := p.parseUnary()
		if !ok {
			return 0, false
		}

		if operator == '*' {
			left *= right
		} else {
			if right == 0 {
				return 0, false
			}

			left /= right
		}
	}
}

func (p *mathParser) parseUnary() (float64, bool) {
	p.skipSpace()

	if p.pos >= len(p.src) {
		return 0, false
	}

	switch p.src[p.pos] {
	case '+':
		p.pos++

		return p.parseUnary()
	case '-':
		p.pos++

		value, ok := p.parseUnary()

		return -value, ok
	default:
		return p.parsePrimary()
	}
}

func (p *mathParser) parsePrimary() (float64, bool) {
	p.skipSpace()

	if p.pos >= len(p.src) {
		return 0, false
	}

	switch {
	case p.src[p.pos] == '(':
		p.pos++

		value, ok := p.parseExpr()
		if !ok || !p.consume(')') {
			return 0, false
		}

		return value, true
	case isMathLetter(p.src[p.pos]):
		return p.parseFunc()
	default:
		return p.parseLengthToken()
	}
}

func (p *mathParser) parseFunc() (float64, bool) {
	start := p.pos

	for p.pos < len(p.src) && isMathLetter(p.src[p.pos]) {
		p.pos++
	}

	name := strings.ToLower(p.src[start:p.pos])

	if !p.consume('(') {
		return 0, false
	}

	switch name {
	case "calc":
		return p.parseCalc()
	case "min":
		return p.parseMinMax(false)
	case "max":
		return p.parseMinMax(true)
	case "clamp":
		return p.parseClamp()
	default:
		return 0, false
	}
}

// parseCalc parses the body of calc(): one expression plus the closing ')'.
func (p *mathParser) parseCalc() (float64, bool) {
	value, ok := p.parseExpr()
	if !ok || !p.consume(')') {
		return 0, false
	}

	return value, true
}

func (p *mathParser) parseMinMax(wantMax bool) (float64, bool) {
	value, ok := p.parseArg()
	if !ok {
		return 0, false
	}

	for p.consume(',') {
		next, ok := p.parseArg()
		if !ok {
			return 0, false
		}

		if wantMax {
			value = math.Max(value, next)
		} else {
			value = math.Min(value, next)
		}
	}

	if !p.consume(')') {
		return 0, false
	}

	return value, true
}

func (p *mathParser) parseClamp() (float64, bool) {
	low, found := p.parseArg()
	if !found || !p.consume(',') {
		return 0, false
	}

	value, found := p.parseArg()
	if !found || !p.consume(',') {
		return 0, false
	}

	high, found := p.parseArg()
	if !found || !p.consume(')') {
		return 0, false
	}

	return math.Max(low, math.Min(value, high)), true
}

func (p *mathParser) parseArg() (float64, bool) {
	value, ok := p.parseExpr()
	if !ok {
		return 0, false
	}

	p.skipSpace()

	return value, true
}

func (p *mathParser) parseLengthToken() (float64, bool) {
	p.skipSpace()
	start := p.pos

	for p.pos < len(p.src) && (isMathDigit(p.src[p.pos]) || p.src[p.pos] == '.') {
		p.pos++
	}

	if p.pos == start {
		return 0, false
	}

	number, err := strconv.ParseFloat(p.src[start:p.pos], 64)
	if err != nil {
		return 0, false
	}

	unitStart := p.pos

	for p.pos < len(p.src) && (isMathLetter(p.src[p.pos]) || p.src[p.pos] == '%') {
		p.pos++
	}

	return mathUnitPx(number, strings.ToLower(p.src[unitStart:p.pos]), p.env)
}

// mathUnitPx converts a number plus unit into CSS pixels.
func mathUnitPx(number float64, unit string, env MathEnv) (float64, bool) {
	switch unit {
	case "", "px":
		return number, true
	case "em":
		return number * env.FontSizePx, true
	case unitRem:
		return number * rootFontSizePx, true
	case "ex", "ch":
		return number * env.FontSizePx * exChToEmFactor, true
	default:
		return mathAbsolutePx(number, unit, env)
	}
}

// mathAbsolutePx converts the absolute length units pt, pc, in, cm, and mm.
func mathAbsolutePx(number float64, unit string, env MathEnv) (float64, bool) {
	switch unit {
	case "pt":
		return number * mathPxPerInch / pointsPerInch, true
	case "pc":
		return number * mathPxPerPica, true
	case "in":
		return number * mathPxPerInch, true
	case "cm":
		return number * mathPxPerInch / cmPerInch, true
	case "mm":
		return number * mathPxPerInch / mmPerInch, true
	default:
		return mathViewportPx(number, unit, env)
	}
}

// mathViewportPx converts the viewport-relative units vw, vh, dvh, svh, lvh,
// vmin, and vmax, plus percentages.
func mathViewportPx(number float64, unit string, env MathEnv) (float64, bool) {
	switch unit {
	case "%":
		return number * env.PercentPx / percentScale, true
	case "vw":
		return number * env.ViewportW / percentScale, true
	case "vh", "dvh", "svh", "lvh":
		return number * env.ViewportH / percentScale, true
	case "vmin":
		return number * math.Min(env.ViewportW, env.ViewportH) / percentScale, true
	case "vmax":
		return number * math.Max(env.ViewportW, env.ViewportH) / percentScale, true
	default:
		return 0, false
	}
}

func (p *mathParser) skipSpace() {
	for p.pos < len(p.src) {
		switch p.src[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

func (p *mathParser) consume(ch byte) bool {
	p.skipSpace()

	if p.pos < len(p.src) && p.src[p.pos] == ch {
		p.pos++

		return true
	}

	return false
}

func isMathDigit(ch byte) bool { return ch >= '0' && ch <= '9' }

func isMathLetter(ch byte) bool {
	return ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z'
}
