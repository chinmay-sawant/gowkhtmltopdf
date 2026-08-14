## Summary

Improves print fidelity for dense HTML (Wikipedia-style articles): long URL wrapping, awards table pagination and Ref cites, reference-list underlines and spacing, and a critical fix that removes document-wide gap packing which had interleaved/overlapped body and reference text. Regenerates `output/wiki-ana-de-armas.pdf` as the live smoke sample.

---

## Motivation / context

- Plans: `plans/phases/pending-phase-items/` · prior print/layout work on `feature/pending-phase-items`
- Branch: `feature/pending-phase-items` → `master`
- Constraint: pure-Go layout; no site-specific wiki CSS hacks — generic CSS/print engine fixes only
- Driver: visual QA of `output/wiki-ana-de-armas.pdf` (Ana de Armas Wikipedia smoke)

---

## Changes

### Inline / overflow / links

- Honor CSS `overflow-wrap` / `word-wrap` / `word-break` with **inheritance** onto text nodes
- Emergency wrap for tokens wider than a line (print edge safety); soft breaks at URL punctuation
- Sticky punctuation no longer glues multi-em bare URLs onto the previous line
- Float tails that fit one full-width line clear below the float (orphan “big time.” fix)
- `overflow-wrap: break-word` does not mid-break words that fit the next full line (captions)
- Adjacent cite markers get a hair space when not forced to stack by markup
- Link underlines: coalesce same-href runs on a line; clamp stroke weight; **skip bare URL text** so reference lists are not a rule forest
- Force-underline for `a[href]` kept for prose links when cascade decoration is `none`

### Tables

- Leading all-`<th>` rows treated as repeating headers when no `<thead>`
- Empty / padding-only rows collapsed; border-collapse grid emitted **per row** and bound into row op ranges (stops phantom empty bands across page breaks)
- Multi-cite nowrap clusters expand min-content so `[127][128]` can stay horizontal when markup allows
- **Rowspan Ref cells** with `<br>` between cites (wiki awards): spread line boxes across full cell height so markers align with both rows instead of overlapping at the top
- Continuation-page table fragments: seal incomplete tops/bottoms under repeated thead (`capTablePageBreaks`)

### Pagination / avoid packing

- `preferSplitOverBlank`: prefer splitting short `page-break-inside: avoid` boxes over large empty bands (dense reference `li` lists)
- **`packAvoidGaps` no-op**: removed document-global `shiftOpsBelowY` compaction that crushed line spacing and interleaved body paragraphs
- Sibling packing path left disabled/harmless; blank-band control stays on prefer-split

### Tests & sample

- New/extended: overflow wrap, thead-from-th, ref gaps, table empty rows, multi-cite / rowspan cites, continuation borders, underline coalesce, overlap packing
- `output/wiki-ana-de-armas.pdf` regenerated (live Wikipedia smoke)

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Negligible; pagination still O(ops × fixpoint iters); global gap pack removed (slightly less post-work) |
| **Memory** | Negligible |
| **Behavior / correctness** | Major print correctness: no text overlap from gap pack; better URL wrap, tables, underlines |
| **API / CLI** | None |
| **Dependencies** | None |
| **Binary size / build time** | Tests + sample PDF only |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | Defaults unchanged. Operators still control page numbers / HF via CLI. |

---

## Test plan

- [x] `go test ./internal/layout/ -count=1`
- [x] `go test ./internal/convert/ -run 'TestGoldenCorpusAllFixtures/fixture' -count=1` (fixture corpus)
- [x] Visual QA of regenerated `output/wiki-ana-de-armas.pdf` (pages 1–12): body flow, awards Ref cites, refs underlines, no interleaved body text
- [ ] `make test` full tree (recommended on CI)
- [ ] `make lint` / `go vet ./...`

### Commands

```sh
go test ./internal/layout/ -count=1
go test ./internal/convert/ -run 'TestGoldenCorpusAllFixtures/fixture' -count=1
go build -o bin/gowkhtmltopdf ./cmd/gowkhtmltopdf
./bin/gowkhtmltopdf --use-system-fonts --zoom 0.666667 \
  'https://en.wikipedia.org/wiki/Ana_de_Armas' \
  output/wiki-ana-de-armas.pdf
```

---

## Screenshots / sample output

Live smoke (network): Wikipedia *Ana de Armas* → `output/wiki-ana-de-armas.pdf`

| Metric (post-fix) | Notes |
|--------------------|--------|
| Page count | ~10–12 depending on packing; Chrome ~10 |
| Body line collisions | **0** tight baselines after disabling gap pack |
| Awards `[127]`/`[128]` | Spread across rowspan (~17pt dy) |
| Ref max same-page gap | Prefer-split only (~25–38pt residual air, not 100–150pt cascades) |
| Bare URL underlines | Suppressed; title/prose links still underlined |

---

## Related issues

- Relates to pending-phase print/layout fidelity work on `feature/pending-phase-items`
- Relates to epic rendering quality / Wikipedia-class print (see `plans/PR/issues/` when filed)

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`bug`, `enhancement`)
- [x] Related issues filled
- [x] Filled body under `plans/PR/pr-print-layout-fidelity.md`

---

## Follow-ups (out of scope)

- Safer *scoped* mid-item hole healing (ops limited to avoid-box range only) if dense refs need tighter packing again
- IPA multi-script face consistency without `--use-system-fonts` fragmentation
- Caption mid-word breaks in extremely narrow float boxes
- Optional page numbers via HF flags in the wiki Makefile smoke recipe

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented (none expected)
- [ ] New layout rules have unit coverage
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets; sample PDF is intentional smoke output
