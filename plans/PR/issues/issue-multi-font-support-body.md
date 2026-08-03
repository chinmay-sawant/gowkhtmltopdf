## Context

**Parent epic:** #2 — [epic: post-MVP rendering quality — image mode, fonts, CSS for real sites](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/2)

**Siblings under #2:** #3 (image-mode PNG) · #4 (font spacing) · #5 (Wikipedia CSS) · #6 (multi-font)

Today the engine effectively validates **one** embedded face:

| Mode | Font |
|------|------|
| PDF | Liberation Sans **Regular** only (`internal/pdf/assets/LiberationSans-Regular.ttf`), subset embedded |
| Image | 5×7 ASCII bitmap (`internal/imageout/font.go`) — not even the TTF |

Bold is **faked** (PDF: fill+stroke text mode; image: double draw). Italic is largely upright.  
Non-Latin / CJK / many Unicode punctuation paths fold to `?` or WinAnsi stand-ins (`pdfString` / `winAnsiFold`).

Users expect:

- Bold / italic / bold-italic families  
- Optional additional faces (serif/mono) via system or bundled fonts  
- Longer-term: broader Unicode (CID/Type0) for Wikipedia language lists, etc.

## Scope (in)

1. Font registry: map CSS `font-family` / `font-weight` / `font-style` to loaded faces.  
2. Bundle or discover at least: Regular + **Bold** (+ Italic if feasible) of Liberation (or equivalent OFL fonts) under stdlib-only embedding.  
3. Wire layout measurement and PDF subsetting per face (multiple `ensureFont` subsets / resource names `F0`/`F1`/…).  
4. Remove or reduce fake-bold when a real bold face is selected.  
5. Tests: bold heading width ≠ regular; ToUnicode/subset still valid; fixture visual smoke.  
6. Docs + matrix: which families/weights are supported; still no HarfBuzz shaping claim.  
7. Stretch (can split later): simple Type0/CID or multi-byte path for common Unicode beyond Latin-1.

## Out of scope

- Full OpenType shaping / Arabic/Indic reordering  
- Downloading arbitrary webfonts from the network without ACL policy  
- Commercial font licensing beyond OFL/SIL bundled assets  
- Image-mode TTF raster (coordinate with image-mode issue; may share registry)

## Success criteria

- [ ] At least **two** PDF faces validated (e.g. Regular + Bold) end-to-end  
- [ ] CSS `font-weight: bold` / `<b>` / `<strong>` use bold face when available  
- [ ] Fake bold not required for default report fixtures  
- [ ] `make samples` / golden tests green  
- [ ] Documentation lists supported families and Unicode limits  

## Plan

- Parent epic: #2  
- Code: `internal/pdf/{fonts.go,fontpdf.go,assets}`, `internal/layout/style.go`  
- Sibling: font spacing, CSS Wikipedia, image-mode raster  

## References

- Relates to #2 (parent epic)
- Asset: `internal/pdf/assets/LiberationSans-Regular.ttf`  
- Encoding: `pdfString` / simple font Latin-1 limits  
- Deferred: README “CJK fonts / complex-script shaping”  

