package app

import (
	"context"
	"errors"
	"fmt"
	"io"

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/convert"
	"gowkhtmltopdf/internal/imageout"
)

// RunImage is the command-facing image adapter. It keeps command parsing and
// output ownership at the application boundary while imageout remains a
// CLI-independent request engine.
func RunImage(ctx context.Context, cmd *cli.Command, log io.Writer) error {
	if cmd == nil {
		return ErrNilCommand
	}

	if ctx == nil {
		return ErrNilContext
	}

	img := cmd.Image

	format, err := imageout.ResolveFormat(img.Format, cmd.Output)
	if err != nil {
		return fmt.Errorf("app: resolve image format: %w", err)
	}

	img.Format = format
	request := convert.NewImageRequest(cmd.Global, img, cmd.Objects, io.Discard)

	if err := request.ValidateImage(); err != nil {
		if errors.Is(err, convert.ErrNoRenderableObjects) {
			return ErrNoPageObjects
		}

		return fmt.Errorf("app: validate image: %w", err)
	}

	out, closeOut, err := cmd.OpenOutput()
	if err != nil {
		return fmt.Errorf("app: open image output: %w", err)
	}

	request.Output = out
	runErr := imageout.RunRequest(ctx, request, log)

	if runErr != nil {
		runErr = fmt.Errorf("app: image: %w", runErr)
	}

	return errors.Join(runErr, closeOut())
}
