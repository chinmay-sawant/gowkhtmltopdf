# Command-line interface

gowkhtmltopdf is a pure-Go, no-cgo HTML→PDF / HTML→image engine for
**controlled reports**. It is **not a browser**. There is **no JavaScript**
(`<script>` is stripped at load). Unknown wkhtmltopdf flags are **errors**,
not silent no-ops.

This page is the user-facing CLI contract. The per-feature support table is
[compatibility-matrix.md](compatibility-matrix.md). Library callers should
read [library-api.md](library-api.md) instead of wrapping these binaries.

## Two binaries

One parser, two modes (`internal/cli`). Flags that do not belong to a mode
fail at parse time (`option --… is not supported in pdf/image mode`).

| Binary | Build | What it writes |
|--------|-------|----------------|
| `gowkhtmltopdf` | `make build` → `bin/gowkhtmltopdf` | Multi-object PDF |
| `gowkhtmltoimage` | `make build` → `bin/gowkhtmltoimage` | One PNG or JPEG |

`--help`, `--version`, `--license`, and `--extended-help` always print the
product name **`gowkhtmltopdf`**, even from the image binary. `--extended-help`
is the same text as `--help`.

```sh
gowkhtmltopdf --version    # Name: gowkhtmltopdf / Version: (VERSION file)
gowkhtmltopdf --license    # MIT + attribution; also -L
gowkhtmltopdf --help
```

`-h` / `--help`, `-V` / `--version`, `-L` / `--license`, and `-E` /
`--extended-help` print to stdout and exit **0**. They do not convert.

`--` ends option parsing. Everything after it is positional (object keywords,
inputs, output).

## PDF grammar

```text
gowkhtmltopdf [GLOBAL OPTIONS] [OBJECT]... <output>
  OBJECT:
    [PAGE OPTIONS]  page <input>
    [TOC OPTIONS]   toc
    [COVER OPTIONS] cover <input>
```

Rules:

- The **last positional argument is the output path**. Use `-` for stdout.
  Omitting output is an error (`no output file specified`).
- `page` is optional for implicit pages: every leftover positional except
  the last one becomes a page object.
- A document must contain **at least one non-TOC object with a page input**.
  `toc` alone is rejected (`you need to specify at least one input file`).
- `cover` is a page with `IsCover`: no outline entry, empty header/footer
  (it does **not** inherit global HF).
- `toc` does **not** consume pending page-scoped flags (`UseOutline=false`).
- Pair flags (`--cookie`, `--custom-header`, `--post`, `--replace`)
  **immediately create** a page object. See [Pitfalls](#pitfalls).

### Inputs (not stdin)

Each `<input>` is one of:

| Form | Meaning |
|------|---------|
| Existing local path | Loaded as `file://…` (still subject to the [local-file ACL](#local-files)) |
| `http://` / `https://` | Fetched by the process (see [Remote URL security](#remote-url-security)) |
| `file://` | Local file URL (ACL applies; host must be empty or `localhost`) |
| `data:…` | Inline document |
| `inline:…` | In-memory HTML (prefix kept as the synthetic base) |
| Token starting with `<` | Treated as raw HTML (`inline:<html>…`) |

**There is no stdin HTML.** `-` as an *input* is not “read stdin”. If a file
named `-` exists in the working directory it is that file; otherwise
`GuessURL` turns the token into `http://-`. `-` as *output* is stdout.

A relative path that **does not exist** is not an error at parse time. The
loader treats it as `http://<token>` (so `missing.html` becomes
`http://missing.html`). Prefer an existing path, `./file.html`, or an
explicit `file://` / `http(s)://` URL.

### PDF examples

```sh
# Implicit first page (local file: ACL opt-in)
gowkhtmltopdf --enable-local-file-access report.html report.pdf

# Explicit page keyword (page-scoped flags after `page`)
gowkhtmltopdf page --enable-local-file-access --zoom 1.1 report.html report.pdf

# Page size, orientation, margins (long form — see -L pitfall)
gowkhtmltopdf --page-size A4 --orientation Landscape \
  --margin-top 15mm --margin-left 12mm report.html out.pdf

# Headers / footers / outline (outline is already on, depth 4)
gowkhtmltopdf \
  --header-left "Report" --header-right "[page]/[topage]" \
  --footer-center "[section]" --footer-line \
  --outline-depth 2 --title "Invoice Report" \
  --enable-local-file-access chapter.html book.pdf

# TOC + body (TOC appearance flags BEFORE `toc`)
gowkhtmltopdf --enable-local-file-access \
  --toc-header-text "Contents" --disable-dotted-lines \
  toc page body.html book.pdf

# Cover + TOC + chapters
gowkhtmltopdf --enable-local-file-access \
  cover cover.html toc page ch1.html page ch2.html book.pdf

# PDF to stdout
gowkhtmltopdf --enable-local-file-access report.html -

# Inline HTML (quoted so the shell does not eat `<`)
gowkhtmltopdf 'inline:<html><body><h1>Hi</h1></body></html>' out.pdf
gowkhtmltopdf '<html><body><h1>Hi</h1></body></html>' out.pdf

# Remote URL (compatible network policy by default)
gowkhtmltopdf 'https://example.com/report.html' out.pdf

# Decent-print attempt (opt-in chrome strip; not browser parity)
gowkhtmltopdf --simplify-dom --use-system-fonts --timeout 60 \
  --load-error-handling ignore \
  'https://example.com/article' article.pdf
```

`--dump-default-toc-xsl` is a **terminal** action: no input, no output, no
`--dump-outline`. It prints the built-in TOC stylesheet description to
stdout and exits 0.

`--dump-outline` is a **boolean**. It writes wkhtmltopdf-shaped outline XML
to **stdout**. It cannot be combined with PDF output `-` (one stream cannot
be both PDF and XML). Write the PDF to a file:

```sh
gowkhtmltopdf --dump-outline --enable-local-file-access report.html report.pdf
# report.pdf = PDF, stdout = <?xml … <outline xmlns="http://wkhtmltopdf.org/outline">
```

## Image grammar

```text
gowkhtmltoimage [OPTIONS] <input> <output>
gowkhtmltoimage [OPTIONS] page <input>            # output omitted → stdout
gowkhtmltoimage [OPTIONS] page <input> -          # stdout
```

The same object grammar is parsed, then the image adapter requires
**exactly one** input object. `cover` / `toc` / a second `page` fail
validation (`app: multiple image objects` or “you need to specify at least
one input file”).

If the output positional is omitted (`page <input>` with nothing after it)
or is `-`, pixels go to **stdout**. `--format` wins; otherwise the output
extension is sniffed (`.jpg` / `.jpeg` → JPEG, everything else including
empty/stdout → PNG).

`--allow` is **not registered** in image mode. Use
`--enable-local-file-access` (and `--allow-host` / `--restrict-network` for
HTTP). `--page-size`, headers, TOC, outline, and other PDF-only flags are
rejected.

### Image examples

```sh
# Default viewport 1024 CSS px, smart-width on, PNG from extension
gowkhtmltoimage --enable-local-file-access report.html report.png

# Fixed width, content height, JPEG
gowkhtmltoimage --width 800 --format jpg --quality 80 \
  --enable-local-file-access report.html report.jpg

# Transparent PNG (ignored for JPEG — white canvas + warning)
gowkhtmltoimage --transparent --no-smart-width --width 1024 \
  --enable-local-file-access badge.html badge.png

# Crop after raster (pixels)
gowkhtmltoimage --crop-x 0 --crop-y 0 --crop-w 400 --crop-h 300 \
  --enable-local-file-access report.html crop.png

# Pixels on stdout
gowkhtmltoimage --enable-local-file-access page report.html > report.png
```

| Flag | Default | Notes |
|------|---------|-------|
| `--width` | 1024 | Viewport width in CSS pixels; `<= 0` also means 1024 |
| `--height` | 0 | Minimum canvas height; **0 = content height** |
| `--crop-x` / `--crop-y` / `--crop-w` / `--crop-h` | unset (`-1`) | Applied after raster; all four must be set to crop |
| `--format` | sniff / PNG | `png` or `jpg` (`jpeg` accepted) |
| `--quality` | 94 | JPEG only (1–100); PNG ignores it |
| `--transparent` | off | PNG alpha; JPEG warns and uses white |
| `--smart-width` / `--no-smart-width` | on | Grow the viewport (×1.5, bounded) until content fits |

Image text uses the same TTF outline raster path as PDF (coverage AA).

## Multi-object documents

Objects are stored in argv order and painted in that order (TOC pages are
then prepended so page numbers include the TOC).

| Keyword | Pending page flags | Header/footer | Outline |
|---------|--------------------|---------------|---------|
| `page` | Consumes pending into this object | Inherits global HF unless object-level HF flags were set after this keyword | Included (default) |
| `cover` | Consumes pending | **Wiped** (empty HF, `HeaderSet`/`FooterSet` true — global HF is not inherited) | Excluded |
| `toc` | **Not consumed** (pending waits for the next real page/cover) | Inherits global HF | `UseOutline=false`; TOC is generated from body headings |

```sh
# Two implicit pages: a.html then b.html
gowkhtmltopdf --enable-local-file-access a.html b.html book.pdf

# Per-page flags after each `page`
gowkhtmltopdf \
  --enable-local-file-access \
  page --zoom 1.2 ch1.html \
  page --zoom 1.0 ch2.html \
  book.pdf

# Cover does not show the global footer
gowkhtmltopdf --footer-center "[page]/[topage]" \
  --enable-local-file-access \
  cover cover.html page body.html book.pdf
```

`--copies N` (default 1, max 1000) and `--collate` / `--no-collate`
(default collate on) repeat the finished document. `--page-offset` is added
to `[page]` / TOC page numbers.

The engine also caps objects (10 000) and pages (100 000). Those limits are
not parser errors; they fail at convert time.

## Flag groups

Only flags registered in `internal/cli/flags.go` exist. Values may be
`--flag value` or `--flag=value`. Boolean flags accept
`true` / `1` / `yes` / `on` and `false` / `0` / `no` / `off`, and most can
be negated as `--no-<flag>`. A few flags **ignore** the boolean value — see
[Pitfalls](#pitfalls).

### Documentation flags (both binaries)

| Flag | Short | Effect |
|------|-------|--------|
| `--help` | `-h` | Usage + mode-filtered flag list; exit 0 |
| `--extended-help` | `-E` | **Same text as `--help`** |
| `--version` | `-V` | `Name: gowkhtmltopdf` + version; exit 0 |
| `--license` | `-L` | License banner; exit 0 |

**`-L` is license, not margin-left.** Use `--margin-left`.

### PDF page / document

Defaults: A4, portrait, 10 mm margins, 1 copy, collate on, compression on,
smart-shrinking on, background on, outline on (depth 4).

| Flag | Short | Mode | Notes |
|------|-------|------|-------|
| `--quiet` | `-q` | Both | Suppress info/warning on stderr; errors still print |
| `--collate` / `--no-collate` | | PDF | Default on |
| `--copies` | `-c` | PDF | Integer ≥ 1 |
| `--orientation` | `-O` | PDF | `Portrait` or `Landscape` (case-insensitive) |
| `--page-size` | `-s` | PDF | Named size (below). Custom size: `--page-width` **and** `--page-height` |
| `--page-width` / `--page-height` | | PDF | Millimetres (suffix optional). Both must be > 0 to override `--page-size` |
| `--margin-top` | `-T` | PDF | Millimetres (e.g. `15` or `15mm`) |
| `--margin-bottom` | `-B` | PDF | |
| `--margin-left` | **none** | PDF | **Not `-L`** |
| `--margin-right` | `-R` | PDF | |
| `--grayscale` | `-g` | PDF | Whole-document grayscale |
| `--title` | `-t` | PDF | PDF `/Title` **and** `[title]`. HTML `<title>` is `[doctitle]` only |
| `--no-pdf-compression` | | PDF | Uncompressed streams |
| `--page-offset` | | PDF | Added to `[page]` and TOC numbers |
| `--enable-smart-shrinking` | | PDF | Default is already on |
| `--disable-smart-shrinking` | | PDF | There is **no** bare `--smart-shrinking` |

Named `--page-size` values (case-insensitive): `A0`–`A6`, `B0`–`B6`,
`C5E`, `Comm10E`, `DLE`, `Executive`, `Folio`, `Ledger`, `Legal`,
`Letter`, `Tabloid`.

### Load, auth, and links

| Flag | Mode | Scope | Notes |
|------|------|-------|-------|
| `--enable-local-file-access` | Both | Global + first/current page | Dual-write. See [Local files](#local-files) |
| `--disable-local-file-access` | Both | Global + page | Always blocks (ignores bool value) |
| `--allow <prefix>` | **PDF only** | Global | Repeatable ACL prefix. Missing in image mode |
| `--restrict-network` | Both | Global | Restricted policy (private + cross-host redirects denied; `http`/`https` only) |
| `--allow-host <host>` | Both | Global | Repeatable exact or `*.example.com` allowlist. **Sets the explicit policy on.** Prefer combining with `--restrict-network` |
| `--proxy <url>` | Both | **Global live** | Absolute `http://` or `https://` proxy URL. Object `load.proxy` is stored and **ignored** |
| `--username` / `--password` | Both | Page | HTTP basic auth |
| `--timeout <sec>` | Both | Page | HTTP **response** timeout. `0` / unset = 60 s. Connect is always 30 s |
| `--zoom <factor>` | Both | Page | Operator layout scale (not stylesheet emulation). `0` / unset = 1 |
| `--load-error-handling` | Both | Global + page | `abort` (default) / `skip` / `ignore` |
| `--cookie <name> <value>` | Both | Page | Pair flag — **creates a page object** |
| `--custom-header <name> <value>` | Both | Page | Pair flag — creates a page object |
| `--post <name> <value>` | Both | Page | Repeatable form field; pair flag — creates a page object |
| `--external-links` / `--no-external-links` | PDF | Page | URI annotations for `http(s)` hrefs |
| `--internal-links` / `--no-internal-links` | PDF | Page | `#id` GoTo when geometry exists |
| `--resolve-relative-links` | PDF | Global | Default **on**: resolve relative `href` against the page URL |
| `--keep-relative-links` | PDF | Global | Forces resolve off. **Ignores** the bool value |

`--load-error-handling`:

| Value | Main document |
|-------|----------------|
| `abort` | Fail the conversion. HTTP 404 → exit 2, HTTP 401 → exit 3 |
| `skip` | Omit that object (warning) |
| `ignore` | Continue with the error response (body may be empty) |

Failed CSS `<link>` and image **subresources** are isolated (warning +
continue). There is no `--images` CLI flag; images stay on unless a library
caller sets `web.images=false`.

### Web / media

| Flag | Mode | Default | Notes |
|------|------|---------|-------|
| `--simplify-dom` / `--no-simplify-dom` | Both | **off** | Opt-in chrome strip. See [URL mode](#url-mode--chrome-strip---simplify-dom) |
| `--simplify-dom-profile` | Both | `""` | `""` = landmarks only; `mediawiki` (also `wiki` / `mw`) adds `#mw-navigation` / `.mw-jump-link` |
| `--print-link-underline` | Both | off | After cascade, underline `a[href]`. Author `text-decoration` wins otherwise |
| `--print-media-type` / `--no-print-media-type` | Both | PDF already lays out as `print`; image default is `screen` | `--print-media-type` forces print. `--no-print-media-type` only clears that override |
| `--media-type` | Both | | `print` or `screen` (object/global). Print-media-type override wins |
| `--background` / `--no-background` | Both | on | Body background paint. `--no-background` **ignores** the bool value |

### Headers and footers (PDF)

Set **before any object keyword** → stored on the global settings and
inherited by every non-cover object. Set **after** `page` / `toc` → that
object only. `cover` always has empty HF.

| Flag | Notes |
|------|-------|
| `--header-left` / `--header-center` / `--header-right` | Text (placeholders allowed) |
| `--footer-left` / `--footer-center` / `--footer-right` | |
| `--header-font-name` / `--footer-font-name` | Stored; text HF currently paints with embedded Liberation Sans |
| `--header-font-size` / `--footer-font-size` | Points (default 12) |
| `--header-spacing` / `--footer-spacing` | Extra band space |
| `--header-line` / `--footer-line` | Separator at the content edge |
| `--header-html` / `--footer-html` | **URL or path**, not raw markup. Nested child layout, clipped to the reserved margin band |
| `--replace <token> <value>` | Pair flag — creates a page object. Literal substring replace **before** `[page]`-style expansion |

`--header-html testdata/golden/fixture-36-header.html` is resolved like a
top-level page (CWD / absolute / `http(s)`), under the same ACL. A value
that looks like HTML (`<…`) is **ignored** with a warning.

HTML HF supports a body CSS subset (including flex/grid/images and local
`@font-face` under the ACL). `#id` links go to **body** destinations; ids
inside the HF tree are not destinations.

### TOC (PDF)

Put appearance flags **before** the `toc` keyword. Boolean TOC fields
**OR-merge** with the global defaults — a `false` set on the TOC object
cannot turn off a global `true`. See [TOC and outline](#toc-and-outline).

| Flag | Default | Notes |
|------|---------|-------|
| `--toc-header-text` | `Table of Contents` | Caption |
| `--toc-text-size-shrink` | `0.8` | Entry font scale |
| `--toc-level-indentation` | `1em` | Per heading level |
| `--disable-dotted-lines` | dotted **on** | Must be before `toc` to win the OR merge |
| `--disable-toc-links` | forward links **off** | Clears forward links |
| `--toc-forward-links` | off | Wrap TOC entries in `#anchor` links |
| `--toc-back-links` | off | Back-links from headings to the TOC |
| `--xsl-style-sheet` | unused | **Does not run XSLT.** Warns and uses the built-in Go template |

### Outline (PDF)

| Flag | Default | Notes |
|------|---------|-------|
| `--outline` / `--no-outline` | **on** | PDF `/Outlines` from `h1`–`h6` |
| `--outline-depth` | 4 | Heading depth |
| `--dump-outline` | off | **Boolean.** XML to stdout. Not a path. Cannot combine with PDF `-` |
| `--dump-default-toc-xsl` | off | Terminal dump; no operands |

### Fonts (both)

| Flag | Default | Notes |
|------|---------|-------|
| `--font-path <dir>` | none | Repeatable extra TTF/OTF directories |
| `--use-system-fonts` | **off** | Opt-in OS font scan (determinism) |

Short notes and CJK/Arabic limits: [fonts.md](fonts.md).

### Image-only

See [Image grammar](#image-grammar). `--width`, `--height`, `--crop-*`,
`--format`, `--quality`, `--transparent`, `--smart-width`,
`--no-smart-width`.

### Short flags

| Short | Long |
|-------|------|
| `-q` | `--quiet` |
| `-g` | `--grayscale` |
| `-O` | `--orientation` |
| `-s` | `--page-size` |
| `-T` | `--margin-top` |
| `-B` | `--margin-bottom` |
| `-R` | `--margin-right` |
| `-c` | `--copies` |
| `-t` | `--title` |
| `-h` | `--help` |
| `-V` | `--version` |
| `-L` | `--license` (**not** `--margin-left`) |
| `-E` | `--extended-help` |

Short flags do not take `--flag=value` form and do not cluster (`-qg` is
`unknown option -qg`).

## Local files

Local paths and `file://` are **denied by default**
(`load.blocklocalfileaccess=true`). A read is allowed when:

1. `--allow <prefix>` matches the real, symlink-resolved path (PDF only;
   independent of the enable/block pair), **or**
2. **Both** the global enable is on **and** this object’s block flag is off.

`--enable-local-file-access` sets the global enable **and** unblocks the
**current** page (or, if no object keyword has been seen, the **first real
page/cover**). It is **not** applied to later pages unless you repeat it
after each `page` / `cover`.

```sh
# Unblocks only the first page
gowkhtmltopdf --enable-local-file-access \
  page one.html page two.html out.pdf
# two.html is still blocked

# Repeat per page, or allow a tree
gowkhtmltopdf \
  page --enable-local-file-access one.html \
  page --enable-local-file-access two.html \
  out.pdf

gowkhtmltopdf --allow /srv/reports \
  page /srv/reports/a.html page /srv/reports/b.html out.pdf
```

`toc` does not consume that pending enable, so

```sh
gowkhtmltopdf --enable-local-file-access toc page body.html out.pdf
```

unblocks `body.html` (no ghost page before the TOC). Header/footer flags
before any object stay **global**.

`--allow` prefixes are compared after `EvalSymlinks`. A symlink inside an
allowed directory cannot escape it. `file://` hosts other than empty /
`localhost` are refused. Image mode has no `--allow`; use
`--enable-local-file-access`.

Treat any job that enables local files as able to read whatever the process
user can read. See [THREAT-MODEL.md](THREAT-MODEL.md) §3.

## Remote URL security

An `http` / `https` input makes **this process** an HTTP client for the
primary URL **and** every linked stylesheet, image, and local `@font-face`
the document names. That is the same threat class as embedding the library
in a web handler.

| Risk | What happens | Mitigation |
|------|----------------|------------|
| **SSRF** | The CLI host fetches attacker-chosen hosts (localhost, metadata, RFC1918) | Do not pass arbitrary user `?url=` / argv URLs without isolation. Prefer `--restrict-network` and `--allow-host` |
| **Untrusted HTML** | Hostile markup triggers second-hop fetches and CPU/RAM work up to loader caps | Convert HTML you author, or isolate the job; keep local-file ACL off |
| **Credentials** | `--username` / `--password` / `--cookie` / `--custom-header` / `--proxy` leave the box on every matching request | Only on trusted jobs |
| **Local files** | `file://` and path reads | Keep `--enable-local-file-access` / `--allow` off |

There is no `--insecure`. TLS verification stays on.

### Network policy

| Mode | How you get it | Behaviour |
|------|----------------|-----------|
| **Compatible** (default) | No `--restrict-network` / `--allow-host` | Any `http`/`https` host, including localhost and RFC1918. Redirects may cross hosts |
| **Restricted** | `--restrict-network` | Private / loopback / link-local denied unless an **exact** `--allow-host` match. Cross-host redirects denied. Schemes `http` and `https` only |
| **Broken half-policy** | `--allow-host` **alone** | Sets `NetworkPolicySet` **without** filling allowed schemes. HTTP(S) is then denied as an unknown scheme. **Always pair** `--restrict-network --allow-host HOST` |

`--allow-host` accepts an exact hostname or a label-boundary wildcard
(`*.example.com` matches `a.example.com`, not `example.com` and not
`notexample.com`). Exact allowlisted hosts may be private (trusted internal
services). Wildcard suffixes still resolve and **block private records**.

```sh
# Untrusted HTML / open URL
gowkhtmltopdf --restrict-network \
  --allow-host reports.example.test \
  'https://reports.example.test/invoice' out.pdf
```

Loader limits (not flags, except `--timeout`):

| Limit | Value |
|-------|------:|
| Connect timeout | 30 s |
| Response timeout | 60 s (`--timeout`, `0` = 60) |
| Redirects | 10 |
| Body (HTTP and file) | 100 MiB (rejected, not truncated) |

Library callers set `GlobalSettings.SetNetworkPolicy` with
`CompatibleNetworkPolicy()` or `RestrictedNetworkPolicy()`. See
[THREAT-MODEL.md](THREAT-MODEL.md) and
[integration-security.md](integration-security.md).

Fetching a URL is **not** “decent print.” See [URL mode](#url-mode--chrome-strip---simplify-dom)
and [fidelity.md — Arbitrary websites](fidelity.md#arbitrary-websites-phase-21).

## Headers, footers, and placeholders

Text HF is left / center / right in the reserved margin band, plus an
optional rule. Placeholders are substituted **after** `--replace` and
**after** copies, so `[topage]` is the final page count.

Only these tokens are recognized:

| Token | Expands to |
|-------|------------|
| `[page]` | 1-based page index + `--page-offset` |
| `[frompage]` | 1-based index within this object’s pages |
| `[topage]` | Final document page count |
| `[date]` | `YYYY-MM-DD` (wall clock; library `Now` can pin it) |
| `[time]` | `HH:MM:SS` |
| `[title]` | `--title` / PDF title |
| `[doctitle]` | HTML `<title>` |
| `[webpage]` | This object’s input string |
| `[section]` | Nearest outline heading on this page (empty on TOC pages) |
| `[subsection]` | Next-deeper heading |
| `[subject]` | Always empty |

There is **no** `[isodate]`, `[sitepage]`, or `[sitepages]`. Unknown
`[word]` tokens stay literal.

```sh
gowkhtmltopdf \
  --title "Q3 statement" \
  --header-left "[title]" --header-right "[date] [time]" \
  --footer-center "[page] / [topage]" \
  --replace '[dept]' 'Finance' \
  --enable-local-file-access statement.html statement.pdf
```

`--replace` substitutes the **exact** token you pass (include `[…]` if that
is what you wrote). Because it is a pair flag, put it **after** `page` (or
rely on implicit filling of the object it created). See [Pitfalls](#pitfalls).

## TOC and outline

`toc` inserts a generated table of contents built from body `h1`–`h6`
(respecting `--outline-depth` and per-object outline gates). It is **not**
XSLT. `--xsl-style-sheet` prints a warning and the built-in template still
runs. `--dump-default-toc-xsl` only dumps a description of that template.

Defaults: caption `Table of Contents`, scale `0.8`, indent `1em`, dotted
leaders on, forward/back links off.

```sh
gowkhtmltopdf \
  --toc-header-text "Contents" \
  --toc-text-size-shrink 0.85 \
  --toc-level-indentation 1.5em \
  --disable-dotted-lines \
  --toc-forward-links \
  --outline --outline-depth 3 \
  --enable-local-file-access \
  toc page report.html book.pdf
```

**Boolean OR merge:** `effectiveTOC` does
`object.Flag || global.Flag`. A flag set *after* `toc` writes the object
field. `--disable-dotted-lines` after `toc` sets object `DottedLines=false`
while global stays `true` → result is still dotted. Put TOC appearance
flags **before** `toc`.

Outline bookmarks are **on by default** (depth 4). `--no-outline` disables
them. Covers never appear in the outline. `--dump-outline` writes

```xml
<?xml version="1.0" encoding="UTF-8"?>
<outline xmlns="http://wkhtmltopdf.org/outline">
  …
</outline>
```

to stdout (1-based page attributes, no CDATA).

## Fonts

By default every PDF embeds Liberation Sans (Regular / Bold / Italic /
BoldItalic). That is enough for Latin report templates.

- `--font-path DIR` — repeatable; scan for `.ttf` / TrueType-flavored `.otf`
- `--use-system-fonts` — also scan common OS font dirs (off by default)

Use these for CJK, IPA, or Wikipedia-style Unicode. Local `@font-face`
`url(….ttf|otf|woff)` follows the same ACL as images. Remote WOFF2 / `https`
/ `data:` faces are skipped. Details and shaping limits:
[fonts.md](fonts.md).

## Pagination notes

These are layout facts operators hit from the CLI, not extra flags.

- **`<thead>` / `table-header-group`** repeats on continuation pages
  (fixture-23). Nested-table edges are best-effort.
- **`position: sticky`** clamps to the **page content box** (print
  scrollport) inside the containing block. There are no browser-style
  continuation clones. Inside `overflow: auto|scroll|hidden|clip` the
  sticky scrollport is that box at scroll offset 0 (PDF has no user
  scroll).
- **`--zoom`** scales layout lengths (forwarded as `Options.Zoom`). Values
  below 1 shrink. Unset / `0` means 1.
- **Smart-shrinking** (default on) may **re-layout** with an effective zoom
  when content is wider than the page content box (overflow ≳ 0.1 pt). User
  `--zoom` multiplies that factor. `--disable-smart-shrinking` skips the
  pass. There is no bare `--smart-shrinking`.
- **Orphans / widows:** CSS `orphans` / `widows` are parsed (integer ≥ 1,
  inherit, initial 2) and enforced when line boxes are countable. A
  geometric short-block heuristic remains when line counts are unavailable
  (fixtures 30 / 37).

<a id="url-mode--chrome-strip---simplify-dom"></a>

## URL mode & chrome strip (`--simplify-dom`)

<a id="url-mode--chrome-strip---simplify-dom"></a>

`--simplify-dom` defaults to **off**. Controlled invoices/reports should
leave it off so author CSS is unchanged.

Paste-any-URL prints are best-effort (“decent print”, not visual parity).
The engine does not run JavaScript; SPA shells and JS-only content will be
empty or incomplete. Pick a recipe:

**1. Raw honesty smoke** — live page as-is (this is the Ana
`make samples` artifact; **no** `--simplify-dom`):

```sh
gowkhtmltopdf --use-system-fonts --zoom 0.666667 --timeout 60 \
  --load-error-handling ignore \
  'https://en.wikipedia.org/wiki/Ana_de_Armas' output/wiki-ana-de-armas.pdf
```

**2. Decent-print attempt** — opt-in chrome strip + fonts (still not Chrome
parity):

```sh
gowkhtmltopdf --simplify-dom --simplify-dom-profile mediawiki \
  --use-system-fonts --timeout 60 \
  --load-error-handling ignore \
  'https://en.wikipedia.org/wiki/Example' out.pdf
```

| Flag / setting | Role |
|----------------|------|
| `--simplify-dom` | Injects `display:none !important` on `nav` / `footer` / `aside`, `[role=navigation|contentinfo|complementary]`, and `nav.site-nav`. Nodes stay in the tree; no extra origins are fetched |
| `--simplify-dom-profile mediawiki` | Also hide `#mw-navigation` / `.mw-jump-link`. Empty profile = landmarks only |
| `--no-simplify-dom` | Explicit off (the default) |
| `--print-link-underline` | Opt-in underline after cascade |
| `--use-system-fonts` / `--font-path` | Glyph coverage for open-web text ([fonts.md](fonts.md)) |
| `--zoom` | Operator scale. Ana smoke may use `0.666667` to densify author `12pt` body |
| `--print-media-type` | PDF layout already uses `Media: "print"` (`@media print` + size features). Prefer `--simplify-dom` for chrome beyond print CSS |
| `--timeout` / `--load-error-handling` | Loader policy for flaky public sites |

Security for remote URLs: [Remote URL security](#remote-url-security).
Acceptance bar and non-claims: [fidelity.md](fidelity.md#arbitrary-websites-phase-21).
Sample regeneration: [samples.md](samples.md).

## Exit codes

| Code | Meaning |
|-----:|---------|
| 0 | Success, including `--help` / `--version` / `--license` / `--extended-help` / `--dump-default-toc-xsl` |
| 1 | General error (parse, I/O, HTTP other than 404/401, layout, policy, …) |
| 2 | HTTP **404** on a failing main-document load (`abort`) |
| 3 | HTTP **401** on a failing main-document load (`abort`) |

Wrapped load errors still map through `errors.As`. Progress / info lines
(`Loading pages (1/N)`, smart-shrink notices) go to stderr unless
`--quiet`. Conversion errors are always printed to stderr, even with
`--quiet`.

`SIGINT` / `SIGTERM` cancel the conversion context.

## Pitfalls

1. **`--enable-local-file-access` placement.** Before any object it unblocks
   only the first real page/cover (and is not consumed by `toc`). Later
   pages stay blocked unless you repeat the flag or use `--allow`.
2. **`-L` is `--license`**, not `--margin-left`. The long form is the only
   way to set the left margin. (`-L` is intercepted before the short-flag
   table.)
3. **`--dump-outline` is a bool to stdout**, not `--dump-outline file.xml`.
   Combining it with PDF output `-` is rejected.
4. **No stdin HTML.** `-` as input becomes `http://-` unless a file named
   `-` exists. Pipe HTML via a temp file, `inline:…`, or the library API.
5. **Pair flags create page objects.** `--cookie`, `--custom-header`,
   `--post`, and `--replace` call `object()` immediately. Before `page` they
   open an object; a later `page` keyword opens **another** one. That
   breaks image mode (`exactly one input object`) and can add an empty PDF
   object that then tries to load `http://`. Put pair flags **after**
   `page`, or use implicit `in.html out.pdf` so the created object is
   filled by the first URL.
6. **Some flags ignore the bool value.** `--keep-relative-links`,
   `--no-background`, and `--disable-local-file-access` always apply their
   “off / block” side — `--keep-relative-links=false` still disables
   resolution. `--enable-smart-shrinking` / `--disable-smart-shrinking`
   likewise ignore the value.
7. **No bare `--smart-shrinking`.** Unknown option. Use the enable/disable
   pair (default is on).
8. **TOC bool OR merge.** Appearance flags belong **before** `toc`.
   `--disable-dotted-lines` after `toc` loses to the global default `true`.
9. **Cover wipes HF.** Global `--header-*` / `--footer-*` do not appear on
   `cover` pages.
10. **`--allow-host` scheme pitfall.** Used alone it turns on the explicit
    network policy with an **empty scheme list**, so `http`/`https` are
    denied. Use `--restrict-network --allow-host HOST`.
11. **Unknown flags error.** There is no silent ignore of `--dpi`,
    `--enable-javascript`, `--javascript-delay`, `--cookie-jar`,
    `--log-level`, `--user-style-sheet`, `--read-args-from-stdin`,
    `--lowquality`, `--use-xserver`, `--window-status`, `--run-script`,
    `--images`, `--minimum-font-size`, `--produce-forms`,
    `--default-encoding`, `--custom-header-propagation`, and similar
    wkhtmltopdf surface. Check [compatibility-matrix.md](compatibility-matrix.md)
    before migrating scripts.
12. **`--xsl-style-sheet` does not run XSLT.** Warning + built-in TOC
    template.
13. **`--extended-help` ≡ `--help`.**
14. **Image `--allow` is missing.** `option --allow is not supported in
    image mode`. Use `--enable-local-file-access`.
15. **Help always says `gowkhtmltopdf`**, including `gowkhtmltoimage
    --help` / `--version`.
16. **Non-existent relative paths become `http://…`.** `report.html` that
    is not on disk is fetched as `http://report.html`, not a missing-file
    error.

## Related docs

| Document | Why |
|----------|-----|
| [getting-started.md](getting-started.md) | Build, first PDF/PNG, first URL |
| [library-api.md](library-api.md) | `RunPDF` / `RunImage` / settings (no argv) |
| [fonts.md](fonts.md) | `--font-path`, system fonts, `@font-face`, shaping limits |
| [fidelity.md](fidelity.md) | What “good” means; URL decent-print bar |
| [samples.md](samples.md) | Fixtures, `make samples`, Ana smoke |
| [THREAT-MODEL.md](THREAT-MODEL.md) | ACL, network policy, timeouts |
| [integration-security.md](integration-security.md) | Embedding behind HTTP (SSRF) |
| [compatibility-matrix.md](compatibility-matrix.md) | Per-flag / per-CSS contract |
