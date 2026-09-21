// Command pdf converts one HTML file to PDF through the target Document API.
//
// Usage:
//
//	go run ./examples/pdf [options] input.html output.pdf
//
// Options:
//
//	--page-size <name>          e.g. A4, Letter (default A4)
//	--orientation <p|l>         portrait or landscape
//	--margin-top <mm>           top margin in mm
//	--allow-local-files         allow local files (needed for file inputs)
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"

	gowkhtmltopdf "github.com/chinmay-sawant/gowkhtmltopdf"
)

const pdfArgumentCount = 2

var errPDFArguments = errors.New("need exactly one input and one output file")

func usage() {
	fmt.Fprintln(os.Stderr, `usage: pdf [options] <input.html> <output.pdf>

options:
  --page-size <name>          e.g. A4, Letter (default A4)
  --orientation <p|l>         portrait or landscape
  --margin-top <mm>           top margin in mm
  --allow-local-files         allow local files (needed for file inputs)
  --help                      show this help`)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "pdf:", err)
		os.Exit(1)
	}
}

func run(argv []string) error {
	flags := flag.NewFlagSet("pdf", flag.ContinueOnError)
	flags.Usage = usage
	pageSize := flags.String("page-size", "A4", "e.g. A4, Letter")
	orientation := flags.String("orientation", "portrait", "portrait or landscape")
	marginTop := flags.String("margin-top", "", "top margin in mm")
	allowLocalFiles := flags.Bool("allow-local-files", false, "allow local files (needed for file inputs)")

	if err := flags.Parse(argv); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return fmt.Errorf("parse flags: %w", err)
	}

	if flags.NArg() != pdfArgumentCount {
		usage()

		return errPDFArguments
	}

	var topMargin float64

	if *marginTop != "" {
		parsed, err := strconv.ParseFloat(*marginTop, 64)
		if err != nil {
			return fmt.Errorf("margin-top: %w", err)
		}

		topMargin = parsed
	}

	var doc gowkhtmltopdf.Document
	doc.Pages = make([]gowkhtmltopdf.Page, 1)
	doc.Pages[0].Source.File = flags.Arg(0)
	doc.PageSize = *pageSize
	doc.Orientation = *orientation
	doc.Margin.Top = topMargin
	doc.AllowLocalFiles = *allowLocalFiles

	output, err := os.Create(flags.Arg(1))
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer output.Close()

	if err := doc.WritePDF(context.Background(), output); err != nil {
		return fmt.Errorf("write PDF: %w", err)
	}

	fmt.Fprintf(os.Stdout, "pdf: wrote %s\n", flags.Arg(1))

	return nil
}
