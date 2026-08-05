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
```

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
