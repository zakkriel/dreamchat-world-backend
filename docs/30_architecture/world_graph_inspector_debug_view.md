# World Graph Inspector (creator/debug mode) — minimal spec

**Status:** Accepted 2026-06-10 (from Rules Register gap G-1). Deliberately minimal — a read-only debug tool, not a product feature.
**Rule basis:** C-4 permits authoritative state in creator/debug mode only. This view never appears in play mode.

## Purpose
Answer, for any event/mutation/perception: **"where did this come from, and what depends on it?"** — the question that surfaced this gap ("I cannot see how canon events relate in dependency").

## Scope (and nothing more)
- **Input:** one record id (event, mutation, or perception), or one entity (showing its recent records).
- **Output:** a two-direction graph around that record:
  - **Upstream:** provenance edges (`derived_from`, `inferred_from`, `reported_by`, `witnessed_by`, `compensates`, `supersedes`) + mandatory `source_event_id` links.
  - **Downstream:** records whose provenance points here; from Phase 4, causal bundles where this record is an input (rendered as a bundle node with its typed inputs — trigger/enabler/blocker, polarity — and one effect).
- **Depth:** default 2 hops, max 4. Read-only. No editing, no filtering UI beyond depth and kind toggles.

## Implementation notes
- Pure queries over existing tables (`provenance_edge`, `causal_bundle`, `causal_bundle_input`, `perception_record.source_event_id`). **No new tables, no new writes, no engine changes** — consistent with the frozen engine set.
- Rendering: simplest available (server-rendered SVG/DOT or a tiny client graph). Status coloring: accepted / invalidated / pending_review / dirty.
- Phase availability: useful from Phase 0A (provenance exists from day one); bundles appear automatically when Phase 4 starts writing them.

## Non-goals
World map visualization · relationship graphs · player-facing anything · graph editing · analytics. If it grows past one screen, it has exceeded this spec.
