# Exploration 01 — Architecture & Core Pipeline

> **Agent:** explore · architecture  
> **Source tree:** `wkhtmltopdf/` v0.12.7-dev  
> **Date:** planning session

---

## What this project is

Thin C++ orchestration over **Qt WebKit + QPrinter**, not a browser. Application code in `src/` ≈ **10.6k LOC**.

## Layout

| Path | Role |
|------|------|
| `src/lib/` | libwkhtmltox: converters, loader, outline, settings, C API |
| `src/pdf/` | `wkhtmltopdf` CLI |
| `src/image/` | `wkhtmltoimage` CLI |
| `src/shared/` | CLI arg handlers, help outputters, progress |
| `qt/` | empty placeholder for patched Qt |

## PDF phases (patched Qt)

0 Loading pages → 1 Counting pages → 2 Loading TOC → 3 Resolving links → 4 Loading headers/footers → 5 Printing pages → 6 Done

Key types: `PdfConverterPrivate`, `PageObject`, `MultiPageLoader`, `Outline`, `QWebPrinter` (patched).

## Critical Qt dependencies to replace

- `QWebPage` / layout / JS  
- `QWebPrinter` (pageCount, spoolPage, elementLocation)  
- `QPrinter` + patched `QPainter` (links, forms, outline)  
- `QNetworkAccessManager` stack  
- `QXmlQuery` XSLT for TOC  

## Pure-Go takeaway

Orchestration layer is portable; **rendering and PDF emission are the product**.
