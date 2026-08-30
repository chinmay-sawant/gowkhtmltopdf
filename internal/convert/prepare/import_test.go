package prepare_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/prepare"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

func TestCollectSheetsImportAppliesImportedRules(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"/a.css": `@import url("b.css");
.from-a { color: #f00 }`,
		"/b.css": `.from-b { color: #00f }`,
	}
	sheets, _, hits := collectImportSheets(t, importFixture{
		html:  `<html><head><link rel="stylesheet" href="a.css"></head><body></body></html>`,
		files: files,
		media: "print",
	})

	if hits["/a.css"] != 1 || hits["/b.css"] != 1 {
		t.Fatalf("fetch hits = %v, want a.css and b.css once", hits)
	}

	assertSheetClasses(t, sheets, []string{"from-b", "from-a"})
}

func TestCollectSheetsImportFromInlineStyle(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"/b.css": `.from-b { color: #00f }`,
	}
	sheets, _, hits := collectImportSheets(t, importFixture{
		html: `<html><head><style>@import url("b.css"); .from-style { color: #f00 }</style></head>` +
			`<body></body></html>`,
		files: files,
		media: "print",
	})

	if hits["/b.css"] != 1 {
		t.Fatalf("fetch hits = %v, want b.css once", hits)
	}

	assertSheetClasses(t, sheets, []string{"from-b", "from-style"})
}

func TestCollectSheetsImportResolvesAgainstSheetBase(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"/css/a.css": `@import url("b.css");
.from-a { color: #f00 }`,
		"/css/b.css": `.from-b { color: #00f }`,
		"/b.css":     `.wrong { color: #0f0 }`,
	}
	sheets, _, hits := collectImportSheets(t, importFixture{
		html:  `<html><head><link rel="stylesheet" href="css/a.css"></head><body></body></html>`,
		files: files,
		media: "print",
	})

	if hits["/css/b.css"] != 1 {
		t.Fatalf("fetch hits = %v, want /css/b.css once", hits)
	}

	if hits["/b.css"] != 0 {
		t.Fatalf("resolved import against document base; hits = %v", hits)
	}

	assertSheetClasses(t, sheets, []string{"from-b", "from-a"})
}

func TestCollectSheetsImportNestedOrder(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"/a.css": `@import url("b.css");
.from-a { color: #f00 }`,
		"/b.css": `@import url("c.css");
.from-b { color: #00f }`,
		"/c.css": `.from-c { color: #0f0 }`,
	}
	sheets, _, _ := collectImportSheets(t, importFixture{
		html:  `<html><head><link rel="stylesheet" href="a.css"></head><body></body></html>`,
		files: files,
		media: "print",
	})
	assertSheetClasses(t, sheets, []string{"from-c", "from-b", "from-a"})
}

func TestCollectSheetsImportSkipsCycle(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"/a.css": `@import url("b.css");
.from-a { color: #f00 }`,
		"/b.css": `@import url("a.css");
.from-b { color: #00f }`,
	}
	sheets, _, hits := collectImportSheets(t, importFixture{
		html:  `<html><head><link rel="stylesheet" href="a.css"></head><body></body></html>`,
		files: files,
		media: "print",
	})

	if hits["/a.css"] != 1 || hits["/b.css"] != 1 {
		t.Fatalf("cycle re-fetched; hits = %v", hits)
	}

	assertSheetClasses(t, sheets, []string{"from-b", "from-a"})
}

func TestCollectSheetsImportFailedFetchDoesNotFail(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"/a.css": `@import url("missing.css");
.from-a { color: #f00 }`,
	}
	sheets, logBuf, hits := collectImportSheets(t, importFixture{
		html:  `<html><head><link rel="stylesheet" href="a.css"></head><body></body></html>`,
		files: files,
		media: "print",
	})

	if hits["/missing.css"] != 1 {
		t.Fatalf("missing import not fetched; hits = %v", hits)
	}

	assertSheetClasses(t, sheets, []string{"from-a"})

	if !strings.Contains(logBuf.String(), `skipping @import "missing.css"`) {
		t.Fatalf("log = %q, want skipping @import warning", logBuf.String())
	}
}

func TestCollectSheetsImportMediaGate(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"/a.css": `@import url("print.css") print;
@import url("screen.css") screen;
.from-a { color: #f00 }`,
		"/print.css":  `.from-print { color: #00f }`,
		"/screen.css": `.from-screen { color: #0f0 }`,
	}

	printSheets, _, printHits := collectImportSheets(t, importFixture{
		html:  `<html><head><link rel="stylesheet" href="a.css"></head><body></body></html>`,
		files: files,
		media: "print",
	})
	if printHits["/print.css"] != 1 || printHits["/screen.css"] != 0 {
		t.Fatalf("print media hits = %v", printHits)
	}

	assertSheetClasses(t, printSheets, []string{"from-print", "from-a"})

	screenSheets, _, screenHits := collectImportSheets(t, importFixture{
		html:  `<html><head><link rel="stylesheet" href="a.css"></head><body></body></html>`,
		files: files,
		media: "screen",
	})
	if screenHits["/screen.css"] != 1 || screenHits["/print.css"] != 0 {
		t.Fatalf("screen media hits = %v", screenHits)
	}

	assertSheetClasses(t, screenSheets, []string{"from-screen", "from-a"})
}

func TestCollectSheetsImportDepthCap(t *testing.T) {
	t.Parallel()

	files := map[string]string{}

	for idx := range maxImportDepth + 2 {
		name := fmt.Sprintf("/l%d.css", idx)
		body := fmt.Sprintf(".l%d { color: red }", idx)

		if idx < maxImportDepth+1 {
			body = fmt.Sprintf("@import url(\"l%d.css\");\n%s", idx+1, body)
		}

		files[name] = body
	}

	sheets, logBuf, hits := collectImportSheets(t, importFixture{
		html:  `<html><head><link rel="stylesheet" href="l0.css"></head><body></body></html>`,
		files: files,
		media: "print",
	})

	want := make([]string, 0, maxImportDepth+1)
	for idx := maxImportDepth; idx >= 0; idx-- {
		want = append(want, fmt.Sprintf("l%d", idx))
	}

	assertSheetClasses(t, sheets, want)

	if hits["/l8.css"] != 1 {
		t.Fatalf("depth-8 sheet not fetched; hits = %v", hits)
	}

	if hits[fmt.Sprintf("/l%d.css", maxImportDepth+1)] != 0 {
		t.Fatalf("depth cap exceeded; hits = %v", hits)
	}

	if !strings.Contains(logBuf.String(), "depth exceeds") {
		t.Fatalf("log = %q, want depth-cap warning", logBuf.String())
	}
}

const maxImportDepth = 8

type importFixture struct {
	html  string
	files map[string]string
	media string
}

func collectImportSheets(
	t *testing.T,
	fixture importFixture,
) ([]*css.Stylesheet, bytes.Buffer, map[string]int) {
	t.Helper()

	hits := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		hits[request.URL.Path]++

		body, ok := fixture.files[request.URL.Path]
		if !ok {
			http.NotFound(responseWriter, request)

			return
		}

		responseWriter.Header().Set("Content-Type", "text/css")
		_, _ = responseWriter.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	root, err := html.Parse(fixture.html)
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}

	loader := load.NewLoader(settings.LoadGlobal{}) //nolint:exhaustruct // default HTTP loader

	var logBuf bytes.Buffer
	sheets := prepare.CollectSheets(
		t.Context(),
		loader,
		root,
		server.URL+"/index.html",
		settings.DefaultLoadPage(),
		prepare.SheetOptions{ //nolint:exhaustruct // test viewport/media only
			ViewportW: 600, ViewportH: 800, MediaType: fixture.media,
		},
		&logBuf,
	)

	return sheets, logBuf, hits
}

func assertSheetClasses(t *testing.T, sheets []*css.Stylesheet, want []string) {
	t.Helper()

	got := sheetClasses(sheets)
	if !slices.Equal(got, want) {
		t.Fatalf("sheet classes = %v, want %v (%d sheets)", got, want, len(sheets))
	}
}

func sheetClasses(sheets []*css.Stylesheet) []string {
	out := make([]string, 0)

	for _, sheet := range sheets {
		if sheet == nil {
			continue
		}

		for _, rule := range sheet.Rules {
			for _, sel := range rule.Selectors {
				for _, part := range sel.Parts {
					out = append(out, part.Classes...)
				}
			}
		}
	}

	return out
}
