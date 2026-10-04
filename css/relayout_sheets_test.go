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

// firstSheet returns the first parsed sheet. Callers check sheetCount first,
// so the index is safe.
func firstSheet(t *testing.T, doc *css.Document) any {
	t.Helper()

	got, ok := pubstate.StyledOf(doc)
	if !ok {
		t.Fatal("not a styled document")
	}

	return got.Sheets[0]
}
