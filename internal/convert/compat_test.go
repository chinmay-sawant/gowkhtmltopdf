//nolint:testpackage // compatibility adapters exercise private engine seams.
package convert

import (
	"context"
	"fmt"
	"io"

	"gowkhtmltopdf/internal/cli"
)

// These test-only adapters preserve the historical same-package test seam.
// Production command translation belongs in internal/app; the conversion
// engine stays independent of the CLI parser.
func RunPDF(cmd *cli.Command, log io.Writer) error {
	return RunPDFContext(context.Background(), cmd, log, nil)
}

func RunPDFContext(ctx context.Context, cmd *cli.Command, log io.Writer, progress func(phase string, percent int)) (err error) { //nolint:lll // compatibility test adapter
	if cmd == nil {
		return errNilCommand
	}

	if ctx == nil {
		return errNilContext
	}

	outline := cmd.OutlineWriter
	if outline == nil {
		outline = io.Discard
	}

	req := NewPDFRequest(cmd.Global, cmd.Objects, io.Discard, outline)
	if cmd.DumpOutline {
		req.Global.DumpOutline = true
	}

	if err := req.ValidatePDF(); err != nil {
		return err
	}

	out, closeOut, err := cmd.OpenOutput()
	if err != nil {
		return fmt.Errorf("open output: %w", err)
	}

	defer func() {
		if closeErr := closeOut(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	req.Output = out

	return Run(ctx, req, log, progress)
}
