package settings_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

func TestStampCoverDefaults(t *testing.T) {
	t.Parallel()

	obj := settings.DefaultPdfObject()
	obj.IncludeInOutline = true
	obj.Header.Left = "inherit-me"
	obj.Footer.Center = "inherit-me"
	settings.StampCover(&obj)

	if !obj.IsCover {
		t.Fatal("IsCover = false")
	}

	if obj.IncludeInOutline {
		t.Fatal("IncludeInOutline = true")
	}

	if !obj.HeaderSet || !obj.FooterSet {
		t.Fatalf("HF set bits = header:%v footer:%v", obj.HeaderSet, obj.FooterSet)
	}

	assertEmptyHF(t, obj.Header, obj.Footer)
}

func TestStampTOCDefaults(t *testing.T) {
	t.Parallel()

	obj := settings.DefaultPdfObject()
	obj.IncludeInOutline = true
	obj.UseOutline = true
	settings.StampTOC(&obj)

	if !obj.IsTableOfContent {
		t.Fatal("IsTableOfContent = false")
	}

	if obj.UseOutline {
		t.Fatal("UseOutline = true")
	}

	if obj.IncludeInOutline {
		t.Fatal("IncludeInOutline = true")
	}
}

func TestStampEmptyHFOverrideBlocksGlobalFallthrough(t *testing.T) {
	t.Parallel()

	obj := settings.DefaultPdfObject()
	settings.StampEmptyHFOverride(&obj)

	global := settings.DefaultPdfGlobal()
	global.Header.Left = "global-header"
	global.Footer.Center = "global-footer"

	if got := obj.HeaderFor(global); got.Left != "" {
		t.Fatalf("HeaderFor after empty stamp = %+v", got)
	}

	if got := obj.FooterFor(global); got.Center != "" {
		t.Fatalf("FooterFor after empty stamp = %+v", got)
	}
}

func assertEmptyHF(t *testing.T, header, footer settings.HeaderFooter) {
	t.Helper()

	if header.Left != "" || header.Center != "" || header.Right != "" ||
		footer.Left != "" || footer.Center != "" || footer.Right != "" ||
		len(header.Replace) != 0 || len(footer.Replace) != 0 {
		t.Fatalf("HF must be empty: header=%+v footer=%+v", header, footer)
	}
}
