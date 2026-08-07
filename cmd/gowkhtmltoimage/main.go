// Command gowkhtmltoimage converts HTML to an image (PNG or JPEG).
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"gowkhtmltopdf/internal/app"
	"gowkhtmltopdf/internal/cli"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	cmd, err := cli.Parse(argv, cli.ModeImage)
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

	if err := app.RunImage(context.Background(), cmd, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "gowkhtmltoimage: %v\n", err)

		return cli.ExitCode(err)
	}

	return cli.ExitOK
}
