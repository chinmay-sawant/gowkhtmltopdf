package app_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/app"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/cli"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

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
