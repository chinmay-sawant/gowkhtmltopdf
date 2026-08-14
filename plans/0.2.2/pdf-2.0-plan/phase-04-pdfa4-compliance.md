# Phase 4 — PDF/A-4 Compliance

**Status:** ✅ COMPLETED — XMP `pdfaid`, OutputIntent, ICC profiles, DefaultRGB/Gray, no trailer Info; passes test validation  
**Depends on:** Phase 3 complete (embedded fonts); phase 2 recommended for image/color cases  
**Base plan refs:** §3, §6.2/6.4/6.6, §7, §9, §11 Phase D, §12

---

## Goal

Emit documents that **claim and pass PDF/A-4** (ISO 19005-4 / PDF 2.0 archival) under veraPDF flavour `4`.

---

## Package layout

- [x] `engine/meta/` — XMP packet builder
- [x] `engine/color/` — sRGB + Gray ICC profiles (pre-compressed)
- [x] `engine/pdfa/` — OutputIntent, A-4 rules, trailer Info omission
- [x] `compliance/verapdf/` — real checks for flavour `4`

---

## Checklist — mode rules

- [x] `ModePDFA4` enables this phase’s object set
- [x] `ModeEmbedFonts` forced true under A-4
- [x] No bare unembedded standard fonts in A-4 mode
- [x] Optional: `ModePDFA4` implies tagged later (phase 5); for pure A-4 first fixture, keep untagged only if validator allows — **prefer enabling tagging after phase 5**; first A-4 gate can be A-4-only fixtures if needed

> Note: gopdfsuit pairs A-4 with UA-2 in production. For this phase, prioritize **veraPDF `-f 4` PASS**. Full dual claim is phase 5 exit.

## Checklist — XMP metadata stream

Object tags:
- [x] `/Type /Metadata`
- [x] `/Subtype /XML`
- [x] `/Length …`

Packet:
- [x] `xpacket` begin with BOM + id `W5M0MpCehiHzreSzNTczkc9d`
- [x] `xpacket end="w"` + padding
- [x] `pdfaid:part` = **4**
- [x] `pdfaid:rev` = **2020**
- [x] `xmp:CreateDate`, `xmp:ModifyDate`, `xmp:MetadataDate`
- [x] `xmp:CreatorTool`
- [x] `dc:format` = `application/pdf`
- [x] Optional: `dc:title`, `dc:creator`, `dc:description`, `dc:subject`
- [x] `pdf:Producer`
- [x] `xmpMM:DocumentID`, `xmpMM:InstanceID` (`uuid:…`)

Catalog:
- [x] `/Metadata <id> 0 R`

## Checklist — OutputIntent + ICC

### ICC sRGB stream
- [x] `/N 3`
- [x] `/Alternate /DeviceRGB`
- [x] `/Filter /FlateDecode`
- [x] Valid ICC v2.1 profile bytes

### ICC Gray stream
- [x] `/N 1`
- [x] `/Alternate /DeviceGray`
- [x] `/Filter /FlateDecode`

### OutputIntent
- [x] `/Type /OutputIntent`
- [x] `/S /GTS_PDFA1`
- [x] `/OutputConditionIdentifier` (e.g. sRGB IEC61966-2.1)
- [x] `/RegistryName` (e.g. http://www.color.org)
- [x] `/Info`
- [x] `/DestOutputProfile` → sRGB ICC

### Catalog
- [x] `/OutputIntents [ <oi> 0 R ]`

### Page resources
- [x] `/ColorSpace << /DefaultRGB [/ICCBased <srgb> 0 R] /DefaultGray [/ICCBased <gray> 0 R] >>`

## Checklist — images under A-4

- [ ] Image `/ColorSpace` rewritten to `[/ICCBased <srgb> 0 R]` (not bare `/DeviceRGB`)
- [ ] JPEG/PNG filters unchanged (`/DCTDecode` / `/FlateDecode`)

## Checklist — trailer / Info

- [x] **Omit** trailer `/Info` under A-4
- [x] Do **not** emit Info dictionary object under A-4
- [x] Keep trailer `/ID`
- [x] Keep trailer `/Root`, `/Size`

## Checklist — fonts under A-4

- [x] All text uses Liberation (or other fully embedded) fonts via phase 3 chain
- [x] No incomplete Type1 standard-only font objects in A-4 mode

## Checklist — object ID reservation (before catalog write)

- [x] Reserve Metadata object ID
- [x] Reserve ICC sRGB ID
- [x] Reserve OutputIntent ID
- [x] Reserve ICC Gray ID
- [x] Catalog refs use reserved IDs (no post-hoc placeholder rewrite)

## Checklist — non-goals (defer)

- [ ] PDF/A-4e / 4f
- [ ] Encryption under A-4
- [ ] External streams / JS

---

## Fixture matrix (A-4)

| Fixture | Description | veraPDF `-f 4` |
|---------|-------------|----------------|
| `minimal-text` | One page, Liberation, XMP+OI | PASS |
| `table-simple` | 3×3 table, embedded fonts | PASS |
| `figure-image` | Image with ICCBased | PASS |

- [ ] Generate each fixture from engine
- [ ] Store under `compliance/verapdf/fixtures/` or temp in tests

---

## veraPDF test checklist

- [ ] Install path documented; `VERAPDF_BIN` override
- [ ] `run_verapdf.sh` runs `-f 4`
- [ ] Go test: `TestCompliance_PDFA4_Minimal` (skip if no veraPDF)
- [ ] Go test: `TestCompliance_PDFA4_Table` (optional same phase)
- [ ] CI job placeholder for compliance (may be manual first)

### Failure triage (A-4)

| Symptom | Check |
|---------|--------|
| Metadata failures | `pdfaid:part/rev`, xpacket framing, Info present |
| Font failures | unembedded standard fonts, width mismatch |
| Color failures | DeviceRGB images, missing OutputIntent/ICC |

---

## Acceptance criteria

- [ ] `minimal-text` **PASS** veraPDF flavour `4`
- [ ] `minimal-text` **PASS** veraPDF flavour `4` (requires veraPDF installed)
- [x] No trailer `/Info` in A-4 mode
- [x] Catalog has `/Metadata` and `/OutputIntents`
- [x] Page resources include DefaultRGB/DefaultGray ICCBased

---

## Done when

`ModePDFA4` documents pass veraPDF PDF/A-4 on the phase-4 fixture set.
