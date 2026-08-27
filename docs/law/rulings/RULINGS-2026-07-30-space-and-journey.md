# RULINGS — 2026-07-30 (space, candidates, the Journey)

**Why this file exists.** During Station F's exit prep, the founder made two rulings that evolve the
action-contracts spec. Both are recorded here with example + reasoning per the standing rule, so an
implementing agent who wasn't in the room cannot misread them.

---

## 1. The candidate whitelist = everything the actor PERCEIVES (not actors-only)

The decomposer's candidate whitelist — the set of ids an action may bind to — is **every entity the
acting actor perceives**, not just present actors and the current location. Artifacts you can see are
bindable: the bar you approach, the crate you grab, the note you carry.

**Example.** In the Drowned Lantern, "approach the bar", "grab the crate", "give the note to Mara" all
bind — the bar, the crate, and the note are perceived entities in (or on) the actor, so they are
candidates. Before this ruling the whitelist was arbitrarily actors + location only, and "approach the
bar" had no id to bind (dead end).

**The bound stays — this does NOT open to every id.** Scoped by perception/knowledge, exactly as
naming-reach already rules (RULINGS-2026-07-23 §3): *"nobody can bind an entity they have no
perception/knowledge path to."* You can bind the bar you see and things you know one hop out; you still
cannot bind the harbormaster you have never seen. The fix removes an arbitrary *under*-inclusion
(artifacts), it does not remove the wall.

**Mechanically:** candidates = co-located actors + co-located/perceived artifacts + items the actor
carries/holds + the current location + one-hop-known entities — each labeled by the actor's own
`fn_display_name` (viewer-relative naming, RULINGS-2026-07-30 naming reach), bounded by perception.

---

## 2. Over-budget is not a REJECT — it is the JOURNEY (initiative passes to the world)

**Evolves FINAL-action-contracts §2** ("Over-budget move = REJECT (v1)") and §6 (the tension budget).
An action that exceeds the current beat's budget is **not rejected and the actor is not blocked from
acting**. Instead the action becomes a **journey**: a sequence of beats, each beat a world-action slot.

**The founder's worked example (tavern → your house, mid-tension):**
- The action spans multiple beats. Each beat, the world acts first (its slot): it may telegraph, cut
  in, or redirect the actor — or do nothing.
- **Nobody acts this slot → the action makes progress and carries to the next slot.** It is not "done"
  and it is not rejected; it continues.
- **Context re-evaluates as the actor moves.** The moment you leave the tavern the tense scene is
  behind you — the next slot runs under the new (lower) tension, and you are now at a waypoint (the
  tavern door) looking onward.
- Across the slots the journey needs, the world had **multiple chances** to stop or redirect the
  actor. If it never did → the actor **arrives** ("gets home"). "It tried to get out and managed to,
  but there were multiple world-action slots in the middle that could have stopped or forced a change
  of plans — or not, and it just resolves."

**Why this replaces REJECT.** "You can't even try to leave" is dramatically dead — the exact
player-centric wall this system refuses. Over-budget meaning *the world goes first, throughout* is
alive: fleeing a standoff is possible, it just gives everyone else their move.

**This IS journey-split** — the subsystem §2 explicitly deferred ("journey-split later when
between-places topology exists"). It is now RULED IN, and it is its own body of work (per-beat
progress + resume state; a world slot per beat via the existing world-first loop; tension re-read at
each waypoint; interruption via the existing telegraph → held-outcome → reaction-beat machinery;
arrival at threshold — the "Way A" accumulate-to-threshold shape from §6, applied to movement). It is
NOT a Station-F tweak; it is built as its own plan ("the Journey"), same discipline as every station.

**Station F interim (honest, not a half-build):** F keeps the spec's current over-budget behavior
until the Journey is built; F's exit seed is tuned so the playable moves fit the budget (stepping out
the tavern door is a short move). No partial journey mechanic is jammed into F.

---

## Sequencing (founder, 2026-07-30): "F and the Journey"

Finish Station F now (perceived-candidate binding + seed geometry tune → the playable exit), then
build the Journey as the next dedicated plan. Both ruled; only the order is set.
