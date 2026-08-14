# Pending — Phase 1: `:link` / `:visited` print semantics (blue links)

> **Parent:** [`README.md`](README.md)  
> **Status:** done (2026-08-05)  
> **Estimated effort:** 0.5–2 days  
> **Prior plan coverage:** Matrix listed `:link`/`:visited` as accepted & **ignored** — **no prior implement checklist**. Closest: Phase 17 selectors / Phase 21 §21.3  

---

## Overview

Wikipedia Vector (and many sites) color hyperlinks with `a:link` / `a:visited`.
Previously those pseudos were dropped at parse time, so `a:link` degraded to bare
`a` (equal specificity with author `a { color: … }`) and interactive
`a:hover` could also degrade to bare `a`.

**Print semantics (shipped):** `:link` and `:visited` match any `a` with
non-empty `href`. `:hover` / `:active` / `:focus` parse but never match.

### Smoke proof

```sh
./bin/gowkhtmltopdf 'https://en.wikipedia.org/wiki/Ana_de_Armas' output/wiki-ana-de-armas.pdf
```

**Note (2026-08-05):** Ana live smoke still shows mostly black body text —
Vector likely needs additional selector/media coverage (Phases 2/4/6). Synthetic
`a:link { color: #0645ad }` renders blue (`TestLinkPseudoColor` + manual
`/tmp/linkcolor.pdf`).

---

## Phase 1 checklist

### 1.1 Parse & match

- [x] Parse `:link` and `:visited` on compounds in `internal/css` (append to `Pseudos`)
- [x] `Match`: `:link` / `:visited` succeed when element is `a` with non-empty `href`
- [x] `Match`: `:link` / `:visited` fail when no `href` or not an `a`
- [x] `:hover` / `:active` / `:focus` parsed but `matchPseudo` → false (no degrade to bare `a`)

### 1.2 Cascade / UA

- [x] UA `a { color: #0000ee; text-decoration: underline }` still in `style.go`
- [x] Author `a:link` outranks bare `a` via specificity (`TestLinkVisitedPseudos`)
- [x] Bare `<a>` without href keeps author `a { … }` (not `:link`)

### 1.3 Tests

- [x] `TestLinkVisitedPseudos` (`internal/css`)
- [x] `TestLinkPseudoColor` (`internal/layout`)
- [x] Regression: `go test ./internal/css ./internal/layout ./internal/convert -count=1` OK

### 1.4 Docs & matrix

- [x] Matrix: `:link`/`:visited` → Partial; hover/active/focus → never match
- [x] Ana residual noted (Phases 2/4/6)

### 1.5 Gates

- [x] `make lint` → `go vet ./...` OK (2026-08-05)
- [x] `make test` → `go test ./...` OK (2026-08-05)
- [x] Optional smoke: regenerated `output/wiki-ana-de-armas.pdf` (raw); links still mostly black pending skin CSS
- [x] Status → done; README order-1 → done

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| `internal/css` Match / cascade | Phase 2 open-web CSS (more rules fire) |
| UA `a` defaults | Visible blue without author CSS |

---

## Out of scope

- Real visited-history tracking
- Full Vector link chrome without Phases 2/4/6
