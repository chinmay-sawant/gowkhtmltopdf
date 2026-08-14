# Phase 01 - Settings Model & CLI Skeleton

> **Parent:** `plans/00-canonical-pure-go-rewrite.md`  
> **Status:** complete (2026-08-03)  
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
- [x] `PdfGlobal` defaults match `pdfsettings.cc` (A4, portrait, color, dpi 96, collate true, outline true, depth 4, compression true, imageDPI 600, imageQuality 94, margins)
- [x] `PdfObject` defaults (external/local links true, includeInOutline true, pagesCount true, produceForms false)
- [x] `HeaderFooter` defaults (font Arial 12, spacing 0)
- [x] `TableOfContent` defaults (dotted true, caption "Table of Contents", fontScale 0.8, indentation "1em")

#### Load / web / image
- [x] `LoadPage` defaults: jsdelay 200, blockLocal true, stopSlowScripts true, loadError abort, media ignore, printMediaType false
- [x] `Web` defaults: background/images/JS true, smart shrink true, plugins false
- [x] `ImageGlobal` defaults: width 1024, height 0, quality 94, smartWidth true, crop -1s

#### Parsers
- [x] `ParseUnitReal` + `FormatUnitReal` (all units from `pdfsettings.cc`)
- [x] `ParsePageSize` / `ParseOrientation` / `ParseProxy` / `ParseLoadErrorHandling` / `ParseLogLevel`
- [x] Reflection: `Global.Set(name, value)`, `Object.Set(name, value)` with dotted keys from `reflect.cc` / C API

### 1.2 CLI (`internal/cli`, `cmd/gowkhtmltopdf`)

#### Grammar
- [x] Parse global switches until first object
- [x] Objects: `cover URL`, `toc`, optional `page` + URL
- [x] Last arg = output path (`-` = stdout)
- [x] Page switches after each object bind to that object (address remapping semantics)
- [x] Cover: clear header/footer fields; `includeInOutline=false`

#### Flags (accept all; store in settings)
- [x] Doc flags: help, version, license, extended-help (man/html can stub)
- [x] Global PDF: quiet, log-level, collate, copies, orientation, page-size, grayscale, lowquality, title, margins, dpi, page-width/height, image-quality/dpi, no-pdf-compression, use-xserver (no-op warn), cookie-jar, read-args-from-stdin
- [x] Outline: outline/no-outline, outline-depth, dump-outline, dump-default-toc-xsl
- [x] Page/web/load: full shared set from `commonarguments.cc` + `pdfarguments.cc`
- [x] HF: header/footer left/center/right/font/line/spacing/html + replace
- [x] TOC: xsl-style-sheet, toc-header-text, disable-toc-links, disable-dotted-lines, toc-text-size-shrink, toc-level-indentation

#### Runtime UX
- [~] Stderr progress bars when log ≥ Info - deferred to Phase 5 (progress phases ship with the convert pipeline)
- [x] Exit codes: 0 success; 2 on HTTP 404; 3 on 401; 1 otherwise - `settings.HttpErrorCode` + `HttpError` wired; full path live once loads land (Phase 2)
- [~] `--read-args-from-stdin` batch loop with shell-like tokenize - flag accepted; batch loop deferred (Phase 9 tooling)

### 1.3 Tests
- [x] Table-driven: flag string → expected settings field
- [x] Multi-object grammar fixtures
- [x] Default-value snapshot test
- [x] UnitReal edge cases (cm→mm, invalid unit)

### 1.4 Closure
- [x] `make test` pass
- [x] `make lint` pass
- [x] Binary: `gowkhtmltopdf --help` and `--version` work
      Evidence 2026-08-03: `--version` → `0.1.0-dev`; `--help` prints synopsis; `--bogus` → unknown option, exit 1; no-input convert → explicit error, exit 1.
- [x] Convert path returns explicit `errEngineNotReady` until later phase
      `internal/convert.ErrEngineNotReady` returned by `RunPDF`; replaced in Phase 5.

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

- [x] `--no-custom-header-propagation` bug (sets true): **fixed** - our `--no-` negation sets the option false; documented as an intentional divergence (upstream typo in `pdfcommandlineparser.cc`)
- [x] Bare `--` end-of-options typo (`argv[arg][2] == '0'`): **fixed** - `--` is treated as end-of-options; remaining args are positionals
