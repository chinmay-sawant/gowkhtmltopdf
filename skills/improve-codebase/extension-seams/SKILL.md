---
name: improve-codebase-extension
description: >
  Audit how a Go codebase is extended: half-wired features, forked
  adapters, missing proof, and the table or dispatch that should own the
  next CSS property, flag, API method, or layout rule. Emits structured
  findings for a phase-wise checklist. Use when adding a feature, asking
  where to extend, reviewing an incomplete extension, or running
  /improve-codebase-extension. Do not use for architecture-only DAG
  reviews, ponytail deletion, or a single visual template bug.
---

# Improve-codebase — extension seams

The question is **where the next honest feature goes**, and whether
today's code will force a fork. There is no plugin hook. Extension is
editing the owning table, then proving it.

Do not implement. Do not add a framework so the next feature is "easier".

Read first: `../references/finding-schema.md`. If the module path is
`gowkhtmltopdf`, also read `../references/gowkhtmltopdf.md` and
`CONTRIBUTING.md` § Where to change code.

## 1. Name the change classes this tree actually has

From source, list the **dispatch tables** — the switch, registry, or
`register*` that new behavior must join. Typical classes in a renderer
or library:

- used-value / property
- selector or parse rule
- input allowlist (HTML tag, flag, dotted key)
- formatting / pagination / write-out adapter
- public constructor or `With*`
- trust-boundary policy

A class with two tables that have already drifted is the finding.

## 2. Half-wired detector

For each recent or incomplete feature, walk the full path. File
**one** finding for the missing step, not one per file.

| If this exists | This must exist |
|---|---|
| Typed config field | registry + engine read + round-trip test |
| Flag / dotted key | a consumed field (not an ignore-list-only name) |
| Parsed property / enum | apply/dispatch **and** a consumer |
| Compatibility matrix **Implemented** | a code path **and** a test or fixture |
| Display-list opcode | every write-out adapter + shared paint policy |
| Selector / pseudo | parse **and** match; unknown tokens must not degrade to the host |
| Allowlisted input token | UA/default box and/or builder consumer |
| Typed `With*` / constructor | thin public wrapper + invalid-input test |
| Mode-only knob | that mode's settings + sink; do not fork the shared front half |

A typed field with no engine consumer is a defect. Parking a
compatibility name on an ignore list is correct.

On `gowkhtmltopdf`, resolve owner files from
`../references/gowkhtmltopdf.md` § Extension tables.

## 3. Fork detector

Two adapters that share a front half must share:

- prepare / stylesheet gathering
- used-value resolution
- display-list ops
- paint policy (`PaintOrder`, style-of, fake-bold)

They may fork at write-out (PDF objects vs raster encode) and at
mode-only knobs (crop, JPEG quality, TOC).

File `friction` when a helper is copy-pasted per adapter
(`mediaFor`, font-registry construction, `Op` switch). File `defect`
when the copies have already diverged.

## 4. Proof detector

A feature without the smallest honest proof is not done. Do not
demand a heavier class than the change warrants.

| Change | Smallest proof |
|---|---|
| Used value, no pagination | unit: apply + one box/op |
| Print / break / new formatting context | unit that failed before the fix + structural golden + visual |
| Selector / pseudo | match **and** non-degrade unit |
| Flag / dotted key | Set/Get parity + parse + engine assertion |
| Typed API | builder snapshot + `errors.Is` + one run that honors it |
| Writer object (PDF annot, font, image) | writer unit + convert needle, not byte-identity |
| Mode-only sink behavior | that sink's unit (pixels, magic bytes) |
| Trust / ACL | allow **and** deny unit; never a golden output file |

On `gowkhtmltopdf`, goldens are structural; new fixtures need a
`fixturePageBounds` row; overlap is `make samples`. See the
calibration file § Proof that is law here.

## 5. Wrong-place detector

These are always findings if you see them in current source:

- Site-specific class/id heuristics in cascade or layout (operator
  policy belongs in flags).
- Document-global Y shifts of all ops below a baseline.
- Stripping unknown pseudos so `li:target` becomes `li`.
- New dependency outside the allowlist.
- Promoting an `Ignored` key to a typed stub.
- Encoding a look the CSS does not ask for as an engine default.

## 6. If the user is about to extend (design mode)

When the prompt is "add X" rather than "audit":

1. Name the change class and the **one** owner table.
2. List the files in pipeline order (parse → apply → consume → prove →
   claim).
3. State the proof class from §4.
4. State what you will **not** touch (the other sink, the matrix, JS,
   a new package).
5. Stop. Do not implement unless they already asked to implement.

That list is the extension plan. Promote gaps in the *current* tree
to findings; do not invent future tables.

## 7. Emit findings

Finding-schema records. Cap a solo run at **15**. Prefer `EXT-` IDs
after dedupe.

If invoked **by** `/improve-codebase`, return the finding list only.

If invoked **alone**, continue to
`skills/phase-wise-checklist/SKILLS.md` and write:

`plans/reviews/improve-codebase/extension-<YYYY-MM-DD>/phase-wise-checklist.md`

Phase order for this lens: trust/security → half-wired public surface →
shared paint/prepare forks → used-value/layout honesty → matrix/docs.
Do not put a matrix-row update in a phase before the engine consumer.
