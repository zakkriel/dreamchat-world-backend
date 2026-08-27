# 01 — Canonical Epistemic Enum (normative for all Compendium PRDs)

**Status:** Accepted 2026-06-10 (product decision: the engine's `epistemic_type` is the single canonical enum).
**Replaces:** the per-PRD `source_type` lists, which drifted across documents.

## The canonical enum (engine `perception_record.epistemic_type`)

`direct` · `shared` · `told` · `overheard` · `public` · `rumor` · `inference` · `mistaken` · `confirmed` · `disputed`

## Mapping from old PRD `source_type` values

| Old PRD value | Canonical | Medium/detail goes to |
|---|---|---|
| direct, observation | `direct` | `sensory_mode` ('visual','auditory','deduction',…) |
| told_by_actor, told_by_other | `told` | `source_actor_id` + source metadata |
| rumor | `rumor` | source metadata |
| public_record, document, broadcast, institutional_record | `public` | source metadata (`source_text/reference` carries the medium: record, document, broadcast, …) |
| inference | `inference` | provenance edges (`inferred_from`) |
| unknown | — removed | a perception with unknown epistemics is a modeling smell; if genuinely needed, use `rumor` with low confidence |

**Rule:** *what kind of knowing* is the enum; *through what medium* is metadata. Common Knowledge (Glossary) enters as `public` with a common-knowledge source marker. Distortion and reliability live in `confidence` / `distortion_level`, not in new enum values.
