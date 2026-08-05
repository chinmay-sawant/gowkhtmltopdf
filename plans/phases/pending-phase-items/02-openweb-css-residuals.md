# Pending — Phase 2: Open-web CSS residuals (Phase 17 / 21 leftovers)

> **Parent:** [`README.md`](README.md)  
> **Status:** not started  
> **Estimated effort:** 1–3 weeks (slice by symptom)  
> **Prior plan coverage:** **Yes** — Tier-2 pending-3 shipped float/table/multicol/etc.; Phase 21 §21.3 still lists **residuals** (infobox float, large skins)  

---

## Overview

Core Phase 17 pending-3 work is done, but live Wikipedia still shows weak
infobox packing, odd spacing, and ~3× Chrome page count. This phase attacks
**open-web residuals** with evidence from Ana smoke + vendored fixtures — not
Chrome pixel parity.

### Smoke proof

```sh
./bin/gowkhtmltopdf 'https://en.wikipedia.org/wiki/Ana_de_Armas' output/wiki-ana-de-armas.pdf
```

Track: page count band vs prior smoke; infobox beside lead; less catastrophic
whitespace. Reference: `output/chrome_ana.pdf`.

---

## Phase 2 checklist

### 2.1 Inventory (evidence-first)

- [ ] Capture concrete CSS/layout failures from Ana PDF vs Chrome (list with screenshots or page/notes): float/infobox, citation spacing, section rules, nav chrome volume
- [ ] Map each item to code seam (`float.go`, `style.go`, table, paint) or “needs selector/media (other phase)”
- [ ] Prefer fixes that help `testdata/web/wiki-like-article.html` + marketing fixture too

### 2.2 Float / infobox

- [ ] Improve `table.infobox { float: right }` + following prose wrap on vendored wiki-like fixture (extend fixture if needed)
- [ ] Clear/intrusion edge: content after infobox does not leave huge empty bands without cause
- [ ] Proof: layout and/or convert test + optional Ana smoke note

### 2.3 Tables / citation chrome

- [ ] Reduce pathological spacing in `[n]` / superscript-like citation runs where caused by our inline/box model (not missing CSS)
- [ ] Infobox table cell stacking: labels/values readable (Partial OK)

### 2.4 Pagination density

- [ ] Identify top contributors to page-count inflation (unused chrome, large margins, failed `display:none`, missing print CSS → Phase 4)
- [ ] Fix at least one density bug with before/after page-count note on Ana smoke or fixture

### 2.5 Docs

- [ ] Update Phase 21 §21.3 inventory with what was fixed vs remains
- [ ] Matrix notes only if behavior claim changes

### 2.6 Gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Smoke command recorded (page count before/after if measured)
- [ ] Status → done or Partial with remaining bullets listed

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 1 `:link` (optional but helpful) | Visible link affordance while tuning layout |
| Pending-3 float/table | Baseline |
| Phase 4 print media | Further density |

---

## Out of scope

- Matching Chrome’s exact 10-page pagination
- Full Vector skin fidelity
