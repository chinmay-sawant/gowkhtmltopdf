# Pending — Phase 8: SVG-as-`img` (wiki logo / icons)

> **Parent:** [`README.md`](README.md)  
> **Status:** done (out of scope by design — 2026-08-05)  
> **Estimated effort:** n/a unless product amends  
> **Prior plan coverage:** **Yes** — matrix / fidelity: SVG-as-`img` not decodable by stdlib  

---

## Overview

Chrome prints Wikipedia logo (SVG). We skip SVG images. Restoring logos needs
an SVG subset renderer or rasterization dependency — **product amendment**.

---

## Phase 8 checklist

### 8.1 Honesty (no implement)

- [x] Document: SVG-as-`img` unsupported — matrix / fidelity already state this
- [x] Ana “missing header logo” attributed to SVG (and skin CSS), not a Phase 1–7 regression
- [x] Next gate if amended: new phase under pending-phase-items or Phase 23 tooling

### 8.2 Gates

- [x] No code change required for this ledger pass
- [x] Status → **out of scope by design**

---

## Out of scope

- Implementing SVG here without explicit product amendment
