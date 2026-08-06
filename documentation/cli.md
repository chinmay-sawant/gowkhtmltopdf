# Command-line interface

## gowkhtmltopdf

```text
gowkhtmltopdf [GLOBAL OPTIONS] [OBJECT]... <output>
  OBJECT:
    [PAGE OPTIONS]  page <input>
    [TOC OPTIONS]   toc
    [COVER OPTIONS] cover <input>
```

- Output path required; use `-` for stdout  
- Input may be a path, URL, or `-` (stdin)  
- Run `--help` / `--extended-help` for the full flag list  

### Common examples

```sh
# Local file (ACL opt-in)
gowkhtmltopdf --enable-local-file-access report.html report.pdf

# Remote URL (HTTPS) — see Remote URL security below
gowkhtmltopdf 'https://example.com/report.html' out.pdf

# Page size and margins
gowkhtmltopdf --page-size A4 --margin-top 15mm report.html out.pdf

# Headers / footers / outline
gowkhtmltopdf \
  --header-left "Report" --header-right "page [page]/[topage]" \
  --footer-center "[section]" \
  --outline --outline-depth 2 \
  --enable-local-file-access chapter.html book.pdf

# TOC + body
gowkhtmltopdf --enable-local-file-access \
  --outline \
  toc page body.html book.pdf

# Explicit page keyword (page-scoped flags after page)
gowkhtmltopdf page --enable-local-file-access in.html out.pdf

# URL → decent print (opt-in chrome strip; see URL mode below)
gowkhtmltopdf --simplify-dom 'https://example.com/article' article.pdf
```

### Remote URL security

HTTP(S) inputs are a first-class CLI path:

```sh
gowkhtmltopdf 'https://example.com/report.html' out.pdf
```

That makes the **process running the CLI** an HTTP client for the primary URL
and for every linked stylesheet/image the document references. Treat this as
the same threat class documented for embedding:

| Risk | What happens | Mitigation |
|------|----------------|------------|
| **SSRF** | Server/CLI host fetches attacker-chosen hosts (localhost, metadata, RFC1918) when the URL or HTML is untrusted | Do not pass arbitrary user `?url=` / CLI args without host allowlists and network isolation |
| **Untrusted HTML** | Hostile markup can trigger second-hop fetches (`img`/`link`) and CPU/RAM DoS up to loader caps | Convert only HTML you author, or isolate the job; keep local-file ACL off |
| **Local files** | `file://` and path reads are denied by default | Keep `--enable-local-file-access` / `--allow` off for untrusted input |

Defaults that help: connect/response timeouts, redirect limits, body size caps,
TLS verify on (unless `--insecure`), no JavaScript execution. There is **no**
network egress allowlist inside the converter — input trust is the control.

Fidelity for public sites is a separate question: URL fetch ≠ “decent print”
acceptance. See [fidelity.md — Arbitrary websites](fidelity.md#arbitrary-websites-phase-21),
[THREAT-MODEL.md](THREAT-MODEL.md) (§5–§7.1), and
[integration-security.md](integration-security.md).

### Placeholders (text headers/footers)

`[page]`, `[topage]`, `[frompage]`, `[date]`, `[time]`, `[title]`,
`[doctitle]`, `[webpage]`, `[section]`, `[subsection]`  
Custom: `--replace name value`.

### Pagination & tables

- Multi-page tables with `<thead>` / `table-header-group` **repeat** the header
  row(s) on continuation pages (fixture-23).
- `position: sticky` clamps to the page content box (print scrollport) within
  the containing block (fixture-31). Inside `overflow: auto|scroll|hidden|clip`,
  that box is the sticky scrollport at scroll offset 0 (PDF has no user scroll;
  no page-edge clones for overflow-contained sticky).
- `--zoom` scales layout (forwarded to the layout engine).
- `--smart-shrinking` may **re-layout** with an effective zoom when content is
  wider than the page.
- Orphan/widow control: CSS `orphans` / `widows` are parsed (integer ≥1,
  inherit, initial 2) and enforced when line boxes are countable; a geometric
  short-block heuristic remains when line counts are unavailable.

### Fonts & links

- `--font-path <dir>` adds font search directories; `--use-system-fonts` opts
  into system font dirs (off by default for determinism). Same flags and
  local `@font-face` ACL apply to `gowkhtmltoimage`.
- `--resolve-relative-links` / `--keep-relative-links` control whether relative
  `href` values are resolved against the page URL.
- Body `#id` internal links emit GoTo annotations when geometry is available;
  HTML header/footer `#id` links resolve to **body** GoTo destinations (ids
  inside the HF document tree are not destinations).
- `--header-html` / `--footer-html` run a nested child layout (body CSS subset,
  flex/grid/images, local `@font-face` under the same ACL), clipped to the
  reserved margin band — not a browser nested browsing context.

### URL mode & chrome strip (`--simplify-dom`)

Paste-any-URL prints are best-effort (“decent print”, not visual parity).
Use one of two recipes depending on the goal:

**1. Raw honesty smoke** (what the engine does with the live page as-is — also
the Ana `make samples` artifact; **no** `--simplify-dom`):

```sh
gowkhtmltopdf --use-system-fonts --zoom 0.666667 --timeout 60 \
  --load-error-handling ignore \
  'https://en.wikipedia.org/wiki/Ana_de_Armas' output/wiki-ana-de-armas.pdf
```

**2. Decent-print attempt** (opt-in chrome strip + fonts; still not Chrome
parity):

```sh
gowkhtmltopdf --simplify-dom --use-system-fonts --timeout 60 \
  --load-error-handling ignore \
  'https://en.wikipedia.org/wiki/Example' out.pdf
```

| Flag / setting | Role |
|----------------|------|
| `--simplify-dom` | Opt-in chrome-strip (default **off**). Injects landmark `display:none` (`nav`/`footer`/`aside` + ARIA roles). Nodes stay in the tree; no extra origins are fetched. |
| `--simplify-dom-profile mediawiki` | With `--simplify-dom`, also hide MediaWiki `#mw-navigation` / `.mw-jump-link`. Empty profile = landmarks only. |
| `--no-simplify-dom` | Explicit off (also the default). Keep for invoices/reports and for raw smoke artifacts. |
| `--print-link-underline` | Opt-in: underline `a[href]` after cascade (default **off**). Author `text-decoration` wins otherwise. |
| `--use-system-fonts` | Opt-in OS font scan so IPA/Unicode glyphs can fall back (see [fonts.md](fonts.md)). Prefer this for open-web URLs; invoices often need only embedded Liberation. |
| `--font-path <dir>` | Explicit extra face directories (e.g. DejaVu) when system scan is undesirable. |
| `--zoom` | Operator layout scale (not stylesheet emulation). Ana smoke may use `0.666667` to densify author `12pt` body. |
| `--print-media-type` | Accepted for wkhtmltopdf compatibility; layout already runs with `Media: "print"` (site `@media print` + size features via `MediaMatches`). Prefer `--simplify-dom` for chrome reduction beyond print CSS. |
| `--timeout <sec>` | HTTP response timeout (default 60s). Connect timeout is 30s. |
| `--load-error-handling ignore\|skip\|abort` | Main document load policy. |
| Images | On by default. Library/embedders: `web.images=false` skips fetch/paint (no dedicated `--images` CLI flag yet). |
| JS flags | No JS engine; scripts are stripped at load. |

Loader limits (verified existing behavior in `internal/load/load.go`): body
cap **100 MiB** (`DefaultMaxBodySize`), connect **30s**, response **60s**
(or `--timeout`), redirect cap **10**. Failed CSS `<link>` and image
subresources are isolated (warning + continue; body text still emits) —
see `TestSubresourceFailureIsolation`. Progress phases print to stderr
unless `--quiet`.

Security notes for remote URLs: see [Remote URL security](#remote-url-security)
above. Sample regeneration: [samples.md](samples.md).

### Page-scoped flags and `toc`

Flags such as `--enable-local-file-access` may appear **before** any object
keyword. They apply to the **first real page/cover**, not to a phantom empty
object, and not consumed by `toc`. Header/footer flags before any object
remain **global**.

## gowkhtmltoimage

```sh
gowkhtmltoimage [OPTIONS] <input> <output.png|jpg>
```

Useful options: `--width`, `--height`, `--format`, `--quality`,
`--transparent`, `--crop-x/y/w/h`, `--enable-local-file-access`.

Default viewport width is approximately **1024** CSS pixels (smart-width).
Text uses the same TTF outline raster path as PDF (coverage AA); a 5×7 bitmap
fallback applies only when an op has no font face.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | HTTP 404 (when applicable) |
| 3 | HTTP 401 (when applicable) |

## Samples target

```sh
make samples   # regenerates output/*.pdf and sample PNG
```

See [samples.md](samples.md).
