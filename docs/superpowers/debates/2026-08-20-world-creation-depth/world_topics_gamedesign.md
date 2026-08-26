# What a full world must contain — gamedesign

**Checks applied before writing** (handover §1.2): brief-traceability — no identifier below could come
from one brief (`doing`, `wants`, `binds`, `pace_class` are grammar words; `tide`, `slate`, `caste` are
values). No exemption lists — every rule below is a comparison. Vocabulary/grammar — the seat mints
names, never slots. Engine gap ⇒ engine program — five of my ten clusters need engine work and I name
which. Every leaf reaches a reader — the ones that don't, I mark as blocked, not as design.

## 1. The clusters, by the beat they produce

**Tier A — already live, already universal. These are the ones that work.**

1. **Place graph + ways.** Without ≥2 places and one way, `ActorMoved` has no legal target
   (`fn_portal_permits`). *Table moment:* beat 1 — the only things the player can name are this room, its
   portals, and what's on the far side (`beathandler.go:384-397`). Live.
2. **Tension per place.** Class → beat budget → journey (`tension.go:28-45`). *Table moment:* the first
   act that doesn't fit the beat and becomes travel instead. Live; this is the proof-shape.
3. **One private thing per person.** Not flavour: `fn_isolated_npcs` gives an NPC her own cognition call
   *iff* she holds a private record touching the action (`schema.sql:2359-2360`). A world with no secrets
   runs every mind in the batch seat forever. *Table moment:* the first answer that is a deflection.
4. **Asymmetric memory of one shared past event.** `history[].knowledge`, per holder, per epistemic type.
   *Table moment:* beats 3-5 — two people describe the same night differently and the player can work the
   gap. This is the deepest thing the engine already does.
5. **Objects as leverage, not scenery.** `carried_by` plus the fact sheet's `reachable`/`weight`/`contents`.
   *Table moment:* beat 2 — take it, ask for it, or notice who won't put it down.

**Tier B — "already running". This is the actual gap, and all three need engine work.**

6. **What each person is doing right now.** *Table moment:* beat 1 — you interrupt something instead of
   being greeted. This is the difference between a place and a set, and it is the cheapest one to get.
   **No reader today** (`REFERENCE:149`). Destination: the cognition prompt's present block and the
   narrator's scene state. **Engine work.**
7. **What each person wants next — and at least two of those wants incompatible.** Traits give *how* a
   mind acts; nothing gives *what it is trying to get*, so minds are reactive and the world revolves
   around the player — the exact thing the cognition design forbids
   (`FINAL-world-npc-cognition.md:3`). *Table moment:* beat 2-4, an NPC acts without being addressed.
   **No reader today.** Same destination as 6 — one reader serves both, plus `role`
   (`REFERENCE:152-154`). **Engine work.**
8. **One scheduled change that alters state.** *Table moment:* beat 3-5 — the water rises while you are
   still talking, and it was going to happen whether you came or not. `pending_event` is read every clock
   crossing and written by tests only (handover §3.3). **Engine work** (a writer, plus §5 below).

**Tier C — the rules and the type of world.**

9. **Norms + who they bind, and collectives whose membership is legible or not.** *Table moment:* an NPC
   refuses something physically possible — the only way a rule is ever *felt* — and the moment the player
   mis-reads who someone is. Norms must **not** land in `personality_core.traits`: traits drift under
   `malleability` (handover §3.4), so a law stored there decays per mind. **Engine work:** one non-trait
   per-mind channel.
10. **Motion vocabulary + conditions that hinder it.** *Table moment:* distance stops being one number —
    the same stair is nothing to one person and impossible for another. `fn_move_duration_actor`
    hardcodes `'walk'` (handover §3.3) and there is no actor↔type binding. **Engine work**, and it is the
    founder's own transposition insight, not mine.

Five of ten need engine work. Shipping any of 6-10 as schema before its reader exists is
`cast[].standing` again, five times.

## 2. Where the GA-2 line sits

**A universal is legal iff its absence leaves an engine mechanism dark. It must be stated as a slot the
beat loop reads, never as a kind of thing worlds contain.**

That is why the existing universals do not feel like an ontology: "≥2 places" exists because movement has
no target otherwise; "everyone hides one thing" exists because the private-cognition path is unreachable
without it (`schema.sql:2359-2360`). Neither says what a world *is*. By the same test "≥1 collective" is
**illegal** — no mechanism requires a group — as are "someone in authority" and "a scarce resource".

This answers the comprehensiveness trap. Coverage comes not from enumerating fiction but from enumerating
**the machinery that has an input and currently gets nothing** — presence, intention, scheduled change,
private knowledge, motion. That list is derivable from the engine and finite; the brief fills the slots
and the fiction stays free. The honest consequence: anything with no mechanism behind it does not belong
in the document, however good it sounds.

## 3. What I would cut from the reference shape

- **`regard`** (`REFERENCE:150`). `relationship_state` has zero readers and the `[RELATIONSHIPS]` block
  is unrendered. This is the most seductive field in the document and the purest repeat of the defect.
- **`role`** — merge into the one per-mind situation block with `doing`/`wants`. Three leaves competing
  for one unbuilt reader will lose three separate fights.
- **`collectives[].description`** — no destination named; `descriptor` + `legibility` carry the play.
- **`places[].kind`** — dropped today (handover §3.3) and it is a free-text invitation to genre nouns.
- **`mood`/`ornament` as arrays** — they reach `world_theme` and stop there. Keep single words; do not let
  decoration grow while Tier B has no reader.

## 4. The single biggest omission

**Nothing in the shape states that two present people want incompatible things right now.**

`wants` is per-person and unopposed; `regard` gestures at relations and reaches nothing. With Tier A alone,
the player is the only source of motion in the room for five beats — the world actor's pressure roll
(`fn_pressure_chance`) is the sole autonomous mover, and it is random rather than situated. Authored
opposition is what makes a room keep moving when the player says nothing, and it costs one field on the
same reader Tier B already needs. Its player-facing half is equally missing: nothing says what happens
**to the player** in the next minute if they do nothing.

## 5. Reference-file failures against handover §1.2

1. **`role`, `wants`, `doing`, `regard` fail "every authored leaf reaches a reader"** (`REFERENCE:149-150`).
   Blocked, not designed.
2. **`near_future[].sets` cannot become a `pending_event`.** The payload is `{actor_id, attempt}`
   (`ledger.go:16-19`) — an acting world entity — and the example's subject is a *place*
   (`"subject": "Vaunt Shallows"`). Nothing can be attributed as the actor of the tide. Either the mover
   is minted as an entity or this cluster does not land: today it is authored-but-inert by construction.
3. **`ways.impedes` sits in the IR while its own signature change is unruled** (handover §5.1). That is an
   engine program; including it here violates "never patched at world creation".
4. **`conditions[].hinders` can only ever bind movement** — `status_modifier.action_type CHECK
   (action_type IN ('move'))` (handover §3.4). A condition hindering speech or perception has nowhere to
   go, so the cluster invites authoring that cannot land.
5. **Shape drift in `arrival`:** no `stated` (the player's single perception), and two candidates share
   one `canonical_name` where the live schema requires exactly three distinct ones
   (`world_genesis.v1.schema.json:305-307`).
