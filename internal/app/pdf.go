// Package app owns command-to-engine adapters. Keeping this translation here
// lets command mains stay orchestration-only while internal/convert remains a
// CLI-independent engine for library callers.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"

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
// process-global stdout.
//
//nolint:cyclop // command preflight must remain before sink creation.
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

	// OpenOutput maps an empty path and "-" to os.Stdout. Reject both forms
	// when dump-outline also targets stdout; a single stream cannot be both a
	// standalone XML document and a valid PDF. OutputWriter is an explicit
	// library sink and therefore bypasses this CLI-path guard.
	if req.Global.DumpOutline && cmd.OutputWriter == nil && (cmd.Output == "" || cmd.Output == "-") {
		return ErrConflictingOutputSinks
	}

	if len(cmd.Objects) == 0 {
		return ErrNoPageObjects
	}

	out, closeOut, err := cmd.OpenOutput()
	if err != nil {
		return fmt.Errorf("app: open output: %w", err)
	}

	defer func() {
		if closeErr := closeOut(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	req.Output = out

	if err := convert.Run(ctx, req, log, progress); err != nil {
		return fmt.Errorf("app: pdf conversion: %w", err)
	}

	return nil
}
