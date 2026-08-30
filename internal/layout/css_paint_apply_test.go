//nolint:testpackage,varnamelen,cyclop,funlen // cascade paint apply proofs
package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

func TestCurrentColor(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="outer">
			<p class="inner">x</p>
			<p class="own">y</p>
			<p class="outline">z</p>
			<p class="fold">w</p>
		</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.outer { color: red }
		.inner { border-color: currentColor }
		.own { color: currentColor }
		.outline { outline-color: currentColor }
		.fold { border-top-color: currentcolor }
	`)}, "print", testViewport, 800)

	red := [3]float64{1, 0, 0}

	inner := styleByClass(t, styles, "inner")
	if inner.Color != red {
		t.Fatalf("inner Color = %v, want red inherited", inner.Color)
	}

	if inner.BorderTop.Color != red || inner.BorderRight.Color != red ||
		inner.BorderBottom.Color != red || inner.BorderLeft.Color != red {
		t.Fatalf("inner border-color currentColor = T%v R%v B%v L%v, want red",
			inner.BorderTop.Color, inner.BorderRight.Color, inner.BorderBottom.Color, inner.BorderLeft.Color)
	}

	own := styleByClass(t, styles, "own")
	if own.Color != red {
		t.Fatalf("own color:currentColor = %v, want parent red", own.Color)
	}

	outline := styleByClass(t, styles, "outline")
	if outline.OutlineColor != red || !outline.OutlineColorSet {
		t.Fatalf("outline-color currentColor = %v set=%v, want red", outline.OutlineColor, outline.OutlineColorSet)
	}

	fold := styleByClass(t, styles, "fold")
	if fold.BorderTop.Color != red {
		t.Fatalf("border-top-color currentcolor = %v, want red", fold.BorderTop.Color)
	}
}

func TestOutlineParse(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="sh">x</div>
		<div class="width">x</div>
		<div class="thin">x</div>
		<div class="thick">x</div>
		<div class="dashed">x</div>
		<div class="dotted">x</div>
		<div class="none">x</div>
		<div class="color">x</div>
		<div class="offset">x</div>
		<div class="box">x</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.sh { outline: 2pt solid red }
		.width { outline-width: medium }
		.thin { outline-width: thin }
		.thick { outline-width: thick }
		.dashed { outline-style: dashed }
		.dotted { outline-style: dotted }
		.none { outline-style: none }
		.color { outline-color: blue }
		.offset { outline-offset: 4pt }
		.box { width: 100pt; height: 40pt; outline: 8pt solid red }
	`)}, "print", testViewport, 800)

	sh := styleByClass(t, styles, "sh")
	if !near(sh.OutlineWidth, 2) {
		t.Fatalf("outline width = %.3f, want 2pt", sh.OutlineWidth)
	}

	if sh.OutlineStyle != solidKeyword {
		t.Fatalf("outline style = %q, want solid", sh.OutlineStyle)
	}

	if sh.OutlineColor != ([3]float64{1, 0, 0}) || !sh.OutlineColorSet {
		t.Fatalf("outline color = %v set=%v, want red", sh.OutlineColor, sh.OutlineColorSet)
	}

	if !near(styleByClass(t, styles, "width").OutlineWidth, borderWidth(mediumKeyword, defaultFontSizePt)) {
		t.Fatalf("outline-width:medium = %.3f, want medium", styleByClass(t, styles, "width").OutlineWidth)
	}

	if !near(styleByClass(t, styles, "thin").OutlineWidth, borderWidth("thin", defaultFontSizePt)) {
		t.Fatalf("outline-width:thin = %.3f", styleByClass(t, styles, "thin").OutlineWidth)
	}

	if !near(styleByClass(t, styles, "thick").OutlineWidth, borderWidth("thick", defaultFontSizePt)) {
		t.Fatalf("outline-width:thick = %.3f", styleByClass(t, styles, "thick").OutlineWidth)
	}

	if got := styleByClass(t, styles, "dashed").OutlineStyle; got != "dashed" {
		t.Fatalf("outline-style:dashed = %q", got)
	}

	if got := styleByClass(t, styles, "dotted").OutlineStyle; got != "dotted" {
		t.Fatalf("outline-style:dotted = %q", got)
	}

	if got := styleByClass(t, styles, "none").OutlineStyle; got != cssDisplayNone {
		t.Fatalf("outline-style:none = %q", got)
	}

	blue := styleByClass(t, styles, "color")
	if blue.OutlineColor != ([3]float64{0, 0, 1}) || !blue.OutlineColorSet {
		t.Fatalf("outline-color:blue = %v set=%v", blue.OutlineColor, blue.OutlineColorSet)
	}

	if !near(styleByClass(t, styles, "offset").OutlineOffset, 4) {
		t.Fatalf("outline-offset = %.3f, want 4pt", styleByClass(t, styles, "offset").OutlineOffset)
	}

	box := styleByClass(t, styles, "box")
	if !near(box.Width, 100) || !near(box.Height, 40) {
		t.Fatalf("outline changed layout size: width=%.3f height=%.3f", box.Width, box.Height)
	}

	if !near(box.OutlineWidth, 8) || box.OutlineStyle != solidKeyword {
		t.Fatalf("box outline = width %.3f style %q", box.OutlineWidth, box.OutlineStyle)
	}
}

func TestRadiusLonghand(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="tl">x</div>
		<div class="tr">x</div>
		<div class="br">x</div>
		<div class="bl">x</div>
		<div class="pct">x</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.tl { border-top-left-radius: 8pt }
		.tr { border-top-right-radius: 6pt }
		.br { border-bottom-right-radius: 4pt }
		.bl { border-bottom-left-radius: 2pt }
		.pct { border-top-left-radius: 50% }
	`)}, "print", testViewport, 800)

	tl := styleByClass(t, styles, "tl")
	if !near(tl.BorderRadiusTopLeft, 8) {
		t.Fatalf("border-top-left-radius = %.3f, want 8pt", tl.BorderRadiusTopLeft)
	}

	if !near(tl.BorderRadiusPercent, -1) {
		t.Fatalf("absolute longhand should clear BorderRadiusPercent, got %.3f", tl.BorderRadiusPercent)
	}

	if !near(styleByClass(t, styles, "tr").BorderRadiusTopRight, 6) {
		t.Fatalf("border-top-right-radius = %.3f, want 6pt", styleByClass(t, styles, "tr").BorderRadiusTopRight)
	}

	if !near(styleByClass(t, styles, "br").BorderRadiusBottomRight, 4) {
		t.Fatalf("border-bottom-right-radius = %.3f, want 4pt", styleByClass(t, styles, "br").BorderRadiusBottomRight)
	}

	if !near(styleByClass(t, styles, "bl").BorderRadiusBottomLeft, 2) {
		t.Fatalf("border-bottom-left-radius = %.3f, want 2pt", styleByClass(t, styles, "bl").BorderRadiusBottomLeft)
	}

	pct := styleByClass(t, styles, "pct")
	if !near(pct.BorderRadiusPercent, 50) {
		t.Fatalf("border-top-left-radius:50%% BorderRadiusPercent = %.3f, want 50", pct.BorderRadiusPercent)
	}

	if !near(pct.BorderRadiusTopLeft, 50) {
		t.Fatalf("border-top-left-radius:50%% corner = %.3f, want 50", pct.BorderRadiusTopLeft)
	}
}

func TestBackgroundImageParse(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="dq">x</div>
		<div class="sq">x</div>
		<div class="bare">x</div>
		<div class="none">x</div>
		<div class="grad">x</div>
		<div class="sh">x</div>
		<div class="clear">x</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.dq { background-image: url("logo.png") }
		.sq { background-image: url('mark.jpg') }
		.bare { background-image: url(icon.svg) }
		.none { background-image: url("x.png"); }
		.none { background-image: none }
		.grad { background-image: linear-gradient(red, blue) }
		.sh { background: url("banner.png") red }
		.clear { background: none }
	`)}, "print", testViewport, 800)

	if got := styleByClass(t, styles, "dq").BackgroundImage; got != "logo.png" {
		t.Fatalf("double-quoted url = %q, want logo.png", got)
	}

	if got := styleByClass(t, styles, "sq").BackgroundImage; got != "mark.jpg" {
		t.Fatalf("single-quoted url = %q, want mark.jpg", got)
	}

	if got := styleByClass(t, styles, "bare").BackgroundImage; got != "icon.svg" {
		t.Fatalf("bare url = %q, want icon.svg", got)
	}

	if got := styleByClass(t, styles, "none").BackgroundImage; got != "" {
		t.Fatalf("background-image:none = %q, want empty", got)
	}

	if got := styleByClass(t, styles, "grad").BackgroundImage; got != "linear-gradient(red, blue)" {
		t.Fatalf("gradient background = %q, want linear-gradient(red, blue)", got)
	}

	sh := styleByClass(t, styles, "sh")
	if sh.BackgroundImage != "banner.png" {
		t.Fatalf("background shorthand url = %q, want banner.png", sh.BackgroundImage)
	}

	if sh.BGColor != ([4]float64{1, 0, 0, 1}) {
		t.Fatalf("background shorthand color = %v, want red", sh.BGColor)
	}

	if got := styleByClass(t, styles, "clear").BackgroundImage; got != "" {
		t.Fatalf("background:none image = %q, want empty", got)
	}
}

func TestListStylePositionParse(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<ul class="outer">
			<li class="in">a</li>
			<li class="out">b</li>
			<li class="sh">c</li>
			<li class="type">d</li>
		</ul>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.outer { list-style-position: inside }
		.in { list-style-position: inside }
		.out { list-style-position: outside }
		.sh { list-style: square inside }
		.type { list-style-type: decimal }
	`)}, "print", testViewport, 800)

	if got := styleByClass(t, styles, "in").ListStylePosition; got != listPosInside {
		t.Fatalf("list-style-position:inside = %q", got)
	}

	if got := styleByClass(t, styles, "out").ListStylePosition; got != listPosOutside {
		t.Fatalf("list-style-position:outside = %q", got)
	}

	sh := styleByClass(t, styles, "sh")
	if sh.ListStylePosition != listPosInside {
		t.Fatalf("list-style shorthand position = %q, want inside", sh.ListStylePosition)
	}

	if sh.ListStyleType != listStyleSquare {
		t.Fatalf("list-style shorthand type = %q, want square", sh.ListStyleType)
	}

	inherited := styleByClass(t, styles, "type")
	if inherited.ListStylePosition != listPosInside {
		t.Fatalf("list-style-position did not inherit (type-only child) = %q", inherited.ListStylePosition)
	}

	if inherited.ListStyleType != listStyleDecimal {
		t.Fatalf("list-style-type override = %q, want decimal", inherited.ListStyleType)
	}
}

func TestCounterResetIncrement(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="outer">
			<p class="child">x</p>
		</div>
		<div class="inc">y</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.outer { counter-reset: section 0 }
		.inc { counter-increment: section }
	`)}, "print", testViewport, 800)

	outer := styleByClass(t, styles, "outer")
	if outer.CounterReset != "section 0" {
		t.Fatalf("counter-reset = %q, want %q", outer.CounterReset, "section 0")
	}

	if outer.CounterIncrement != "" {
		t.Fatalf("outer CounterIncrement = %q, want empty", outer.CounterIncrement)
	}

	child := styleByClass(t, styles, "child")
	if child.CounterReset != "" {
		t.Fatalf("counter-reset inherited = %q, want empty", child.CounterReset)
	}

	if child.CounterIncrement != "" {
		t.Fatalf("counter-increment inherited = %q, want empty", child.CounterIncrement)
	}

	inc := styleByClass(t, styles, "inc")
	if inc.CounterIncrement != "section" {
		t.Fatalf("counter-increment = %q, want section", inc.CounterIncrement)
	}

	if inc.CounterReset != "" {
		t.Fatalf("inc CounterReset = %q, want empty", inc.CounterReset)
	}
}

func TestQuotesParse(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="q">
			<p class="child">x</p>
		</div>
		<div class="none">y</div>
		<div class="own">z</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.q { quotes: "«" "»" }
		.none { quotes: none }
		.own { quotes: '‹' '›' }
	`)}, "print", testViewport, 800)

	q := styleByClass(t, styles, "q")
	if q.QuotesOpen != "«" || q.QuotesClose != "»" {
		t.Fatalf("quotes = %q %q, want « »", q.QuotesOpen, q.QuotesClose)
	}

	child := styleByClass(t, styles, "child")
	if child.QuotesOpen != "«" || child.QuotesClose != "»" {
		t.Fatalf("quotes did not inherit: %q %q", child.QuotesOpen, child.QuotesClose)
	}

	none := styleByClass(t, styles, "none")
	if none.QuotesOpen != "" || none.QuotesClose != "" {
		t.Fatalf("quotes:none = %q %q, want empty", none.QuotesOpen, none.QuotesClose)
	}

	own := styleByClass(t, styles, "own")
	if own.QuotesOpen != "‹" || own.QuotesClose != "›" {
		t.Fatalf("single-quoted quotes = %q %q, want ‹ ›", own.QuotesOpen, own.QuotesClose)
	}
}

func TestOutlineDoesNotInherit(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="outer"><p class="child">x</p></div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.outer { outline: 3pt dashed blue }
	`)}, "print", testViewport, 800)

	child := styleByClass(t, styles, "child")
	if child.OutlineStyle != "" || child.OutlineWidth != 0 {
		t.Fatalf("outline inherited: style=%q width=%.3f", child.OutlineStyle, child.OutlineWidth)
	}
}
