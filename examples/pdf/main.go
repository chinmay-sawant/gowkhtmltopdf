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
	"errors"
	"flag"
	"fmt"
	"os"

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
	fs := flag.NewFlagSet("pdf", flag.ContinueOnError)
	fs.Usage = usage
	pageSize := fs.String("page-size", "A4", "e.g. A4, Letter")
	orientation := fs.String("orientation", "portrait", "portrait or landscape")
	marginTop := fs.String("margin-top", "", "top margin in mm")
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

	c := gowkhtmltopdf.NewConverter()
	mustSet := func(name, value string) error {
		if err := c.Global().Set(name, value); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		return nil
	}
	if err := mustSet("size.pagesize", *pageSize); err != nil {
		return err
	}
	if err := mustSet("orientation", *orientation); err != nil {
		return err
	}
	if *marginTop != "" {
		if err := mustSet("margin.top", *marginTop); err != nil {
			return err
		}
	}

	obj := gowkhtmltopdf.NewObjectSettings().SetPage(input)
	if *localFiles {
		if err := mustSet("enablelocalfileaccess", "true"); err != nil {
			return err
		}
		if err := obj.Set("load.blocklocalfileaccess", "false"); err != nil {
			return err
		}
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
