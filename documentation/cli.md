# Command-line interface

This page documents the 0.2.4 CLI target from Phase 36.

> **Implementation status:** the target parser, help text, and root Document
> API are complete and validated. The -o/--output, --html, --url, --cover,
> --toc, and --allow-local-files examples below are the supported 0.2.4
> contract. There is no legacy compatibility mode.

The CLI and library describe the same model:

~~~text
gowkhtmltopdf argv → Document → Document.WritePDF
gowkhtmltoimage argv → ImageDocument → ImageDocument.WriteImage
~~~

The engine remains a pure-Go, no-cgo PDF engine based on HTML templates —
without any wrappers. It does not execute JavaScript and is not a browser.

## Binaries

| Binary | Target input model | Output |
|---|---|---|
| gowkhtmltopdf | One or more page files, or one explicit --html / --url source; optional cover and TOC | PDF |
| gowkhtmltoimage | Exactly one page file, --html, or --url source | PNG or JPEG |

Build from a checkout:

~~~sh
make build
./bin/gowkhtmltopdf --help
./bin/gowkhtmltoimage --help
~~~

--help, --version, and --license are terminal actions and exit without
converting. --version reports the project release from VERSION; it is not
the library's LibraryVersion compatibility identifier.

## PDF grammar

~~~text
gowkhtmltopdf [GLOBAL OPTIONS] -o OUTPUT PAGE...
gowkhtmltopdf [GLOBAL OPTIONS] -o OUTPUT --html HTML
gowkhtmltopdf [GLOBAL OPTIONS] -o OUTPUT --url URL
gowkhtmltopdf [GLOBAL OPTIONS] --cover COVER -o OUTPUT [--toc] PAGE...
~~~

-o and --output are aliases. The output path is required; -o - writes
PDF bytes to stdout. Page files are positional and are converted in order.
--html, --url, and positional page files are mutually exclusive.

A Document must have at least one renderable page. --toc adds a TOC to the
document; it does not replace a page. --cover PATH adds a cover before the
TOC and pages. The target order is always:

~~~text
cover → toc → pages
~~~

Examples:

~~~sh
# Local HTML file. Local reads are denied unless explicitly enabled.
gowkhtmltopdf --allow-local-files -o report.pdf report.html

# Inline HTML. Quote it so the shell does not interpret < and >.
gowkhtmltopdf -o hello.pdf \
  --html '<html><body><h1>Hello</h1></body></html>'

# Remote server-rendered page.
gowkhtmltopdf -o remote.pdf \
  --url https://example.test/reports/monthly

# Cover, generated TOC, and two body pages.
gowkhtmltopdf --allow-local-files \
  --cover cover.html --toc \
  -o book.pdf chapter-1.html chapter-2.html

# PDF bytes on stdout.
gowkhtmltopdf --quiet --allow-local-files -o - report.html > report.pdf
~~~

There is no implicit page, cover, or toc positional token in the target
grammar. There is also no stdin HTML shorthand: use --html or --url.

## Target PDF flags

The 0.2.4 CLI exposes named flags that map to Document fields. It does not
expose a generic --set key=value escape hatch.

| Flag | Document field / behavior |
|---|---|
| --page-size NAME | PageSize; named sizes such as A4, Letter, and Legal |
| --page-width MM, --page-height MM | WidthMM / HeightMM; both are required for a custom size |
| --orientation portrait or landscape | Orientation |
| --margin-top, --margin-right, --margin-bottom, --margin-left | Margin values in millimetres |
| --title TEXT | Title and PDF /Title |
| --copies N | Copies; must be at least 1 |
| --collate, --no-collate | Collate |
| --outline, --no-outline | Outline |
| --outline-depth N | OutlineDepth |
| --pdf-version 1.4, 1.7, or 2.0 | PDFVersion; version alone is not a conformance claim |
| --pdf-profile PROFILE | PDFProfile, for example a3a-ua1 or a4-ua2 |
| --background, --no-background | Background |
| --enable-smart-shrinking, --disable-smart-shrinking | SmartShrinking |
| --no-pdf-compression | Compression |
| --keep-relative-links | ResolveRelLinks false |
| --font-path PATH | Repeatable FontPaths entry. Prefer a **directory** (scanned to depth 2 for `.ttf`/`.otf`). A bare `.ttf`/`.otf` file is accepted as one face; other files warn and skip |
| --use-system-fonts | UseSystemFonts (opt-in OS font dirs; not Fontconfig aliases) |
| --use-metric-font-aliases | UseMetricFontAliases (opt-in Registry accept map; default off) |
| --allow-local-files | AllowLocalFiles |
| --restrict-network | Restricted network policy |
| --allow-host HOST | Network host allowlist entry |
| --quiet | Suppress informational output; errors remain visible |

Header and footer text flags map to the document-level Header and Footer,
with page-specific variants reserved for a later flag-group decision. The
supported placeholders remain [page], [topage], [frompage], [date],
[time], [title], [doctitle], [webpage], [section], and
[subsection]. [subject] expands to an empty value.

## Image grammar

~~~text
gowkhtmltoimage [GLOBAL OPTIONS] -o OUTPUT PAGE
gowkhtmltoimage [GLOBAL OPTIONS] -o OUTPUT --html HTML
gowkhtmltoimage [GLOBAL OPTIONS] -o OUTPUT --url URL
~~~

Image mode requires exactly one source. It has no cover, TOC, pages,
outlines, copies, or PDF-only flags.

~~~sh
gowkhtmltoimage --allow-local-files \
  --width 1024 --format png \
  -o invoice.png invoice.html

gowkhtmltoimage --width 800 --quality 85 --format jpg \
  -o invoice.jpg --html '<h1>Invoice</h1>'

gowkhtmltoimage --transparent --no-smart-width \
  -o badge.png badge.html
~~~

Image flags map to ImageDocument:

| Flag | Field / behavior |
|---|---|
| --width PX | Width |
| --height PX | Height |
| --format png or jpg | Format |
| --quality 1..100 | Quality; JPEG only |
| --smart-width, --no-smart-width | SmartWidth |
| --transparent | Transparent PNG canvas |
| --crop-x, --crop-y, --crop-w, --crop-h | Crop |
| --allow-local-files | AllowLocalFiles |
| --restrict-network, --allow-host | Network policy |

--transparent has no effect on JPEG other than selecting a non-transparent
canvas. Image output defaults to PNG when no format is specified.

## Sources and security

The target CLI has three explicit source forms:

| Form | Meaning |
|---|---|
| Positional existing path | Local file; subject to --allow-local-files |
| --html HTML | In-memory HTML; no URL guessing |
| --url URL | http:// or https:// document |

Local files are blocked by default, including linked CSS, images, and fonts.
Enable them only for trusted files with --allow-local-files. For untrusted
remote input, prefer --restrict-network and an explicit --allow-host
allowlist. TLS verification remains enabled; there is no --insecure flag.

The CLI runs in the calling process and is an HTTP client when given a URL.
Do not pass arbitrary user URLs into a server-side conversion command without
authorization, host policy, and resource limits. See
[THREAT-MODEL.md](THREAT-MODEL.md).

## Exit codes

The target keeps the useful HTTP distinctions from the existing CLI:

| Exit code | Meaning |
|---:|---|
| 0 | Help/version/license or successful conversion |
| 1 | Usage, validation, rendering, I/O, or unexpected failure |
| 2 | Main document returned HTTP 404 |
| 3 | Main document returned HTTP 401 |

Phase 36 owns the final parser and exit-code tests. Until that phase closes,
the binary help and phase implementation are authoritative for the current
working tree.

## Migrating from the old CLI

The old grammar:

~~~sh
gowkhtmltopdf --allow-local-files \
  cover cover.html toc page chapter.html old.pdf
~~~

becomes:

~~~sh
gowkhtmltopdf --allow-local-files \
  --cover cover.html --toc -o new.pdf chapter.html
~~~

Other common changes:

| 0.2.3 style | 0.2.4 target |
|---|---|
| Output as the final positional argument | Required -o OUTPUT / --output OUTPUT |
| page input.html | Positional input.html |
| cover cover.html | --cover cover.html |
| toc | --toc |
| inline:<html>...</html> | --html '<html>...</html>' |
| URL-looking positional input | --url https://... |
| --enable-local-file-access | --allow-local-files |
| --set key=value or dotted settings | No replacement; use named flags |

For the library migration, see
[MIGRATION-0.2.4.md](MIGRATION-0.2.4.md).
