// Command image converts one HTML file to a PNG or JPEG through the target
// ImageDocument API.
//
// Usage:
//
//	go run ./examples/image [options] input.html output.png
//
// Options:
//
//	--width <px>                  viewport width in pixels (default 1024)
//	--format <png|jpg>            output format (default png)
//	--allow-local-files           allow local files (needed for file inputs)
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

const imageArgumentCount = 2

var (
	errImageArguments    = errors.New("need exactly one input and one output file")
	errUnsupportedFormat = errors.New("unsupported format")
)

func usage() {
	fmt.Fprintln(os.Stderr, `usage: image [options] <input.html> <output.png>

options:
  --width <px>                  viewport width in pixels (default 1024)
  --format <png|jpg>            output format (default png)
  --allow-local-files           allow local files (needed for file inputs)
  --help                        show this help`)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "image:", err)
		os.Exit(1)
	}
}

func run(argv []string) error {
	flags := flag.NewFlagSet("image", flag.ContinueOnError)
	flags.Usage = usage
	widthText := flags.String("width", "", "viewport width in pixels (default 1024)")
	format := flags.String("format", "png", "output format (png or jpg)")
	allowLocalFiles := flags.Bool("allow-local-files", false, "allow local files (needed for file inputs)")

	if err := flags.Parse(argv); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return fmt.Errorf("parse flags: %w", err)
	}

	if flags.NArg() != imageArgumentCount {
		usage()

		return errImageArguments
	}

	if *format != "png" && *format != "jpg" {
		return fmt.Errorf("%w %q (png or jpg)", errUnsupportedFormat, *format)
	}

	var width int

	if *widthText != "" {
		parsed, err := strconv.Atoi(*widthText)
		if err != nil {
			return fmt.Errorf("width: %w", err)
		}

		width = parsed
	}

	var doc gowkhtmltopdf.ImageDocument
	doc.Source.File = flags.Arg(0)
	doc.Width = width
	doc.Format = *format
	doc.AllowLocalFiles = *allowLocalFiles

	output, err := os.Create(flags.Arg(1))
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer output.Close()

	if err := doc.WriteImage(context.Background(), output); err != nil {
		return fmt.Errorf("write image: %w", err)
	}

	fmt.Fprintf(os.Stdout, "image: wrote %s\n", flags.Arg(1))

	return nil
}
