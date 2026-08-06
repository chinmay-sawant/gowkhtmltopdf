// Command gowkhtmltopdf converts HTML to PDF.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/convert"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	cmd, err := cli.Parse(argv)
	if err != nil {
		switch {
		case errors.Is(err, cli.ErrHelp), errors.Is(err, cli.ErrExtHelp):
			cli.PrintHelp(os.Stdout, cli.ModePDF)
			return cli.ExitOK
		case errors.Is(err, cli.ErrVersion):
			cli.PrintVersion(os.Stdout)
			return cli.ExitOK
		case errors.Is(err, cli.ErrLicense):
			cli.PrintLicense(os.Stdout)
			return cli.ExitOK
		}
		fmt.Fprintf(os.Stderr, "gowkhtmltopdf: %v\n", err)
		return cli.ExitError
	}

	// Sole dump home for --dump-default-toc-xsl: cmd.DumpDefaultTOCXSL (CLI).
	// Do not also read Global.DumpDefaultTOCXSL. --dump-outline is convert's job.
	if cmd.DumpDefaultTOCXSL {
		fmt.Fprint(os.Stdout, convert.DefaultTOCXSL())
		return cli.ExitOK
	}

	if cmd.Output == "" {
		fmt.Fprintln(os.Stderr, "gowkhtmltopdf: no output file specified (use '-' for stdout)")
		return cli.ExitError
	}

	// --quiet suppresses progress/info/warning output but never errors, which
	// are always printed to stderr below.
	logw := io.Writer(os.Stderr)
	if cmd.Global.Quiet {
		logw = io.Discard
	}
	if err := convert.RunPDFContext(context.Background(), cmd, logw, nil); err != nil {
		fmt.Fprintf(os.Stderr, "gowkhtmltopdf: %v\n", err)
		if hc, ok := err.(interface{ HttpErrorCode() int }); ok {
			return hc.HttpErrorCode()
		}
		return cli.ExitError
	}
	return cli.ExitOK
}
