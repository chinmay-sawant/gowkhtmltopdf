//nolint:testpackage,wsl,varnamelen,cyclop // cascade apply proofs
package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func TestWordSpacingInherits(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div class="outer"><p class="inner">hello world</p></div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.outer { word-spacing: 3pt }
		.own { word-spacing: 1pt }
	`)}, "print", testViewport, 800)

	inner := styleByClass(t, styles, "inner")
	if !near(inner.WordSpacing, 3) {
		t.Fatalf("inner WordSpacing = %v, want 3 (inherited)", inner.WordSpacing)
	}

	ownRoot := mustParse(t, `<html><body><div class="outer"><p class="own">x</p></div></body></html>`)
	ownStyles := resolveStyles(ownRoot, []*css.Stylesheet{sheet(t, `
		.outer { word-spacing: 3pt }
		.own { word-spacing: 1pt }
	`)}, "print", testViewport, 800)

	own := styleByClass(t, ownStyles, "own")
	if !near(own.WordSpacing, 1) {
		t.Fatalf("own WordSpacing = %v, want 1", own.WordSpacing)
	}
}

func TestWordSpacingWidensRuns(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `p { margin: 0; font-size: 12pt; white-space: nowrap } .wide { word-spacing: 4pt }`)
	plain := layoutHTML(t, `<html><body><p>hello world</p></body></html>`, cssSheet)
	wide := layoutHTML(t, `<html><body><p class="wide">hello world</p></body></html>`, cssSheet)

	plainW := textRunWidth(plain, "hello")
	wideW := textRunWidth(wide, "hello")
	if wideW-plainW < 3.5 {
		t.Fatalf("word-spacing did not widen runs: plain=%.3f wide=%.3f", plainW, wideW)
	}
}

func TestVisibilityHidden(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
		p { margin: 0 }
		.hidden { visibility: hidden; height: 40pt }
		.collapse { visibility: collapse; height: 40pt }
		.gone { display: none; height: 40pt }
	`)
	hidden := layoutHTML(t, `<html><body>
		<p>shown</p>
		<p class="hidden">secret<span>nested</span></p>
		<p>after</p>
	</body></html>`, cssSheet)

	if got := joinedText(hidden); strings.Contains(got, "secret") || strings.Contains(got, "nested") {
		t.Fatalf("hidden subtree painted %q", got)
	}

	if !strings.Contains(joinedText(hidden), "shown") || !strings.Contains(joinedText(hidden), "after") {
		t.Fatalf("visible siblings missing from %q", joinedText(hidden))
	}

	gone := layoutHTML(t, `<html><body>
		<p>shown</p>
		<p class="gone">secret<span>nested</span></p>
		<p>after</p>
	</body></html>`, cssSheet)

	hiddenAfter := textY(t, hidden, "after")
	goneAfter := textY(t, gone, "after")
	if hiddenAfter-goneAfter < 30 {
		t.Fatalf("hidden box did not keep layout size: hidden after Y=%.2f display-none after Y=%.2f",
			hiddenAfter, goneAfter)
	}

	collapsed := layoutHTML(t, `<html><body><p class="collapse">gone</p><p>after</p></body></html>`, cssSheet)
	if strings.Contains(joinedText(collapsed), "gone") {
		t.Fatal("visibility:collapse painted text on a non-table")
	}
}

func TestWhiteSpacePreWrap(t *testing.T) {
	t.Parallel()

	wrapSheet := sheet(t, `pre { white-space: pre-wrap; width: 40pt; margin: 0; font-size: 12pt }`)
	wrapped := layoutHTML(t, "<html><body><pre>aaaa bbbb cccc dddd\nZ</pre></body></html>", wrapSheet)
	texts := opsOfKind(wrapped, OpText)

	if len(texts) < 3 {
		t.Fatalf("pre-wrap should wrap and keep the newline, got %+v", texts)
	}

	var sawZ bool

	for _, op := range texts {
		if strings.TrimSpace(op.Text) == "Z" {
			sawZ = true

			if op.Y <= texts[0].Y {
				t.Fatalf("pre-wrap newline did not start a new line: Z y=%.2f first y=%.2f", op.Y, texts[0].Y)
			}
		}
	}

	if !sawZ {
		t.Fatalf("pre-wrap dropped the preserved newline run, texts=%+v", texts)
	}

	lineSheet := sheet(t, `div { white-space: pre-line; width: 200pt; margin: 0 }`)
	lined := layoutHTML(t, "<html><body><div>a    b\nc</div></body></html>", lineSheet)
	lineTexts := opsOfKind(lined, OpText)

	joined := joinedText(lined)
	if strings.Contains(joined, "    ") {
		t.Fatalf("pre-line should collapse spaces, got %q", joined)
	}

	if len(lineTexts) < two {
		t.Fatalf("pre-line should keep the newline, got %+v", lineTexts)
	}

	if !strings.Contains(joined, "a") || !strings.Contains(joined, "b") || !strings.Contains(joined, "c") {
		t.Fatalf("pre-line dropped content: %q", joined)
	}
}

func TestLogicalMargin(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="sh">x</div>
		<div class="start">x</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.sh { margin-inline: 10pt 20pt; margin-block: 5pt }
		.start { margin-inline-start: 8pt; margin-block-end: 3pt }
	`)}, "print", testViewport, 800)

	sh := styleByClass(t, styles, "sh")
	if !near(sh.MarginLeft, 10) || !near(sh.MarginRight, 20) {
		t.Fatalf("margin-inline = L %.2f R %.2f, want 10/20", sh.MarginLeft, sh.MarginRight)
	}

	if !near(sh.MarginTop, 5) || !near(sh.MarginBottom, 5) {
		t.Fatalf("margin-block = T %.2f B %.2f, want 5/5", sh.MarginTop, sh.MarginBottom)
	}

	start := styleByClass(t, styles, "start")
	if !near(start.MarginLeft, 8) {
		t.Fatalf("margin-inline-start = %.2f, want 8", start.MarginLeft)
	}

	if !near(start.MarginBottom, 3) {
		t.Fatalf("margin-block-end = %.2f, want 3", start.MarginBottom)
	}
}

func TestLogicalPadding(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div class="box">x</div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.box { padding-inline: 4pt 6pt; padding-block-start: 2pt; padding-block-end: 3pt }
	`)}, "print", testViewport, 800)
	box := styleByClass(t, styles, "box")

	if !near(box.PaddingLeft, 4) || !near(box.PaddingRight, 6) {
		t.Fatalf("padding-inline = L %.2f R %.2f, want 4/6", box.PaddingLeft, box.PaddingRight)
	}

	if !near(box.PaddingTop, 2) || !near(box.PaddingBottom, 3) {
		t.Fatalf("padding-block = T %.2f B %.2f, want 2/3", box.PaddingTop, box.PaddingBottom)
	}
}

func TestLogicalInset(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="all">x</div>
		<div class="pair">x</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.all { inset: 10pt 20pt }
		.pair { inset-block: 4pt; inset-inline-start: 7pt }
	`)}, "print", testViewport, 800)

	all := styleByClass(t, styles, "all")
	if all.TopAuto || all.RightAuto || all.BottomAuto || all.LeftAuto {
		t.Fatal("inset should clear auto offsets")
	}

	if !near(all.Top, 10) || !near(all.Bottom, 10) || !near(all.Left, 20) || !near(all.Right, 20) {
		t.Fatalf("inset = T %.2f R %.2f B %.2f L %.2f, want 10/20/10/20",
			all.Top, all.Right, all.Bottom, all.Left)
	}

	pair := styleByClass(t, styles, "pair")
	if !near(pair.Top, 4) || !near(pair.Bottom, 4) || !near(pair.Left, 7) {
		t.Fatalf("inset-block/inline-start = T %.2f B %.2f L %.2f, want 4/4/7",
			pair.Top, pair.Bottom, pair.Left)
	}
}

func TestLogicalSize(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="size">x</div>
		<div class="minmax">x</div>
		<div class="minmaxblock">x</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.size { inline-size: 100pt; block-size: 50pt }
		.minmax { min-inline-size: 40pt; max-inline-size: 80pt }
		.minmaxblock { min-block-size: 30pt; max-block-size: 60pt }
	`)}, "print", testViewport, 800)

	size := styleByClass(t, styles, "size")
	if !near(size.Width, 100) || !near(size.Height, 50) {
		t.Fatalf("logical size = %.2fx%.2f, want 100x50", size.Width, size.Height)
	}

	minmax := styleByClass(t, styles, "minmax")
	if !near(minmax.MinWidth, 40) || !near(minmax.MaxWidth, 80) {
		t.Fatalf("min/max-inline-size = %.2f/%.2f, want 40/80", minmax.MinWidth, minmax.MaxWidth)
	}

	minmaxblock := styleByClass(t, styles, "minmaxblock")
	if !near(minmaxblock.MinHeight, 30) || !near(minmaxblock.MaxHeight, 60) {
		t.Fatalf("min/max-block-size = %.2f/%.2f, want 30/60", minmaxblock.MinHeight, minmaxblock.MaxHeight)
	}
}

func TestCaptionSideParse(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<table class="bottom"><caption class="cap">title</caption><tr><td>1</td></tr></table>
		<table class="plain"><caption>plain</caption><tr><td>2</td></tr></table>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.bottom { caption-side: bottom }
	`)}, "print", testViewport, 800)

	table := styleByClass(t, styles, "bottom")
	if table.CaptionSide != cssVerticalAlignBottom {
		t.Fatalf("table CaptionSide = %q, want bottom", table.CaptionSide)
	}

	captionSty := styleByClass(t, styles, "cap")
	if captionSty.CaptionSide != cssVerticalAlignBottom {
		t.Fatalf("caption did not inherit CaptionSide, got %q", captionSty.CaptionSide)
	}

	plain := styleByClass(t, styles, "plain")
	if plain.CaptionSide != "" {
		t.Fatalf("default CaptionSide = %q, want empty (top)", plain.CaptionSide)
	}

	topRoot := mustParse(t, `<html><body><table class="top"><tr><td>x</td></tr></table></body></html>`)
	topStyles := resolveStyles(topRoot, []*css.Stylesheet{sheet(t, `.top { caption-side: top }`)},
		"print", testViewport, 800)
	if got := styleByClass(t, topStyles, "top").CaptionSide; got != cssVerticalAlignTop {
		t.Fatalf("caption-side:top = %q", got)
	}

	sideRoot := mustParse(t, `<html><body>
		<table class="left"><tr><td>x</td></tr></table>
		<table class="right"><tr><td>y</td></tr></table>
	</body></html>`)
	sideStyles := resolveStyles(sideRoot, []*css.Stylesheet{sheet(t, `
		.left { caption-side: left }
		.right { caption-side: right }
	`)}, "print", testViewport, 800)
	if got := styleByClass(t, sideStyles, "left").CaptionSide; got != floatLeft {
		t.Fatalf("caption-side:left = %q", got)
	}

	if got := styleByClass(t, sideStyles, "right").CaptionSide; got != floatRight {
		t.Fatalf("caption-side:right = %q", got)
	}
}

func styleByClass(t *testing.T, styles map[*html.Node]*ResolvedStyle, class string) *ResolvedStyle {
	t.Helper()

	for node, sty := range styles {
		if node.Type == html.ElementNode && node.Attribute("class") == class {
			return sty
		}
	}

	t.Fatalf("no styled element with class %q", class)

	return nil
}

func TestFontShorthand(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="sh">x</div>
		<div class="full">y</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.sh { font: 16pt/1.5 Helvetica, Arial, sans-serif }
		.full { font: italic bold 14pt/20pt serif }
	`)}, "print", testViewport, 800)

	sh := styleByClass(t, styles, "sh")
	if !near(sh.FontSize, 16) {
		t.Fatalf("font shorthand size = %.2f, want 16", sh.FontSize)
	}

	if !near(sh.LineHeight, 24) {
		t.Fatalf("font shorthand line-height = %.2f, want 24 (16*1.5)", sh.LineHeight)
	}

	if len(sh.FontFamily) == 0 || sh.FontFamily[0] != "Helvetica" {
		t.Fatalf("font shorthand family = %v, want Helvetica first", sh.FontFamily)
	}

	full := styleByClass(t, styles, "full")
	if !full.FontItalic {
		t.Fatal("font shorthand italic not set")
	}

	if full.FontWeight < 700 {
		t.Fatalf("font shorthand weight = %d, want >= 700", full.FontWeight)
	}

	if !near(full.FontSize, 14) {
		t.Fatalf("font shorthand size = %.2f, want 14", full.FontSize)
	}

	if !near(full.LineHeight, 20) {
		t.Fatalf("font shorthand line-height = %.2f, want 20", full.LineHeight)
	}
}

func TestOverflowAxesIndependent(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="x-hidden">x</div>
		<div class="y-scroll">x</div>
		<div class="split">x</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.x-hidden { overflow-x: hidden; }
		.y-scroll { overflow-y: scroll; }
		.split { overflow-x: hidden; overflow-y: visible; }
	`)}, "print", testViewport, 800)

	xHidden := styleByClass(t, styles, "x-hidden")
	if xHidden.OverflowX != overflowHidden || xHidden.OverflowY != visibleKeyword {
		t.Fatalf("x-hidden: OverflowX=%q OverflowY=%q, want hidden/visible", xHidden.OverflowX, xHidden.OverflowY)
	}

	yScroll := styleByClass(t, styles, "y-scroll")
	if yScroll.OverflowX != visibleKeyword || yScroll.OverflowY != overflowScroll {
		t.Fatalf("y-scroll: OverflowX=%q OverflowY=%q, want visible/scroll", yScroll.OverflowX, yScroll.OverflowY)
	}

	split := styleByClass(t, styles, "split")
	if split.OverflowX != overflowHidden || split.OverflowY != visibleKeyword {
		t.Fatalf("split: OverflowX=%q OverflowY=%q, want hidden/visible", split.OverflowX, split.OverflowY)
	}
}

func joinedText(res *Result) string {
	var b strings.Builder

	for _, op := range res.Ops {
		if op.Kind == OpText {
			b.WriteString(op.Text)
		}
	}

	return b.String()
}

func textY(t *testing.T, res *Result, needle string) float64 {
	t.Helper()

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, needle) {
			return op.Y
		}
	}

	t.Fatalf("no text op containing %q", needle)

	return 0
}

func textRunWidth(res *Result, needle string) float64 {
	var total float64

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, needle) {
			total += op.W
		}
	}

	return total
}
