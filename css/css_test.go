package css_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/html"
)

func TestParseAndApply(t *testing.T) {
	t.Parallel()

	doc := mustDoc(t, `<div id="box">Hi</div>`)
	sheet, err := css.Parse("div { width: 10px }")

	if err != nil {
		t.Fatal(err)
	}

	styled, err := css.Apply(t.Context(), doc, css.Options{
		WidthPx:  200,
		HeightPx: 100,
		Media:    "screen",
		Extra:    []*css.Sheet{sheet},
	})
	if err != nil {
		t.Fatal(err)
	}

	if styled == nil {
		t.Fatal("styled document is nil")
	}
}

func TestApplyRejectsBadInput(t *testing.T) {
	t.Parallel()

	doc := mustDoc(t, `<p>Hi</p>`)
	opts := css.Options{WidthPx: 20, HeightPx: 20, Media: "screen", Extra: nil}

	_, err := css.Apply(nil, doc, opts) //nolint:staticcheck // nil context is intentional
	if !errors.Is(err, css.ErrNilContext) {
		t.Fatalf("nil context: %v", err)
	}

	_, err = css.Apply(t.Context(), nil, opts)
	if !errors.Is(err, css.ErrNilDocument) {
		t.Fatalf("nil document: %v", err)
	}

	_, err = css.Apply(t.Context(), doc, css.Options{WidthPx: 0, HeightPx: 20, Media: "screen", Extra: nil})
	if !errors.Is(err, css.ErrBadSize) {
		t.Fatalf("size: %v", err)
	}

	_, err = css.Apply(t.Context(), doc, css.Options{WidthPx: 20, HeightPx: 20, Media: "speech", Extra: nil})
	if !errors.Is(err, css.ErrBadMedia) {
		t.Fatalf("media: %v", err)
	}
}

func TestParseRejectsBrokenSheet(t *testing.T) {
	t.Parallel()

	_, err := css.Parse("div { width: ")
	if err == nil {
		t.Fatal("expected a parse error")
	}

	if !strings.Contains(err.Error(), "css: parse:") {
		t.Fatalf("error = %v", err)
	}
}

func mustDoc(t *testing.T, source string) *html.Document {
	t.Helper()

	doc, err := html.Parse([]byte(source))
	if err != nil {
		t.Fatal(err)
	}

	return doc
}
