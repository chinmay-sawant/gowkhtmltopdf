## Summary

Closes the Tier 2 pending-3 ledger (phases 17–20 leftovers): nested HTML HF as
child documents, CSS `orphans`/`widows`, float↔table packing, multicol lite,
static 2D transforms, `:has()` + `@container` size lite, flex/sticky polish,
and WOFF1 + `halt`/`palt`. Also starts Phase 21 with a product “decent print”
contract, vendored web fixtures, opt-in `--simplify-dom` chrome-strip, and
golden fixture-36 for nested HF.

---

## Motivation / context

- Plans: `plans/phases/tier-2-pending-3/` · `plans/phases/phase-21-arbitrary-websites.md`
- Branch: `feature/tier-2-pending-3` from `master` @ #19 merge
- Constraint: **stdlib layout** + allowlisted `go-text/typesetting`; no CGO HarfBuzz; no browser embed
- Disposition: permanent product boundaries marked `[x]` out of scope (no `[~]` deferrals)

---

## Changes

### Tier 2 pending-3 (phases 17–20)

- **Nested HTML HF:** child layout pipeline, shared font registry / `MergeFontFaces`, link GoTo/URI regressions; golden **fixture-36** + `attachHFCompanions` convention
- **Orphans/widows:** CSS parse/inherit + Rule 3 keep-together (fixture-37)
- **Float↔table:** clear-below tables, float-in-td, blockify edges (fixture-38)
- **Multicol lite:** `column-count` / width / gap / span / fill (fixture-39)
- **Static 2D transforms:** paint CTM + opacity ExtGState; abs/fixed CB (fixture-40)
- **`:has()` + `@container`:** relational match; size queries + two-pass style (fixtures 41–42)
- **Flex/grid remaining:** min-size / `%` polish; Partial subgrid/masonry expand
- **Sticky overflow honesty:** overflow boxes as sticky scrollports at offset 0
- **Fonts:** WOFF1 zlib→SFNT; optional OT `halt`/`palt`; WOFF2 unsupported by design

### Phase 21 kickoff (arbitrary websites)

- Product contract + fidelity/CLI/README honesty (“decent print”, non-claims, URL SSRF)
- Vendored `testdata/web/` wiki-like + marketing fixtures + convert acceptance tests
- Opt-in `--simplify-dom` / `web.simplifydom` chrome-strip stylesheet (default **off**)
- Manual live Wikipedia smoke documented; not required in `make test`

### Docs & plans

- Checklist reconciliation for phases 17–20; pending-3 ledgers marked done
- Compatibility matrix / CLI / samples updated for new surfaces

---

## Impact

| Area | Impact |
|------|--------|
| **Behavior / correctness** | Broader CSS/print layout; nested HF child docs; container two-pass; chrome-strip only when opted in |
| **Performance** | Container two-pass for size containers; simplify-dom cheap CSS inject |
| **Memory** | Negligible for HF companions / synthetic sheet |
| **API / CLI** | New `--simplify-dom` / `--no-simplify-dom` (page-scoped); HF unchanged flags |
| **Dependencies** | None new |
| **Binary size / build time** | WOFF1 decoder + fixtures only |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None (defaults unchanged) | `--simplify-dom` is opt-in; invoice/report HTML unaffected |

---

## Test plan

- [x] `make lint` → `go vet ./...`
- [x] `make test` → `go test ./...`
- [x] Golden corpus including fixtures 36–42
- [x] `go test ./internal/convert -run 'Web|Wiki|Marketing|Simplify'`
- [x] `CGO_ENABLED=0` posture preserved (no new CGO)

### Commands

```sh
make lint
make test
go test ./internal/convert -run 'TestGoldenCorpusAllFixtures/fixture-3[6-9]|fixture-4[0-2]|Web|Simplify' -count=1
```

---

## Screenshots / sample output

- Golden fixtures 36–42 under `testdata/golden/`
- Vendored web fixtures under `testdata/web/` (wiki-like + marketing landing)
- Live Ana de Armas Wikipedia PDF remains **manual** smoke (`documentation/samples.md`)

---

## Related issues

- Relates to #19 (flex/grid Stage A–C merge)
- Relates to #17 (Tier 2 pending)
- Relates to #2 (post-MVP / real-site epic — Phase 21 kickoff)

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [x] Filled body under `plans/PR/pr-tier-2-pending-3.md`

---

## Follow-ups (out of scope)

- Phase 21 parent product sign-off / Phase 22 JS-driven pages (only if amended)
- Style/scroll-state container queries; `cq*` units (permanent Partial non-goals)
- WOFF2 / Brotli; bundled Noto CJK; Chrome layout-test parity
- Full joint-intrinsic subgrid / L3 masonry

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented (`--simplify-dom`)
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
