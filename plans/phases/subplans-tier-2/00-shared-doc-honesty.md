# Tier 2 Subplan 00 - Shared Documentation Honesty (phases 17–20)

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md` + phases 17–20 Pending tables  
> **Status:** done (2026-08-05)  
> **Estimated effort:** 0.5–1 day (docs only)  
> **Constraint:** documentation-only; no `make lint`/`make test` required per skill  
> **Normative file:** `documentation/compatibility-matrix.md`

---

## Overview

Tier 2 core behavior is shipped, but product docs still contradict README’s
deferred table. This subplan is the **single coordinated honesty pass** so
phases 17–20 do not fight over the same matrix rows.

**Source of truth for product status today:** `README.md` `#deferred--not-planned`
(already mostly correct). Matrix / fidelity feature-map / overview lag behind.

## Executive Summary

| Surface | Today | Target after this pass |
|---------|-------|------------------------|
| `compatibility-matrix.md` | Last audit 2026-08-04; denies flex/grid/thead/CJK/@font-face/internal links | Partial/Shipped aligned with code |
| `fidelity.md` feature map | 4+ stale rows | Match product positioning table |
| `overview.md` / README overview | “no flex/grid/CJK” | Report-subset Partial wording |
| CLI / library | Missing thead/zoom/font-path notes | Short additive subsections |
| Code leftovers | — | **Out of this subplan** → 19 audit, 20 HF GoTo |

---

## Phase 1: Freeze ownership

### 1.1 Conflict rule

- [x] Treat this file as the only matrix rewrite owner for Tier 2 pending
- [x] Per-phase subplans mark matrix rows `[x]` only after this pass lands
- [x] Do not open four separate matrix PRs

### 1.2 Normative target list

- [x] Normative: `documentation/compatibility-matrix.md`
- [x] Mirrors: `documentation/fidelity.md`, `documentation/overview.md`, `README.md` overview table
- [x] Operator additives: `documentation/cli.md`, `documentation/library-api.md`
- [x] Fonts refine only if Partial wording differs: `documentation/fonts.md` (owned by fontface agent — left alone)

---

## Phase 2: Matrix rewrite (single session)

Stamp header, then edit in this order (avoids internal contradictions).

### 2.1 Header stamp

- [x] Update **Last honesty audit** date + note “Tier 2 phases 17–20 core on master (#16/#17)”
- [x] Record base commit at edit time (`f5bb754`)

### 2.2 Phase 17 rows — display / position / flex / grid

**Known stale claims (evidence 2026-08-05):**

| Location | Stale text (summary) | Code truth |
|----------|----------------------|------------|
| §2.2 `position` | relative/absolute/fixed Not implemented | `layout.go` `buildAbsolute`/`buildFixed`/`applyRelativeOffset`; fixture-26/28 |
| §2.2 `display` allowlist | omits flex/grid | `style.go` accepts `flex\|inline-flex\|grid\|inline-grid` |
| Feature checklist Flexbox/Grid | **No** | `internal/layout/flex.go`, `grid.go` |
| §5 Explicitly unsupported | “flex/grid ignored → inline” | Must move to Partial |

Checklist:

- [x] §2.2: `position` → **Partial** — static/relative/absolute/fixed lite; sticky = relative-offset only (`applyRelativeOffset`); cite fixture-26/28
- [x] §2.2: extend `display` Implemented/Partial list with flex/grid values
- [x] Feature checklist: Flexbox/Grid → **Partial** + property bullets (see phase-17-pending supported subset)
- [x] Feature checklist: Floats/absolute → float lite + absolute/fixed lite; sticky deferred
- [x] §5: remove “flex/grid ignored” and “absolute/fixed → static”; rephrase as deferred-full / Partial

### 2.3 Phase 18 rows — pagination

**Known stale claims:**

| Location | Stale text | Code truth |
|----------|------------|------------|
| Pagination prose (~L133) | zoom not forwarded; smart-shrink warn-only; thead repeat no; orphan none | `convert.go` Zoom + smart-shrink re-layout; `paint.go` `repeatTableHeaders` + `orphansWidows` |
| §2.6 `orphans`/`widows` | Not implemented | Heuristics yes; CSS props absent ([css-break-3](https://www.w3.org/TR/css-break-3/)) |

Checklist:

- [x] Rewrite Pagination paragraph: zoom forwarded; smart-shrinking re-layouts; thead repeat shipped (`fixture-23`); orphan/widow **heuristics** (not CSS props)
- [x] §2.6: Status → **Partial (heuristics)**; note CSS `orphans`/`widows` still absent from `applyRestProps` / `internal/css`
- [x] Ensure §7.3 / `--zoom` rows stay consistent with new prose (they already say Supported)

### 2.4 Phase 19 rows — fonts / i18n / @font-face

**Known stale claims:**

| Location | Stale text | Code truth |
|----------|------------|------------|
| §4 `@page`, `@font-face` | Not implemented (wrong cite) | `@font-face` parsed; `@page` skipped; `mergeFontFaces` loads local TTF |
| §5 `@font-face` | Ignored → defaults | PDF path Partial under ACL |
| §5 RTL/CJK | Latin-first Phase 3 | Type0 + Arabic `ShapeText` + vertical-rl lite |
| §2.3 `font-family` | “until phase 19” | Registry + `--font-path` shipped |

Checklist:

- [x] §4: **split** `@page` (Not implemented) vs `@font-face` (**Partial**); fix stale `css.go` line cites
- [x] §5: replace `@font-face` Ignored with Partial wording (local TTF/OTF via `FetchSub`; WOFF/network skipped; image mode unwired)
- [x] §2.3: mention registry, `--font-path`, local `@font-face` (PDF)
- [x] §5 RTL/CJK: Type0/CID Identity-H; Arabic presentation-form joining; Hangul needs face; vertical-rl lite; Indic/HarfBuzz not claimed
- [x] Add `writing-mode: vertical-rl\|lr` Partial row (or note under text/fonts)
- [x] Add §7 flag rows: `--font-path`, `--use-system-fonts`

### 2.5 Phase 20 rows — links / HF

**Known stale claims:**

| Location | Stale text | Code truth |
|----------|------------|------------|
| §1 `a` | internal anchors deferred | Body GoTo shipped (`applyInternalLinks`, fixture-24) |
| §7.5 `--internal-links` | Ignored / never emits | Layout emits `#` `OpLinkURI`; convert resolves |
| Missing flags | no resolve/keep-relative rows | `resolveRelativeLinkURIs` shipped |

Checklist:

- [x] §1 `a`: body `#id` GoTo shipped; HTML HF **external URI** carried; HF **fragment GoTo** still limited (point to phase-20-pending)
- [x] §7.5 `--internal-links`: update to Supported/Partial with geometry caveats
- [x] Add `--resolve-relative-links` / `--keep-relative-links` Supported rows
- [x] Optional §7.7 HF note: HTML HF URI annotations on body pages; fragment GoTo pending

### 2.6 Matrix self-consistency gate

- [x] `rg -n 'flex/grid.*[Nn]o|thead repeat.*not implemented|Latin-only|@font-face.*[Ii]gnored|does not forward' documentation/compatibility-matrix.md` → no false negatives vs shipped Tier 2
- [x] Pagination prose does not contradict §7.3 / `--zoom`

---

## Phase 3: Fidelity + product surfaces

### 3.1 `documentation/fidelity.md`

- [x] Feature map: floats/flex/position/grid → Partial / shipped lite (not “No”)
- [x] Feature map: Pagination / thead → shipped (breaks + thead; orphans heuristics)
- [x] Feature map: Fonts / CJK → Partial (Type0 + discovery; not “Latin only / 19 next”)
- [x] Failure modes: narrow “Unsupported display (flex/grid)” to full-Grid / unknown values only
- [x] Soft: Tier “as of” blurb → Tier 2 core shipped

### 3.2 `documentation/overview.md`

- [x] `#what-it-is-not`: remove “no flex/grid/floats/position” and “no CJK/CID yet”
- [x] Replace with honest report-subset / Partial language + matrix link

### 3.3 `README.md` overview table

- [x] Align “Full CSS” / “CJK” overview rows with deferred table (Partial / shipped lite)
- [x] Do **not** invent claims beyond deferred table + matrix

### 3.4 Plans hygiene (same honesty PR or follow-up)

- [x] `plans/10-canonical-post-mvp-roadmap.md` “Current product facts” flex/grid/position bullets
- [x] Point phase 17–20 Pending “matrix/fidelity” rows at this subplan as `[~]` covered / or `[x]` when done

---

## Phase 4: CLI / library additives

### 4.1 `documentation/cli.md`

- [x] Add short **Pagination & tables**: thead repeat; `--zoom` / smart-shrinking; orphans/widows = heuristics not CSS
- [x] Add **Fonts & links**: `--font-path`, `--use-system-fonts`, `--resolve-relative-links`
- [x] Fix adjacent stale “bitmap font” image-mode line if still present (Phase 15 honesty)

### 4.2 `documentation/library-api.md`

- [x] Under settings: one paragraph or bullets pointing at matrix §2.6 / fonts / links
- [x] Mention thead repeat + Zoom / smart-shrinking settings keys

### 4.3 Optional inventory

- [x] `documentation/samples.md`: fixture range includes 23+ (not stuck at fixture-21)

---

## Phase 5: Closure

### 5.1 Done when

- [x] Matrix no longer denies flex/grid, thead, Type0/CJK, local `@font-face`, or body internal links
- [x] Fidelity feature-map matches product positioning
- [x] Overview + README overview match deferred table
- [x] Parent phase Pending “matrix/fidelity” items can flip to `[x]` (or `[~]` pointer here)

### 5.2 Explicitly not this subplan

- [~] Explicitly not this subplan:
  - Phase 19 `@font-face` E2E / image-mode → phase-19-pending
  - Phase 20 HF fragment GoTo → phase-20-pending
  - (fixtures 29/30 already added on branch)

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Shipped Tier 2 code on master | Honest Partial labels |
| README deferred table | Wording source of truth |
| This pass | Phase 17–20 pending doc rows closable |

---

## Out of scope

- Implementing sticky, full Grid/Flex, CSS `orphans`/`widows` parsing, HarfBuzz, WOFF
- Rewriting exploration/ historical plans except known-stale phase-06 limitations (owned by phase-20-pending)
