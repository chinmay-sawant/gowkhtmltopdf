# Pending — Phase 9: Remote webfonts (WOFF2 / HTTPS `@font-face`)

> **Parent:** [`README.md`](README.md)  
> **Status:** in progress (reopened — must ship, not defer)  
> **Estimated effort:** days (needs allowlist amendment for Brotli or WOFF2 decoder)  
> **Prior plan coverage:** **Yes** — Phase 19 / fonts-remaining permanent non-goal — **superseded by this ledger**  

---

## Overview

Chrome loads wiki webfonts. Implement remote `https://` `@font-face` under the
existing loader ACL/timeouts, plus WOFF2 decode (Brotli). Phase 3 local fallback
remains the offline path.

---

## Phase 9 checklist

### 9.1 Implement

- [ ] Amendment: allow Brotli (or `go-text` WOFF2 path) as direct module if required
- [ ] Fetch `https://` `@font-face` src via `FetchSub` (same ACL as images/CSS)
- [ ] Decode WOFF2 → SFNT; register face like WOFF1
- [ ] Tests: local WOFF2 fixture; optional httptest remote fetch test
- [ ] fonts.md / matrix updated

### 9.2 Gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Status → done

---

## Out of scope

- Auto-installing system font packages
- CGO HarfBuzz
