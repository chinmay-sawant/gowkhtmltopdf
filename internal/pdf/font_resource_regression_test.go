package pdf

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParseTTFRejectsOversizedInput(t *testing.T) {
	t.Parallel()

	data := make([]byte, maxFontBytes+1)
	_, err := ParseTTF(data)

	if !errors.Is(err, errFontTooLarge) {
		t.Fatalf("ParseTTF error = %v, want errFontTooLarge", err)
	}
}

func TestScanFontDirsSkipsOversizedFontBeforeParsing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "oversized.ttf")

	if err := os.WriteFile(path, []byte{}, 0o600); err != nil {
		t.Fatalf("create font: %v", err)
	}

	if err := os.Truncate(path, int64(maxFontBytes)+1); err != nil {
		t.Fatalf("truncate font: %v", err)
	}

	registry := ScanFontDirs([]string{dir})

	if len(registry.faces) != 0 {
		t.Fatalf("scanned %d oversized faces, want 0", len(registry.faces))
	}
}
