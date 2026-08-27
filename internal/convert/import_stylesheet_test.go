package convert //nolint:testpackage // shares newCommand with convert_test.go

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/prepare"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
)

func TestImportStylesheet(t *testing.T) {
	t.Parallel()

	htmlDoc := `<html><head><link rel="stylesheet" href="a.css"></head>` +
		`<body><p class="from-a from-b">x</p></body></html>`
	cmd, dir := newCommand(t, htmlDoc, filepath.Join(t.TempDir(), "out.pdf"))
	writeImportCSS(t, dir, "a.css", `@import url("b.css");
.from-a { color: #ff0000 }`)
	writeImportCSS(t, dir, "b.css", `.from-b { color: #0000ff }`)

	loader := load.NewLoader(cmd.Global.Load)
	root, err := html.ParseDocument([]byte(htmlDoc))
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}

	page := cmd.Objects[0]
	sheets := prepare.CollectSheets(
		t.Context(),
		loader,
		root,
		"file://"+filepath.ToSlash(dir)+"/",
		page.Load,
		prepare.SheetOptions{ //nolint:exhaustruct // test viewport/media only
			ViewportW: 600, ViewportH: 800, MediaType: mediaPrint,
		},
		io.Discard,
	)

	got := importSheetClasses(sheets)
	want := []string{"from-b", "from-a"}
	if !slices.Equal(got, want) {
		t.Fatalf("imported sheets = %v, want %v (%d sheets)", got, want, len(sheets))
	}

	_ = runPDF(t, cmd)
}

func writeImportCSS(t *testing.T, dir, name, body string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func importSheetClasses(sheets []*css.Stylesheet) []string {
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
