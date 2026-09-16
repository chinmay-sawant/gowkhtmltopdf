package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

// The font shorthand resets style, weight, size, line-height, and family. A
// CSS-wide inherit must expand to all five longhands; otherwise the UA
// h2{font-weight:bold} survives an author `h2{font:inherit}` reset
// (gobyexample finding 1).
func TestFontShorthandInheritExpandsLonghands(t *testing.T) {
	t.Parallel()

	decls, ok := expandFontDeclaration("font", "inherit")
	if !ok {
		t.Fatal("font: inherit did not expand")
	}

	want := map[string]bool{
		"font-weight": false, "font-style": false, "font-size": false,
		"line-height": false, "font-family": false,
	}

	for _, decl := range decls {
		if _, tracked := want[decl.prop]; tracked {
			want[decl.prop] = true
		} else {
			t.Errorf("font: inherit emitted unexpected %s", decl.prop)

			continue
		}

		if decl.val != inheritKeyword {
			t.Errorf("%s = %q, want %q", decl.prop, decl.val, inheritKeyword)
		}
	}

	for prop, seen := range want {
		if !seen {
			t.Errorf("font: inherit did not emit %s", prop)
		}
	}
}

func TestFontShorthandInheritBeatsUABold(t *testing.T) {
	t.Parallel()

	// UA h2{font-weight:bold} must lose to the author reset even though the
	// inherited body weight is regular.
	root := mustParse(t, `<html><body><h2>x</h2></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `h2 { font: inherit }`)}, "print", testViewport, 800)

	h2Node := findElementByName(root, "h2")

	sty := styles[h2Node]
	if sty == nil {
		t.Fatal("h2 has no resolved style")
	}

	if sty.FontWeight != fontWeightNormalValue {
		t.Errorf("h2 weight = %d, want inherited %d (UA bold must not survive)", sty.FontWeight, fontWeightNormalValue)
	}

	// Inherited style, size, and line-height still flow through.
	root = mustParse(t,
		`<html><body style="font-weight:700;font-style:italic;font-size:20px;line-height:1.5"><h2>x</h2></body></html>`)
	styles = resolveStyles(root, []*css.Stylesheet{sheet(t, `h2 { font: inherit }`)}, "print", testViewport, 800)

	h2Node = findElementByName(root, "h2")

	sty = styles[h2Node]
	if sty == nil {
		t.Fatal("h2 has no resolved style")
	}

	if sty.FontWeight != fontWeightBoldValue {
		t.Errorf("h2 weight = %d, want inherited %d", sty.FontWeight, fontWeightBoldValue)
	}

	if !sty.FontItalic {
		t.Error("h2 font-style did not inherit italic")
	}

	if !near(sty.FontSize, 15) { // 20px at 96dpi
		t.Errorf("h2 size = %v, want inherited 15pt", sty.FontSize)
	}

	if !near(sty.LineHeightUnitless, 1.5) {
		t.Errorf("h2 line-height unitless = %v, want inherited 1.5", sty.LineHeightUnitless)
	}
}

// Other font forms must keep expanding as before.
func TestFontShorthandFormsStillExpand(t *testing.T) {
	t.Parallel()

	decls, ok := expandFontDeclaration("font", "italic bold 12px/1.4 Arial, sans-serif")
	if !ok {
		t.Fatal("font italic bold form did not expand")
	}

	got := map[string]string{}
	for _, d := range decls {
		got[d.prop] = d.val
	}

	if got["font-style"] != "italic" || got["font-weight"] != "bold" ||
		got["font-size"] != "12px" || got["line-height"] != "1.4" ||
		got["font-family"] != "Arial, sans-serif" {
		t.Fatalf("font shorthand expansion = %#v", got)
	}
}
