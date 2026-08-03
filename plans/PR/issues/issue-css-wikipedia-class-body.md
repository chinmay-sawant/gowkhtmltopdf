## Context

**Parent epic:** #2 — [epic: post-MVP rendering quality — image mode, fonts, CSS for real sites](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/2)

**Siblings under #2:** #3 (image-mode PNG) · #4 (font spacing) · #5 (Wikipedia CSS) · #6 (multi-font)

MVP CSS is a **documented subset** for controlled reports (boxes, tables, basic text). Real websites—especially Wikipedia—use richer CSS (flex/grid-ish layouts via modern rules, complex selectors, multi-column chrome, language lists, sticky nav, etc.). Converting them produces multi-page PDFs that open but **do not look like the site**.

### Concrete example

```sh
go run ./cmd/gowkhtmltopdf \
  "https://en.wikipedia.org/wiki/Ana_de_Armas" \
  output/wiki-ana-de-armas.pdf
```

| Result | Notes |
|--------|--------|
| File | `output/wiki-ana-de-armas.pdf` (~1.3 MB, many pages) committed as smoke artifact |
| Latin body text | Partially present; nav/TOC chrome dominates early pages |
| Layout | Sidebar/nav/search/appearance UI dumped as linear flow |
| Non-Latin language names | Often `?` (single Latin font — sibling font issue) |
| JS UI | Ignored (no JS engine — by design) |

URL: **https://en.wikipedia.org/wiki/Ana_de_Armas**

This is the **acceptance fixture** for “most-used website CSS” progress: not full pixel parity, but readable article body with reduced chrome noise and better structure.

## Scope (in)

1. Inventory Wikipedia (Vector/Minerva-ish) CSS constructs that break layout; map to matrix gaps (flex, grid, float, position, media queries, complex selectors).  
2. Prioritize a **report-friendly subset expansion** that also helps common sites (e.g. simple flex row, float lite, `display` values, `:nth-child`, attribute selectors).  
3. Optionally add “chrome stripping” / reader-mode heuristics for known wiki DOM (product decision — document in issue comments).  
4. Golden or smoke: store minimal HTML snapshot or documented command for Ana de Armas; assert page count band + presence of title string in content stream.  
5. Update `documentation/compatibility-matrix.md` as properties land.

## Out of scope

- Executing JavaScript / SPA hydration  
- Full Chrome layout parity  
- CID/CJK fonts (sibling issue; link only)  
- Scraping or caching Wikipedia in CI without network policy (prefer vendored HTML fixture for CI)

## Success criteria

- [ ] Documented CSS additions implemented with tests  
- [ ] Ana de Armas (or vendored snapshot) PDF shows **clear article title + body**, less useless nav-only leading pages  
- [ ] Matrix updated; README deferred list adjusted  
- [ ] Still stdlib-only / no browser embed  

## Plan

- Parent epic: #2  
- Plans: `plans/phases/phase-04-html-css-layout.md`, intermediate roadmap phase-09  
- Artifact: `output/wiki-ana-de-armas.pdf`  

## References

- Relates to #2 (parent epic)
- URL: https://en.wikipedia.org/wiki/Ana_de_Armas  
- Sample PDF: `output/wiki-ana-de-armas.pdf`  
- Docs: `documentation/compatibility-matrix.md`, `documentation/overview.md`  
- Related PR #1 samples section  

