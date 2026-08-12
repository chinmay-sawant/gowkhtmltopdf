package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Version is stamped by the build (ldflags -X) and defaults to 0.1.0-dev.
var Version = "0.1.0-dev" //nolint:gochecknoglobals // ldflags-stamped build variable

// PrintHelp writes usage text for the given Mode.
func PrintHelp(writer io.Writer, mode Mode) {
	output := "PDF"
	if mode == ModeImage {
		output = "PNG/JPEG image"
	}

	fmt.Fprintf(writer, `Name:
  gowkhtmltopdf - Convert HTML to %s with the pure-Go report renderer

Synopsis:
  gowkhtmltopdf [GLOBAL OPTIONS] [OBJECT]... <output file>

  OBJECT is one of:
    [PAGE OPTIONS] page <input url>
    [TOC OPTIONS] toc
    [COVER OPTIONS] cover <input url>
  The last positional argument is the output file; use "-" for stdout.
  "page" is optional for the first object.

Examples:
  gowkhtmltopdf --page-size A4 --orientation Landscape report.html report.pdf
  gowkhtmltopdf cover cover.html toc page chapter1.html page chapter2.html book.pdf

Description:
  Controlled HTML to %s conversion with a documented CSS subset
  (see documentation/compatibility-matrix.md). No JavaScript, browser process,
  or Qt/WebKit engine is used.

%s
`, output, output, flagList(mode))
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

	for _, n := range names {
		spec := flagTable[n]
		if spec.kind == flagBool {
			fmt.Fprintf(&buf, "  --%s\n", n)
		} else {
			fmt.Fprintf(&buf, "  --%s <value>\n", n)
		}
	}

	return buf.String()
}
