package app_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/app"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/cli"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

func TestRunImageResolvesFormatPerExecution(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cmd := &cli.Command{
		Global: settings.DefaultPdfGlobal(),
		Image:  settings.DefaultImageGlobal(),
		Objects: []settings.PdfObject{{
			Load: settings.LoadPage{
				InlineHTML: []byte("<h1>format</h1>"),
			},
		}},
		Output: filepath.Join(dir, "first.png"),
	}

	if err := app.RunImage(t.Context(), cmd, nil); err != nil {
		t.Fatalf("PNG execution: %v", err)
	}

	png, err := os.ReadFile(cmd.Output)

	if err != nil {
		t.Fatalf("read PNG: %v", err)
	}

	if !bytes.HasPrefix(png, []byte("\x89PNG")) {
		t.Fatalf("first output is not PNG: %q", png[:min(len(png), 8)])
	}

	cmd.Output = filepath.Join(dir, "second.jpg")
	if err := app.RunImage(t.Context(), cmd, nil); err != nil {
		t.Fatalf("JPEG execution: %v", err)
	}

	jpg, err := os.ReadFile(cmd.Output)

	if err != nil {
		t.Fatalf("read JPEG: %v", err)
	}

	if !bytes.HasPrefix(jpg, []byte{0xff, 0xd8, 0xff}) {
		t.Fatalf("second output is not JPEG: %x", jpg[:min(len(jpg), 8)])
	}
}

func TestRunImageDelegatesPreflightBeforeOpeningOutput(t *testing.T) {
	t.Parallel()

	output := filepath.Join(t.TempDir(), "out.png")
	cmd := &cli.Command{
		Global: settings.DefaultPdfGlobal(),
		Image:  settings.DefaultImageGlobal(),
		Output: output,
	}

	err := app.RunImage(t.Context(), cmd, nil)
	if !errors.Is(err, app.ErrNoPageObjects) {
		t.Fatalf("RunImage() = %v, want errors.Is(..., %v)", err, app.ErrNoPageObjects)
	}

	if _, statErr := os.Stat(output); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("output stat error = %v, want os.ErrNotExist", statErr)
	}
}

func TestRunImageRejectsNegativeWidthBeforeOpeningOutput(t *testing.T) {
	t.Parallel()

	output := filepath.Join(t.TempDir(), "out.png")
	existing := []byte("pre-existing output must survive")

	if err := os.WriteFile(output, existing, 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := &cli.Command{
		Global: settings.DefaultPdfGlobal(),
		Image:  settings.DefaultImageGlobal(),
		Objects: []settings.PdfObject{{
			Load: settings.LoadPage{InlineHTML: []byte("<h1>bad width</h1>")},
		}},
		Output: output,
	}
	cmd.Image.Width = -1

	if err := app.RunImage(t.Context(), cmd, nil); err == nil {
		t.Fatal("RunImage() = nil, want negative width error")
	}

	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("output was removed or unreadable after preflight: %v", err)
	}

	if !bytes.Equal(got, existing) {
		t.Fatalf("output changed to %q, want untouched %q", got, existing)
	}
}

func TestRunImageRejectsMultipleObjectsBeforeOpeningOutput(t *testing.T) {
	t.Parallel()

	output := filepath.Join(t.TempDir(), "out.png")
	cmd := &cli.Command{
		Global: settings.DefaultPdfGlobal(),
		Image:  settings.DefaultImageGlobal(),
		Objects: []settings.PdfObject{
			{Page: "first.html"},
			{Page: "second.html"},
		},
		Output: output,
	}

	err := app.RunImage(t.Context(), cmd, nil)
	if !errors.Is(err, app.ErrMultipleImageObjects) {
		t.Fatalf("RunImage() = %v, want errors.Is(..., %v)", err, app.ErrMultipleImageObjects)
	}

	if _, statErr := os.Stat(output); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("output stat error = %v, want os.ErrNotExist", statErr)
	}
}

// failedImageCommand builds an image command that passes request validation
// and fails during rendering: the loopback fetch is rejected by the network
// policy, which is exactly the window where opening the output up front would
// have truncated it.
func failedImageCommand(t *testing.T, output string) *cli.Command {
	t.Helper()

	cmd, err := cli.Parse([]string{
		"--restrict-network",
		"--quiet",
		"http://127.0.0.1/",
		"--output", output,
	}, cli.ModeImage)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	return cmd
}

func TestRunImageFailurePreservesExistingOutput(t *testing.T) {
	t.Parallel()

	output := filepath.Join(t.TempDir(), "out.png")
	previous := []byte("previous artifact bytes that a failed conversion must not clobber")

	if err := os.WriteFile(output, previous, 0o600); err != nil {
		t.Fatalf("seed output: %v", err)
	}

	runErr := app.RunImage(t.Context(), failedImageCommand(t, output), nil)
	if !errors.Is(runErr, load.ErrNetworkPolicy) {
		t.Fatalf("RunImage = %v, want errors.Is(..., load.ErrNetworkPolicy)", runErr)
	}

	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", output, err)
	}

	if !bytes.Equal(got, previous) {
		t.Fatalf("a failed conversion clobbered the output: got %d bytes %q, want %d bytes",
			len(got), got, len(previous))
	}
}

func TestRunImageFailureDoesNotCreateOutput(t *testing.T) {
	t.Parallel()

	output := filepath.Join(t.TempDir(), "out.png")

	runErr := app.RunImage(t.Context(), failedImageCommand(t, output), nil)
	if !errors.Is(runErr, load.ErrNetworkPolicy) {
		t.Fatalf("RunImage = %v, want errors.Is(..., load.ErrNetworkPolicy)", runErr)
	}

	if _, statErr := os.Stat(output); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("failed conversion left an artifact at %q (stat error = %v)", output, statErr)
	}
}

func TestRunImageSuccessReplacesExistingOutput(t *testing.T) {
	t.Parallel()

	output := filepath.Join(t.TempDir(), "out.png")

	if err := os.WriteFile(output, []byte("stale bytes"), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := &cli.Command{
		Global: settings.DefaultPdfGlobal(),
		Image:  settings.DefaultImageGlobal(),
		Objects: []settings.PdfObject{{
			Load: settings.LoadPage{InlineHTML: []byte("<h1>overwrite</h1>")},
		}},
		Output: output,
	}

	if err := app.RunImage(t.Context(), cmd, nil); err != nil {
		t.Fatalf("RunImage: %v", err)
	}

	png, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", output, err)
	}

	if !bytes.HasPrefix(png, []byte("\x89PNG")) {
		t.Fatalf("output is not PNG after a successful run: %q", png[:min(len(png), 8)])
	}
}

// TestRunImageRejectsBadOutputPathBeforeConversion keeps the early destination
// check: a missing parent directory fails before the render starts even though
// the file itself is no longer opened up front.
func TestRunImageRejectsBadOutputPathBeforeConversion(t *testing.T) {
	t.Parallel()

	cmd := &cli.Command{
		Global: settings.DefaultPdfGlobal(),
		Image:  settings.DefaultImageGlobal(),
		Objects: []settings.PdfObject{{
			Load: settings.LoadPage{InlineHTML: []byte("<h1>output preflight</h1>")},
		}},
		Output: filepath.Join(t.TempDir(), "missing-dir", "out.png"),
	}

	err := app.RunImage(t.Context(), cmd, nil)
	if err == nil || !strings.HasPrefix(err.Error(), "app: open image output:") {
		t.Fatalf("RunImage() = %v, want app: open image output error", err)
	}
}
