//nolint:testpackage // compatibility adapter exercises private engine seams.
package imageout

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/cli"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/errs"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// imageLoadGlobalCmd preserves the white-box test helper while the production
// image engine remains independent of the CLI package.
func imageLoadGlobalCmd(cmd *cli.Command) settings.LoadGlobal {
	return imageLoadGlobal(cmd.Global, cmd.Image)
}

// Run preserves the historical same-package test seam. Production callers
// use app.RunImage, which owns command parsing and output lifecycle.
func Run(ctx context.Context, cmd *cli.Command, log io.Writer) error {
	if cmd == nil {
		return errs.ErrNilCommand
	}

	if ctx == nil {
		return errNilContext
	}

	img := cmd.Image

	format, err := resolveFormat(img.Format, cmd.Output)
	if err != nil {
		return err
	}

	img.Format = format
	req := NewRequest(cmd.Global, img, cmd.Objects, io.Discard)

	if err := req.Validate(); err != nil {
		return fmt.Errorf("imageout: validate: %w", err)
	}

	out, closeOut, err := cmd.OpenOutput()
	if err != nil {
		return fmt.Errorf("imageout: open output: %w", err)
	}

	req.Output = out

	return errors.Join(RunRequest(ctx, req, log), closeOut())
}
