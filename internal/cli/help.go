package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Version is the project release stamped by the build
// (ldflags -X github.com/chinmay-sawant/gowkhtmltopdf/internal/cli.Version=$(cat VERSION)).
// The unstamped default matches the VERSION file so tests and local
// `go test`/`go run` agree with the release number. It is not
// LibraryVersion, which is the upstream wkhtmltopdf compatibility id.
var Version = "0.2.5" //nolint:gochecknoglobals // ldflags-stamped build variable

// PrintHelp writes usage text for the given Mode.
func PrintHelp(writer io.Writer, mode Mode) {
	output := "PDF"
	command := "gowkhtmltopdf"
	examples := fmt.Sprintf(
		"  %s --page-size A4 --orientation Landscape -o report.pdf report.html\n"+
			"  %s -o book.pdf --cover cover.html --toc chapter1.html chapter2.html\n"+
			"  %s --html '<html><body><h1>Report</h1></body></html>' -o report.pdf",
		command, command, command,
	)

	if mode == ModeImage {
		output = "PNG/JPEG image"
		command = "gowkhtmltoimage"
		examples = fmt.Sprintf(
			"  %s --width 800 --format png -o report.png report.html\n"+
				"  %s --url https://example.test/preview -o preview.png\n"+
				"  %s --html '<html><body><h1>Preview</h1></body></html>' -o preview.png",
			command, command, command,
		)
	}

	fmt.Fprintf(writer, `Name:
  %s - Convert HTML to %s with the pure-Go report renderer

Synopsis:
  %s [GLOBAL OPTIONS] -o <output> [PAGE...]
  %s [GLOBAL OPTIONS] --html <html> -o <output>
  %s [GLOBAL OPTIONS] --url <url> -o <output>

Examples:
%s

Description:
  Controlled HTML to %s conversion with a documented CSS subset
  (see documentation/compatibility-matrix.md). No JavaScript, browser process,
  or Qt/WebKit engine is used.

%s
`, command, output, command, command, command, examples, output, flagList(mode))
}

// PrintVersion writes the version banner.
func PrintVersion(w io.Writer) {
	fmt.Fprintf(w, "Name: gowkhtmltopdf\nVersion: %s\n", Version)
}

// PrintLicense writes the license banner.
func PrintLicense(w io.Writer) {
	fmt.Fprintf(w, `gowkhtmltopdf %s
Copyright (C) 2026 gowkhtmltopdf contributors

This program is an independent, clean-room reimplementation of the
wkhtmltopdf command-line interface and is licensed under the MIT License.
The original wkhtmltopdf is Copyright (C) 2010-2020 wkhtmltopdf authors and
is licensed under the LGPL.

See LICENSE for the full text of the MIT License.
`, Version)
}

// flagList renders a "--name" list filtered by Mode.
func flagList(mode Mode) string {
	names := make([]string, 0, len(flagTable))

	for name, spec := range flagTable {
		if spec.mod&mode == 0 {
			continue
		}

		names = append(names, name)
	}

	sort.Strings(names)

	var buf strings.Builder

	for _, name := range names {
		spec := flagTable[name]

		if isDocFlagName(name) {
			continue
		}

		if name == "output" {
			fmt.Fprintln(&buf, "  -o, --output <path>")

			continue
		}

		if spec.kind == flagBool {
			fmt.Fprintf(&buf, "  --%s\n", name)
		} else {
			fmt.Fprintf(&buf, "  --%s <value>\n", name)
		}
	}

	return buf.String()
}
