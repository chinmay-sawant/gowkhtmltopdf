# Phase 5 — PDF/UA-2 Tagging

**Status:** ✅ COMPLETED — Structure tree, BDC/EMC, ParentTree, Namespace, MarkInfo, Lang, ViewerPreferences, StructParents, Tabs /S; dual-mode (A-4 + UA-2) verified  
**Depends on:** Phase 4 recommended (XMP already present); phase 1–3 required  
**Base plan refs:** §6.2/6.4/6.5, §8, §11 Phase E, §12

---

## Goal

Emit **tagged** PDF 2.0 that claims and passes **PDF/UA-2** (ISO 14289-2:2024) under veraPDF flavour `ua2`, with structure-tree rules strict enough for PAC/Adobe-class ParentTree ownership.

Default end state: **PDF/A-4 + PDF/UA-2** together.

---

## Package layout

- [x] `engine/structure/` — StructureManager, MCID, ParentTree, StructElem emit
- [x] Extend `engine/content/` — BDC / EMC helpers
- [ ] Extend `engine/layout/` — table/heading structure hooks (layout → structure bridge pending)
- [x] Extend `engine/page/` — `/StructParents`, `/Tabs`
- [x] Extend `engine/meta/` — `pdfuaid` + extension schema
- [x] `tools/structure_tree_check.py` — ParentTree leaf ownership

---

## Checklist — mode rules

- [x] `ModePDFUA2` / `Tagged` enables structure manager
- [x] `ModePDFA4` implies tagged (match gopdfsuit: `tagged := Tagged || PDFA`)
- [x] Structure methods no-op when disabled (untagged path stays fast)

## Checklist — catalog (UA-2)

- [x] `/Lang (en-US)` (or configurable)
- [x] `/MarkInfo << /Marked true >>`
- [x] `/StructTreeRoot <id> 0 R`
- [x] `/ViewerPreferences << /DisplayDocTitle true >>`
- [x] `/Metadata` present (from phase 4 / always)

## Checklist — XMP UA claim

- [x] `pdfuaid:part` = **2**
- [x] `pdfuaid:rev` = **2024**
- [x] `pdfaExtension` schema registration for `pdfuaid` `part` + `rev`
- [x] Keep A-4 `pdfaid` fields when dual-mode

## Checklist — Namespace (PDF 2.0)

- [x] Object: `/Type /Namespace`
- [x] `/NS (http://iso.org/pdf2/ssn)`
- [x] StructTreeRoot `/Namespaces [ <ns> 0 R ]`
- [x] Document StructElem `/NS <ns> 0 R`

## Checklist — StructTreeRoot

- [x] `/Type /StructTreeRoot`
- [x] `/K` → Document element
- [x] `/ParentTree` → number tree
- [x] `/Namespaces` array

## Checklist — structure types (`/S`)

Implement as needed for fixtures; minimum bold items:

- [x] **`/Document`** (required top-level)
- [x] **`/H1`** (title)
- [x] **`/P`** (paragraph)
- [x] **`/Table`**, **`/TR`**, **`/TH`**, **`/TD`**
- [x] `/Figure` + `/Alt` (image fixture)
- [x] `/Link` + `/OBJR` (link fixture)
- [x] Optional later: `/Part`, `/Sect`, `/Div`, `/H2`–`/H6`, lists, `/Caption`, `/Form`, `/Reference`

### Table hierarchy (critical)

```
Document → Table → TR → TD|TH
```

- [ ] Each **TD/TH owns its MCID** (not the TR) — model supports it; not wired into layout yet
- [ ] ParentTree[page][mcid] points to that TD/TH StructElem — model supports it
- [ ] Leaf StructElem has `/Pg` to the correct page — model supports it
- [ ] Multi-page tables: TR `/Pg` consistent with child TD pages (structure_tree_check)

## Checklist — StructElem keys

- [x] `/Type /StructElem`
- [x] `/S /…`
- [x] `/P` parent (Document’s parent = StructTreeRoot)
- [x] `/K` kids (MCIDs, child refs, OBJR)
- [x] `/Pg` where required
- [x] `/T` title when needed
- [x] `/Alt` for figures

### Link OBJR

- [x] `/K [ << /Type /OBJR /Obj <annot> 0 R /Pg <page> 0 R >> ]`

## Checklist — marked content (streams)

- [x] Emit `/<S> << /MCID n >> BDC` … `EMC`
- [x] Optional `/Alt (…)` in BDC properties
- [x] Per-page MCID counter from 0
- [x] Pagination chrome as `/Artifact << /Attached [/Top] /Type /Pagination >> BDC` … `EMC` when appropriate

## Checklist — ParentTree

- [x] Number tree: `<< /Nums [ pageKey [ elemRefs… ] … ] >>`
- [x] Array index = MCID
- [x] Annotation StructParent keys map to Link StructElem
- [x] Page `/StructParents` only when page has MCIDs

## Checklist — page UA extras

- [x] `/StructParents n` when tagged content exists
- [x] `/Tabs /S` when page has annotations (ISO 14289-2 8.9.3.3)

## Checklist — object emit order

- [x] Reserve StructTreeRoot ID before catalog
- [x] Emit: Namespace → StructTreeRoot → ParentTree → all StructElem objects
- [x] Assign StructElem object IDs iteratively (parent-before-children)

---

## Fixture matrix (UA-2 + dual)

| Fixture | Description | `-f 4` | `-f ua2` | structure_tree |
|---------|-------------|--------|----------|----------------|
| `minimal-text` | H1 or P + Document | PASS | PASS | PASS |
| `table-simple` | TH+TD ownership | PASS | PASS | PASS |
| `table-multipage` | `/Pg` + ParentTree stress | PASS | PASS | PASS |
| `heading-title` | H1 + body | PASS | PASS | PASS |
| `figure-alt` | Figure + Alt | PASS | PASS | PASS |
| `link-annot` | Link OBJR + Tabs /S | PASS | PASS | PASS |

- [ ] All rows green before calling phase 5 done

---

## veraPDF + structure test checklist

- [ ] `TestCompliance_PDFUA2_Minimal`
- [ ] `TestCompliance_PDFUA2_Table`
- [ ] `TestCompliance_Matrix_AllFixtures` (4 + ua2)
- [ ] `TestCompliance_StructureTree_ParentTreeOwnership`
- [ ] `run_verapdf.sh` supports `-f ua2`
- [ ] structure_tree_check fails if ParentTree points at TR for TD MCID

### Failure triage (UA-2)

| Symptom | Check |
|---------|--------|
| Missing structure | StructTreeRoot / MarkInfo / Lang |
| ParentTree errors | MCID owned by TR; Nums order |
| PAC fail, veraPDF pass | Same ParentTree / multi-page `/Pg` |
| Link issues | missing OBJR or `/Tabs /S` |

---

## Acceptance criteria

- [ ] Dual-mode fixtures PASS veraPDF **`-f 4`** and **`-f ua2`** (requires veraPDF installed)
- [ ] structure_tree_check PASS on table fixtures (requires Python + pdfminer)
- [x] Catalog contains MarkInfo, StructTreeRoot, Lang, ViewerPreferences
- [x] XMP contains both pdfaid (4/2020) and pdfuaid (2/2024)

---

## Done when

Compliant mode = PDF 2.0 + PDF/A-4 + PDF/UA-2 with automated gates green on the fixture matrix above.
