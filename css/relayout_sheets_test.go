package css_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pubstate"
)

func sheetCount(t *testing.T, doc *css.Document) int {
	t.Helper()

	got, ok := pubstate.StyledOf(doc)
	if !ok {
		t.Fatal("not a styled document")
	}

	return len(got.Sheets)
}

func sheetAt(t *testing.T, doc *css.Document, index int) any {
	t.Helper()

	got, ok := pubstate.StyledOf(doc)
	if !ok {
		t.Fatal("not a styled document")
	}

	return got.Sheets[index]
}
