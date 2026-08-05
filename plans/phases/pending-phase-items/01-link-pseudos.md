# Pending — Phase 1: `:link` / `:visited` print semantics (blue links)

> **Parent:** [`README.md`](README.md)  
> **Status:** not started  
> **Estimated effort:** 0.5–2 days  
> **Prior plan coverage:** Matrix lists `:link`/`:visited` as accepted & **ignored** — **no prior implement checklist**. Closest: Phase 17 selectors / Phase 21 §21.3  

---

## Overview

Wikipedia Vector (and many sites) color hyperlinks with `a:link` / `a:visited`.
Our matcher ignores those pseudos, so skin rules never apply and links often
render as body text (black) despite UA default `a { color: #0000ee }`.

**Print semantics:** treat `:link` and `:visited` as matching any `a[href]`
(visited state is meaningless in static PDF). Do **not** implement `:hover` /
`:active` / `:focus` interaction.

### Smoke proof

```sh
./bin/gowkhtmltopdf 'https://en.wikipedia.org/wiki/Ana_de_Armas' output/wiki-ana-de-armas.pdf
```

Expect article body links to paint a distinct link color (wiki blue / cascaded
`color`), not identical to surrounding prose — compare visually to
`output/chrome_ana.pdf` (parity not required).

---

## Phase 1 checklist

### 1.1 Parse & match

- [ ] Parse `:link` and `:visited` on compounds in `internal/css` (same path as other simple pseudos)
- [ ] `Match`: `:link` / `:visited` succeed when element is `a` with non-empty `href` (any scheme including `#` / relative)
- [ ] `Match`: `:link` / `:visited` fail when no `href` or not an `a`
- [ ] Still ignore `:hover`, `:active`, `:focus` (no match; do not break compounds that include them if already ignored)

### 1.2 Cascade / UA

- [ ] Confirm UA `a { color: #0000ee; text-decoration: underline }` in `style.go` still applies when no author override
- [ ] Author `a:link { color: … }` wins over UA per normal cascade/specificity
- [ ] `a { color: inherit }` or article-color rules without `:link` still apply as today

### 1.3 Tests

- [ ] Unit: stylesheet `a:link { color: #0645ad }` colors an `<a href="…">` text op (not black/default body)
- [ ] Unit: `a:visited` same as `:link` for print
- [ ] Unit: bare `<a>no href</a>` does not match `:link`
- [ ] Regression: existing link URI / GoTo tests (`fixture-24`, HF links) still green

### 1.4 Docs & matrix

- [ ] Matrix Selectors: `:link` / `:visited` → Partial (print = has-href; no history)
- [ ] Note in fidelity / Phase 21 residuals if needed

### 1.5 Gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Optional smoke: regenerate `output/wiki-ana-de-armas.pdf` with the raw command; record visual note
- [ ] Flip this file **Status** → done; README order-1 row → done

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| `internal/css` Match / cascade | Phase 2 open-web CSS (more rules fire) |
| UA `a` defaults | Visible blue without author CSS |

---

## Out of scope

- Real visited-history tracking
- `:hover` / `:active` / `:focus`
- Underline-only vs color site quirks beyond cascade
