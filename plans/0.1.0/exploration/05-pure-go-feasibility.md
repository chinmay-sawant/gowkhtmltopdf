# Exploration 05 - Pure-Go Feasibility (Stdlib Only)

> **Agent:** explore · feasibility  
> **Constraint:** Go standard library only - no modules, no Chrome, no cgo

---

## Verdict

| Scope | Feasible? | Solo senior | 2 seniors | Person-months |
|-------|-----------|-------------|-----------|---------------|
| **MVP** controlled reports, no JS | Hard but yes | 18–30 months | 10–18 months | 18–28 PM |
| **Intermediate** richer CSS, HTML HF, better tables | Severe gaps ok | 3.5–6 years | 2–3.5 years | 45–70 PM |
| **Full WebKit-like parity** | **No** | Not credible | Not credible | 200–500+ PM incomplete |

## Component matrix (summary)

| Component | Stdlib? | MVP PM |
|-----------|---------|--------|
| HTTP/file loader | Mostly | 0.5–2 |
| HTML parser | No (build subset) | 1–6 |
| CSS + layout | No | 8–14+ |
| JavaScript | No - **omit** | 0 |
| Fonts/shaping | Almost no | 3–6 Latin |
| Images JPEG/PNG/GIF | Yes | 0.5–1 |
| SVG/WebP | No | defer |
| PDF writer | No (build) | 2–4 + fonts |
| Page breaks | No (build) | 2–4 |
| HF/TOC/outline | App logic | 2–5 |

## Showstoppers for full parity

1. CSS layout + fragmentation  
2. JS + DOM  
3. HarfBuzz-level shaping  
4. Stdlib-only forbids reusing pure-Go OSS (inflates schedule)  
5. Infinite web platform surface  

## Honest MVP definition

**In:** server-rendered HTML, CSS subset, tables, PNG/JPEG, A4/Letter, multi-page, text HF, outlines, selectable Latin text  

**Out:** SPAs, JS, modern CSS, arbitrary websites, XSLT TOC, full forms, pixel parity with Qt  

## Strategic recommendation

Plan the **report island** (Phases 0–6 + 9). Do not date “full parity.” If arbitrary HTML is required later, pure stdlib is the wrong vehicle (headless Chrome is industry standard).
