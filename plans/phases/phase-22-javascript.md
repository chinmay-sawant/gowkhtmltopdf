# Phase 22 — JavaScript Support (Staged, Stdlib-Only)

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** not started  
> **Estimated effort:** Stage A–B weeks; Stage C research months; Stage D **not planned** under pure stdlib  
> **Depends on:** Phase 21 layout breadth; security model  
> **Unblocks:** “JS-driven pages” only within staged capability  
> **Tier:** 2→3 · **Constraint:** **no third-party JS engines or browser embeds**

---

## Overview

Users asked for **full JavaScript support**. Under the project rule (Go stdlib only, no plugins, no Chrome), a complete ES + DOM + Web APIs stack is a **multi-year product**—closer to writing a browser than a PDF tool. This phase is **honest and staged**: ship only what can be proven safe and useful; gate “full support” behind a formal plan amendment if constraints change.

## Executive Summary

| Today | Evidence |
|-------|----------|
| `<script>` stripped at load | `internal/html` / load pipeline |
| `--enable-javascript` accepted + warned | CLI flags |
| `jsdelay` / `windowStatus` / `runScript` | settings stubs |
| No VM | feasibility: stdlib has no JS engine |

**Product truth:** full WebKit-class JS is **Tier 3**. Stages A–C may still help report templates.

---

## Phase 22 checklist

### 22.0 Constraint & security freeze (must complete first)

- [ ] Written policy: scripts **off by default**; enable only with explicit flag
- [ ] Threat model update: script can be SSRF/exfil vector if network APIs exposed—default deny
- [ ] Decision record: remain pure-stdlib **or** amend plan to allow an engine (would be a new ledger)
- [ ] No stage may `os/exec` a browser or load third-party `.so`

### 22.1 Stage A — Honest flags & delay semantics (quick win)

- [ ] Document all JS-related flags as: ignored / sleep-only / stripped
- [ ] `--javascript-delay`: keep as post-load **sleep** only; test timing
- [ ] `--window-status`: document unsupported until Stage C+
- [ ] `--run-script`: document unsupported or queue for Stage B
- [ ] Matrix + CLI help strings accurate (no false “enabled”)
- [ ] Effort: days–1 week

### 22.2 Stage B — Report-oriented non-JS alternatives (quick product value)

Many “JS needs” for invoices are actually:

- [ ] Server-side data already in HTML (document pattern)
- [ ] Optional **Go template** pre-pass for embedders (library helper)—not browser JS
- [ ] CSS print rules instead of JS show/hide where possible
- [ ] Do **not** claim this is JavaScript
- [ ] Effort: 1–2 weeks if product wants

### 22.3 Stage C — Research spike: pure-Go ES subset (go/no-go)

- [ ] Spike goals: parse + evaluate tiny expression language **or** minimal ES5 subset **without** deps
- [ ] Scope candidates (pick one for spike):
  - expressions only (no DOM)
  - scripts that set `document.title` / simple textContent on ids
  - no `eval` of network-fetched code by default
- [ ] Estimate LOC / years for DOM (`querySelector`, layout readback, timers)
- [ ] **Go/no-go gate:** write ADR in `plans/` or `documentation/`:
  - **Go:** define Stage C.1 implementable subset with tests
  - **No-go:** mark full JS `[~]` → point to Phase 23 / external tools
- [ ] Effort: 2–6 weeks research; implementation separate estimate

### 22.4 Stage C.1 — Minimal script subset (only if Stage C go)

- [ ] Enable path: `--enable-javascript` actually runs subset after parse, before layout
- [ ] Sandbox: no filesystem, no process, no raw network (or network only via existing loader ACL)
- [ ] DOM surface allowlist documented (e.g. getElementById text only)
- [ ] Time/memory caps; stop-slow-scripts behavior real
- [ ] Fuzz parser/evaluator
- [ ] Tests: script changes text node → PDF reflects change
- [ ] Security review checklist signed
- [ ] Effort: months for even a tiny DOM

### 22.5 Stage D — Full browser JS/DOM (deferred)

- [~] Full ES + HTML5 DOM + layout readback + timers + XHR  
  **Reason:** unbounded; stdlib-only reimplementation is the wrong tool  
  **Next gate:** product amendment allowing Chrome headless / embedded runtime **or** multi-year dedicated engine project  
  **Pointer:** `phases/phase-23-tier3-deferred.md`  
  **Do not** start Stage D work inside Tier 1/2 without amendment

### 22.6 Docs for any shipped stage

- [ ] Matrix JavaScript row: exact stage language
- [ ] Fidelity: “JS-driven pages” only if Stage C.1+ shipped
- [ ] README deferred table updated
- [ ] Threat model + integration-security updated

### 22.7 Closure gates (per stage)

- [ ] Stage A: docs-only or flag wording — lint/test if code touched
- [ ] Stage B/C.1: `make lint` + `make test` recorded
- [ ] Parent Phase 22 rows reflect stage status (`[x]` / `[~]`)
- [ ] Handoff: if no-go, active work ends at Phase 21 + Tier 2 polish

```
# closure evidence per stage
# stage:
# make lint →
# make test →
```

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Loader script strip points | Execution hook site |
| Layout/DOM tree | Any script that mutates tree |
| Threat model | Safe defaults |

---

## Out of scope (unless plan amended)

- V8/JavaScriptCore/QuickJS as dependencies
- Puppeteer/Playwright/Chrome headless integration inside this module
- Plugins / NPAPI / Java applets
- Service workers, WebAssembly browser APIs
