# 00 — Time & Mutability Rules (normative for all Compendium PRDs)

**Status:** Accepted 2026-06-10 (product decision). Applies to every schema in this folder. Where any PRD text conflicts with this doc, **this doc wins**.
**Source of authority:** Canon Engine ADR-001 (immutable canon events), ADR-006 (three time axes; invalidation never deletion), ADR-030 (fictional time is logical: tick + label, never `TIMESTAMPTZ`).

---

## The rule

> **Canon and perception are forward-moving and append-only. Nothing in the domain is ever updated in place.**

There is no such thing as `updated_at_in_world_time`. A record that can be "updated" is a record that can silently lose history — which breaks replay, audit, correction, and the product promise itself.

## What replaced the old fields

| Old field (pre-engine, banned) | Replacement | Semantics |
|---|---|---|
| `updated_at_in_world_time` | `as_of_tick` | **Derived**, never stored as mutable state: the tick of the latest committed event/version this view reflects. Lives on *projections*, not on domain truth. |
| `created_at_in_world_time` | `created_at_tick` | Logical tick (+ display label) at which the element entered canon/perception. Written once, immutable. |
| `last_updated_at` (perceptions) | `latest_version_tick` | **Derived**: tick of the newest Perception Version. Perceptions change by appending versions, never by editing. |
| `first_perceived_at` | `first_perceived_tick` | Written once, immutable. |
| `in_world_time` (on records) | `occurred_at_tick` + `display_label` | Tick for ordering/logic; human label ("Day 3, Morning") for display. The label is presentation, the tick is truth. |
| `last_confirmed_at_in_world_time` (Carry State) | `last_confirmed_tick` | Derived from the latest supporting perception record. Drives decay language ("last known…"), never visibility. |

## The three time axes (never conflate)

1. **In-world time** — logical tick + display label, owned by the World Clock (engine doc 10). The only time users see.
2. **Event order** — the append sequence of canon events/perception versions. Drives derivation and replay.
3. **System wall-clock** — `created_at TIMESTAMPTZ` etc. on database rows. Operational telemetry only (cache freshness, debugging, ops). **Never rendered in the product UI, never used for world logic.**

A projection row carrying a wall-clock `updated_at` is fine — that is axis 3 metadata about a cache, not domain truth.

## How change is expressed (instead of updates)

- An Actor "changes" → a new canon event is committed; projections re-derive; `as_of_tick` advances.
- Understanding changes → a new Perception Version is appended; the page synthesis derives from the latest version; older versions remain queryable ("how my understanding evolved").
- Something becomes wrong → it is **invalidated** by a correction event (present-forward), never deleted or overwritten.
- An Artifact moves hands → a new Carry State derivation from new events; the old state remains in history.
