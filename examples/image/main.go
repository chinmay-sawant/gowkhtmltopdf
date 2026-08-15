// Command image converts one HTML file to a PNG (or JPEG) image through the
// public gowkhtmltopdf library API.
//
// Usage:
//
//	go run ./examples/image [options] input.html output.png
//
// Options:
//
//	--width <px>                  viewport width in pixels (default 1024)
//	--format <png|jpg>            output format (default png)
//	--enable-local-file-access    allow local files (needed for file inputs)
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	gowkhtmltopdf "github.com/chinmay-sawant/gowkhtmltopdf"
)

func usage() {
	fmt.Fprintln(os.Stderr, `usage: image [options] <input.html> <output.png>

options:
  --width <px>                  viewport width in pixels (default 1024)
  --format <png|jpg>            output format (default png)
  --enable-local-file-access    allow local files (needed for file inputs)
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
	width := fs.String("width", "", "viewport width in pixels (default 1024)")
	format := fs.String("format", "png", "output format (png or jpg)")
	localFiles := fs.Bool("enable-local-file-access", false, "allow local files (needed for file inputs)")
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
	input, output := fs.Arg(0), fs.Arg(1)
	if *format != "png" && *format != "jpg" {
		return fmt.Errorf("unsupported format %q (png or jpg)", *format)
	}

	c := gowkhtmltopdf.NewImageConverter()
	mustSet := func(name, value string) error {
		if err := c.Set(name, value); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		return nil
	}
	if *width != "" {
		if err := mustSet("width", *width); err != nil {
			return err
		}
	}
	if err := mustSet("format", *format); err != nil {
		return err
	}
	c.AddObject(input)
	if *localFiles {
		c.Global().EnableLocalFileAccess()
		c.Object().EnableLocalFileAccess()
	}

	if err := c.Convert(context.Background()); err != nil {
		return err
	}
	if err := os.WriteFile(output, c.Output(), 0o644); err != nil {
		return err
	}
	fmt.Printf("image: wrote %s (%d bytes)\n", output, len(c.Output()))
	return nil
}
