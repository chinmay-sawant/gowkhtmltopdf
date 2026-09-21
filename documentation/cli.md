# Command-line interface

`gowkhtmltopdf` and `gowkhtmltoimage` share one document model with the Go
library. `VERSION` is 0.2.6. The flags below are what the binaries accept.
There is no legacy `page` / `cover` / `toc` object grammar.

The CLI and library describe the same model:

~~~text
gowkhtmltopdf argv → Document → Document.WritePDF
gowkhtmltoimage argv → ImageDocument → ImageDocument.WriteImage
~~~

The engine is a pure-Go, no-cgo PDF engine for HTML templates. It does not
execute JavaScript, and it is not a browser.

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

There is no `page`, `cover`, or `toc` positional token. There is also no
stdin HTML shorthand: use `--html` or `--url`.

## PDF flags

Named flags map to `Document` fields. There is no `--set key=value` escape
hatch.

| Flag | Document field / behavior |
|---|---|
| --page-size NAME | PageSize; named sizes such as A4, Letter, and Legal |
| --page-width MM, --page-height MM | WidthMM / HeightMM; both are required for a custom size |
| --orientation portrait or landscape | Orientation |
| --margin-top, --margin-right, --margin-bottom, --margin-left | Margin values in millimetres |
| --title TEXT | Title and PDF /Title |
| --copies N | Copies; must be at least 1 |
| --collate, --no-collate | Collate |
| --outline, --no-outline | Outline; default on (`internal/settings/settings.go` `Outline: true`) |
| --outline-depth N | OutlineDepth; default 4 |
| --pdf-version 1.4, 1.7, or 2.0 | PDFVersion; version alone is not a conformance claim |
| --pdf-profile PROFILE | PDFProfile, for example a3a-ua1 or a4-ua2 |
| --background, --no-background | Background |
| --enable-smart-shrinking, --disable-smart-shrinking | SmartShrinking |
| --no-pdf-compression | Compression |
| --keep-relative-links | ResolveRelLinks false |
| --font-path PATH | Repeatable FontPaths entry |
| --use-system-fonts | UseSystemFonts |
| --allow-local-files | AllowLocalFiles |
| --restrict-network | Restricted network policy |
| --allow-host HOST | Network host allowlist entry |
| --quiet | Suppress informational output; errors remain visible |
| --header-left, --header-center, --header-right | Document header text |
| --footer-left, --footer-center, --footer-right | Document footer text |
| --header-line, --footer-line | Rule under the header or over the footer |
| --header-html, --footer-html | HTML URL for the header or footer |
| --header-font-name, --header-font-size, --header-spacing | Header face, size, and spacing (footer twins use the `footer-` prefix) |
| --toc | Insert a generated table of contents |
| --simplify-dom, --no-simplify-dom | Opt-in landmark chrome strip; default off |
| --simplify-dom-profile NAME | `mediawiki` adds MediaWiki selectors; empty keeps landmarks only |
| --print-link-underline | Opt-in underline on `a[href]` after the cascade; default off |
| --zoom FLOAT | Layout scale |
| --timeout DURATION | HTTP response timeout |
| --media-type, --print-media-type, --no-print-media-type | Media used for the cascade; PDF default stays `print` |

Header and footer flags apply to the whole document (`hfFlag` in
`internal/cli/flags.go`). Placeholders: `[page]`, `[topage]`, `[frompage]`,
`[date]`, `[time]`, `[title]`, `[doctitle]`, `[webpage]`, `[section]`,
`[subsection]`. `[subject]` expands empty. Custom substitutions:
`--replace key value`.

## URL mode, chrome strip, simplify-dom

`--url` fetches one `http` or `https` document. Connect timeout is 30 s,
response timeout is 60 s (`--timeout` overrides the response timeout),
redirects stop at 10, and the body cap is 100 MiB. TLS verification stays
on. There is no `--insecure`.

Live public pages stay exploratory until Phase 21 acceptance against
vendored fixtures. See [fidelity.md](fidelity.md#arbitrary-websites-phase-21).

`--simplify-dom` is off by default. Turn it on to hide landmark chrome
through `prepare.SimplifyChromeCSS` (`internal/convert/prepare/simplify.go`).
`--simplify-dom-profile=mediawiki` adds MediaWiki selectors on top of those
landmarks. The flag does not run JavaScript.

`--print-link-underline` is off by default. Turn it on to underline
`a[href]` after the cascade. With the flag off, `text-decoration: none`
on a link stays off.

A raw Wikipedia smoke (no chrome strip) is:

```sh
./bin/gowkhtmltopdf --use-system-fonts --zoom 0.666667 \
  --url 'https://en.wikipedia.org/wiki/Ana_de_Armas' \
  -o output/wiki-ana-de-armas.pdf
```

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
| --zoom FLOAT | Zoom factor applied to layout; values below 1 shrink the document to fit the raster budget |
| --format png or jpg | Format |
| --quality 1..100 | Quality; JPEG only |
| --smart-width, --no-smart-width | SmartWidth |
| --transparent | Transparent PNG canvas |
| --crop-x, --crop-y, --crop-w, --crop-h | Crop |
| --allow-local-files | AllowLocalFiles |
| --restrict-network, --allow-host | Network policy |

--transparent has no effect on JPEG other than selecting a non-transparent
canvas. Image output defaults to PNG when no format is specified.

## Image raster budget

Image mode renders one canvas, so it enforces a hard envelope: each side
of the output image is at most 8,192 CSS px, and the final image is at most
16,777,216 CSS pixels (16M). Painting runs at 2x supersampling, so the
internal caps are 16,384 px per side, 67,108,864 px (64M), and 256 MiB of
NRGBA backing bytes.

A document that crosses the envelope fails with a resource-budget error
("imageout: raster exceeds resource budget") that reports the size that
overflowed. There is no tiling and no automatic zoom fallback. To fit one,
lower --zoom (or the library ImageDocument.Zoom field); for example the
golden template complex-css is about 11,208 CSS px tall and fits at
--zoom 0.73 or lower. Lowering --width helps when the width is what pushes
the canvas over. Four golden templates cross the envelope at 1x:
complex-css, font-examples, fixture-56, and fixture-60. PDF conversion of
the same document is not subject to this cap.

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

`cli.ExitCode` (`internal/cli/cli.go`) maps the result:

| Exit code | Meaning |
|---:|---|
| 0 | Help, version, license, or a successful conversion |
| 1 | Usage, validation, rendering, I/O, or any other error |
| 2 | Main document returned HTTP 404 |
| 3 | Main document returned HTTP 401 |

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

| 0.2.3 style | Current CLI |
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
