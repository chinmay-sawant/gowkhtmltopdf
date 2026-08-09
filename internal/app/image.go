package app

import (
	"context"
	"fmt"
	"io"
	"strings"

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/convert"
	"gowkhtmltopdf/internal/imageout"
	"gowkhtmltopdf/internal/settings"
)

// RunImage is the command-facing image adapter. It keeps command parsing and
// output ownership at the application boundary while imageout remains usable
// through its compatibility adapter and library request seam.
func RunImage(ctx context.Context, cmd *cli.Command, log io.Writer) error {
	if cmd == nil {
		return errNilCommand
	}

	if ctx == nil {
		return errNilContext
	}

	// Validate the request before imageout opens the command's output sink.
	// The discard writer preserves the same explicit output invariant without
	// starting the rendering pipeline twice.
	request := convert.NewImageRequest(cmd.Global, cmd.Image, cmd.Objects, io.Discard)
	if err := request.ValidateImage(); err != nil {
		return fmt.Errorf("app: validate image: %w", err)
	}

	if !hasImageInput(cmd.Objects) {
		return fmt.Errorf("app: validate image: no input page")
	}

	if err := imageout.Run(ctx, cmd, log); err != nil {
		return fmt.Errorf("app: image: %w", err)
	}

	return nil
}

func hasImageInput(objects []settings.PdfObject) bool {
	for _, object := range objects {
		if object.IsTableOfContent {
			continue
		}

		return strings.TrimSpace(object.Page) != "" || len(object.Load.InlineHTML) > 0
	}

	return false
}
