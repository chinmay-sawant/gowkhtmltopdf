# v0.2.4 Font Resolution and Compliance Follow-up

> **Status:** planned — implementation not started
> **Parent:** [`plans/0.2.4/README.md`](../README.md)
> **Relationship to the parent ledger:** additive follow-up track; the completed
> phases 31–39 remain closed until this work is explicitly implemented and
> validated.

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

All implementation checklist rows in this folder intentionally remain
unchecked until the corresponding code, rendered artifacts, and compliance
proof exist. Creating this plan does not constitute implementation evidence.
