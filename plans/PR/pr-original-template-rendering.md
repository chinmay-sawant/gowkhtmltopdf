# fix(renderer): improve original template rendering fidelity

## Summary

This PR improves rendering of the repository's original, static print templates without requiring renderer-specific HTML/CSS workarounds. It adds deterministic bundled font support and carries key CSS typography and geometry properties through layout and PDF painting, with fixture coverage for the affected output.

## Motivation / context

- Issue: [#30](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/30)
- The fixture 55 masthead exposed a mismatch between the source HTML's typography and the generated PDF, particularly in the top-right operations brief.
- The renderer should preserve valid template CSS instead of requiring authors to rewrite the source markup.

## Changes

### Renderer and CSS support

- Add deterministic bundled Liberation Sans, Serif, Mono, and DejaVu font assets with regular, bold, and italic face resolution.
- Preserve `letter-spacing` and `text-transform` from the CSS cascade through inline layout, display-list operations, and PDF text painting.
- Carry border-radius into painted geometry and keep positioned/footer and flex layout calculations aligned with the final containing geometry.
- Improve CSS parsing, flex measurement, inline collection, smart-shrink helpers, and pagination-related paint behavior used by the original fixtures.

### Fixtures and regression coverage

- Add the fixture 54 and fixture 55 original HTML/PDF artifacts.
- Add fixture 55 masthead typography regression coverage and regenerate its PDF output.
- Keep the source fixture markup unchanged while validating renderer behavior.

### Repository hygiene

- Resolve the repository's existing `make lint` findings with targeted suppressions where fixture-oriented or state-machine code requires them.
- Ignore generated `.pi-subagents` artifacts.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Adds bundled font-face resolution and PDF character-spacing operations; no intentional layout algorithmic expansion beyond the affected paths. |
| **Memory** | Embeds additional deterministic font assets in the binary. |
| **Behavior / correctness** | Generated PDFs now preserve supported CSS typography and rounded geometry more faithfully for controlled print templates. |
| **API / CLI** | No public API or CLI changes. |
| **Dependencies** | No new runtime dependencies. |
| **Binary size / build time** | Bundled font assets increase binary size; existing font loading remains cached. |

## Breaking changes / migration

| Item | Migration |
|-----------|-----------|
| None | - |

## Test plan

- [x] `make lint`
- [x] `GOCACHE=/tmp/gowk-go-cache go test ./... -count=1`
- [x] `git diff --check`
- [x] Regenerated and visually compared `output/fixture-55-lantern-cooperative-report.pdf` against its source HTML.

### Commands

```sh
make lint
GOCACHE=/tmp/gowk-go-cache go test ./... -count=1
git diff --check
```

## Screenshots / sample output

The regenerated sample is available at:

```text
output/fixture-54-ember-harbor-storybook.pdf
output/fixture-55-lantern-cooperative-report.pdf
```

## Related issues

- Relates to [#30](https://github.com/chinmay-sawant/gowkhtmltopdf/issues/30)

## PR metadata checklist (author)

- [x] Self-assigned (`@me`)
- [x] Labels applied
- [x] Related issues filled with a real ticket ID
- [x] Filled body committed under `plans/PR/pr-original-template-rendering.md`

## Follow-ups (out of scope)

- Complete the remaining issue #30 fixture 50/52 geometry and documentation success criteria in follow-up work.
- Continue broader browser-fidelity work outside the controlled static-template scope.

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New regression coverage is present for the typography mismatch
- [ ] PR has assignee and labels
- [ ] Related issues use the correct `Relates to` keyword
- [ ] No secrets or unintended generated artifacts are committed
