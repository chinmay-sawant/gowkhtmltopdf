# Tier 2 Subplan - Phase 20 Pending (HTML HF fragment GoTo + link honesty)

> **Parent:** [`plans/phases/phase-20-hf-links-edges.md`](../phase-20-hf-links-edges.md) — Pending (after #17)  
> **Status:** **done** (HF fragment GoTo + matrix/fidelity honesty)  
> **Estimated effort:** 1–3 days code + tests; docs via shared pass  
> **Depends on:** Phase 6 locations map; [00-shared-doc-honesty.md](00-shared-doc-honesty.md) for matrix link rows  
> **Constraint:** stdlib-only; no full nested HTML HF documents

---

## Overview

Phase 20 **core is shipped**: body inline `#` GoTo, `resolveRelativeLinks`,
`[topage]` under copies, dump-outline TOC offset, HTML HF **external URI**
annotations on body pages, and **same-document fragment GoTo from HTML
headers/footers** (`href="#id"` → body `AddLinkDest`). Matrix/fidelity still
lag on several already-shipped link behaviors (shared honesty pass).

## Executive Summary

| Pending item (parent) | Disposition | Primary work |
|-----------------------|-------------|--------------|
| HTML HF → body fragment (`#id`) GoTo | **Shipped (code)** | `drawHTMLHF` → `AddLinkDest` via `buildBodyIDIndex` |
| Shared matrix/fidelity refresh | **[x] done** | Shared Pass 0 |
| Full HTML HF as nested documents | Out of scope | Keep `[~]` → v0.3.0 |

---

## Phase 1: Evidence baseline (scanned 2026-08-05)

### 1.1 Pipeline order (`internal/convert/convert.go`)

1. Body/TOC paint
2. `applyTOCLinks` → `applyInternalLinks` (body `#frag` → GoTo) — **before** HF
3. `materializeCopies` (if copies > 1)
4. `drawHeadersFooters` — after copies so `[page]`/`[topage]` are final

### 1.2 Current HF link behavior (`internal/convert/hf.go`)

| Concern | Body path | HTML HF path |
|---------|-----------|--------------|
| External URI | `drawLink` → `AddLinkURI` | `drawHTMLHF` → `AddLinkURI` (ExternalLinks gated) |
| `#id` GoTo | `applyInternalLinks` → `AddLinkDest` | `drawHTMLHF` → `AddLinkDest` (LocalLinks; copies-aware) |
| `resolveRelativeLinkURIs` | Yes | Yes (HF base) |
| `ExternalLinks` / `LocalLinks` | Yes | Yes |
| Locations / id index | Body `res.Locations` via `buildBodyIDIndex` | Same index threaded into HF draw |

### 1.3 Why `#` cannot stay as URI

Per PDF / ISO 32000 Link annotations: internal navigation uses **GoTo** (or `/Dest`)
with an explicit destination such as `[page_ref /XYZ left top zoom]`. Emitting
`AddLinkURI(rect, "#id")` is **not** a same-document GoTo. Existing API:
`Page.AddLinkDest(rect, pageIdx, x, y)` in `internal/pdf/pdf.go`.

### 1.4 Root cause (not missing layout emission) — fixed

1. HF HTML is a **separate** layout (`loadHTMLHF`) — body ids live on `objectState.res.Locations`
2. `applyInternalLinks` only walks body ops and runs **before** HF draw
3. ~~Hard `#` skip in `drawHTMLHF`~~ → resolves via `buildBodyIDIndex` + `AddLinkDest`
4. HF draws **after** copies → dest page indices remapped with `remapPageForCopies`

---

## Phase 2: Documentation honesty (shared + phase-06 hygiene)

### 2.1 Shared matrix / fidelity

Owned by [00-shared-doc-honesty.md](00-shared-doc-honesty.md) §2.5:

- [x] §1 `a`: body internal anchors shipped; HF URI + fragment GoTo shipped
- [x] §7.5 `--internal-links` wording fixed
- [x] Add `--resolve-relative-links` / `--keep-relative-links` rows
- [x] Optional HF HTML link note under header/footer section
- [x] Fidelity feature-map cross-cutting rows (includes HF fragment GoTo shipped)

### 2.2 Phase-06 known-limitations rewrite

- [x] `plans/phases/phase-06-headers-toc-outline.md`: rewrite Known limitations / §6.5 — HF fragment GoTo shipped; no stale deferred claims
- [x] Proof: `rg -n 'not carried|resolveRelativeLinks deferred|topage.*copies' plans/phases/phase-06-headers-toc-outline.md` → no stale claims
- [x] Qualify `plans/PR/pr-tier-2.md` HF bullet — URI + fragment GoTo both shipped (matrix/README are source of truth)

### 2.3 Already honest

- [x] `README.md` deferred: HF URI + fragment GoTo shipped; nested HF docs out of scope

---

## Phase 3: Implementation — HF `#id` → body GoTo

### 3.1 Shared id index

- [x] Extract `buildBodyIDIndex(bodies []*objectState)` from `applyInternalLinks` loop in `internal/convert/links.go`
  - Key: `loc.Node.Attribute("id")`
  - Value: `{st *objectState, loc layout.ElementLocation}`
- [x] Reuse from `applyInternalLinks` (no behavior change for body links)
- [x] Proof: existing `go test ./internal/convert -run 'TestInternalLinkDest|TestExternalLinks|TestResolve' -count=1`

### 3.2 Emit GoTo from `drawHTMLHF`

- [x] Thread id index (+ page-index helper) into `drawHeadersFooters` → `drawHTMLHF`
- [x] On `OpLinkURI` with `uri[0]=='#'`:
  - Resolve id (strip `#`)
  - If missing id → silent skip (parity with body)
  - If `!LocalLinks` → skip
  - Compute dest: `destPage = tocTotal + dest.st.offset + dest.loc.Page` then **remap for copies**
  - Dest point: reuse `destPoint(dest.loc, dest.st.geom)` (or equivalent)
  - Source rect: already computed HF band rect (pad W/H ≤0 to 10pt as today)
  - Emit: `page.AddLinkDest(rect, destPageIdx, dx, dy)` — **not** `AddLinkURI`
- [x] Proof: `TestHTMLHeaderFragmentGoTo` in `hf_links_test.go`

### 3.3 Copies-aware dest pages (risk hotspot)

HF is drawn on **post-copy** pages. Body `applyInternalLinks` annotates pre-copy
pages and relies on `DuplicatePage` to copy annots. HF must target final indices.

- [x] Map logical body page → final index using same owner/copy rules as `drawHeadersFooters`
  - Collate: `p % logicalN` / copy group
  - Non-collate: `p / copies` style ownership (match existing HF page loop exactly)
- [x] Test: `TestHTMLHeaderFragmentGoToCopies` + `TestHTMLHeaderFragmentGoToCopiesNonCollate`
- [x] Assert `/Dest` present and **no** `/URI` for that clickable HF rect

### 3.4 SHOULD parity (small, same feature)

- [x] Apply `resolveRelativeLinkURIs` to HF ops (base = `htmlHFLayout.base`); leave `#frag` alone
- [x] Gate external HF links with `ExternalLinks` (parity with body)
- [x] Optional: `TestHTMLHeaderExternalURI` asserts HF external URI annotation

---

## Phase 4: Fixtures / tests

### 4.1 Existing coverage

| Asset | Covers | Gap |
|-------|--------|-----|
| fixture-06 | Body external URI | No HF |
| fixture-24 | Body `#id` GoTo | No `/Dest` assert in golden; no HF |
| `links_resolve_test.go` | Relative resolve | Body only |
| `TestHTMLHeader*` | HF text/placeholders | No links |

### 4.2 New tests (required)

- [x] `TestHTMLHeaderFragmentGoTo`:
  1. Body with `<h2 id="target">` forced to page 2 (`page-break-before`)
  2. Separate `--header-html` / `--footer-html` file with `<a href="#target">…</a>`
  3. `--enable-local-file-access` + HF URL path (not raw markup)
  4. Assert PDF has `/Dest` / GoTo (and preferably no `/URI` for that rect)
- [x] `TestHTMLHeaderFragmentGoToCopies` (see 3.3)

### 4.3 Optional golden

- [~] `testdata/golden/fixture-29-hf-fragment-link.html` + envelope
- [~] Golden runner may need `gotos`/`dests` flag — or keep asserts in convert tests only
- [~] Optional strengthen fixture-24 golden to assert `/Dest` (quality, not HF feature)

---

## Phase 5: Closure gates

### 5.1 Required

- [x] HF fragment GoTo tests green
- [x] Shared doc-honesty link rows landed
- [x] Parent Phase 20 Pending HF fragment row → `[x]`
- [x] Convert package tests + lint recorded below

### 5.2 Done when

- [x] Clicking `#id` in HTML HF navigates to body element page (GoTo/XYZ)
- [x] External HF URIs still work
- [x] Missing id / LocalLinks=false → no bogus annotation
- [x] Copies > 1 dest page index correct for tested collate mode(s)
- [x] Phase-06 docs no longer claim body internal links / resolveRelativeLinks as deferred

### 5.3 Next

- [ ] Phase 21 arbitrary websites (product next)

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 6 `Locations` + `AddLinkDest` | Dest geometry |
| Phase 18 page numbers / copies order | Consistent HF page indices |
| Shared doc-honesty | Matrix link honesty |
| This GoTo slice | Close last Phase 20 functional leftover |

---

## Risks

- **Copies × GoTo:** easiest bug — wrong dest after collate/non-collate
- **HF-only ids:** `#id` means **body** destination, not an id inside the HF HTML tree
- **CLI LocalLinks defaults:** pre-existing quirk; don’t block HF GoTo on fixing CLI
- **Silent skip** on missing id — document in matrix honesty

---

## Out of scope

- Full HTML HF as nested full browser documents (Phase 6 model stands)
- PDF named actions beyond GoTo/URI
- `name=` destinations (body index is `id` only today)
- Making HF Locations participate in body layout
