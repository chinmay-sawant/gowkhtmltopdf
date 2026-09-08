package app_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/app"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/cli"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

func TestRunImageResolvesFormatPerExecution(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cmd := &cli.Command{ //nolint:exhaustruct // repeated format resolution only needs one source
		Global: settings.DefaultPdfGlobal(),
		Image:  settings.DefaultImageGlobal(),
		Objects: []settings.PdfObject{{ //nolint:exhaustruct // inline source only
			Load: settings.LoadPage{ //nolint:exhaustruct // inline source only
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
	cmd := &cli.Command{ //nolint:exhaustruct // focused invalid command
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

func TestRunImageRejectsMultipleObjectsBeforeOpeningOutput(t *testing.T) {
	t.Parallel()

	output := filepath.Join(t.TempDir(), "out.png")
	cmd := &cli.Command{ //nolint:exhaustruct // focused invalid command
		Global: settings.DefaultPdfGlobal(),
		Image:  settings.DefaultImageGlobal(),
		Objects: []settings.PdfObject{
			{Page: "first.html"},  //nolint:exhaustruct // only page source matters
			{Page: "second.html"}, //nolint:exhaustruct // only page source matters
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
