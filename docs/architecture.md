# Architecture

## Package map

| Package | Responsibility |
|---------|----------------|
| `gowkhtmltopdf` (root) | Public library API (`Converter`, `ImageConverter`) |
| `cmd/gowkhtmltopdf` | PDF CLI entrypoint |
| `cmd/gowkhtmltoimage` | Image CLI entrypoint |
| `internal/cli` | Argv parse, multi-object grammar (`page` / `cover` / `toc`), help |
| `internal/settings` | wkhtmltopdf-style settings, UnitReal, dotted `Get`/`Set` |
| `internal/load` | URL guess, HTTP(S)/file/`data:`, ACL, cookies, auth, POST |
| `internal/html` | Allowlisted HTML tokenizer + tree |
| `internal/css` | CSS subset parse, selectors, cascade |
| `internal/layout` | Style resolve, block/inline/table layout, display list, paint |
| `internal/outline` | Headings → outline tree / dump XML |
| `internal/convert` | PDF job orchestration (HF, TOC, links, copies) |
| `internal/pdf` | PDF 1.4 writer, TTF subset, images, annotations |
| `internal/imageout` | Raster path for PNG/JPEG |

## Conversion pipeline (PDF)

1. **Parse CLI / library settings** → `cli.Command` or converter state  
2. **Load** each body object (`load.Loader`)  
3. **Parse HTML** → style with CSS sheets (inline + linked if allowed)  
4. **Layout** → display-list ops (text, rects, images, links)  
5. **Paginate** using page size, margins, `page-break-*`  
6. **Paint** ops into `pdf.Document` pages  
7. **TOC** (if any): build outline tree, two-iteration page-count fixpoint  
8. **Reorder** so TOC pages come first when present  
9. **Outline / link annotations / headers-footers** final pass  
10. **Write** PDF with zlib Flate streams, embedded font subset, xref  

Image mode shares load → parse → layout, then rasterizes the display list
with a bitmap font (`internal/imageout`).

## PDF writer notes

- PDF **1.4**, pure Go  
- Streams: `/Filter /FlateDecode` via **zlib** (RFC 1950)  
- Fonts: embedded **Liberation Sans** subset; simple font + WinAnsi-style
  Latin-1 codes; `/Widths` in **1000 units/em**  
- Outlines: Catalog `/Outlines N 0 R` assigned after outline object refs exist  

## Security defaults

- Local file access **denied** unless `--enable-local-file-access` / settings  
- `--allow` prefixes for selective local paths  
- HTTP: connect/response timeouts, max redirects, max body size  

Details: [THREAT-MODEL.md](THREAT-MODEL.md).

## Extension points (intentionally small)

The public API is settings-driven (dotted names), not a plugin framework.
New CSS properties or elements require changes inside `internal/css` and
`internal/layout` and an update to the compatibility matrix.
