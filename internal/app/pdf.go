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

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/convert"
)

// errNilCommand guards the command-facing adapters against nil dereferences.
var errNilCommand = errors.New("app: nil command")

// BuildPDFRequest translates a parsed CLI command into the stable engine
// request. The caller owns output-sink creation and supplies both document
// and optional outline sinks explicitly.
func BuildPDFRequest(cmd *cli.Command, output, outline io.Writer) (*convert.Request, error) {
	if cmd == nil {
		return nil, errNilCommand
	}

	req := convert.NewPDFRequest(cmd.Global, cmd.Objects, output, outline)
	if cmd.DumpOutline {
		req.Global.DumpOutline = true
	}

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("app: validate: %w", err)
	}

	return req, nil
}

// RunPDF is the command-facing adapter. It owns opening the document sink and
// selecting stdout for --dump-outline; convert.Run only receives explicit
// writers and never reaches into process-global stdout.
func RunPDF(ctx context.Context, cmd *cli.Command, log io.Writer, progress func(string, int)) error {
	if cmd == nil {
		return errNilCommand
	}

	out, closeOut, err := cmd.OpenOutput()
	if err != nil {
		return fmt.Errorf("app: open output: %w", err)
	}

	outline := io.Writer(nil)
	if cmd.DumpOutline || cmd.Global.DumpOutline {
		outline = os.Stdout
	}

	req, err := BuildPDFRequest(cmd, out, outline)
	if err == nil {
		err = convert.Run(ctx, req, log, progress)
	}

	if closeErr := closeOut(); closeErr != nil && err == nil {
		err = closeErr
	}

	return err
}
