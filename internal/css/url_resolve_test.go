package css

import (
	"testing"
)

// sheetURL mirrors the learncpp dashicons case: a stylesheet two directories
// below the host root whose url(../...) references must climb out of css/.
const sheetURL = "https://cdn.example.com/blog/wp-includes/css/dashicons.min.css"

func TestResolveURLsRewritesRelativeRefs(t *testing.T) {
	t.Parallel()

	sheet := mustSheet(t, `
		.a { background-image: url("../img/marker.png") }
		.b { background: url(../img/bg.png) no-repeat }
		.c { list-style-image: url('icons/bullet.png') }
		.d { border-image-source: url(../img/border.png) }
		.e { background-image: url(a.png), url(b.png) }
		@font-face {
			font-family: Dashicons;
			src: local("Dashicons"), url(../fonts/dashicons.eot?v=1) format("embedded-opentype"),
				url(../fonts/dashicons.ttf?v=1) format("truetype");
		}
		p { color: red }
	`)
	sheet.ResolveURLs(sheetURL)

	if sheet.Base != sheetURL {
		t.Fatalf("Base = %q, want %q", sheet.Base, sheetURL)
	}

	const (
		root     = "https://cdn.example.com/blog/wp-includes/"
		cssDir   = root + "css/"
		wantURL1 = root + "img/marker.png"
		wantURL2 = root + "img/bg.png"
		wantURL3 = cssDir + "icons/bullet.png"
		wantURL4 = root + "img/border.png"
		wantURL5 = cssDir + "a.png"
		wantURL6 = cssDir + "b.png"
	)

	wantDecls := []struct{ prop, value string }{
		{"background-image", `url("` + wantURL1 + `")`},
		{"background", `url("` + wantURL2 + `") no-repeat`},
		{"list-style-image", `url("` + wantURL3 + `")`},
		{"border-image-source", `url("` + wantURL4 + `")`},
		{"background-image", `url("` + wantURL5 + `"), url("` + wantURL6 + `")`},
	}

	if len(sheet.Rules) != len(wantDecls)+1 {
		t.Fatalf("rules = %d, want %d", len(sheet.Rules), len(wantDecls)+1)
	}

	for i, want := range wantDecls {
		decl := sheet.Rules[i].Decls[0]
		if decl.Prop != want.prop || decl.Value != want.value {
			t.Errorf("decl %d = %s: %s, want %s: %s", i, decl.Prop, decl.Value, want.prop, want.value)
		}
	}

	fontSrc := sheet.FontFaces[0].Src
	if got := FontFaceURLs(fontSrc); len(got) != 2 ||
		got[0] != root+"fonts/dashicons.eot?v=1" || got[1] != root+"fonts/dashicons.ttf?v=1" {
		t.Errorf("font-face urls = %v, want sheet-relative absolute urls", got)
	}
}

func TestResolveURLsKeepsBaseIndependentRefs(t *testing.T) {
	t.Parallel()

	sheet := mustSheet(t, `
		.a { background-image: url("data:image/png;base64,AAAA") }
		.b { filter: url(#blur) }
		.c { background-image: url(https://other.example/x.png) }
		.d { background-image: url("") }
		.e { content: "url(x.png)" }
	`)
	sheet.ResolveURLs(sheetURL)

	want := []string{
		`url("data:image/png;base64,AAAA")`,
		`url(#blur)`,
		`url(https://other.example/x.png)`,
		`url("")`,
		`"url(x.png)"`,
	}

	if len(sheet.Rules) != len(want) {
		t.Fatalf("rules = %d, want %d", len(sheet.Rules), len(want))
	}

	for i, wantValue := range want {
		if got := sheet.Rules[i].Decls[0].Value; got != wantValue {
			t.Errorf("decl %d = %q, want %q", i, got, wantValue)
		}
	}
}

func TestResolveURLsProtocolRelativeGetsBaseScheme(t *testing.T) {
	t.Parallel()

	sheet := mustSheet(t, `.x { src: url(//cdn.example.com/font.woff) }`)
	sheet.ResolveURLs(sheetURL)

	want := `url("https://cdn.example.com/font.woff")`
	if got := sheet.Rules[0].Decls[0].Value; got != want {
		t.Fatalf("value = %q, want %q", got, want)
	}
}

func TestResolveURLsMarkerCaseInsensitive(t *testing.T) {
	t.Parallel()

	sheet := mustSheet(t, `.x { background-image: URL(../img/upper.png) }`)
	sheet.ResolveURLs(sheetURL)

	want := `url("https://cdn.example.com/blog/wp-includes/img/upper.png")`
	if got := sheet.Rules[0].Decls[0].Value; got != want {
		t.Fatalf("value = %q, want %q", got, want)
	}
}

func TestResolveURLsLeavesImportsRaw(t *testing.T) {
	t.Parallel()

	sheet := mustSheet(t, `@import url("child.css") print; p { color: red }`)
	sheet.ResolveURLs(sheetURL)

	if len(sheet.Imports) != 1 || sheet.Imports[0].URL != "child.css" || sheet.Imports[0].Media != "print" {
		t.Fatalf("imports = %+v, want the raw child.css/print pair", sheet.Imports)
	}
}

func TestResolveURLsUnusableBaseIsNoop(t *testing.T) {
	t.Parallel()

	for _, base := range []string{"", "css/site.css", "data:text/css,"} {
		sheet := mustSheet(t, `.x { background-image: url(img/a.png) }`)
		sheet.ResolveURLs(base)

		if sheet.Base != base {
			t.Errorf("Base = %q, want %q", sheet.Base, base)
		}

		if got := sheet.Rules[0].Decls[0].Value; got != "url(img/a.png)" {
			t.Errorf("base %q changed the value to %q", base, got)
		}
	}
}
