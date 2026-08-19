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
	fs := flag.NewFlagSet("pdf", flag.ContinueOnError)
	fs.Usage = usage
	pageSize := fs.String("page-size", "A4", "e.g. A4, Letter")
	orientation := fs.String("orientation", "portrait", "portrait or landscape")
	marginTop := fs.String("margin-top", "", "top margin in mm")
	allowLocalFiles := fs.Bool("allow-local-files", false, "allow local files (needed for file inputs)")
	if err := fs.Parse(argv); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 {
		usage()
		return fmt.Errorf("need exactly one input and one output file")
	}

	var topMargin float64
	if *marginTop != "" {
		parsed, err := strconv.ParseFloat(*marginTop, 64)
		if err != nil {
			return fmt.Errorf("margin-top: %w", err)
		}
		topMargin = parsed
	}

	doc := gowkhtmltopdf.Document{
		Pages: []gowkhtmltopdf.Page{{
			Source: gowkhtmltopdf.Content{File: fs.Arg(0)},
		}},
		PageSize:        *pageSize,
		Orientation:     *orientation,
		Margin:          gowkhtmltopdf.Margin{Top: topMargin},
		AllowLocalFiles: *allowLocalFiles,
	}

	output, err := os.Create(fs.Arg(1))
	if err != nil {
		return err
	}
	defer output.Close()

	if err := doc.WritePDF(context.Background(), output); err != nil {
		return err
	}

	fmt.Printf("pdf: wrote %s\n", fs.Arg(1))
	return nil
}
