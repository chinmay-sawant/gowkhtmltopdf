package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/cli"
)

//nolint:wsl // stdout swapping necessarily interleaves assignments and checks.
func captureStdout(t *testing.T, action func() int) (int, []byte) {
	t.Helper()

	old := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = writer
	code := action()
	if err := writer.Close(); err != nil {
		t.Fatalf("close stdout: %v", err)
	}
	os.Stdout = old

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close captured stdout: %v", err)
	}

	return code, data
}

//nolint:paralleltest,wsl // stdout is process-global and tests must remain serial.
func TestRunDumpDefaultTOCXSLIsStandaloneTerminalAction(t *testing.T) {
	code, output := captureStdout(t, func() int {
		return run([]string{"--dump-default-toc-xsl"})
	})

	if code != cli.ExitOK {
		t.Fatalf("run exit code = %d, want %d", code, cli.ExitOK)
	}
	if !bytes.HasPrefix(output, []byte("<?xml")) || !bytes.Contains(output, []byte("<xsl:stylesheet")) {
		t.Fatalf("stdout = %q, want standalone default XSL", output)
	}
}

//nolint:paralleltest,wsl // stdout is process-global and tests must remain serial.
func TestRunRejectsDumpOutlineWithPDFStdout(t *testing.T) {
	code, output := captureStdout(t, func() int {
		return run([]string{
			"--quiet",
			"--dump-outline",
			"--html", "<html><body><h1>stdout conflict</h1></body></html>",
			"--output", "-",
		})
	})

	if code != cli.ExitError {
		t.Fatalf("run exit code = %d, want %d", code, cli.ExitError)
	}
	if len(output) != 0 {
		t.Fatalf("stdout bytes = %d, want zero for rejected mixed sinks", len(output))
	}
}

//nolint:paralleltest,wsl // stdout is process-global and tests must remain serial.
func TestRunKeepsPDFFileAndOutlineXMLSeparate(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "out.pdf")
	code, outline := captureStdout(t, func() int {
		return run([]string{
			"--quiet",
			"--dump-outline",
			"--html", "<html><body><h1>separate outputs</h1></body></html>",
			"--output", outputPath,
		})
	})

	if code != cli.ExitOK {
		t.Fatalf("run exit code = %d, want %d", code, cli.ExitOK)
	}
	pdf, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", outputPath, err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("PDF prefix = %q, want %%PDF-", pdf[:min(len(pdf), 16)])
	}
	if !bytes.HasPrefix(outline, []byte("<?xml")) || !bytes.Contains(outline, []byte("<outline")) {
		t.Fatalf("outline stdout = %q, want standalone XML outline", outline)
	}
}

//nolint:paralleltest,wsl // stdout is process-global and tests must remain serial.
func TestRunRejectsConflictingTerminalOptions(t *testing.T) {
	code, output := captureStdout(t, func() int {
		return run([]string{"--dump-default-toc-xsl", "--dump-outline"})
	})

	if code != cli.ExitError {
		t.Fatalf("run exit code = %d, want %d", code, cli.ExitError)
	}
	if len(output) != 0 {
		t.Fatalf("stdout bytes = %d, want zero for conflicting terminal options", len(output))
	}
}
