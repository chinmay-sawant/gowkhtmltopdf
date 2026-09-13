# Phase 5 design: display-list append and capacity (PERF2-22..25)

Read-only design note for `plans/0.2.6/perf-improve/phase-wise-checklist.md`
phase 5. Evidence boundary: current source plus the 2026-09-11 captures under
`plans/0.2.6/perf-improve/profiles/`. No git command, benchmark, or make target
ran while writing this.

## 0. What the profiles measured

- `(*engine).add` (`internal/layout/layout.go:842-863`, append at `:861`):
  flat 490 ms, 3.79 percent of 500-page samples. Named callers:
  `emitGridVerticals` (through `rowGridStroker.vline`) 190 ms,
  `emitInlineTextRun` 170 ms, `rowGridStroker.hline` 110 ms
  (`profiles/warm-pdf-cpu.md` section 6 and section 9 candidate 3).
- `Op` is 472 bytes by DWARF; the line that generates them,
  `layout.go:1003` `ops = make([]Op, 0, estimateOpCapacity(root))`, is charged
  91.17 MB flat in the CPU capture's allocation cross-check
  (`profiles/warm-pdf-cpu.md` section 7) and 95,600,640 B in the probe
  (`profiles/warm-pdf-alloc.md` section 5).
- The same probe measured DOM nodes 135,022, estimate 202,533, final ops
  174,000, boxes 54,503 at 500 pages. Requested prealloc 95,595,576 B, waste
  13,467,576 B requested and 13,472,640 B charged; the charged line is 86
  percent useful (`warm-pdf-alloc.md` section 5).
- Cap on the capacity lever: 13.47 MB. The 500-page B/op total is 321.10 MB
  (`phase-wise-checklist.md` overview), so the hard ceiling is about 4.2
  percent of bytes.

## 1. Op fields by size class

Tally from `internal/layout/layout.go:374-468`. Offsets were reconstructed
from the declared field order; the sum matches the measured 472 bytes.

| Class | Fields | Bytes | Share |
|---|---|---:|---:|
| float64 scalar | X, Y, W, H, R, G, B, Alpha, Width, Size, LetterSpacing, RotateDeg, InkDescent, PaintOpacity | 112 | 23.7% |
| float64 radius | Radius, RadiusTopLeft, RadiusTopRight, RadiusBottomRight, RadiusBottomLeft, RadiusY, and the four corner Y radii | 80 | 16.9% |
| Matrix2D | Xform (six float64, `transform.go:32-34`) | 48 | 10.2% |
| string | Text, TextTransform, FontFeatures, URI, Alt, BlendMode | 96 | 20.3% |
| slice | Image | 24 | 5.1% |
| int-sized | ID (uint64), Kind (OpKind, int), ImgW, ImgH, StickyID, ZIndex | 48 | 10.2% |
| pointer | Font, StructElem | 16 | 3.4% |
| flags | 9 bool (Bold, NoFakeBold, IsJPEG, IsBackground, Fixed, Pinned, ZIndexSet, Positioned, XformSet), 2 uint8 (StrokeMask, LineInset) | 11 | 2.3% |
| alignment padding | gaps at offsets 90, 154, 233, 259, 282, 353 | 37 | 7.8% |
| total | | 472 | 100% |

Pointer-bearing fields the GC scans: Text, Font, TextTransform, FontFeatures,
URI, Image, Alt, BlendMode, StructElem (nine pointer words, matching the
profile's "about ten"). The float64 fields (192 bytes) and the 37 padding
bytes are dead weight for the GC but are copied on every append and every
`for _, op := range ops` range.

## 2. Pointer fields: candidates for sharing or index storage

| Field | Bytes | Candidate change | Semantics risk |
|---|---:|---|---|
| Font `*pdf.Font` | 8 | None. Already a shared pointer | none |
| Text | 16 | Keep. Per-op unique; interned text would not shrink the header | none |
| URI | 16 | Keep for now. Rewritten after paint (`internal/convert/convert.go:744-750`) | high churn |
| TextTransform, FontFeatures, BlendMode | 48 | One side table per `Result` with small integer indices; these are mostly empty and low-cardinality | medium, readers are spread over paint |
| Alt | 16 | Fold into the image side record | medium, only meaningful with Image |
| Image `[]byte` | 24 | `int32` index into a `Result.Images` table. `cloneOps` currently copies bytes per op (`layout.go:230`) | medium; image paint and fragment cloning read it |
| StructElem `*pdf.StructElem` | 8 | A parallel `[]*pdf.StructElem` on `Result`, allocated only when `doc.IsUA()`. `cloneOps` nils it (`layout.go:233`); `clearStructureElements` and `buildStructureTree` own it | low, covered by tagging and golden tests |
| radius corner block | 80 | Keep `Radius` and `RadiusY` as the fast uniform path; move the eight corner floats behind one `*opRadii` allocated on first use. Saves 64 B, adds 8 B pointer, net 56 B (11.9%) | medium, readers at `paint_pagination_split.go:181-224`, `inline_paint.go:319-338`, `layout_chrome.go:418-652`, `box_shadow.go:549-552`, and the PDF painter |
| Xform + XformSet | 49 | `*Matrix2D` side record. Saves 41 B net (8.7%). Readers at `transform.go:996-1114` and `restampBoxTransforms` | medium |
| 9 bool + 2 uint8 | 11 | Reorder fields only, no field removed. All flags contiguous removes most of the 37 padding bytes. Saves up to 32 B (6.8%) | none beyond `unsafe.Sizeof` assertions and any unkeyed `Op` literal the compiler flags |

Safe first combination: reorder-only packing. Estimated 472 to about 440,
roughly 6.8 percent fewer bytes on every copy, with no reader changes beyond
any unkeyed `Op` literal the compiler flags. The `StructElem` parallel slice
is a separate, narrow change guarded by the tagging tests. The radius and
Xform moves are the larger wins but touch paint hot paths and should be
separate, reversible commits.

## 3. Capacity tightening math and growth risk

`estimateOpCapacity` (`layout.go:1236-1256`) is
`max(64, nodes * 3 / 2)` capped at 2^20. At 500 pages the measured ratio is
`len(Ops)/nodes = 174,000/135,022 = 1.2883`. Options:

| Multiplier | Capacity | Waste slots | Waste bytes | Saved vs today | Headroom over 1.2883 |
|---:|---:|---:|---:|---:|---:|
| 3/2 (today) | 202,533 | 28,533 | 13,467,576 | 0 | 16.4% |
| 7/5 | 189,030 | 15,030 | 7,094,160 | 6,373,416 | 8.7% |
| 27/20 | 182,279 | 8,279 | 3,907,688 | 9,559,888 | 4.8% |
| 4/3 | 180,029 | 6,029 | 2,845,688 | 10,621,888 | 3.5% |
| 13/10 | 175,528 | 1,528 | 721,216 | 12,746,360 | 0.9% |

Growth risk. Go grows a large slice by roughly 1.25x. One under-estimate at
500 pages requests a new array of about
`(202,533 + (202,533+768)/4) * 472 = 119.6 MB` while the old 95.6 MB array
becomes garbage (`warm-pdf-alloc.md` section 5 says "about 120 MB"). One
under-estimate therefore adds about 106 MB net B/op, roughly nine times the
13.5 MB the lever can save, plus a 174,000-element copy. There is a second
reason the multiplier is not globally safe: ops per node are shape-dependent.
A run of long text in one paragraph emits one line op per line over very few
DOM nodes, so the current 1.5 factor is already an under-estimate for
text-heavy documents and relies on append growth. Tightening the global
multiplier makes that growth more frequent, not just possible.

Hard rule from PERF2-23: total 500-page B/op must fall; a growth reallocation
is acceptable only when the total is lower than today. A single 120 MB growth
can never satisfy that against a 13.5 MB ceiling.

Conclusion: the capacity lever is not worth a phase by itself. It is worth
exactly one measured change if and only if a corpus-wide probe shows the
maximum ops per node over every golden fixture and the benchmark is at most
1.30, in which case 4/3 is safe and saves about 10.6 MB at 500 pages. If the
corpus max lands between 1.30 and 1.35, the best safe multiplier is 27/20 and
the saving is about 9.6 MB. If it lands above 1.35, stop. Even at best the win
is 3.3 percent of the 321.10 MB total.

## 4. Two falsifiable changes

Change A, capacity to the corpus-proven multiplier.

- Step: a test-only probe lays out every fixture and logs `len(Ops)`, node
  count, `cap(Ops)` and whether append grew. If max `len/nodes` over the corpus
  is at most 1.30, set the multiplier to 4/3; otherwise stop and record why.
- Exact measurement: 500-page `-test.benchmem` row; expect B/op to fall from
  321.10 MB by at least 8 MB (about 10.6 MB if the multiplier is 4/3), and
  `make golden` 65/65 with no page or text change.
- Rollback: any fixture whose final `len(Ops)` exceeds the preallocated `cap`
  after layout, or a 500-page B/op drop under 8 MB, or a warm-time regression
  over 2 percent.

Change B, Op field packing.

- Step 1 (low risk): reorder fields so all flags and uint8 land in one
  contiguous block; assert `unsafe.Sizeof(Op{}) <= 440`; no reader changes.
- Step 2 (bounded risk): move the eight corner radii behind one `*opRadii`,
  keeping `Radius`/`RadiusY` direct; target `unsafe.Sizeof(Op{}) <= 384`.
- Exact measurement: `unsafe.Sizeof` assertion, 2/10/50/500-page B/op rows,
  and the `add` flat time from the 500-page `-list` profile. The prealloc is
  charged on capacity, so the byte saving is size reduction times 202,533
  slots: about 6.5 MB for step 1 (32 B) and about 11.3 MB more for step 2
  (56 B net). `add` flat should fall because the append copies shrink too;
  target below 0.40 s.
- Rollback: any golden diff, a size over target, B/op up at any size, or
  `add` flat not below 0.40 s. Do not take step 2 if step 1 alone does not
  move `add`.

Note: shrinking `Op` also reduces the prealloc charge, the append copy, and
every per-op range copy. `rowInkBand` (`paint_flow_tables.go:166`, 120 ms) and
`collectBodyNavigation` (`internal/convert/links.go:97`, 110 ms) are already
measured per-op copies that benefit without any code change when the record
shrinks (`warm-pdf-cpu.md` section 9 candidate 5).

## 5. Ranked recommendation and smallest first step

1. Run the corpus capacity probe first. If max ops per node is above 1.30,
   reject the capacity lever in one measurement and keep the phase focused on
   Change B. If it is at most 1.30, land Change A as a single small commit.
2. Land the reorder-only packing (Change B step 1) if the probe is ambiguous;
   it has no reader risk and shrinks every copy. Take the radius side record
   only with a separate rollback point and its own `add` measurement.
3. Do not rewrite emission into chunks or a pointer-based `add` in this phase.
   `add` costs 3.79 percent and the capacity cap is 13.5 MB; a chunked display
   list adds a new representation for every reader to chase. Revisit only if
   the packing steps do not move `add` below 0.40 s and phases 3 and 6 have
   already landed their wins.

Smallest first step: add the test-only corpus capacity probe, record
`len(Ops)/nodes`, `cap >= len` per fixture, and the 500-page numbers, then
decide A yes or no from that single table.
