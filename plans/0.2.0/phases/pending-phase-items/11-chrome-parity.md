# Pending — Phase 11: Chrome / Playwright parity

> **Parent:** [`README.md`](README.md)  
> **Status:** in progress (reopened — ship comparison harness + close visible Ana gaps)  
> **Estimated effort:** ongoing; full pixel parity may remain Partial honesty  
> **Prior plan coverage:** **Yes** — Phase 23 Tier 3 deferred — **harness lives here first**  

---

## Overview

Do not treat Chrome parity as “someone else’s phase.” This ledger ships:

1. Automated visual/structure compare vs `output/chrome_ana.pdf`
2. Closure of gaps found (links, density, SVG, fonts) until decent-print bar

Pixel-identical Chrome is still unlikely under stdlib constraints — but work
happens **here** until the compare harness is green on agreed metrics.

---

## Phase 11 checklist

- [x] Script/tool: `scripts/compare_chrome_ana.py` (pages, links, underline stats vs chrome_ana)
- [x] Record Ana metrics after fix wave (ours ~32pp vs chrome 10pp; underlines via horiz lines)
- [ ] Drive remaining open items from compare failures (density, SVG logo, WOFF2)
- [ ] Explicit metrics for “done enough” (not “ignore”)

### Gates

- [ ] Harness runnable from repo
- [ ] `make lint` / `make test` when code lands
- [ ] Status → done when harness criteria met (document numbers)
