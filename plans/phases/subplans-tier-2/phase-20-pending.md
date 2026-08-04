# Tier 2 Subplan - Phase 20 Pending (HTML HF fragment GoTo + link honesty)

> **Parent:** [`plans/phases/phase-20-hf-links-edges.md`](../phase-20-hf-links-edges.md) — Pending (after #17)  
> **Status:** not started  
> **Estimated effort:** 1–3 days code + tests; docs via shared pass  
> **Depends on:** Phase 6 locations map; [00-shared-doc-honesty.md](00-shared-doc-honesty.md) for matrix link rows  
> **Constraint:** stdlib-only; no full nested HTML HF documents

---

## Overview

Phase 20 **core is shipped**: body inline `#` GoTo, `resolveRelativeLinks`,
`[topage]` under copies, dump-outline TOC offset, HTML HF **external URI**
annotations on body pages. The real leftover is **same-document fragment GoTo
from HTML headers/footers** (`href="#id"`). Matrix/fidelity still lag on several
already-shipped link behaviors (shared honesty pass).

## Executive Summary

| Pending item (parent) | Disposition | Primary work |
|-----------------------|-------------|--------------|
| HTML HF → body fragment (`#id`) GoTo | **Must (code)** | Replace skip in `drawHTMLHF` with `AddLinkDest` |
| Shared matrix/fidelity refresh | **Must (docs)** | Shared Pass 0 |
| Full HTML HF as nested documents | Out of scope | Keep `[~]` |

---

## Phase 1: Evidence baseline (scanned 2026-08-05)

### 1.1 Pipeline order (`internal/convert/convert.go`)

1. Body/TOC paint
2. `applyTOCLinks` → `applyInternalLinks` (body `#frag` → GoTo) — **before** HF
3. `materializeCopies` (if copies > 1)
4. `drawHeadersFooters` — after copies so `[page]`/`[topage]` are final

### 1.2 Current HF link behavior (`internal/convert/hf.go`)

```356:379:internal/convert/hf.go
		case layout.OpLinkURI:
			// Carry HTML HF link annotations onto the body page band.
			// ...
			if len(uri) > 0 && uri[0] == '#' {
				// Same-document fragments in HF: best-effort URI leave as-is;
				// convert may not resolve HF GoTo targets to body ids.
				break
			}
			page.AddLinkURI(rect, uri)
```

| Concern | Body path | HTML HF path |
|---------|-----------|--------------|
| External URI | `drawLink` → `AddLinkURI` | `drawHTMLHF` → `AddLinkURI` |
| `#id` GoTo | `applyInternalLinks` → `AddLinkDest` | **Explicit skip** |
| `resolveRelativeLinkURIs` | Yes | **No** |
| `ExternalLinks` / `LocalLinks` | Yes | **Not applied** to HF ops |
| Locations / id index | Body `res.Locations` | HF result unused for GoTo |

### 1.3 Why `#` cannot stay as URI

Per PDF / ISO 32000 Link annotations: internal navigation uses **GoTo** (or `/Dest`)
with an explicit destination such as `[page_ref /XYZ left top zoom]`. Emitting
`AddLinkURI(rect, "#id")` is **not** a same-document GoTo. Existing API:
`Page.AddLinkDest(rect, pageIdx, x, y)` in `internal/pdf/pdf.go`.

### 1.4 Root cause (not missing layout emission)

1. HF HTML is a **separate** layout (`loadHTMLHF`) — body ids live on `objectState.res.Locations`
2. `applyInternalLinks` only walks body ops and runs **before** HF draw
3. Hard `#` skip in `drawHTMLHF`
4. HF draws **after** copies → dest page indices must use the **final** page set

---

## Phase 2: Documentation honesty (shared + phase-06 hygiene)

### 2.1 Shared matrix / fidelity

Owned by [00-shared-doc-honesty.md](00-shared-doc-honesty.md) §2.5:

- [ ] §1 `a`: body internal anchors shipped; HF URI carried; HF fragment still limited
- [ ] §7.5 `--internal-links` wording fixed
- [ ] Add `--resolve-relative-links` / `--keep-relative-links` rows
- [ ] Optional HF HTML link note under header/footer section
- [ ] Fidelity feature-map cross-cutting rows (owned shared; includes HF Partial)

### 2.2 Phase-06 known-limitations rewrite

- [ ] `plans/phases/phase-06-headers-toc-outline.md`: rewrite Known limitations / §6.5 so the **only** remaining link gap called out is HF fragment GoTo
- [ ] Proof: `rg -n 'not carried|resolveRelativeLinks deferred|topage.*copies' plans/phases/phase-06-headers-toc-outline.md` → no stale claims
- [ ] Qualify `plans/PR/pr-tier-2.md` HF bullet as URI shipped / fragment pending (if still wrong)

### 2.3 Already honest

- [x] `README.md` deferred: resolveRelativeLinks shipped; HF URI carried; fragment limited

---

## Phase 3: Implementation — HF `#id` → body GoTo

### 3.1 Shared id index

- [ ] Extract `buildBodyIDIndex(bodies []*objectState)` from `applyInternalLinks` loop in `internal/convert/links.go`
  - Key: `loc.Node.Attribute("id")`
  - Value: `{st *objectState, loc layout.ElementLocation}`
- [ ] Reuse from `applyInternalLinks` (no behavior change for body links)
- [ ] Proof: existing `go test ./internal/convert -run 'TestInternalLinkDest|TestExternalLinks|TestResolve' -count=1`

### 3.2 Emit GoTo from `drawHTMLHF`

- [ ] Thread id index (+ page-index helper) into `drawHeadersFooters` → `drawHTMLHF`
- [ ] On `OpLinkURI` with `uri[0]=='#'`:
  - Resolve id (strip `#`)
  - If missing id → silent skip (parity with body)
  - If `!LocalLinks` → skip
  - Compute dest: `destPage = tocTotal + dest.st.offset + dest.loc.Page` then **remap for copies**
  - Dest point: reuse `destPoint(dest.loc, dest.st.geom)` (or equivalent)
  - Source rect: already computed HF band rect (pad W/H ≤0 to 10pt as today)
  - Emit: `page.AddLinkDest(rect, destPageIdx, dx, dy)` — **not** `AddLinkURI`
- [ ] Proof: new `TestHTMLHeaderFragmentGoTo` in `phase6_test.go` (or dedicated file)

### 3.3 Copies-aware dest pages (risk hotspot)

HF is drawn on **post-copy** pages. Body `applyInternalLinks` annotates pre-copy
pages and relies on `DuplicatePage` to copy annots. HF must target final indices.

- [ ] Map logical body page → final index using same owner/copy rules as `drawHeadersFooters`
  - Collate: `p % logicalN` / copy group
  - Non-collate: `p / copies` style ownership (match existing HF page loop exactly)
- [ ] Test: `TestHTMLHeaderFragmentGoToCopies` with `--copies 2` (collate and/or non-collate)
- [ ] Assert `/Dest` present and **no** `/URI` for that clickable HF rect

### 3.4 SHOULD parity (small, same feature)

- [ ] Apply `resolveRelativeLinkURIs` to HF ops (base = `htmlHFLayout.base`); leave `#frag` alone
- [ ] Gate external HF links with `ExternalLinks` (parity with body)
- [ ] Optional: unit assert HF external URI annotation exists (gap today — only text HF tests)

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

- [ ] `TestHTMLHeaderFragmentGoTo`:
  1. Body with `<h2 id="target">` forced to page 2 (`page-break-before`)
  2. Separate `--header-html` / `--footer-html` file with `<a href="#target">…</a>`
  3. `--enable-local-file-access` + HF URL path (not raw markup)
  4. Assert PDF has `/Dest` / GoTo (and preferably no `/URI` for that rect)
- [ ] `TestHTMLHeaderFragmentGoToCopies` (see 3.3)

### 4.3 Optional golden

- [~] `testdata/golden/fixture-29-hf-fragment-link.html` + envelope
- [~] Golden runner may need `gotos`/`dests` flag — or keep asserts in convert tests only
- [~] Optional strengthen fixture-24 golden to assert `/Dest` (quality, not HF feature)

---

## Phase 5: Closure gates

### 5.1 Required

- [ ] HF fragment GoTo tests green
- [ ] Shared doc-honesty link rows landed
- [ ] Parent Phase 20 Pending HF fragment row → `[x]` (or `[~]` if only partial copies path)
- [ ] Full suite: `make lint` → ; `make test` → ; if golden added `make golden` → ; record outcomes

### 5.2 Done when

- [ ] Clicking `#id` in HTML HF navigates to body element page (GoTo/XYZ)
- [ ] External HF URIs still work
- [ ] Missing id / LocalLinks=false → no bogus annotation
- [ ] Copies > 1 dest page index correct for tested collate mode(s)
- [ ] Docs no longer claim body internal links / resolveRelativeLinks as deferred

### 5.3 Next

- [ ] Phase 21 arbitrary websites (product), or close Tier 2 pending entirely

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
