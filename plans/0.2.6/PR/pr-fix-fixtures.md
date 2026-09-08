## Summary

Fix fixture rendering regressions in borders, tables, transforms, opacity, and native form controls. The change also refreshes the affected fixture source files and sample PDFs so the generated output matches the corrected layout behavior.

## Motivation / context

- Fixtures 56 and 60 exposed layout and paint differences that were visible in the generated PDFs.
- The fixes keep logical layout coordinates separate from paint geometry where a thick border needs to be inset.
- No public API or CLI behavior changes are intended.

## Changes

### Layout and paint engine

- Expand border shorthands with one to four values and keep unequal-width border corners inside their boxes.
- Paint mixed-width straight borders inward so thick strokes do not extend beyond the border box.
- Preserve signed line geometry for checkbox tick segments while applying border-only paint insets.
- Add native checkbox and radio widget painting, including accent-color support and aligned checked ticks.
- Correct table border-spacing offsets and repeat table headers on normal continuation pages.
- Keep fixture 60's explicitly requested final continuation page without a table header.
- Prevent compounded opacity and restamp transforms after pagination.

### Fixtures and generated output

- Normalize fixture 60 rows and add focused regression tests for property 29 checkbox alignment and property 111 border containment.
- Update the fixture 60 HTML for the intended continuation-page behavior.
- Use a solid border for the fixture 56 divergence cards and regenerate its sample PDF.
- Regenerate the fixture 60, 61, and 62 sample PDFs.
- Update the repository guidance to use the capped Makefile test commands for full-suite validation.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | No material change expected |
| **Memory** | No material change expected |
| **Behavior / correctness** | More accurate border, table, transform, opacity, checkbox, and radio rendering |
| **API / CLI** | None |
| **Dependencies** | None |
| **Binary size / build time** | No material change expected |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

## Test plan

- [x] `make test`
- [x] `make lint`
- [x] `make golden`
- [x] `go test ./internal/layout -run TestFixture60AccentCheckboxTickAndAlignment`
- [x] Visual inspection of the regenerated fixture 60 PDF for the property 29 checkbox and property 111 thick border

### Commands

```sh
make test
make lint
make golden
go test ./internal/layout -run TestFixture60AccentCheckboxTickAndAlignment
```

## Screenshots / sample output

- Regenerated samples: `output/fixture-56-architecture-diagram.pdf`, `output/fixture-60-implemented-props-a.pdf`, `output/fixture-61-implemented-props-b.pdf`, and `output/fixture-62-implemented-props-c.pdf`.
- Fixture 60 property 29 now shows a complete, centered white check on the green checkbox.
- Fixture 60 property 111 keeps the thick right border inside the effect box.

## Related issues

- None identified for this branch.

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues reviewed; none identified
- [x] Filled body under `plans/PR/pr-fix-fixtures.md`

## Follow-ups (out of scope)

- Add pixel-level PDF comparisons if the project adopts a stable raster baseline for these fixtures.

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
