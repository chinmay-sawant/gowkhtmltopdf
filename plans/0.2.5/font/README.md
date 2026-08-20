# v0.2.5 Font Resolution and Compliance

> **Status:** complete — validated 2026-08-19
> **Parent:** [`plans/0.2.5/README.md`](../README.md)
> **Predecessor:** completed v0.2.4 Document API / CLI work
> ([`plans/0.2.4/README.md`](../../0.2.4/README.md), phases 31–39 closed)
> **Evidence sources:** `documentation/fonts.md`,
> `knowledge-base/wiki/syntheses/fonts-and-typography.md`,
> codebase scan refreshed 2026-08-19 (see canonical §Current-state findings)

This track makes CSS font resolution predictable and useful for real-world
documents while preserving the existing pure-Go PDF writer and its compliance
contracts. It is driven by the fixture-55 comparison:

| Renderer | `Georgia` in the compared artifact |
|----------|------------------------------------|
| Chrome on the supplied Windows host | Actual subsetted `Georgia` |
| WeasyPrint on this Linux host | Fontconfig-selected `Gelasio Italic` |
| Gowkhtmltopdf without font flags | Bundled `Liberation Serif` |

The goal is not to make host font discovery the default. The goal is to make
the resolution contract explicit:

1. An explicitly supplied, supported face wins when its internal family name
   matches the CSS family.
2. CSS family stacks continue in author order when a named face is absent.
3. Generic families use deterministic bundled faces.
4. Host aliases such as Fontconfig's `Georgia → Gelasio` are not imported into
   the renderer's default behavior.
5. Every selected face remains parseable, subsettable, embeddable, and valid
   under the existing PDF/A and PDF/UA profiles.

## Intentional divergence from WeasyPrint / Fontconfig (not a bug)

`ae49f3f` closed this track **away from** WeasyPrint host parity, not toward
it. Post-0.2.5 default behavior for `font-family: Georgia, serif` with no
Georgia face supplied is **Liberation Serif via the `serif` generic**. That
is the intended contract.

| Engine | Typical Linux outcome for bare `Georgia` |
|--------|------------------------------------------|
| WeasyPrint | Fontconfig metric substitute (e.g. Gelasio) |
| Chrome (with Georgia installed) | Real Georgia |
| Gowkhtmltopdf (no font flags) | Author stack → `serif` → Liberation Serif |

Do **not** treat Gelasio / Cousine differences vs WeasyPrint as a regression
to fix in this ledger. Metric-compatible host substitution
(`Georgia → Gelasio`, `Courier New → Cousine` when those faces are present)
is a **separate opt-in feature** on top of v0.2.5 — see
[`plans/0.2.6/woff2-metric-aliases/`](../../0.2.6/woff2-metric-aliases/README.md).
Do not reopen phases 01–08 here. The 0.2.6 track must keep the no-flag
Liberation-via-stack default.

## Start here

- [00-canonical-font-resolution-plan.md](00-canonical-font-resolution-plan.md)
- [phase-01-resolution-contract.md](phase-01-resolution-contract.md)
- [phase-02-resolver-seam.md](phase-02-resolver-seam.md)
- [phase-03-discovery-and-font-face.md](phase-03-discovery-and-font-face.md)
- [phase-04-font-format-and-asset-policy.md](phase-04-font-format-and-asset-policy.md)
- [phase-05-pdf-embedding-and-compliance.md](phase-05-pdf-embedding-and-compliance.md)
- [phase-06-corpus-and-engine-comparison.md](phase-06-corpus-and-engine-comparison.md)
- [phase-07-docs-and-operator-contract.md](phase-07-docs-and-operator-contract.md)
- [phase-08-validation-and-closure.md](phase-08-validation-and-closure.md)

## Related work (out of this track)

| Item | Location | Relationship |
|------|----------|--------------|
| HTTPS/local TTF/OTF/WOFF1 `@font-face` | already shipped | Reuse; do not re-plan as unimplemented |
| WOFF2 / Brotli decode | [`plans/0.2.6/woff2-metric-aliases/`](../../0.2.6/woff2-metric-aliases/README.md) | Successor track (supersedes remaining pending-09 WOFF2 rows) |
| Opt-in metric-compatible aliases (Gelasio/Cousine-style) | [`plans/0.2.6/woff2-metric-aliases/`](../../0.2.6/woff2-metric-aliases/README.md) | Same successor track; must stay opt-in and must not become default |
| Normative operator docs | [`documentation/fonts.md`](../../../documentation/fonts.md) | Authoritative shipped contract |
| Compiled KB synthesis | [`knowledge-base/wiki/syntheses/fonts-and-typography.md`](../../../knowledge-base/wiki/syntheses/fonts-and-typography.md) | Keep aligned with shipped resolver |

Phases 01–08 are closed with implementation evidence in each phase file.
Creating or amending this README is not new implementation evidence.
