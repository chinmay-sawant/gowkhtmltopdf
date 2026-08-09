package settings_test

import (
	"testing"

	"gowkhtmltopdf/internal/settings"
)

//nolint:wsl // typed builder assertions
func TestPdfGlobalOptionsBuildsIndependentTypedSnapshot(t *testing.T) {
	t.Parallel()

	const pageSize = "Letter"

	options := settings.NewPdfGlobalOptions().
		WithPageSize(pageSize).
		WithMargins(1, 2, 3, 4).
		WithTitle("typed").
		WithCopies(2, false)

	got := options.Build()
	if got.PageSize != pageSize || got.Size.PageSize != pageSize {
		t.Fatalf("page size = %q/%q, want Letter/Letter", got.PageSize, got.Size.PageSize)
	}
	if got.Margin.Top != 1 || got.Margin.Right != 2 ||
		got.Margin.Bottom != 3 || got.Margin.Left != 4 {
		t.Fatalf("margins = %#v", got.Margin)
	}
	if got.Title != "typed" || got.Copies != 2 || got.Collate {
		t.Fatalf("options = %#v", got)
	}
}
