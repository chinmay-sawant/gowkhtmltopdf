// Command gowkhtmltopdf converts HTML to PDF.
package main

import (
	"errors"
	"fmt"
	"os"

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/convert"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	cmd, err := cli.Parse(argv, os.Stderr)
	if err != nil {
		switch {
		case errors.Is(err, cli.ErrHelp):
			cli.PrintHelp(os.Stdout, cli.ModePDF)
			return cli.ExitOK
		case errors.Is(err, cli.ErrVersion):
			cli.PrintVersion(os.Stdout)
			return cli.ExitOK
		case errors.Is(err, cli.ErrLicense):
			cli.PrintLicense(os.Stdout)
			return cli.ExitOK
		case errors.Is(err, cli.ErrExtHelp):
			cli.PrintExtendedHelp(os.Stdout, cli.ModePDF)
			return cli.ExitOK
		}
		fmt.Fprintf(os.Stderr, "gowkhtmltopdf: %v\n", err)
		return cli.ExitError
	}

	if cmd.DumpDefaultTOCXSL {
		fmt.Fprint(os.Stdout, convert.DefaultTOCXSL())
		return cli.ExitOK
	}

	if cmd.Output == "" {
		fmt.Fprintln(os.Stderr, "gowkhtmltopdf: no output file specified (use '-' for stdout)")
		return cli.ExitError
	}

	if err := convert.RunPDF(cmd, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "gowkhtmltopdf: %v\n", err)
		if hc, ok := err.(interface{ HttpErrorCode() int }); ok {
			return hc.HttpErrorCode()
		}
		return cli.ExitError
	}
	return cli.ExitOK
}
