// Package app owns command-to-engine adapters. Keeping this translation here
// lets command mains stay orchestration-only while internal/convert remains a
// CLI-independent engine for library callers.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/cli"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/errs"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// DefaultTOCXSL returns the built-in TOC stylesheet description used by
// --dump-default-toc-xsl. Command mains import only app and cli; the
// convert package stays behind this adapter.
func DefaultTOCXSL() string {
	return convert.DefaultTOCXSL()
}

// Shared app-level sentinel errors; exported so callers can match with errors.Is.
var (
	ErrNilCommand    = errs.ErrNilCommand
	ErrNoPageObjects = settings.ErrNoRenderableObjects
	ErrNilContext    = errs.ErrNilContext
	// ErrConflictingOutputSinks reports a CLI stdout request that would append
	// outline XML to the PDF byte stream. Library callers with independently
	// supplied writers remain valid.
	ErrConflictingOutputSinks = errors.New("app: pdf stdout conflicts with outline XML stdout")
)

// BuildPDFRequest translates a parsed CLI command into the stable engine
// request. The caller owns output-sink creation and supplies both document
// and optional outline sinks explicitly.
func BuildPDFRequest(cmd *cli.Command, output, outline io.Writer) (*convert.Request, error) {
	if cmd == nil {
		return nil, ErrNilCommand
	}

	req := convert.NewPDFRequest(cmd.Global, cmd.Objects, output, outline)

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("app: validate: %w", err)
	}

	return req, nil
}

// RunPDF is the command-facing adapter. It validates the request before
// opening the document sink and receives the optional outline sink explicitly.
// convert.Run only receives explicit writers and never reaches into
// process-global stdout. File outputs are opened on the first write, so a
// conversion that fails during render leaves an existing artifact untouched.
func RunPDF(
	ctx context.Context,
	cmd *cli.Command,
	log io.Writer,
	progress func(string, int),
	outline io.Writer,
) (err error) {
	if cmd == nil {
		return ErrNilCommand
	}

	if ctx == nil {
		return ErrNilContext
	}

	// Validate the complete command before creating or truncating a file. The
	// discard sink satisfies the request's explicit output contract while
	// keeping validation side-effect free.
	req, err := BuildPDFRequest(cmd, io.Discard, outline)
	if err != nil {
		return err
	}

	// Output resolution maps an empty path and "-" to os.Stdout. Reject both
	// forms when dump-outline also targets stdout; a single stream cannot be
	// both a standalone XML document and a valid PDF. OutputWriter is an
	// explicit library sink and therefore bypasses this CLI-path guard.
	if req.Global.DumpOutline && cmd.OutputWriter == nil && (cmd.Output == "" || cmd.Output == "-") {
		return ErrConflictingOutputSinks
	}

	out, closeOut, err := openLazyOutput(cmd)
	if err != nil {
		return fmt.Errorf("app: open output: %w", err)
	}

	defer func() {
		err = errors.Join(err, closeOut())
	}()

	req.Output = out

	if err = convert.Run(ctx, req, log, progress); err != nil {
		return fmt.Errorf("app: pdf conversion: %w", err)
	}

	return nil
}

// openLazyOutput resolves the output destination for both the PDF and image
// adapters without creating or truncating a file up front. A path is validated
// now (unwritable file, missing parent directory) so destination mistakes
// still surface before a long render, but the file itself is opened on the
// first write: a conversion that fails while rendering must not clobber an
// existing artifact or leave a 0-byte file.
func openLazyOutput(cmd *cli.Command) (io.Writer, func() error, error) {
	if cmd.OutputWriter != nil {
		return cmd.OutputWriter, func() error { return nil }, nil
	}

	if cmd.Output == "" || cmd.Output == "-" {
		return os.Stdout, func() error { return nil }, nil
	}

	if err := checkOutputPath(cmd.Output); err != nil {
		return nil, nil, err
	}

	out := &lazyFileWriter{path: cmd.Output, file: nil}

	return out, out.Close, nil
}

// checkOutputPath rejects destinations that cannot receive bytes without
// touching the destination itself: an existing path must be openable for
// writing, and a new path's parent directory must exist.
func checkOutputPath(path string) error {
	_, err := os.Stat(path)

	switch {
	case err == nil:
		f, openErr := os.OpenFile(path, os.O_WRONLY, 0)
		if openErr != nil {
			return fmt.Errorf("output %q: %w", path, openErr)
		}

		if closeErr := f.Close(); closeErr != nil {
			return fmt.Errorf("output %q: %w", path, closeErr)
		}

		return nil
	case errors.Is(err, os.ErrNotExist):
		if _, statErr := os.Stat(filepath.Dir(path)); statErr != nil {
			return fmt.Errorf("output %q: %w", path, statErr)
		}

		return nil
	default:
		return fmt.Errorf("output %q: %w", path, err)
	}
}

// lazyFileWriter opens its path with os.Create on the first write. Close is a
// no-op when nothing was written, so a failed conversion leaves no file.
type lazyFileWriter struct {
	path string
	file *os.File
}

func (w *lazyFileWriter) Write(data []byte) (int, error) {
	if w.file == nil {
		f, err := os.Create(w.path)
		if err != nil {
			return 0, fmt.Errorf("output %q: %w", w.path, err)
		}

		w.file = f
	}

	n, err := w.file.Write(data)
	if err != nil {
		return n, fmt.Errorf("output %q: %w", w.path, err)
	}

	return n, nil
}

func (w *lazyFileWriter) Close() error {
	if w.file == nil {
		return nil
	}

	if err := w.file.Close(); err != nil {
		return fmt.Errorf("output %q: %w", w.path, err)
	}

	return nil
}
