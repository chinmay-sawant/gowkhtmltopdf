package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/cli"
)

func TestRunHTMLToPNGUsesExplicitOutput(t *testing.T) {
	t.Parallel()

	output := filepath.Join(t.TempDir(), "out.png")
	code := run([]string{
		"--quiet",
		"--format", "png",
		"--html", "<html><body><h1>image smoke</h1></body></html>",
		"--output", output,
	})

	if code != cli.ExitOK {
		t.Fatalf("run exit code = %d, want %d", code, cli.ExitOK)
	}

	data, err := os.ReadFile(output)

	if err != nil {
		t.Fatalf("ReadFile(%q): %v", output, err)
	}

	if !bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatalf("output prefix = %q, want PNG magic", data[:min(len(data), 16)])
	}
}

func TestRunRequiresExplicitOutput(t *testing.T) {
	t.Parallel()

	if code := run([]string{"--html", "<html><body>missing output</body></html>"}); code != cli.ExitError {
		t.Fatalf("run exit code = %d, want %d", code, cli.ExitError)
	}
}
