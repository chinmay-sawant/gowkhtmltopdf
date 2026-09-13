# Phase 6 design: deterministic concurrency for independent page blocks

Read-only design note for `plans/0.2.6/perf-time/phase-wise-checklist.md`
(PERFT-24..27). Evidence boundary: the committed source tree read at write
time, the four profiles under `plans/0.2.6/perf-time/profiles/`, and the
committed phase-7 baseline (1,228.72 ms / 234.92 MB at 500 pages). Phase 2
(style memo, `internal/layout/style_memo.go`) and phase 5 (retained parallel
flate, `internal/pdf/flate_parallel.go`) were landing in the working tree
while this note was written; phases 3 and 4 had not landed. All concurrency
milliseconds below are arithmetic on measured phase shares, not measurements.
The only measured concurrent wall win in the repo is phase 5 stream
compression: median -150.5 ms per 500-page conversion with byte-identical
output (`profiles/pagination-paint-deep.md` section 5.2).

Hard rules used: no git command ran; no benchmark or make target ran; no code,
test, or checklist file was touched; sibling agents own `internal/layout` and
`internal/pdf`.

## 0. Plain-words summary

The 500-page benchmark is 500 `<section>` blocks, each one page tall, each
starting on a new page (`testdata/golden/benchmarks/templates/report.html.tmpl`).
Sections do not touch each other. Today all 500 are built in one pass on one
goroutine and then paginated and painted in one pass. The design splits the
build into 500 independent builds, runs them on a bounded worker pool, and
stitches the 500 display lists back into one list in document order. The
stitched list is handed to the existing pagination and paint code unchanged.
That keeps every hard invariant (page assignment, locations, links, outline,
structure tags, headers/footers) inside code that already passes `make golden`.

Stitching is exact only when the isolated build of a section produces the same
geometry as the same section inside the whole-document flow. The detector in
section 2 decides when that is true. Anything else runs today's single-flow
layout, untouched.

A second, more aggressive stage runs pagination itself per section and paints
pages as they finish. That removes the 270 ms pagination pass from the serial
part of the timeline but changes where page offsets come from, so it is gated
on a differential probe before it ships.

What is rejected: appending ops into one shared slice from many goroutines
(order becomes nondeterministic), a style store per section (style bytes grow
by roughly 110 MB at 500 pages, section 3.2), and one temporary PDF document
per section (page adoption changes object numbering and cannot stay
byte-identical).

## 1. Measured ingredients

| Ingredient | Measured value | Source |
|---|---:|---|
| Warm 500p baseline | 1,228.72 ms / 234.92 MB | plan overview, phase-7 matrix |
| Warm 500p B/op ceiling | 234.92 MB | phase 1 rules |
| Style storage pin | 221,208 B per conversion | `style-deep.md` section 4.1 |
| Style share at 500p | 29 to 36 percent, budget 440 ms | plan overview |
| Build share at 500p | 22 to 25 percent, budget 275 ms | plan overview |
| Pagination plus seal | 20 to 22 percent, budget 270 ms | plan overview |
| Paint share | 6 to 7 percent, budget 77 ms | plan overview |
| Finalize plus compress | 8 to 10 percent, 116 to 156 ms | plan overview |
| GC mark workers | 3 to 5 percent, 51 to 78 ms | plan overview |
| Ops at 500p | 174,000 ops x 440 B = 76.56 MB, prealloc 202,533 slots = 89.1 MB | `display-deep.md` section 3, `pagination-paint-deep.md` section 3.3 |
| Boxes at 500p | 54,503 boxes x 304 B = 16.57 MB | `display-deep.md` section 4 |
| Peak live heap at 500p | 210 to 212 MB | `time-cpu.md` section 4, `pagination-paint-deep.md` section 3.1 |
| Parallel flate | median -150.5 ms, output 1,419,234 bytes | `pagination-paint-deep.md` section 5.2 |
| Phase 6 target | <= 615 ms if the design ships | PERFT-27 |

The profiles' honest combined reading is 1.4x to 1.7x from sequential levers
(`time-cpu.md` section 6). Concurrency is the only ingredient that attacks
style, build, pagination, and paint on one timeline, which is why it is the
only path with a 2x shot. It buys wall time only; total CPU work is unchanged
except where per-section pagination removes superlinear scans.

## 2. Independent-block detection

### 2.1 What a candidate is

A candidate list is an ordered, gap-free run of element children of `body`
(today: direct children, matching the fixture shape). Every candidate must be:

- a block-level in-flow box (no `float`, no `position: absolute|fixed`, not
  `display: none`, not a flex/grid item because its parent is flex/grid);
- the only owner of its own subtree: no descendant is positioned against a
  containing block outside the candidate, no ancestor transform or opacity
  wraps it (see 2.3);
- separated from its predecessor by `page-break-before: always` (the first
  candidate may open at document start), and by a zero vertical margin gap;
- free of features the isolated build cannot reproduce (2.2) and, for the
  aggressive stage 2, free of features the isolated pagination cannot
  reproduce (2.4).

Detection runs after the cascade, because `page-break-before` comes from CSS.
The resolver already runs first in `layoutContext`
(`internal/layout/layout.go:1020`), so `ParallelLayout` can certify with real
styles and fall back to the serial build without a second resolution pass.

### 2.2 Build-isolation conditions (stage 1)

These are the conditions that make an isolated build of the candidate equal to
the build of the same candidate inside the document flow, up to a translation.

| # | Condition | Why it is required | Anchor |
|---|---|---|---|
| B1 | Every candidate's used `margin-top` and `margin-bottom` are 0, and the previous candidate's bottom margin is 0 | the merge reconstructs sibling stacking from box heights; collapsed margins between candidates would change the continuous offsets | `buildBlock` starts at `posY` (`layout.go:1508`) |
| B2 | `body` and `html` have zero margin, padding, and border, and `body` is a normal block container (no flex, grid, multicol, or writing-mode change) | the merge has no shell build to contribute the body content origin or shell chrome ops; a non-zero shell offsets every section and emits ops the merged list would miss, and a non-block body changes how children flow | `finalizeResult` builds the root box once (`layout.go:1057`) |
| B3 | No `float: left|right` in the body flow before or inside a candidate that can extend past the candidate top | the float state (`engine.bfcFloats`) is per flow context; an isolated section starts with no floats | `pushBFCFloats` (`layout.go:1525`), `float.go` |
| B4 | No absolutely positioned descendant resolves its containing block outside the candidate | the engine defers abs children into `absCBHeights` of the nearest positioned ancestor; an external CB is not built in isolation | `buildOutOfFlow` (`layout.go:2128`) |
| B5 | No ancestor above the candidate establishes a stacking context: no `z-index` on a positioned box, no `opacity < 1`, `transform`, `mix-blend-mode`, `isolation: isolate`, or filter | the engine carries `zIndex`, `positioned`, `blendMode`, `transformCBDepth` down the recursion (`layout.go:912-954`); an isolated engine starts clean | `pushZ` / `enterStackingContext` |
| B6 | No `@container` rule matches inside the candidate against a container outside it | container gates are measured per document (`resolveStylesForLayoutContext`, `layout.go:1108`); the global resolution path keeps this exact only if the candidate does not depend on an external container | `internal/css/container.go` |
| B7 | `content:` does not use `counter()` / `counters()` in or before a candidate | counter state is recomputed by a walk from the document root (`counter.go:318,437`). Building from the original tree stays correct but the walk is O(preceding nodes) per generated pseudo, so counters are disqualified for cost and for stage 2 isolation | `contentEnvAt` |
| B8 | No op in the candidate carries `Fixed`, `XformSet`, or `Pinned` | fixed ops are stamped on every page by the painter (`paintPages`, `paint.go:434`); baked transforms are stamped against local origins and would need rebaking after the merge; pinned ops only exist after pagination | `Op` flags (`layout.go:473-492`) |

The benchmark satisfies B1 through B8: sections have `padding: 2mm 0` and no
margins, `body { margin: 0 }`, no floats, no positioned content, no transforms,
no counters, no fixed ops. The ops probe confirms: `Fixed`, `Pinned`,
`XformSet`, `Positioned`, `URI`, `Image` counts are all zero
(`display-deep.md` section 3).

### 2.3 Additional selector conditions for per-candidate style resolution

Stage 1 resolves styles once on the real tree and passes the read-only map to
every worker. Selector matching therefore sees the real ancestors and
siblings, and B5 remains the only ancestor condition. If a later variant
resolves styles per candidate (section 3.2), it must additionally disqualify:

- sibling combinators (`+`, `~`) and structural pseudo-classes
  (`:first-child`, `:last-child`, `:nth-child`, `:nth-of-type`, `:only-child`)
  whose match changes when the candidate becomes the only body child;
- `:has()` selectors with a sibling or outside-subtree argument;
- inline `style` attributes and `--print-link-underline` are already key
  inputs of the memo (`style_memo.go`), so they stay safe;
- `:target` is document state, not tree state, and stays safe.

The clone-based certification already used by the benchmark
(`internal/convert/islands/plan.go:89`) sidesteps structural matching by
certifying one fixture shape. The general detector does not need clones once
styles are resolved globally.

### 2.4 Pagination-isolation conditions (stage 2 only)

If pagination runs per candidate, the candidate must also be a pagination
island:

| # | Condition | Why |
|---|---|---|
| P1 | No `position: sticky` and no `position: fixed` inside a candidate | sticky clamps to the page content box and fixed replays on every page; both are global page state (`sticky.go`, `paintPages`) |
| P2 | No `keep-heading-with-next` / `page-break-after: avoid` on the last box of a candidate | the serial fixpoint would keep the candidate's tail with the next candidate's head; an isolated candidate has no next box (`paint_pagination_fixpoint.go:117`) |
| P3 | No `page-break-inside: avoid` subtree taller than one page | the serial fixpoint may move the whole subtree; isolation keeps it where it fits (`paint_flow_breaks.go:36`) |
| P4 | No table whose rows start in one candidate and continue in the next | impossible with a forced break, but the detector checks table op ownership spans within the candidate only (`opOwnedBy`, `internal/layout/op_ownership.go:35`) |
| P5 | Page-name transitions (`page:`) only on candidate starts | `applyNamedPageBreaks` compares sibling names on the merged box tree (`page_named.go:11`); a transition inside a candidate stays local, a transition at the boundary is the candidate's own break |
| P6 | No orphans/widows setting whose policy would move a box across the candidate boundary | same reason as P2 (`paint_flow_orphans.go`) |

The benchmark satisfies all six: no sticky or fixed ops, no break-after-avoid
at section ends, sections fit one page, one table per section, one page name.

### 2.5 What disqualifies a document

The document falls back to today's single-flow layout when any of these hold:

- fewer than two candidates, or any body child outside the candidate run
  (text nodes with non-whitespace, comments are allowed only when blank,
  scripts, unsupported elements);
- any B-condition or P-condition fails for any candidate (P-conditions are
  only consulted when stage 2 is enabled);
- the run does not cover the body: a candidate list that starts above or ends
  below leftover content is rejected rather than merged around a hole;
- page-height-dependent layout ran: `Result.pageSnapHeight != 0`
  (`multicol.go` snapping);
- the document is PDF/UA-1 or PDF/UA-2: `buildStructureTree` mutates document
  structure state (`tagging.go:29`) and the structure walk starts at the real
  root box; stage 1 falls back for UA so the structure tree is built exactly as
  today. PDF/A is not disqualified because structure is not built for it.

The fallback is the current code path, not a reimplementation: after
resolution, `ParallelLayout` detects and, when not certified, calls the same
`finalizeResult(newEngine(ctx, opts, faces, font, styles, containers, ops),
root, opts)` used by `layoutContext` (`layout.go:1033,1057`) with the same
`make([]Op, 0, estimateOpCapacity(root))` prealloc. `LayoutContext` callers
(`internal/convert/convert.go:646`, `internal/imageout/imageout.go:263`,
`internal/convert/hf.go:399`) keep their current behavior; the parallel path is
opt-in behind the new entry point.

## 3. Parallel section layout

### 3.1 Pipeline

Stage 1, inside `internal/layout`:

1. Resolve styles once for the whole tree (existing
   `resolveStylesForLayoutContext`).
2. Certify the candidate run (section 2) with the resolved styles.
3. Fan out candidates over `W = min(8, max(1, GOMAXPROCS))` workers.
4. Each worker builds one candidate with a fresh `engine` that shares the
   read-only `styles` and `containers` maps and owns its ops, boxes, face
   caches, and workspace buffer.
5. Merge in document order into one `Result` (section 4) and return it.
6. `convert` calls the existing `layout.PaintContext` on the merged `Result`.
   Pagination, locations, page names, structure, links, outline, and
   headers/footers are unchanged.

The worker count is a knob, not a contract; output must be identical for
W = 1, 2, and 8 (probe in section 6).

### 3.2 Style resolution and the shared styleStore

Today `resolveStylesCtx` creates a local store per resolution
(`internal/layout/style.go:609`) and `styleStore.append` interns into 64-record
chunks (`style.go:705,723`). The pin is one chunk: 64 x 3,432 B = 219,648 B
plus overhead = 221,208 B.

The arithmetic that rules out per-section stores: every section resolves the
same 15 distinct styles, and the first insert allocates a full 64-record chunk
regardless of how many records are used. 500 sections x 219,648 B is about
109.8 MB of extra B/op, which is over the 234.92 MB ceiling on its own. Even a
hypothetical exact-size chunk would hold 15 x 3,432 B x 500 = 25.7 MB. The
plan's rule is explicit: style storage stays at 221,208 B unless a lever
provably lowers it (`phase-wise-checklist.md` non-negotiable rules, and
`style-deep.md` section 7 rejects parallel subtree resolution for this reason).

Three shapes were considered:

| Shape | Style bytes at 500p | Determinism | Verdict |
|---|---:|---|---|
| Resolve once on the real tree, workers only read the map | 221,208 B | exact, selector matching unchanged | recommended for stage 1 |
| One shared store behind a mutex; workers resolve their own candidates and intern into it | about 221,208 B (15 records total) | intern returns the content-equal record; pointer identity may vary with insertion order but values do not | stretch, for the parallel-style variant |
| Store per section, merged afterwards | 25.7 MB minimum, 109.8 MB with chunks | needs a pointer remap over every style map and box | reject |

The shared-store variant is small: `styleStore` gains a `sync.Mutex`; `append`
already takes `ResolvedStyle` by value and stores pointers into chunks that are
never moved or reallocated (`styleStoreChunkSize` comment, `style.go:702`), so
a lock around the intern-map read, the bucket append, and the chunk append is
enough. The `candidate` field stays in the per-worker `styleContext`. The
`style_memo` (phase 2) is per `styleContext`, so each worker's memo misses on
its first occurrence of a shape and then hits; that is correct, not shared
state. The 221,208 B pin is measured on the whole-document store, so the
probe counts chunk allocations and requires exactly one.

Font side, no new lock is needed. `pdf.Font` lazy fields are `sync.Once`
guarded (`internal/pdf/fonts.go:74-89`), `Registry` lookups take an RWMutex
(`internal/pdf/registry.go:17-21,118`), and the process font-file cache takes a
mutex (`internal/pdf/font_file_cache.go:35-42`). Layout's per-run face maps
(`engine.faceByStyle`, `faceByRune`, `layout.go:600-604`) and image caches are
per engine, so workers do not share them; a shared read-only face cache can be
pre-warmed from the style set's distinct families before fan-out if the
per-worker maps show up in a profile. The image rasterizer's `glyphAtlas` is
per run and single-threaded (`internal/imageout/imageout.go:453,561`); if
image mode adopts section concurrency later, each worker keeps its own atlas.

### 3.3 html.Node read-only safety

After `prepare`, the tree is read-only for layout. The layout package writes no
`html.Node` field outside tests (the only writes found are tests and the clone
builder in `internal/convert/islands/plan.go:106,124`). Anonymous flex and
multicol wrappers are `&html.Node{}` values held by the engine, never attached
to the parsed tree (`flex.go:135`, `multicol.go:158`). Counters and quotes
walk `Parent` links read-only (`counter.go:310`). The shared trees therefore
need no clone for stage 1, which is what keeps `Locations[i].Node` pointing at
the real nodes so heading lookup by node pointer (`outline.Lookup`,
`internal/outline/outline.go:197`) and link collection keep working unchanged.

Two process-global read caches have locks and are safe to share: the CSS
sibling cache (`internal/css/match.go:32-71`, mutex) and the parsed font file
cache. Neither needs a new rule.

### 3.4 Op storage and the B/op trap

The naive merge is a copy: workers allocate per-section `[]Op` slices, the
merger appends them into one result slice. That pays for the section slices
plus the result slice in the same iteration. Serial allocates one 89.1 MB
prealloc (`layout.go:1028`). Per-section estimates sum to roughly the same
89.1 MB (the estimate is 1.5 x node count per section), so the naive merge
would roughly double op bytes and push 500p B/op from 234.92 MB to about
324 MB.

Two mechanisms keep it flat, in order of preference:

1. **Per-call free list plus one preallocated result array.** Each worker
   borrows `[]Op` from a free list that lives for the one `ParallelLayout`
   call, sized to its own `estimateOpCapacity(section)`; the merger copies each
   finished section into
   `result.Ops = make([]Op, 0, estimateOpCapacity(root))` and returns the
   buffer to the list. Peak extra bytes are `W` buffers (about 8 x 0.18 MB =
   1.4 MB for the benchmark shape), so expected B/op delta is single-digit MB,
   not double. The list is per call, not process-wide, so nothing is retained
   across conversions and a GC between sections cannot inflate the accounting
   the way a process `sync.Pool` can. The result array is the same allocation
   the serial path makes, and it never grows because the root estimate covers
   174,000 ops with room to spare.
2. **Windowed single allocation.** Allocate the root-sized array once and give
   each worker a three-index window
   (`ops[base:base:base+estimateOpCapacity(section)]`) so an overflowing
   append allocates outside the shared array instead of spilling into the
   neighbor. Sections that fit cost zero copy; the merger compacts windows
   forward in place (destinations never pass an uncopied source when every
   section fits its window). A section that overflows its window breaks the
   compact-in-place assumption; the ceiling-safe fallback is to rebuild that
   document on the serial path, and the measured rows (benchmark 500p, library
   500, image 250/500) are the rows that must be shown overflow-free. Choose
   this only if mechanism 1 misses the B/op ceiling.

Both mechanisms are measured by the same probe: 500p `-benchmem` B/op at or
below 234.92 MB and at or below the same free tree's serial B/op, plus a
counter proving one op array of root capacity is allocated per conversion.

### 3.5 Boundary geometry

The merge reconstructs the continuous canvas, not the paginated canvas. In the
serial layout the body stacks candidates back to back, so candidate i starts
at `dy_i`, the sum of the outer heights of candidates 0..i-1 plus their
collapsed margins. B1 and B2 make that gap zero: `dy_i` is the sum of previous
candidate box heights, which is exactly the serial sibling stacking for the
certified shape. Forced page breaks are not applied at build time in either
path (they are a paint-time policy, `paint_flow_breaks.go:499`), so the merged
canvas is the serial canvas and the existing `beforeAlways` pass applies the
same 499 breaks to the same geometry. This is the single most important
choice in the design: it removes page-offset arithmetic from stage 1
entirely.

Isolated widths match because the body content width equals `opts.Width` under
B2 (no shell margins or padding). Percentage heights resolve against the same
`opts.Height`. `pageSnapHeight` is checked, not merged: any non-zero value
disqualifies (section 2.5).

## 4. Deterministic merge

### 4.1 Merge order and offsets

The merge runs once, on the main goroutine, consuming section results in index
order. It never looks at completion order.

| Surface | Merge rule | Invariant |
|---|---|---|
| `Ops` | append in section order; `op.Y += dy_i`; renumber `op.ID = globalIndex+1` | merged op order and Y equal the serial display list |
| `box.opStart/opEnd` | `+= opBase_i` (op count before the section) | per-box op ranges stay exact |
| `box.y`, `box.x` | `box.y += dy_i`, `box.x` unchanged | merged box geometry equals serial |
| `StickyID` | `+= stickyBase_i` (0 for the certified shape because sticky is disqualified) | fragment lookup by id stays unique |
| `root` | synthetic box with the section roots as children, in order, `y = 0`, `height = last bottom` | later tree walks see document order |
| `boxes` | concatenation of each part's preorder list, same offsets | `flowBoxList` and `populateLocations` walk in document order |
| `Width/Height` | `opts.Width`, last section bottom | matches serial bounds |
| `Pages`, `Locations` | not merged; they are filled by the serial `PaintContext` after the merge | one owner for the page model |
| `pageNames` | not merged; `namedPageNames` runs on the merged result (`paint.go:228`) | page names derive from merged boxes |
| structure elements | not merged; `buildStructureTree` runs once after paint for UA documents, which are disqualified from the parallel path | one document structure tree |
| links and navigation | not merged; `collectBodyNavigation` runs once on the merged result (`convert.go:667`) | link intents resolve document-wide |
| outline headings | not merged; `collectObjectHeadings` runs once on the merged result (`convert.go:666`) | node pointers are the real nodes |
| flow indexes | empty stores on the merged result; `PaintContext` builds them from scratch | no stale index can survive (see fixture 56, section 6) |

Header/footer pipeline: no change in stage 1. `assembleHeadersFooters` runs
after `RenderObjects` returns, using the final `doc.PageCount()` and
`plan.OwnerOf` (`internal/convert/pdf_pipeline.go:265`,
`internal/convert/hf.go:737`). Since the merged result is painted by the
existing `PaintContext` into the one document in document order, `[page]`,
`[topage]`, `[frompage]`, named-page margins, and `:first` / `:left` /
`:right` geometry resolve exactly as today.

### 4.2 Stage 2: per-section pagination and the PaintContext split

Stage 2 gives each worker the pagination passes as well, so the serial
pagination window shrinks. It requires:

- `PaintContext` (`internal/layout/paint.go:135`) split at the `paintPages`
  call (`paint.go:235`) into `paginateResult(ctx, res, opts)` (lines 165 to
  233: named breaks, fixpoint, seals, splits, sticky, locations, structure,
  page names, restamp) and `paintResult(ctx, doc, res, opts, contentH)`
  (lines 235 to 245).
- The worker neutralizes the candidate's own leading break before paginating
  (an engine-local style override, the same effect as the benchmark island CSS
  override at `internal/convert/page_islands.go:18`), so the candidate starts
  at local page 0.
- `applyNamedPageBreaks` still runs on the merged box tree before the workers
  paginate, so name changes across candidates are decided once.
- Structure building and `restampBoxTransforms` stay on the main goroutine
  after the merge, because both mutate document or absolute geometry state.
- Merge offsets become page-based: `pageOffset_i` is the sum of previous
  candidates' page counts, `dy_i = pageOffset_i * contentH`, page buckets are
  concatenated with an op-index base, `Locations.Page += pageOffset_i` and
  `Locations.Y += dy_i`, page names are concatenated with
  `mergePageNames` (`page_islands.go:159`), and navigation is merged with
  `appendIslandNavigation` (`page_islands.go:174`).
- Painting stays on one goroutine, in global page order, so
  `doc.AddPage`/`PageAt` and the font resource naming order are unchanged.
  `paintResult` may start once the prefix of candidates needed for the next
  page range is merged (section 5).
- PDF/UA documents stay on stage 1 (or serial): structure elements are
  document-owned and `buildStructureTree` assigns them from the merged tree.

The existing benchmark island renderer is the working precedent for all of
this at object scale: it renders each section as its own root, paints it into
the shared document, and merges page names, headings, and navigation with
offsets (`internal/convert/page_islands.go:94-157`). Stage 2 generalizes that
path and removes its sequential loop.

### 4.3 Invariants that make the merged output byte-identical

1. **Same inputs.** One style resolution for the tree; the worker sees the
   same `Options`, faces, registry, zoom, and media as the serial engine.
2. **Same build.** Each candidate's box tree and op list are produced by the
   same engine code from the same node subtree and style map; only `posY`
   differs, and `posY` shifts every op in the subtree by the same constant
   (verified by the op-for-op probe).
3. **Same continuous canvas.** Merge offsets reconstruct serial sibling
   stacking; no candidate is page-shifted before pagination.
4. **Same pagination and paint.** Exactly one `PaintContext` call on the
   merged result in stage 1; page assignment, splits, seals, locations,
   structure, link intents, page names, and restamps run once, in the existing
   order.
5. **No shared mutable state across workers.** Shared: style records (never
   mutated after insert), sheets, faces, registry, containers, and the parsed
   tree (read-only). Owned per worker: engine fields, ops, boxes, caches.
   Process caches with locks are the only cross-worker mutation.
6. **Order is an input, not an outcome.** The merger consumes a bounded
   channel in index order; worker completion order, `GOMAXPROCS`, and channel
   depth cannot change the merged list.
7. **One allocation shape.** The merged ops array is one root-capacity
   allocation as in the serial path; section buffers are reused through the
   per-call free list. B/op stays at or below the ceiling.

### 4.4 The probe that proves the merge

A differential test in `internal/layout`, run over every fixture that the
detector certifies plus the benchmark template shape:

- build the document twice, once with `LayoutContext` and once with
  `ParallelLayout` at W = 1, 2, and 8;
- compare the two display lists op for op: `Kind`, `ID`, `X`, `Y`, `W`, `H`,
  `Text`, `Font`, `Size`, `Flags`, and box `opStart/opEnd` and `y`;
- compare `Pages`, `Locations` page and rect, and page names after painting;
- compare the final PDF bytes with a pinned `Now` (date-normalized);
- require the merger's `beforeAlways` changed-target count to equal the serial
  count, so a hidden pre-shift shows up as a count mismatch instead of a
  lucky byte match.

Pass: zero differences at all three worker counts for every certified
fixture, and `output-bytes` 1,419,234 at 500p. Fail: log the first differing
field and either add the disqualifier that covers it or reject the merge rule.

## 5. Pipeline overlap as the companion or fallback

Stage 1 already overlaps work in the natural way: the worker pool keeps all
cores busy across style/build for every section, and phase 5's retained flate
pool keeps compression on the same timeline as finalize. It does not overlap
layout with paint, because pagination is global.

Stage 2 opens the three-stage pipeline:

```
sections -> [workers: build + paginate] -> bounded chan (cap W) -> ordered
merge -> bounded chan (cap 2) -> [painter: AddPage + paintResult in page
order] -> [retained flate pool: compress each finished page stream] ->
Finalize
```

- **Bounded channels.** The result channel caps in-flight section Results at
  `W + cap`; each benchmark section is about 0.2 MB of ops and 0.03 MB of
  boxes, so the cap costs about 2 MB. The page channel caps at two page-ready
  ranges. No unbounded queue, no per-section goroutine left running.
- **Paint with compression.** Phase 5 already compresses all page streams in
  parallel at finalize (`internal/pdf/flate_parallel.go:91-141`). Streaming
  means the painter enqueues each page's raw `Content.Bytes()` into the same
  retained pool as soon as the page is painted and stores the compressed
  result on the page; `finalizePages` then assembles from the stored results.
  Output must remain page-indexed, never completion-ordered; the pool already
  returns outputs in input order (`flate_parallel.go:69-85`). The raw page
  buffers are released per page (`pdf.go:1127`), so overlap does not retain
  them.
- **Memory bound.** Today's peak live heap is 210 to 212 MB and the 500p B/op
  ceiling is 234.92 MB. The pipeline must not add a second full op list or
  retain finished section Results: the merger copies and drops each part, the
  channel keeps at most `W + 2` live, and the painter holds one page at a
  time. The probe records 500p B/op and the gctrace peak; the gate is B/op at
  or below the ceiling with the peak no higher than the same tree's serial
  peak plus the channel bound.
- **Determinism rules.** Section index order, not completion order, defines
  the merge; page index order defines painting and compression; thread count
  is not an input to any content decision.

If stage 2's differential probe fails, stage 1 plus phase 5 compression is the
fallback and the measured result is reported as the 1.x answer, per PERFT-26.

## 6. Risks and falsifiable probes

| Choice | What can kill it | Probe | Pass/fail |
|---|---|---|---|
| Detector accepts a coupling it cannot see | a candidate builds differently in isolation (float leak, external abs CB, collapsing margin) | op-for-op differential over all 65 fixtures at W = 1, 2, 8 | zero field differences; first difference names the missing condition |
| Merge offsets | a Y or index is off by one op or one ulp | same differential plus `beforeAlways` changed-count equality | counts equal, 500p `output-bytes` 1,419,234, page count 500 |
| Shared style store | chunk count grows, or a data race on the intern map | temporary counter on `styleStore.append` chunk allocations; `go test -race ./internal/layout` | exactly one 219,648 B chunk plus overhead, storage 221,208 B; race clean |
| Per-section buffers | B/op doubles | 500p `-benchmem`, rate-1 memprofile | B/op <= 234.92 MB and <= same-tree serial B/op; one root-capacity ops array per conversion |
| Pre-shift / stale index | fixture 56's domain-09 box desync returned when a live bucket was read (phase 6 of perf-improve kept stale buckets and changed fixture 56; `perf-improve/results/phase-6/bucketing.md:65-87`) | fixture-56 byte compare plus `TestPaintReleasesPaginationIndexesAfterReaders` and `TestPageIndexStaleAfterEveryMutator` | bytes equal, stores empty before pagination |
| Pagination merge (stage 2) | page assignment or a split fragment changes | fixtures 31, 60, 62 pin sticky, table spill, and transform restamp (`perf-improve/designs/phase-4-shift-index.md:93-103`) | per-op page assignment equal to serial; golden 65/65 |
| Painter / doc mutation from several goroutines | object numbering, font names, or annotations change | painter is one goroutine; `-race`; bookmarks and annots byte compare | bytes equal, race clean |
| Headers/footers | page numbers or named-page margins resolve against a partial document | HF tests (`internal/convert/hf_links_test.go`) plus a `[page]/[topage]` document at 2 worker counts | identical page text and page names |
| Cancellation | workers outlive a cancelled context | follow the `internal/convert/cancellation_test.go` and `internal/layout/pagination_ctx_test.go` patterns | context error returned, caller waits for all workers, no leak |

Fixture pins used by the probes: fixture 31 (sticky top), fixture 56
(architecture diagram, the stale-index canary), fixture 60 (implemented props
A, thead and seals), fixture 62 (implemented props C, transform restamp); page
envelopes at `internal/convert/golden_test.go:334,418,437,445`.

## 7. Smallest first prototype

P0 is a test-only prototype in `internal/layout`, no production behavior
change:

1. Build two `<section>` blocks with the benchmark CSS (`page-break-before:
   always`, zero margins), parse once.
2. Serial arm: `LayoutContext` + `Paint` with a pinned `Now`, keep the bytes.
3. Parallel arm: resolve styles once; build each section with a fresh engine
   sharing the style map and a `make([]Op, 0, estimateOpCapacity(section))`
   buffer; merge with `dy_0 = 0`, `dy_1 = section0.Height`; synthetic root;
   call the existing `Paint` on the merged result.
4. Compare display lists field by field and PDF bytes.

Pass: byte-identical PDFs, op-for-op equal lists, two pages.
Fail: stop and record the first differing field. Do not proceed to the
500-page prototype.

P1 is the measured 500-page prototype, behind a test-only switch in the
convert benchmark path:

```sh
# warm 500p, generic row, three samples
go test ./internal/convert -run '^$' \
  -bench '^BenchmarkPDFPages$/^generic$/^500Pages$' \
  -benchtime=1x -count=3 -benchmem
# golden corpus
make golden
# race on the packages touched
go test -race ./internal/layout ./internal/convert
```

Pass: `output-bytes` 1,419,234, page count 500, B/op at or below 234.92 MB and
at or below the same tree's serial B/op, golden 65/65, race clean, and the
warm median lower than the same tree's serial median by at least 15 percent.
The 15 percent bar is a floor for the first prototype; the 615 ms target
(2x) applies only after phases 2 to 4 have landed and the concurrency result
is re-measured.

Fail: any metric. Record the failing invariant in the PERFT-24 row and reject
that merge rule.

## 8. Honest ceiling

All numbers below are projections from the measured phase budgets, except the
one measured flate number. The plan's per-phase targets (PERFT-08 style <= 160,
PERFT-13 build <= 175, PERFT-18 pagination plus paint <= 170, PERFT-22
finalize <= 50) sum to about 555 ms, leaving roughly 60 ms for GC and parse
before the 615 ms stretch. The profiles' independent honest ceilings are more
conservative (style 1.40x, build 1.32x, the post-layout group 1.53x to
1.57x). Concurrency multiplies the phases it covers; it cannot beat Amdahl.

| Option | Serial part after phases 2 to 5 | Parallel part | Plausible warm 500p | B/op effect |
|---|---|---:|---:|---|
| Sequential only (phases 2 to 5) | all phases | none | 650 to 830 ms | falls: keys slice, shorthand strings, seal scratch, compression buffers |
| Stage 1: parallel build, serial pagination and paint | style 160 + pagination and paint 170 + finalize 50 + GC/parse 80 | build 175 / 8 workers, about 25 to 40 | 470 to 520 ms | +1 to 4 MB before phase 2/3 savings; net must stay <= 234.92 MB |
| Stage 1 with style still serial at the old 440 budget | 440 + 170 + 50 + 80 | 40 | 780 to 800 ms | same |
| Stage 2: per-section pagination and staged paint | style 160 + paint 60 to 77 + finalize 20 to 50 + GC/parse 80 | build plus paginate 175 + 170, about 45 to 70 | 360 to 440 ms | +2 to 6 MB (page metadata, boundaries); measure |
| Stage 2 plus streaming compression | paint 60 + assembly 20 + GC/parse 80 | as above plus compression hidden | 300 to 400 ms | flat, one page raw and compressed at a time |

Reading the table:

- Stage 1 alone, on the committed tree without phases 2 to 5, saves build only
  (about 240 ms of 1,229) and lands near 990 ms, a 1.2x. It is not the 2x lever
  by itself.
- Stage 1 after phases 2 to 5 is the realistic 2x candidate: about 470 to 520
  ms, 2.4x to 2.6x, with the style phase serial.
- Stage 2 is the only route below 450 ms, and its entire gain depends on the
  per-section pagination invariance proof. Do not credit it before P1 and the
  stage-2 differential probe pass.
- No concurrency wall number here has been measured on this tree. The one
  measured wall win is compression (-150.5 ms median, byte-identical). Every
  other number is share arithmetic and is labeled as such.

## 9. Decision gates

- **PERFT-24:** P0 passes (two-section differential) and P1 passes (500p
  `output-bytes`, B/op, golden, race, 15 percent warm floor). If P1 cannot
  show byte identity, record the failing invariant and reject with the exact
  blocker.
- **PERFT-25:** enable stage 2 only if its per-op page-assignment differential
  passes on fixtures 31, 56, 60, and 62 and the 500p shape. Otherwise ship
  stage 1 as the companion and the phase 5 flate pool as the compression
  overlap, with this note as the fallback record.
- **PERFT-26:** ship only when the merged path keeps the phase 1 contract
  (output-bytes, page counts, ordered text, golden 65/65), B/op stays at or
  below 234.92 MB, and the style pin stays 221,208 B. Otherwise publish the
  measured 1.x number and this design as the rejected option with numbers.
- **PERFT-27:** final capture is the arbiter; the acceptance band is <= 615 ms
  for the stretch and 750 to 830 ms for the measured-lever floor.
