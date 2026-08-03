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
	"fmt"
	"os"
	"strings"

	gowkhtmltopdf "gowkhtmltopdf"
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
	var (
		width      = ""
		format     = "png"
		localFiles = false
	)
	var inputs []string
	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		switch {
		case arg == "--help" || arg == "-h":
			usage()
			return nil
		case arg == "--enable-local-file-access":
			localFiles = true
		case strings.HasPrefix(arg, "--width="):
			width = strings.TrimPrefix(arg, "--width=")
		case arg == "--width" && i+1 < len(argv):
			i++
			width = argv[i]
		case strings.HasPrefix(arg, "--format="):
			format = strings.TrimPrefix(arg, "--format=")
		case arg == "--format" && i+1 < len(argv):
			i++
			format = argv[i]
		case strings.HasPrefix(arg, "-"):
			return fmt.Errorf("unknown option %q", arg)
		default:
			inputs = append(inputs, arg)
		}
	}
	if len(inputs) != 2 {
		usage()
		return fmt.Errorf("need exactly one input and one output file")
	}
	input, output := inputs[0], inputs[1]
	if format != "png" && format != "jpg" {
		return fmt.Errorf("unsupported format %q (png or jpg)", format)
	}

	c := gowkhtmltopdf.NewImageConverter()
	if width != "" {
		c.Set("width", width)
	}
	c.Set("format", format)
	if localFiles {
		c.Global().Set("enablelocalfileaccess", "true")
	}
	c.AddObject(input)
	if localFiles {
		c.Object().Set("load.blocklocalfileaccess", "false")
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
