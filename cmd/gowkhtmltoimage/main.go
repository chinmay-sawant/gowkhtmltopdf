// Command gowkhtmltoimage converts HTML to an image (PNG or JPEG).
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/imageout"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	cmd, err := cli.Parse(argv)
	if err != nil {
		switch {
		case errors.Is(err, cli.ErrHelp), errors.Is(err, cli.ErrExtHelp):
			cli.PrintHelp(os.Stdout, cli.ModeImage)
			return cli.ExitOK
		case errors.Is(err, cli.ErrVersion):
			cli.PrintVersion(os.Stdout)
			return cli.ExitOK
		case errors.Is(err, cli.ErrLicense):
			cli.PrintLicense(os.Stdout)
			return cli.ExitOK
		}
		fmt.Fprintf(os.Stderr, "gowkhtmltoimage: %v\n", err)
		return cli.ExitError
	}

	// imageout.Run opens the output sink, resolves --format from the path,
	// builds convert.Request, and calls RunRequest (P1-1 adapter).
	if err := imageout.Run(context.Background(), cmd, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "gowkhtmltoimage: %v\n", err)
		return cli.ExitCode(err)
	}
	return cli.ExitOK
}
