package app

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/cli"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/imageout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// ErrMultipleImageObjects reports an image command with more than one input
// object. Image mode owns one source and one output canvas.
var ErrMultipleImageObjects = errors.New("app: multiple image objects")

// RunImage is the command-facing image adapter. It keeps command parsing and
// output ownership at the application boundary while imageout remains a
// CLI-independent request engine.
func RunImage(ctx context.Context, cmd *cli.Command, log io.Writer) (err error) {
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
	request := imageout.NewRequest(cmd.Global, img, cmd.Objects, io.Discard)

	if err := request.Validate(); err != nil {
		if errors.Is(err, settings.ErrNoRenderableObjects) {
			return ErrNoPageObjects
		}

		if errors.Is(err, imageout.ErrMultipleInputs) {
			return ErrMultipleImageObjects
		}

		return fmt.Errorf("app: validate image: %w", err)
	}

	out, closeOut, err := cmd.OpenOutput()
	if err != nil {
		return fmt.Errorf("app: open image output: %w", err)
	}

	defer func() {
		err = errors.Join(err, closeOut())
	}()

	request.Output = out

	if err = imageout.RunRequest(ctx, request, log); err != nil {
		return fmt.Errorf("app: image: %w", err)
	}

	return nil
}
