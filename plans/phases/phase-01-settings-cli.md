# Phase 01 — Settings Model & CLI Skeleton

> **Parent:** `plans/00-canonical-pure-go-rewrite.md`  
> **Status:** not started  
> **Estimated effort:** 4–6 weeks solo · 2–3 weeks pair  
> **Depends on:** Phase 0 scaffold  
> **Unblocks:** Phase 2 (load settings), progressive flag wiring for convert

---

## Overview

Port the settings surface and CLI grammar of `wkhtmltopdf` / `wkhtmltoimage` so scripts can parse flags correctly before the engine exists. Convert may stub with a clear error until Phase 5–6.

## Executive Summary

CLI + settings alone are ~8–12 person-weeks of the full project (explore agent 4). Defaults matter: especially `blockLocalFileAccess=true`, margins L/R 10mm, jsdelay 200, outline on.

---

## Checklist

### 1.1 Settings packages (`internal/settings`)

#### PDF global/object
- [ ] `PdfGlobal` defaults match `pdfsettings.cc` (A4, portrait, color, dpi 96, collate true, outline true, depth 4, compression true, imageDPI 600, imageQuality 94, margins)
- [ ] `PdfObject` defaults (external/local links true, includeInOutline true, pagesCount true, produceForms false)
- [ ] `HeaderFooter` defaults (font Arial 12, spacing 0)
- [ ] `TableOfContent` defaults (dotted true, caption "Table of Contents", fontScale 0.8, indentation "1em")

#### Load / web / image
- [ ] `LoadPage` defaults: jsdelay 200, blockLocal true, stopSlowScripts true, loadError abort, media ignore, printMediaType false
- [ ] `Web` defaults: background/images/JS true, smart shrink true, plugins false
- [ ] `ImageGlobal` defaults: width 1024, height 0, quality 94, smartWidth true, crop -1s

#### Parsers
- [ ] `ParseUnitReal` + `FormatUnitReal` (all units from `pdfsettings.cc`)
- [ ] `ParsePageSize` / `ParseOrientation` / `ParseProxy` / `ParseLoadErrorHandling` / `ParseLogLevel`
- [ ] Reflection: `Global.Set(name, value)`, `Object.Set(name, value)` with dotted keys from `reflect.cc` / C API

### 1.2 CLI (`internal/cli`, `cmd/gowkhtmltopdf`)

#### Grammar
- [ ] Parse global switches until first object
- [ ] Objects: `cover URL`, `toc`, optional `page` + URL
- [ ] Last arg = output path (`-` = stdout)
- [ ] Page switches after each object bind to that object (address remapping semantics)
- [ ] Cover: clear header/footer fields; `includeInOutline=false`

#### Flags (accept all; store in settings)
- [ ] Doc flags: help, version, license, extended-help (man/html can stub)
- [ ] Global PDF: quiet, log-level, collate, copies, orientation, page-size, grayscale, lowquality, title, margins, dpi, page-width/height, image-quality/dpi, no-pdf-compression, use-xserver (no-op warn), cookie-jar, read-args-from-stdin
- [ ] Outline: outline/no-outline, outline-depth, dump-outline, dump-default-toc-xsl
- [ ] Page/web/load: full shared set from `commonarguments.cc` + `pdfarguments.cc`
- [ ] HF: header/footer left/center/right/font/line/spacing/html + replace
- [ ] TOC: xsl-style-sheet, toc-header-text, disable-toc-links, disable-dotted-lines, toc-text-size-shrink, toc-level-indentation

#### Runtime UX
- [ ] Stderr progress bars when log ≥ Info (`progressfeedback.cc`)
- [ ] Exit codes: 0 success; 2 on HTTP 404; 3 on 401; 1 otherwise (`utilities.cc`)
- [ ] `--read-args-from-stdin` batch loop with shell-like tokenize

### 1.3 Tests
- [ ] Table-driven: flag string → expected settings field
- [ ] Multi-object grammar fixtures
- [ ] Default-value snapshot test
- [ ] UnitReal edge cases (cm→mm, invalid unit)

### 1.4 Closure
- [ ] `make test` pass
- [ ] `make lint` pass
- [ ] Binary: `gowkhtmltopdf --help` and `--version` work
- [ ] Convert path returns explicit `errEngineNotReady` until later phase (or minimal Phase 3 smoke)

---

## Dependencies

| Needs | From |
|-------|------|
| Module paths | Phase 0 |
| Feeds | Phase 2 LoadPage fields; Phase 8 reflection |

## Source map (upstream)

| Concern | Path |
|---------|------|
| PDF flags | `wkhtmltopdf/src/pdf/pdfarguments.cc` |
| Shared flags | `wkhtmltopdf/src/shared/commonarguments.cc` |
| Parse grammar | `wkhtmltopdf/src/pdf/pdfcommandlineparser.cc` |
| Defaults | `wkhtmltopdf/src/lib/pdfsettings.cc`, `loadsettings.cc`, `websettings.cc` |
| Exit codes | `wkhtmltopdf/src/lib/utilities.cc` |

## Known upstream quirks (decide: mirror vs fix)

- [ ] Document decision on `--no-custom-header-propagation` bug (sets true)
- [ ] Document decision on bare `--` end-of-options typo (`argv[arg][2] == '0'`)
