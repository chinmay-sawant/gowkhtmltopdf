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
	fs := flag.NewFlagSet("image", flag.ContinueOnError)
	fs.Usage = usage
	widthText := fs.String("width", "", "viewport width in pixels (default 1024)")
	format := fs.String("format", "png", "output format (png or jpg)")
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
	if *format != "png" && *format != "jpg" {
		return fmt.Errorf("unsupported format %q (png or jpg)", *format)
	}

	var width int
	if *widthText != "" {
		parsed, err := strconv.Atoi(*widthText)
		if err != nil {
			return fmt.Errorf("width: %w", err)
		}
		width = parsed
	}

	doc := gowkhtmltopdf.ImageDocument{
		Source:          gowkhtmltopdf.Content{File: fs.Arg(0)},
		Width:           width,
		Format:          *format,
		AllowLocalFiles: *allowLocalFiles,
	}

	output, err := os.Create(fs.Arg(1))
	if err != nil {
		return err
	}
	defer output.Close()

	if err := doc.WriteImage(context.Background(), output); err != nil {
		return err
	}

	fmt.Printf("image: wrote %s\n", fs.Arg(1))
	return nil
}
