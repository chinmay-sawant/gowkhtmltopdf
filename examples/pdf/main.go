// Command pdf converts one HTML file to PDF through the public
// gowkhtmltopdf library API.
//
// Usage:
//
//	go run ./examples/pdf [options] input.html output.pdf
//
// Options:
//
//	--page-size <name>            e.g. A4, Letter (default A4)
//	--orientation <p|l>           portrait or landscape
//	--margin-top <mm>             top margin in mm
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
	fmt.Fprintln(os.Stderr, `usage: pdf [options] <input.html> <output.pdf>

options:
  --page-size <name>           e.g. A4, Letter (default A4)
  --orientation <p|l>          portrait or landscape
  --margin-top <mm>            top margin in mm
  --enable-local-file-access   allow local files (needed for file inputs)
  --help                       show this help`)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "pdf:", err)
		os.Exit(1)
	}
}

func run(argv []string) error {
	var (
		pageSize    = "A4"
		orientation = "portrait"
		marginTop   = ""
		localFiles  = false
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
		case strings.HasPrefix(arg, "--page-size="):
			pageSize = strings.TrimPrefix(arg, "--page-size=")
		case arg == "--page-size" && i+1 < len(argv):
			i++
			pageSize = argv[i]
		case strings.HasPrefix(arg, "--orientation="):
			orientation = strings.TrimPrefix(arg, "--orientation=")
		case arg == "--orientation" && i+1 < len(argv):
			i++
			orientation = argv[i]
		case strings.HasPrefix(arg, "--margin-top="):
			marginTop = strings.TrimPrefix(arg, "--margin-top=")
		case arg == "--margin-top" && i+1 < len(argv):
			i++
			marginTop = argv[i]
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

	c := gowkhtmltopdf.NewConverter()
	c.Global().Set("size.pagesize", pageSize)
	c.Global().Set("orientation", orientation)
	if marginTop != "" {
		c.Global().Set("margin.top", marginTop)
	}

	obj := gowkhtmltopdf.NewObjectSettings().SetPage(input)
	if localFiles {
		c.Global().Set("enablelocalfileaccess", "true")
		obj.Set("load.blocklocalfileaccess", "false")
	}
	c.AddObject(obj)

	if err := c.Convert(context.Background()); err != nil {
		return err
	}
	if err := os.WriteFile(output, c.Output(), 0o644); err != nil {
		return err
	}
	fmt.Printf("pdf: wrote %s (%d bytes)\n", output, len(c.Output()))
	return nil
}
