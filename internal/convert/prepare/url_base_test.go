package prepare_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/prepare"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf/assets"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// origin is one test HTTP origin serving fixed bodies by path plus a hit
// count per requested path.
type origin struct {
	url  string
	hits map[string]int
}

func newOrigin(t *testing.T, files map[string]string) origin {
	t.Helper()

	hits := map[string]int{}

	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		hits[request.URL.Path]++

		body, ok := files[request.URL.Path]
		if !ok {
			http.NotFound(responseWriter, request)

			return
		}

		responseWriter.Header().Set("Content-Type", contentTypeFor(request.URL.Path))
		_, _ = responseWriter.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return origin{url: server.URL, hits: hits}
}

func contentTypeFor(path string) string {
	switch {
	case strings.HasSuffix(path, ".css"):
		return "text/css"
	case strings.HasSuffix(path, ".ttf"):
		return "font/ttf"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	default:
		return "text/html"
	}
}

// collectSheetsFrom parses docHTML, builds a resource context rooted at
// documentURL, and collects its stylesheets with print media.
func collectSheetsFrom(t *testing.T, docHTML, documentURL string) ([]*css.Stylesheet, prepare.ResourceContext) {
	t.Helper()

	root, err := html.Parse(docHTML)
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}

	loader, err := load.NewLoaderWithError(settings.LoadGlobal{})
	if err != nil {
		t.Fatalf("new loader: %v", err)
	}

	resources := prepare.NewResourceContext(loader, documentURL, settings.DefaultLoadPage())
	sheets := resources.CollectSheets(
		t.Context(),
		root,
		prepare.SheetOptions{ViewportW: 600, ViewportH: 800, MediaType: "print"},
		io.Discard,
	)

	return sheets, resources
}

// declValue returns the first declaration value for prop across sheets.
func declValue(t *testing.T, sheets []*css.Stylesheet, prop string) string {
	t.Helper()

	for _, sheet := range sheets {
		for _, rule := range sheet.Rules {
			for _, decl := range rule.Decls {
				if decl.Prop == prop {
					return decl.Value
				}
			}
		}
	}

	t.Fatalf("no %q declaration in %d collected sheets", prop, len(sheets))

	return ""
}

// sheetBases returns the recorded source base of each collected sheet.
func sheetBases(sheets []*css.Stylesheet) []string {
	out := make([]string, 0, len(sheets))

	for _, sheet := range sheets {
		out = append(out, sheet.Base)
	}

	return out
}

// TestFontFaceResolvesAgainstSheetBaseAcrossOrigins proves the learncpp
// dashicons case: a document on origin A links a stylesheet on origin B whose
// @font-face uses url(../fonts/...). The font must be fetched from B's
// sheet-relative path, never from A's document-relative path.
func TestFontFaceResolvesAgainstSheetBaseAcrossOrigins(t *testing.T) {
	t.Parallel()

	ttf := assets.LiberationSansRegular()

	docOrigin := newOrigin(t, map[string]string{"/fonts/dashicons.ttf": "decoy"})
	sheetOrigin := newOrigin(t, map[string]string{
		"/blog/wp-includes/css/dashicons.min.css": "@font-face { font-family: Dashicons; " +
			"src: url(../fonts/dashicons.ttf?v=1) }",
		"/blog/wp-includes/fonts/dashicons.ttf": string(ttf),
	})

	docHTML := `<html><head><link rel="stylesheet" href="` +
		sheetOrigin.url + `/blog/wp-includes/css/dashicons.min.css"></head><body></body></html>`

	sheets, resources := collectSheetsFrom(t, docHTML, docOrigin.url+"/index.html")

	sheetURL := sheetOrigin.url + "/blog/wp-includes/css/dashicons.min.css"
	if len(sheets) != 1 || sheets[0].Base != sheetURL {
		t.Fatalf("sheets = %d, Base = %q, want one sheet with Base %q", len(sheets), sheets[0].Base, sheetURL)
	}

	registry := resources.MergeFontFaces(t.Context(), nil, sheets, 1, io.Discard)
	if registry == nil || registry.Lookup([]string{"Dashicons"}, 400, false) == nil {
		t.Fatalf("Dashicons not registered; doc hits = %v, sheet hits = %v", docOrigin.hits, sheetOrigin.hits)
	}

	if sheetOrigin.hits["/blog/wp-includes/fonts/dashicons.ttf"] != 1 {
		t.Fatalf("sheet-relative font not fetched; sheet hits = %v", sheetOrigin.hits)
	}

	if len(docOrigin.hits) != 0 {
		t.Fatalf("document-relative font fetched; doc hits = %v", docOrigin.hits)
	}
}

// TestCollectSheetsAbsolutizesURLValuesAgainstSheetBase proves the same base
// rule for the paint-time url() consumers: background-image,
// list-style-image, and border-image-source values collected from the sheet
// carry the sheet-relative absolute URL, and that URL fetches the sheet's
// resource rather than the document's decoy.
func TestCollectSheetsAbsolutizesURLValuesAgainstSheetBase(t *testing.T) {
	t.Parallel()

	docOrigin := newOrigin(t, map[string]string{
		"/img/marker.png":   "doc marker",
		"/img/border.png":   "doc border",
		"/icons/bullet.png": "doc bullet",
	})
	sheetOrigin := newOrigin(t, map[string]string{
		"/blog/wp-includes/css/site.css": `.marker { background-image: url(../img/marker.png) }
.list { list-style-image: url(../icons/bullet.png) }
.border { border-image-source: url(../img/border.png) }`,
		"/blog/wp-includes/img/marker.png": "sheet marker",
	})

	docHTML := `<html><head><link rel="stylesheet" href="` +
		sheetOrigin.url + `/blog/wp-includes/css/site.css"></head><body></body></html>`

	sheets, resources := collectSheetsFrom(t, docHTML, docOrigin.url+"/index.html")

	imgBase := sheetOrigin.url + "/blog/wp-includes"

	if got := sheets[0].Base; got != sheetOrigin.url+"/blog/wp-includes/css/site.css" {
		t.Fatalf("sheet Base = %q, want the sheet URL", got)
	}

	wantDecls := map[string]string{
		"background-image":    `url("` + imgBase + `/img/marker.png")`,
		"list-style-image":    `url("` + imgBase + `/icons/bullet.png")`,
		"border-image-source": `url("` + imgBase + `/img/border.png")`,
	}

	for prop, want := range wantDecls {
		if got := declValue(t, sheets, prop); got != want {
			t.Errorf("%s = %q, want %q", prop, got, want)
		}
	}

	value := declValue(t, sheets, "background-image")

	urls := css.FontFaceURLs(value)
	if len(urls) != 1 {
		t.Fatalf("background-image urls = %v, want one absolute url", urls)
	}

	if _, err := resources.Fetch(t.Context(), urls[0]); err != nil {
		t.Fatalf("fetch resolved background image: %v", err)
	}

	if sheetOrigin.hits["/blog/wp-includes/img/marker.png"] != 1 {
		t.Fatalf("sheet-relative image not fetched; sheet hits = %v", sheetOrigin.hits)
	}

	if len(docOrigin.hits) != 0 {
		t.Fatalf("document-relative image or decoy fetched; doc hits = %v", docOrigin.hits)
	}
}

// TestCollectSheetsIgnoreBaseHref documents the contract chosen for <base
// href>: it stays ignored. A linked sheet resolves against its own URL and an
// inline <style> against the document URL, never against the <base> element.
func TestCollectSheetsIgnoreBaseHref(t *testing.T) {
	t.Parallel()

	ttf := assets.LiberationSansRegular()

	docOrigin := newOrigin(t, map[string]string{
		"/css/site.css":     `@font-face { font-family: Linked; src: url(../fonts/linked.ttf) }`,
		"/fonts/linked.ttf": string(ttf),
		"/fonts/inline.ttf": string(ttf),
	})
	baseOrigin := newOrigin(t, map[string]string{
		"/fonts/linked.ttf": string(ttf),
		"/fonts/inline.ttf": string(ttf),
	})

	docHTML := `<html><head><base href="` + baseOrigin.url + `/"><link rel="stylesheet" href="css/site.css">` +
		`<style>@font-face { font-family: Inline; src: url(fonts/inline.ttf) }</style></head><body></body></html>`

	sheets, resources := collectSheetsFrom(t, docHTML, docOrigin.url+"/index.html")

	if len(sheets) != 2 ||
		sheets[0].Base != docOrigin.url+"/css/site.css" ||
		sheets[1].Base != docOrigin.url+"/index.html" {
		t.Fatalf("sheet bases = %v, want the linked sheet URL then the document URL", sheetBases(sheets))
	}

	registry := resources.MergeFontFaces(t.Context(), nil, sheets, 1, io.Discard)
	if registry == nil || registry.Lookup([]string{"Linked"}, 400, false) == nil {
		t.Fatalf("linked @font-face not registered; doc hits = %v", docOrigin.hits)
	}

	if registry.Lookup([]string{"Inline"}, 400, false) == nil {
		t.Fatalf("inline @font-face not registered; doc hits = %v", docOrigin.hits)
	}

	if docOrigin.hits["/fonts/linked.ttf"] != 1 || docOrigin.hits["/fonts/inline.ttf"] != 1 {
		t.Fatalf("font hits = %v, want sheet base and document base on the doc origin", docOrigin.hits)
	}

	if len(baseOrigin.hits) != 0 {
		t.Fatalf("<base href> was honored; base-origin hits = %v", baseOrigin.hits)
	}
}

// TestImportedSheetURLsResolveAgainstImportedBase proves an @import chain
// keeps each sheet's own base for its url() values.
func TestImportedSheetURLsResolveAgainstImportedBase(t *testing.T) {
	t.Parallel()

	origin := newOrigin(t, map[string]string{
		"/css/a.css":     `@import url("sub/b.css");`,
		"/css/sub/b.css": `.nested { background-image: url(../img/nested.png) }`,
	})

	docHTML := `<html><head><link rel="stylesheet" href="css/a.css"></head><body></body></html>`
	sheets, _ := collectSheetsFrom(t, docHTML, origin.url+"/index.html")

	want := `url("` + origin.url + `/css/img/nested.png")`
	if got := declValue(t, sheets, "background-image"); got != want {
		t.Fatalf("imported sheet background-image = %q, want %q", got, want)
	}
}
