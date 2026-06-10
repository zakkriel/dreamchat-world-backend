# 10 — World Clock & Temporal Policy

**Status:** Closes the second-priority gap: nothing in the set defined where in-world time comes from, yet every table requires it and the three-axis model rests on it. This spec gives fictional time an owner. Fictional time is **logical** — a tick + label, never a calendar timestamp (ADR-030). Read alongside doc 03 (the logical-time columns) and doc 04 (slow-path extraction, which proposes timing but does not assign it).

---

## 1. Core decision

A **World Clock service** owns `in_world_tick` (the monotonic logical clock) and `in_world_label` (the fiction-facing string). The LLM may *propose* temporal interpretation; **validation assigns** the authoritative tick. Fictional time is logical, not a `TIMESTAMPTZ` (ADR-030): worlds may run on voyages, eras, dream-time, loops, or non-Earth calendars, and a real-world timestamp would silently impose a Gregorian model. This mirrors the project rule (LLM proposes, system judges) applied to time, and keeps fictional time deterministic and replayable.

## 2. Clock state (per world, per scene)

- world current time
- scene current time, scene start time
- last accepted `time_advance` event
- temporal-uncertainty flag (set when timing could not be resolved precisely)

Clock state is itself derived from accepted events (it is a projection like any other), so replay reconstructs it.

## 3. Assignment rules

**Fast path (deterministic):** mechanical events inherit `scene current time` (or current time + a fixed nominal duration for actions with inherent length, configurable). No ambiguity.

**Default for narrative events:** inherit scene current time unless the beat explicitly or strongly implies advancement.

**Time advance:** a `time_advance` event moves the clock. Triggered by explicit phrases ("three hours later," "the next morning," "after two weeks"), strong implication, or user correction. The advance may be explicit, inferred, approximate, or user-set.

**Ambiguous timing:** when duration cannot be resolved safely — keep current scene time, set the uncertainty flag, optionally ask the user *if it matters to play*, and never invent a false-precise timestamp. "Later that evening" becomes current-day + an approximate evening block + uncertainty=true, not a fabricated `21:47:03`.

## 4. Intra-beat ordering (the sharp case)

A single beat can contain multiple ordered events: the theft, then the guard hearing glass, then the guard moving. A single scene timestamp cannot order these, and the causal layer depends on the ordering (a bundle input "darkness *at the moment of* the theft" presupposes intra-beat sequence).

Rule: events within one beat share the scene `in_world_tick` and are ordered by a **monotonic intra-tick sequence number** (`beat_seq`), assigned in extraction order and validated for causal consistency (a cause's `beat_seq` must precede its effect's). Provenance and bundle inputs reference `(in_world_tick, beat_seq)`, not the tick alone. On the Narrative Claim Ledger (doc 12) this ordering lives on the claim row. This keeps the fictional clock coarse (human-meaningful) while giving the causal layer the ordering it needs. *(The `beat_seq` ordering primitive has no prior art in the research inputs; flagged in doc 11 O-3 as worth scrutiny — though the v4.1 review accepted it as the natural home for intra-beat order.)*

## 5. Slow-path extraction contract

Extraction proposes, never assigns:

```json
{
  "temporal_interpretation": {
    "mode": "scene_time | time_advance | ambiguous",
    "relative_phrase": "later that evening",
    "proposed_duration": "PT3H",
    "confidence": 0.73
  }
}
```

Validation maps this to a concrete assignment (or an uncertainty-flagged approximate), creating a `time_advance` event when `mode = time_advance` and the duration passes sanity checks (no negative durations, no advance that would reorder already-accepted later events).

## 6. Phase 0 scope

Phase 0 needs only: manual world/scene tick state, deterministic event-tick assignment, a manual `time_advance` event, `beat_seq` assignment on the scripted scenarios, and replayable clock behavior. **No LLM time parsing in Phase 0.**

## 7. Why this is a core service, not a convenience

Bad time handling causes wrong event ordering, impossible memory timing (a holder "remembering" something before it happened — I-9 catches this), incorrect backstage review triggering, broken long-gap recall, and invalid perception validity windows. The clock is load-bearing for the entire epistemic and causal apparatus.
