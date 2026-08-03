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
Text uses a bitmap font (no anti-aliasing).

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
