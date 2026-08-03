# Exploration 02 - PDF Converter Engine

> **Agent:** explore · PDF deep dive  
> **Primary files:** `src/lib/pdfconverter.cc` (~1215 LOC), `outline.cc`, `pdfsettings.*`

---

## Fact

**wkhtmltopdf does not implement a PDF writer.** One `QPrinter` session paints all objects in order.

## Object model

- `page URL` - normal  
- `cover URL` - page with HF cleared, excluded from outline  
- `toc` - generated HTML from outline XML + XSLT  

## Settings (high level)

**PdfGlobal:** page size/orientation/margins, dpi, copies/collate, outline, compression, image DPI/quality, title, viewport, cookie jar, resolveRelativeLinks  

**PdfObject:** HF, TOC knobs, links, forms, load, web, includeInOutline, tocXsl  

**Not present:** encryption, duplex, PDF version, PDF/A  

## Algorithms to reimplement

| Algorithm | Location |
|-----------|----------|
| Header height auto | `calculateHeaderHeight` |
| HF placeholders | `fillParms` / `hfreplace` |
| Heading outline tree | `Outline::replaceWebPage` |
| TOC fixed-point loop | `loadTocs` / `tocLoaded` |
| Link classify | `findLinks` |
| Copies/collate spool | `printDocument` / `spoolTo` |

## Pure-Go PDF stack estimate (given layout)

~25–45 person-weeks for writer + assembly + links + outline + text HF; fonts/forms add more. HTML layout is separate XXL cost.
