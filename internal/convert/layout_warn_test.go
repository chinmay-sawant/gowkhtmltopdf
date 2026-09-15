package convert

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSkippedImagePayloadWarnsOnceThroughRunLog proves the layout.Options.Warnf
// wiring at the bodyLayoutOpts call site: a fetched payload that is not an
// embeddable image reaches the run log as exactly one warning naming the src.
func TestSkippedImagePayloadWarnsOnceThroughRunLog(t *testing.T) {
	t.Parallel()

	cmd, dir := newCommand(t,
		`<html><body><img src="bad.bin" alt="not an image"></body></html>`,
		filepath.Join(t.TempDir(), "out.pdf"))

	if err := os.WriteFile(filepath.Join(dir, "bad.bin"), []byte("<html>error page</html>"), 0o600); err != nil {
		t.Fatalf("write bad image: %v", err)
	}

	var log bytes.Buffer
	data := runPDFWithLog(t, cmd, &log)

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}

	var warnings []string

	for _, line := range strings.Split(log.String(), "\n") {
		if strings.Contains(line, "bad.bin") {
			warnings = append(warnings, line)
		}
	}

	if len(warnings) != 1 {
		t.Fatalf("log lines naming bad.bin = %d, want exactly 1; log=%q", len(warnings), log.String())
	}

	if !strings.HasPrefix(warnings[0], "warning: ") {
		t.Errorf("skipped-image line is not a warning: %q", warnings[0])
	}

	if !strings.Contains(warnings[0], "unsupported image data") {
		t.Errorf("skipped-image warning does not name the reason: %q", warnings[0])
	}
}

// TestTextOverflowWarnsThroughRunLog proves the layout text-overflow warning
// reaches the run log for a body that paints prose past the page content box.
// Smart shrink ignores text ops, so the warning is the only visible signal.
func TestTextOverflowWarnsThroughRunLog(t *testing.T) {
	t.Parallel()

	cmd, _ := newCommand(t,
		`<html><body><span style="white-space:nowrap;font-size:20pt">`+strings.Repeat("W", 300)+`</span></body></html>`,
		filepath.Join(t.TempDir(), "out.pdf"))

	var log bytes.Buffer
	data := runPDFWithLog(t, cmd, &log)

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}

	if !strings.Contains(log.String(), "text overflows content box") {
		t.Errorf("text overflow did not warn through the run log; log=%q", log.String())
	}
}

// TestSkippedImagePayloadWarnsThroughHTMLLog covers the HTML header/footer
// layout call site (loadHTMLHF): a skipped image inside an HTML header must
// reach the run log like a body skipped image.
func TestSkippedImagePayloadWarnsThroughHTMLLog(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	headerPath := filepath.Join(dir, "header.html")

	if err := os.WriteFile(headerPath, []byte(`<html><body><img src="bad-hf.bin"></body></html>`), 0o600); err != nil {
		t.Fatalf("write header: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "bad-hf.bin"), []byte("<html>error page</html>"), 0o600); err != nil {
		t.Fatalf("write bad header image: %v", err)
	}

	cmd, _ := newCommand(t, `<html><body><p>body</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Header.HTMLURL = headerPath

	var log bytes.Buffer
	data := runPDFWithLog(t, cmd, &log)

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}

	if !strings.Contains(log.String(), "bad-hf.bin") {
		t.Errorf("HTML header skipped image did not warn through the run log; log=%q", log.String())
	}
}
