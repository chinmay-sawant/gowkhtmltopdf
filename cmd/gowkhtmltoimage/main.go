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
	cmd, err := cli.Parse(argv, os.Stderr)
	if err != nil {
		switch {
		case errors.Is(err, cli.ErrHelp):
			cli.PrintHelp(os.Stdout, cli.ModeImage)
			return cli.ExitOK
		case errors.Is(err, cli.ErrVersion):
			cli.PrintVersion(os.Stdout)
			return cli.ExitOK
		case errors.Is(err, cli.ErrLicense):
			cli.PrintLicense(os.Stdout)
			return cli.ExitOK
		case errors.Is(err, cli.ErrExtHelp):
			cli.PrintExtendedHelp(os.Stdout, cli.ModeImage)
			return cli.ExitOK
		}
		fmt.Fprintf(os.Stderr, "gowkhtmltoimage: %v\n", err)
		return cli.ExitError
	}

	if err := imageout.Run(context.Background(), cmd, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "gowkhtmltoimage: %v\n", err)
		if hc, ok := err.(interface{ HttpErrorCode() int }); ok {
			return hc.HttpErrorCode()
		}
		return cli.ExitError
	}
	return cli.ExitOK
}
