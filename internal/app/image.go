package app

import (
	"context"
	"fmt"
	"io"

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/imageout"
)

// RunImage is the command-facing image adapter. It keeps command parsing and
// output ownership at the application boundary while imageout remains usable
// through its compatibility adapter and library request seam.
func RunImage(ctx context.Context, cmd *cli.Command, log io.Writer) error {
	if cmd == nil {
		return fmt.Errorf("app: nil command")
	}
	return imageout.Run(ctx, cmd, log)
}
