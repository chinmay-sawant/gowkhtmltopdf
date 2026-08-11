// Command generate renders the library API architecture template to PDF.
//
// Run from the repository root:
//
//	go run ./testdata/golden/api
//
// The output is written beside the template as architecture-diagram.pdf.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	gowkhtmltopdf "gowkhtmltopdf"
)

const (
	apiDirectory = "testdata/golden/api"
	inputName    = "architecture-diagram.html"
	outputName   = "architecture-diagram.pdf"
	pdfFileMode  = 0o600
	wantPages    = 5
)

var (
	errUnexpectedArguments = errors.New("unexpected command-line arguments")
	errUnexpectedPageCount = errors.New("unexpected generated page count")
	errTemplateNotFound    = errors.New("template not found")
	errTemplateDirectory   = errors.New("template path is a directory")
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "generate:", err)
		os.Exit(1)
	}
}

func run(args []string) error { //nolint:cyclop,funlen // generator phases
	flags := flag.NewFlagSet("generate", flag.ContinueOnError)
	output := flags.String("output", "", "PDF output path")

	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	if flags.NArg() != 0 {
		return fmt.Errorf("%w: %v", errUnexpectedArguments, flags.Args())
	}

	input, defaultOutput, err := resolveTemplatePaths()
	if err != nil {
		return err
	}

	if *output == "" {
		*output = defaultOutput
	}

	converter := gowkhtmltopdf.NewConverter()

	for _, setting := range []struct {
		name  string
		value string
	}{
		{name: "size.pagesize", value: "A4"},
		{name: "margin.top", value: "0"},
		{name: "margin.right", value: "0"},
		{name: "margin.bottom", value: "0"},
		{name: "margin.left", value: "0"},
		{name: "web.background", value: "true"},
		{name: "smartshrinking", value: "false"},
	} {
		if err := converter.Global().Set(setting.name, setting.value); err != nil {
			return fmt.Errorf("set global %q: %w", setting.name, err)
		}
	}

	if err := converter.Global().Set("enablelocalfileaccess", "true"); err != nil {
		return fmt.Errorf("enable local file access: %w", err)
	}

	page := gowkhtmltopdf.NewObjectSettings().SetPage(input)
	if err := page.Set("load.blocklocalfileaccess", "false"); err != nil {
		return fmt.Errorf("allow local template: %w", err)
	}

	converter.AddObject(page)

	if err := converter.Convert(context.Background()); err != nil {
		return fmt.Errorf("convert template: %w", err)
	}

	pdf := converter.Output()
	if got := pageCount(pdf); got != wantPages {
		return fmt.Errorf("%w: %s pages = %d, want %d", errUnexpectedPageCount, input, got, wantPages)
	}

	if err := os.WriteFile(*output, pdf, pdfFileMode); err != nil {
		return fmt.Errorf("write PDF: %w", err)
	}

	if _, err := fmt.Fprintf(os.Stdout, "generated %s (%d pages, %d bytes)\n", *output, wantPages, len(pdf)); err != nil {
		return fmt.Errorf("report generation: %w", err)
	}

	return nil
}

func pageCount(pdf []byte) int {
	return bytes.Count(pdf, []byte("/Type /Page\n"))
}

// resolveTemplatePaths accepts invocation from the repository root, the api
// directory itself, or a compiled copy whose source directory is available.
// The returned input is absolute so the loader cannot reinterpret it as an
// HTTP host when the caller's working directory differs.
func resolveTemplatePaths() (string, string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("resolve working directory: %w", err)
	}

	_, sourceFile, _, sourceOK := runtime.Caller(0)
	sourceDir := ""

	if sourceOK {
		sourceDir = filepath.Dir(sourceFile)
	}

	candidates := []string{
		filepath.Join(workingDir, inputName),
		filepath.Join(workingDir, apiDirectory, inputName),
		filepath.Join(sourceDir, inputName),
	}
	checked := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))

	for _, candidate := range candidates {
		absolute, err := filepath.Abs(candidate)
		if err != nil {
			return "", "", fmt.Errorf("resolve template path %q: %w", candidate, err)
		}

		if _, exists := seen[absolute]; exists {
			continue
		}

		seen[absolute] = struct{}{}

		checked = append(checked, absolute)

		info, err := os.Stat(absolute)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}

		if err != nil {
			return "", "", fmt.Errorf("inspect template %q: %w", absolute, err)
		}

		if info.IsDir() {
			return "", "", fmt.Errorf("%w: %s", errTemplateDirectory, absolute)
		}

		return absolute, filepath.Join(filepath.Dir(absolute), outputName), nil
	}

	return "", "", fmt.Errorf("%w %q; checked %s", errTemplateNotFound, inputName, strings.Join(checked, ", "))
}
