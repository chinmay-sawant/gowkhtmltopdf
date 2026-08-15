// Command generate renders the library API architecture template to PDF.
//
// Run from the repository root (also invoked by `make samples`):
//
//	go run ./testdata/golden/api
//
// Writes (overwriting if present):
//  1. output/architecture-diagram.pdf — sample viewer artifact
//
// It does not write testdata/golden/architecture-diagram.html or
// testdata/golden/api/architecture-diagram.pdf. The HTML corpus fixture is
// separate. fixture-56-architecture-diagram.html is a third, 20-page
// template. The only HTML this command reads is
// testdata/golden/api/architecture-diagram.html. Pass -output to send the
// PDF somewhere else; testdata/golden stays a source tree.
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

	gowkhtmltopdf "github.com/chinmay-sawant/gowkhtmltopdf"
)

const (
	apiDirectory    = "testdata/golden/api"
	sampleDirectory = "output"
	inputName       = "architecture-diagram.html"
	outputName      = "architecture-diagram.pdf"
	pdfFileMode     = 0o600
	wantPages       = 5
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
	output := flags.String("output", "", "PDF output path (default: output/architecture-diagram.pdf)")

	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	if flags.NArg() != 0 {
		return fmt.Errorf("%w: %v", errUnexpectedArguments, flags.Args())
	}

	input, defaultOutput, _, err := resolveTemplatePaths()
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

	if err := writeFile(*output, pdf, pdfFileMode); err != nil {
		return fmt.Errorf("write PDF %s: %w", *output, err)
	}

	if _, err := fmt.Fprintf(os.Stdout, "generated %s (%d pages, %d bytes)\n", *output, wantPages, len(pdf)); err != nil {
		return fmt.Errorf("report generation: %w", err)
	}

	return nil
}

func pageCount(pdf []byte) int {
	return bytes.Count(pdf, []byte("/Type /Page\n"))
}

func writeFile(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	if err := os.WriteFile(path, data, mode); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func uniquePaths(paths ...string) []string {
	out := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))

	for _, path := range paths {
		if path == "" {
			continue
		}

		absolute, err := filepath.Abs(path)
		if err != nil {
			// Fall back to the raw path; writeFile will surface any real error.
			absolute = path
		}

		if _, exists := seen[absolute]; exists {
			continue
		}

		seen[absolute] = struct{}{}
		out = append(out, absolute)
	}

	return out
}

func samePath(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return a == b
	}

	return absA == absB
}

// isAPITemplate reports whether path is the library-API source
// (testdata/golden/api/architecture-diagram.html), not the same-named
// corpus fixture at testdata/golden/architecture-diagram.html.
func isAPITemplate(path string) bool {
	return filepath.Base(filepath.Dir(path)) == "api" && filepath.Base(path) == inputName
}

// resolveTemplatePaths accepts invocation from the repository root, the api
// directory itself, or a compiled copy whose source directory is available.
// The returned input is absolute so the loader cannot reinterpret it as an
// HTTP host when the caller's working directory differs. repoRoot is the
// directory that contains testdata/ and output/.
func resolveTemplatePaths() (input, defaultOutput, repoRoot string, err error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", "", "", fmt.Errorf("resolve working directory: %w", err)
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
			return "", "", "", fmt.Errorf("resolve template path %q: %w", candidate, err)
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
			return "", "", "", fmt.Errorf("inspect template %q: %w", absolute, err)
		}

		if info.IsDir() {
			return "", "", "", fmt.Errorf("%w: %s", errTemplateDirectory, absolute)
		}

		if !isAPITemplate(absolute) {
			// testdata/golden/architecture-diagram.html shares the basename
			// but is a corpus fixture, not this generator's source.
			continue
		}

		apiDir := filepath.Dir(absolute)
		// template lives at <repo>/testdata/golden/api/<file>
		root := filepath.Dir(filepath.Dir(filepath.Dir(apiDir)))

		return absolute, filepath.Join(root, sampleDirectory, outputName), root, nil
	}

	return "", "", "", fmt.Errorf("%w %q; checked %s", errTemplateNotFound, inputName, strings.Join(checked, ", "))
}
