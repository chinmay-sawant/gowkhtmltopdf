## Context

- Fixture-61 rows 108-111 (`testdata/golden/fixture-61-implemented-props-b.html`) applied `list-style`, `list-style-image`, `list-style-position`, and `list-style-type` to plain `div` elements, so the audit PDF could not show any list behavior.
- Rewriting those cells with real `ul`/`ol` demos exposed an engine bug: `ul { list-style: none }` still painted disc bullets.

## Scope (in)

1. Expand the `list-style` shorthand into its type, position, and image longhands at cascade time, carrying the shorthand's own origin and specificity (`internal/layout/style_cascade.go`).
2. Add regression test `TestCascadeListStyleShorthandBeatsUADisc` (`internal/layout/style_cascade_test.go`).
3. Rewrite fixture-61 rows 108-111 with real list demos and regenerate `output/fixture-61-implemented-props-b.pdf`.

## Out of scope

- Full shorthand reset semantics (omitted components keep the prior cascade result; noted in code comment).
- `list-style-image` inheritance (longhand does not inherit parent to child; pre-existing gap).
- Fixture-61 grid rows 74 and 76-81 (item props on container; correct output, weak demos).

## Success criteria

- [x] `ul { list-style: none }` paints zero bullets, via stylesheet and inline style, with `li` inheriting `none`.
- [x] Same-origin source order holds both ways between `list-style` and `list-style-type`.
- [x] Full `internal/layout` suite green; fixture-61 golden passes (8 pages, envelope 5-8).

## Plan

- Branch: `chore/026-review` (already pushed).
- Ship via PR with `Closes` link once opened.

## References

- Fixture: `testdata/golden/fixture-61-implemented-props-b.html:222-225`.
- Engine: `internal/layout/style_cascade.go` (`expandListStyleDeclaration`).
