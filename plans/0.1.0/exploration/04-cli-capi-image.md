# Exploration 04 - CLI, C API, Image Tool

> **Agent:** explore · CLI/C API/image  
> **Primary:** `src/pdf/*`, `src/image/*`, `src/shared/*`, `pdf.h`, `image.h`

---

## PDF CLI grammar

```
[global|page switches]* (cover URL | toc | [page] URL) [page switches]* ... <output>
```

## Image CLI grammar

```
[switches]* <input> <output>
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | success |
| 2 | HTTP 404 |
| 3 | HTTP 401 |
| 1 | other failure |

## C API shape (PDF)

init/deinit → create global/object settings → set string keys → create converter → add_object → convert → get_output → callbacks for log/phase/progress

## Image pipeline

Load → smart width binary search → set height → render to QImage/QSvgGenerator → save with quality. Formats: jpg, png, bmp, svg. Defaults: width 1024, quality 94, smartWidth true.

## CLI-only effort

~8–12 person-weeks for settings + CLI + image plumbing **given** a rasterizer/PDF backend.

## Ship-first recommendation

Flag-compatible CLI skeleton with settings storage → wire high-frequency options → image CLI → advanced objects (cover/toc) → library API.
