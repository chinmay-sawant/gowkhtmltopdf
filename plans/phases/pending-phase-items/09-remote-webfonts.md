# Pending — Phase 9: Remote webfonts (WOFF2 / HTTPS `@font-face`)

> **Parent:** [`README.md`](README.md)  
> **Status:** done (out of scope by design — 2026-08-05)  
> **Estimated effort:** n/a unless product amends ACL/network + Brotli allowlist  
> **Prior plan coverage:** **Yes** — Phase 19 / fonts-remaining permanent non-goal  

---

## Overview

Chrome loads wiki webfonts. We intentionally skip remote `https://` `@font-face`
and WOFF2. **Phase 3 local/system fallback** is the supported alternative.

---

## Phase 9 checklist

### 9.1 Honesty

- [x] Reaffirm remote WOFF2 / HTTPS `@font-face` out of scope (fonts.md / matrix)
- [x] Point implementers to Phase 3 glyph fallback + `--font-path`
- [x] Status → **out of scope by design**

### 9.2 Gates

- [x] No code change in this ledger unless product amends

---

## Out of scope

- CDN webfont auto-fetch
- Brotli/WOFF2 without allowlist amendment
