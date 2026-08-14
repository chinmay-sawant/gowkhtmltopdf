package settings_test

import (
	"testing"

	"gowkhtmltopdf/internal/settings"
)

//nolint:cyclop,wsl // typed builder assertions
func TestPdfGlobalOptionsBuildsIndependentTypedSnapshot(t *testing.T) {
	t.Parallel()

	const pageSize = "Letter"

	options := settings.NewPdfGlobalOptions().
		WithPageSize(pageSize).
		WithMargins(1, 2, 3, 4).
		WithTitle("typed").
		WithCopies(2, false).
		WithPDFVersion("1.7").
		WithPDFProfile("a3a-ua1")

	got := options.Build()
	if got.PdfVersion != "1.7" {
		t.Fatalf("pdf version = %q, want 1.7", got.PdfVersion)
	}
	if got.PdfProfile != "a3a-ua1" {
		t.Fatalf("pdf profile = %q, want a3a-ua1", got.PdfProfile)
	}
	if got.PageSize != pageSize {
		t.Fatalf("page size = %q, want Letter", got.PageSize)
	}
	if got.Margin.Top != 1 || got.Margin.Right != 2 ||
		got.Margin.Bottom != 3 || got.Margin.Left != 4 {
		t.Fatalf("margins = %#v", got.Margin)
	}
	if got.Title != "typed" || got.Copies != 2 || got.Collate {
		t.Fatalf("options = %#v", got)
	}
}

func TestPdfGlobalOptionsWithSettingCoversCompatibilityKeys(t *testing.T) {
	t.Parallel()

	options, err := settings.NewPdfGlobalOptions().
		WithSetting("size.width", "210mm")
	if err != nil {
		t.Fatalf("WithSetting(size.width): %v", err)
	}

	got := options.Build()
	if got.Size.Width != 210 {
		t.Fatalf("size.width = %v, want 210mm", got.Size.Width)
	}

	if _, err := options.WithSetting("not-a-setting", "value"); err == nil {
		t.Fatal("unknown WithSetting key unexpectedly succeeded")
	}
}
