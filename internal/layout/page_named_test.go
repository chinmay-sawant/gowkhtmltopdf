package layout //nolint:testpackage // page-name tests inspect unexported layout state

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

func TestPageNameInherits(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div class="outer"><p class="inner">a</p><p class="own">b</p></div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.outer { page: chapter }
		.own { page: cover }
	`)}, "print", testViewport, 800)

	inner := styleByClass(t, styles, "inner")
	if inner.PageName != "chapter" {
		t.Fatalf("inner PageName = %q, want chapter (used-value inherit)", inner.PageName)
	}

	own := styleByClass(t, styles, "own")
	if own.PageName != "cover" {
		t.Fatalf("own PageName = %q, want cover", own.PageName)
	}

	outer := styleByClass(t, styles, "outer")
	if outer.PageName != "chapter" {
		t.Fatalf("outer PageName = %q, want chapter", outer.PageName)
	}
}

func TestPageNameBreak(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `p { margin: 0 } .ch { page: chapter }`)
	res := layoutHTML(t, `<html><body><p>PAGEONE</p><div class="ch"><p>PAGETWO</p></div></body></html>`, cssSheet)
	doc := pdf.NewDocument()

	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatal(err)
	}

	if pageOf(t, res, "PAGEONE") == pageOf(t, res, "PAGETWO") {
		t.Fatal("page:chapter sibling stayed on the same page")
	}
}

func TestPageNameNoBreakOnBody(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `body { page: chapter } p { margin: 0 }`)
	res := layoutHTML(t, `<html><body><p>ONLY</p></body></html>`, cssSheet)
	doc := pdf.NewDocument()

	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatal(err)
	}

	if n := doc.PageCount(); n != 1 {
		t.Fatalf("body { page: chapter } pages = %d, want 1 (no blank first page)", n)
	}
}
