package settings_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

func TestPdfGlobalFieldSnapshotViaClone(t *testing.T) {
	t.Parallel()

	const pageSize = "Letter"

	global := settings.DefaultPdfGlobal()
	global.PageSize = pageSize
	global.Margin = settings.Margin{Top: 1, Right: 2, Bottom: 3, Left: 4}
	global.Title = "typed"
	global.Copies = 2
	global.Collate = false

	version, err := settings.ParsePDFVersion("1.7")
	if err != nil {
		t.Fatalf("ParsePDFVersion: %v", err)
	}

	global.PdfVersion = version

	profile, err := settings.ParsePDFProfile("a3a-ua1")
	if err != nil {
		t.Fatalf("ParsePDFProfile: %v", err)
	}

	global.PdfProfile = profile

	assertClonedTypedFields(t, settings.ClonePdfGlobal(global), pageSize)
}

func assertClonedTypedFields(t *testing.T, got settings.PdfGlobal, pageSize string) {
	t.Helper()

	checks := []struct {
		ok   bool
		fail string
	}{
		{got.PdfVersion == "1.7", "pdf version"},
		{got.PdfProfile == settings.ProfilePDFA3aPDFUA1, "pdf profile"},
		{got.PageSize == pageSize, "page size"},
		{
			got.Margin.Top == 1 && got.Margin.Right == 2 &&
				got.Margin.Bottom == 3 && got.Margin.Left == 4,
			"margins",
		},
		{got.Title == "typed" && got.Copies == 2 && !got.Collate, "title/copies/collate"},
	}

	for _, check := range checks {
		if !check.ok {
			t.Fatalf("%s snapshot mismatch: %#v", check.fail, got)
		}
	}
}

func TestPdfGlobalSetCoversCompatibilityKeys(t *testing.T) {
	t.Parallel()

	global := settings.DefaultPdfGlobal()

	if err := global.Set("size.width", "210mm"); err != nil {
		t.Fatalf("Set(size.width): %v", err)
	}

	if global.Size.Width != 210 {
		t.Fatalf("size.width = %v, want 210mm", global.Size.Width)
	}

	if err := global.Set("not-a-setting", "value"); err == nil {
		t.Fatal("unknown Set key unexpectedly succeeded")
	}
}
