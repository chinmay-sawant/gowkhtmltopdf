//nolint:testpackage,wsl // white-box PDF resource test
package pdf

import (
	"strings"
	"testing"
)

func TestContentBlendModeUsesPDFExtGState(t *testing.T) {
	t.Parallel()

	content := NewContent()
	content.Save()
	content.SetBlendMode("multiply")
	content.Restore()

	if !strings.Contains(string(content.Bytes()), "/bmMultiply gs\n") {
		t.Fatal("content stream did not select multiply blend mode")
	}
	if got := content.extGState(); got != "/ExtGState << /bmMultiply << /BM /Multiply >> >>" {
		t.Fatalf("ExtGState = %q", got)
	}
}

func TestContentBlendModeIgnoresUnknownMode(t *testing.T) {
	t.Parallel()

	content := NewContent()
	content.SetBlendMode("not-a-pdf-mode")

	if content.Bytes() != nil {
		t.Fatalf("unknown blend mode emitted content: %q", content.Bytes())
	}
	if got := content.extGState(); got != "" {
		t.Fatalf("ExtGState = %q, want empty", got)
	}
}
