# Design — The Living World (Station G / World Actor)

**Status: DESIGN, founder-approved 2026-08-05 (brainstorming session).** Awaiting founder review of this
written spec, then an implementation plan (writing-plans). Grounds the Journey (its own plan, next).

## The problem

The world does not move on its own. Today the clock only advances when the player physically moves; a
scene where you stand and talk **freezes world-time** — no pressure builds, no scheduled event can come
due, the alarm can never cut your conversation. And the foundations left only **sockets** for the world
acting: the `pending_event` ledger table, the `world_pressure` pools table, `fn_due_pending`, and a
`world_actor` seat bound to a fake driver — but **zero behavior**. Nothing fires the ledger on a
clock-crossing, nothing accrues pressure or rolls, and the World Actor is never invoked in the beat.

This station makes the world **advance on its own clock and act on its own** — the two composing
mechanisms of RULINGS-2026-07-23 §7: **known futures** (the pending-events ledger, deterministic) and
**spontaneity** (the World Actor, a mind erupting on rising pressure).

## The principle

The world runs on **world-time, not player steps** (§7b — "a per-step basis would key the world's rhythm
to player behavior, the player-centricity this system refuses"). Time passes every beat; the world takes
its own turn after yours; and disruption at the worst moment is a *feature*, not a bug to filter out (§5).
The engine owns **WHEN** (mechanical time, mechanical rolls); the LLM owns **WHAT** (the authored
intrusion).

## Modular decomposition (the mandate — five units, one job each)

Per the founder's core architecture mandate, the station is five small units with clean interfaces, each
testable in isolation — **not** a lump inside the beat loop:

- **The clock** *(Unit 2)* — computes elapsed world-time and advances it.
- **The scheduled-events ledger** *(Unit 3)* — fires due events on a clock-crossing. Pure SQL, no LLM.
- **The pressure pools + roll** *(Unit 4)* — derives each pool's chance, rolls, decides what fires. Deterministic.
- **The World Actor seat** *(Unit 5)* — world payload in, authored intrusion out. The *only* LLM boundary here.
- **The world's-turn composer** *(Unit 6)* — the thin reusable unit that calls the four above in order; the
  seam the Journey reuses.

Plus two touch-points on existing code: the **decomposer** gains a duration classification *(Unit 1)*, and
the **trace** gains a world's-turn section *(Unit 7)*. The Unit sections below are numbered in build order.

---

## Unit 1 — World-time: every beat costs time

**What it does.** Makes every beat advance world-time, so the world can stir even in a talk-only scene.

- **Beat world-time = the sum of its attempts' durations, with a per-beat floor** (so an empty "I just
  watch" beat still nudges the clock — the floor *is* the `instant` value).
- **Move attempt → `fn_move_duration_actor`** (physics, unchanged — distance ÷ speed).
- **Non-move attempt → a `duration_class`** the decomposer emits: one of
  `instant · short · medium · long · extremely_long`. A **validated enum**, a parse-shape classification
  in the exact spirit of how the decomposer already recognizes a `QUERY` — **not** a raw number, and
  **not** the banned outcome/tension/intent judgment (RULINGS-2026-07-23 §4). "A life story is always
  long"; the decomposer recognizes the *class*, the **engine** owns what "long" means in seconds.
- **Engine maps class → seconds from per-world config** (retunable data, never hardcoded): e.g. instant
  ≈ 1–2 s (nod, glance), short ≈ 5 s (a line, hand over the note), medium ≈ a minute, long ≈ minutes (the
  life story, searching a room), extremely_long ≈ hours. The enum bounds the max by construction — no
  free-text number, so no "I wait 100 years" via a class.

**Sustained-until-threshold acts are NOT a duration class — they are the Journey.** "Lay hidden until
2am", "wait 100 years", "walk home" have no intrinsic length; they run **until a condition is met** (a
time, or an event) and are interruptible throughout. That is the Journey's shape (RULINGS-2026-07-30 §2),
now **expanded beyond travel**: travel resolves on *distance covered*, a vigil resolves on *the clock
reaching a tick*, a watch resolves on *an event firing* — one mechanism, spatial or temporal. The
decomposer recognizes the *"until/for <condition>"* form (a parse-shape, like QUERY) and binds the
condition; **time conditions ride this station's clock, event conditions ride this station's ledger** —
which is why the Living World must exist before the Journey. The uniform test: **span fits the beat's
budget → resolves inline; exceeds it → becomes a Journey**, whether a long monologue or a hundred-year
vigil.

**Interface:** `duration_class` field added to the decomposer's `beat_chain.v2` non-move attempts;
`fn_duration_class_seconds(p_world_id, p_class)` (or a config lookup) maps class → seconds; the beat's
clock-advance sums per-attempt durations with the floor. Depends on: the decompose seat, `fn_move_duration_actor`,
per-world config.

---

## Unit 2 — The clock & the world's turn (ordering)

**What it does.** Defines *where* the world acts in a beat and advances the clock.

The world's two autonomous mechanisms are **time-driven**, which places them differently from the NPC
hook already in the loop:

- The **NPC world-first hook** (existing) fires *before* your action resolves — present minds *react to
  your intent*. Reactive.
- The **World Actor and the ledger** are *not* reacting to you — they fire on **elapsed world-time**, so
  they run *after* your action resolves and the clock advances.

One attempt therefore processes as:

1. NPC world-first (react to intent) — *existing*
2. premise → route → commit (your action resolves) — *existing*
3. clock advances (world-time += the action's duration) — *existing, now non-zero for non-moves*
4. **the world's turn (new):** the world's-turn composer (Unit 6) runs.

**Per-attempt, not end-of-beat.** §5's whole point is disruption *mid-chain* ("the alarm screams while
you're mid-chain"). Run after *each* attempt, an eruption can fire after "cross to the bar" and drop
"slip her the note, ask about the harbormaster". Cost stays bounded: the roll is cheap and usually
doesn't fire; the **LLM is called only when a tier actually erupts**; and a medium/large eruption ends the
beat, so at most one big one per beat.

**Interruption is keyed to magnitude (RULINGS-2026-07-24 §5), not judgment:** **small** commits and the
chain runs on; **medium/large** commit, the narrator delivers, and the **beat ends** — remaining attempts
discarded (same rule as a telegraph). An eruption is **not contestable** — no held outcome, no reaction
beat (§5).

**Interface:** the beat loop calls the world's-turn composer after each attempt's clock advance, passing
`(tickBefore, tickAfter)`; a returned "biggest magnitude that fired" drives the §5 cut.

---

## Unit 3 — The scheduled-events ledger (known futures, 7a)

**What it does.** Fires consequences already caused when the clock crosses their fire-time — the dusk
patrol, the contact arriving, the spell expiring at tick 340. Zero model calls; the world's agenda is
**data**.

- On each clock advance in a slot, find pending events **due in `(tickBefore, tickAfter]`** via the
  existing `fn_due_pending`, ordered by fire-time.
- Each fires: its payload becomes a real canon event through the **normal apply/commit path** (perception
  fan-out included), and the row flips `status = 'fired'`.
- Each pending event carries a **magnitude**; §5's cut applies exactly as for an eruption (small runs on,
  medium/large end the beat).

**Interface:** reuses `pending_event` + `fn_due_pending` (already in schema); a Go step in the world's-turn
composer fires due rows and commits their payloads. Pure SQL + commit; no LLM.

---

## Unit 4 — The pressure pools & the roll (spontaneity engine-side, 7b)

**What it does.** Decides *when* the World Actor gets to act — mechanically, deterministically,
replayably.

**Pressure = world-time elapsed since that tier last erupted.** Nothing to store or tick per beat; it is a
pure function of the canon (each tier's readiness = `now − tick-of-that-tier's-last-eruption`). Every §7b
property falls out for free:

- *chance grows with time since it last acted* → literally the formula;
- *three independent pools* (`small` minutes / `medium` hours / `large` days–weeks) → each tracks its own
  last-eruption tick and its own rate; a drunk stumbling never touches the alarm's clock;
- *drains when it fires* → a fresh eruption is the new last-eruption tick, so readiness resets on its own;
- *carries across scenes* → world-global, not scene-scoped.

**The chance is a capped linear climb:**

```
chance(tier) = min( cap(tier),  climb_rate(tier) × climbs_elapsed )
climbs_elapsed = (now − last_eruption_tick(tier)) / climb_chunk(tier)
```

- **`climb_chunk` is a chunk of world-time, per tier — NOT a beat/message.** Keyed to world-time so a
  chatty player firing ten quick lines does not accelerate the world (§7b). A 3-day journey is thousands
  of climbs → saturates to the cap almost instantly (that is *why* long trips are very likely to erupt);
  three quick tavern lines is ~a minute → a climb or two → barely moves.
- **`cap` (default ~70%, configurable per tier) is a separate dial from `climb_rate`.** The cap sets the
  ceiling; the rate sets how fast you reach it. **Nothing is ever a guaranteed eruption** and there is
  **no hard maximum quiet stretch** — the road can always stay quiet a bit longer, just increasingly
  unlikely. (This **supersedes** the earlier "three days near-guarantees an eruption" wording:
  now *very likely, never guaranteed* — the chance stacks slot-over-slot across a long span, ~96% that
  *something* happened within three saturated slots, but never forced.)

**The roll is deterministic → replayable, never re-rolled, storage-free.** Seed the draw from
`(world_id, tick, tier, last_eruption_tick)` (+ optional per-world salt); fire if the draw is under
`chance(tier)`. Replay recomputes the identical result with nothing to persist. Eruptions are canon
events (fires are already logged); the debug trace recomputes the full roll math for any beat.

**No tension-damping.** Disruption at the worst moment is the feature (§5) — accrual is pure elapsed time;
a tense scene never lowers the odds.

**Config surface (per-world data, never hardcoded):** per-tier `climb_rate`, per-tier `climb_chunk`,
per-tier (or global) `cap`, a **master world-intensity / off switch**, and Unit 1's class→seconds mapping.
The `world_pressure.accrued` column the foundations stubbed becomes vestigial under the derived model —
repurposed as a cache or dropped in the migration.

**Interface:** `fn_pressure_chance(p_world_id, p_tier, p_now)` (derives chance from canon + config) and a
deterministic roll in the world's-turn composer; on fire, no state write is required beyond the eruption
event itself (last-eruption tick is derived from it).

---

## Unit 5 — The World Actor seat (spontaneity LLM-side, 7b)

**What it does.** When a tier fires, authors the intrusion. The **one seat that looks at the whole world**,
not the scene.

**Payload — WORLD-scope.** A bounded world slice (the world-granularity sibling of `gather_slice`): the
pending-events ledger, presence (who-and-where the NPCs are), location/region and faction state, recent
world-level events — **plus the current scene**, so it *can* aim at you if it wants.

**Output — the same six-event vocabulary as everyone else**, attributed to world entities (an arriving
patrol, the alarm bell, the rain), **constrained to the drawn size** (the engine hands it "author a
SMALL/MEDIUM/LARGE intrusion" — a validated enum, like the tension tiers). It enters the **same pipeline**
— adjudicated/committed, **no bypass**. Uniquely, it is the **presence-boundary mover**: it may pull a
*non-present* NPC into the scene — a power no other seat has.

**v1 scope = eruptions manifest perceivably at the current scene** (the alarm heard *here*, the patrol
arriving *here*). Still "unrelated to you" (not caused by or aimed at your action), just perceivable where
you are — which is the point, since the player must *experience* the disruption.

**Built to grow into off-scene ("B") without a rewrite — hard constraints:**
- **The World Actor authors a *truth* event with a location; it never bakes in who perceives it.** It
  authors *"an alarm sounds at «place»"*, never *"the tavern hears an alarm."* In v1 the location is
  always the current scene, so truth and what-you-see coincide.
- **Perception is a separate engine step — the shared fan-out (RULINGS-2026-07-24 §4), no bypass.** In v1
  the fan-out is trivial (you are in the scene, you perceive it). For B, the *only* changes are: the
  event's location may be elsewhere, and the fan-out gains distance/sense fidelity. **The seat's contract
  and the event schema do not change** — B is a fan-out upgrade.
- The payload is already world-scope, so the seat already *has* the reach; v1 only constrains its
  *output* to the scene. Lifting that one constraint = B.

**Interface:** replace `fakeWorldActorDriver` with a real prompt+driver; `fn_world_slice(p_world_id, p_scene)`
assembles the world-scope payload; a `world_actor.txt` rulebook; the drawn size passed as an input
constraint; output routed through the existing adjudicate/commit + fan-out. Depends on: Units 3–4, the
existing pipeline, the perception fan-out.

---

## Unit 6 — The world's-turn composer (the reusable seam)

**What it does.** The thin unit that *is* the world's turn — the seam the Journey reuses verbatim.

Given `(worldID, tickBefore, tickAfter, scene, trace)` it, in order:
1. fires due pending events (Unit 3) in `(tickBefore, tickAfter]`;
2. accrues + rolls each pressure tier (Unit 4);
3. if a tier fires, calls the World Actor (Unit 5) with the drawn size, commits the intrusion through the
   pipeline;
4. returns the **biggest magnitude that fired** so the caller applies the §5 cut (small → continue,
   medium/large → halt).

**The normal beat calls it once per attempt; the Journey calls it once per leg — same unit, zero changes.**
Ordering within a slot: pending events first (scheduled/deterministic), then the pressure roll; if a
medium/large pending event ends the beat, the pressure roll for that slot is skipped.

**This station builds NO Journey logic** — no progress accumulation, no thresholds, no "until" resolution
(that is entirely the Journey plan, mirroring how Station F kept the Journey out). Unit 6 only exposes the
clean world's-turn unit and the clock.

**Interface:** `runWorldTurn(ctx, worldID, tickBefore, tickAfter, scene, trace) → (firedMagnitude, err)`.

---

## Unit 7 — The trace (behind-the-curtain, debug-only)

**What it does.** Extends the existing `reasoning_log` (Grounded Reasoning Unit 3) with a **world's-turn**
section — same rules: debug-only (`DREAMCHAT_MODE=debug`), raw numbers, **absent (not null) when debug is
off**, pure capture with no new LLM call.

Captures, per slot:
- **Clock:** world-time this slot passed (`+6s`, `+3 days`).
- **Scheduled events:** which due events fired (or "none").
- **The pressure roll for all three pools, including the ones that did not fire** —
  `small 42% → rolled 0.71 → no · medium 8% → 0.90 → no · large 0.3% → no`. **The non-firing rolls are
  shown on purpose:** the pressure system is a set of dials you tune (cap, climb rate per pool), and you
  cannot tune what you cannot see. Debug-only, so verbosity is free.
- **The eruption:** if one fired — the drawn size, what the World Actor authored, and whether it cut the
  beat.

**Interface:** a `world_turn` block appended to `BeatTrace`, serialized under `reasoning_log` only when
debug; the play page renders it in the existing collapsible panel.

---

## Data flow (one beat, with the world's turn)

```
input → decompose → chain of [attempt | QUERY | UNRESOLVED]   (attempts now carry duration_class — Unit 1)
  → per attempt:
      NPC world-first (react to intent)          [existing]
      premise → route → commit                    [existing]
      advance clock (+= duration, non-zero now)   [Unit 1/2]
      runWorldTurn(tickBefore, tickAfter):         [Unit 6]
          fire due pending events                  [Unit 3]  → §5 cut on medium/large
          accrue + roll pressure per tier          [Unit 4]
          if fired: World Actor authors intrusion  [Unit 5]  → same pipeline, fan-out edge → §5 cut
  → narrate (includes world events)
  → response { narration, result, reasoning_log? } (reasoning_log gains the world's-turn block — Unit 7)
```

## Corpus grounding (every decision traces to a locked ruling)

- Two composing mechanisms (ledger + World Actor); pressure on world-time not player steps; tiered
  independent pools; logged/replayed roll — **RULINGS-2026-07-23 §7**.
- Magnitude cut (small runs on, medium/large end the beat); eruptions not contestable (no reaction beat);
  perception fan-out / per-receiver hook — **RULINGS-2026-07-24 §5, §4**.
- Same pipeline, no bypass; the perceivable edge; the presence-boundary mover — **§7b**.
- Over-budget = the Journey, here **expanded** to any sustained-until-threshold act — **RULINGS-2026-07-30 §2**.
- Decomposer single-job; `duration_class` is a parse-shape like `QUERY`, not a judgment —
  **RULINGS-2026-07-23 §4** + Grounded Reasoning (QUERY precedent).
- Trace debug-only, absent-not-null, pure capture — **Grounded Reasoning design, Unit 3**.

## Rulings this design makes (evolve the corpus — to fold into a RULINGS file at write-up)

1. **Every beat costs world-time** = sum of per-attempt durations with a floor; non-move duration is a
   decomposer-emitted `duration_class` enum (parse-shape), engine maps class→seconds via config; moves stay
   on physics.
2. **Sustained-until-threshold acts (wait / vigil / travel) are one mechanism = the Journey**, expanding
   RULINGS-2026-07-30 §2 beyond travel; time conditions ride the clock, event conditions ride the ledger.
3. **The eruption chance is capped (~70%, config)** — supersedes the earlier "three days near-guarantees an
   eruption" wording; now *very likely, never guaranteed*.

## Decisions resolved (the forks)

1. **World-time: every beat costs time** — sum-of-durations + floor; `duration_class` enum for non-moves;
   physics for moves; "until"/over-budget → the Journey.
2. **Ordering: the world's turn runs *after* each attempt** (time-driven), per-attempt not end-of-beat,
   distinct from the reactive NPC hook; §5 magnitude cut.
3. **Pressure: derived from canon** (time since last eruption), **capped linear climb** keyed to world-time,
   **deterministic roll**; no tension-damping; all knobs are per-world config.
4. **World Actor: world-scope payload, same-vocabulary output, presence-boundary mover; v1 manifests at the
   scene**, built to grow into off-scene B via author-truth + engine-fan-out.
5. **The world's turn is one reusable unit** the Journey calls per leg; no Journey logic in this station.
6. **Trace: a debug-only world's-turn section** showing the full per-pool roll (including quiet ones) for
   tuning.

## Testing approach

- **Unit 1:** a non-move beat advances world-time by the class→seconds mapping; an empty "I watch" beat
  advances by the floor; a move still advances by `fn_move_duration`; the enum's ceiling bounds the max.
- **Unit 3:** a pending event at tick T fires when a slot crosses T (not before); its payload commits as
  canon; a medium/large pending event ends the beat and discards the rest.
- **Unit 4:** chance derives from time-since-last-eruption and matches `cap`/`climb_rate`/`climb_chunk`
  config; the roll is deterministic (same inputs → same result across replays); pools are independent (a
  small fire does not reset medium/large); a big time-skip saturates to the cap; intensity=off yields no
  eruptions.
- **Unit 5:** the World Actor authors within the drawn size; output routes through adjudicate/commit +
  fan-out (no bypass); it can bring a non-present NPC in; the authored event carries a location and never
  encodes the perceived edge (the B-growth invariant).
- **Unit 6:** `runWorldTurn` returns the biggest fired magnitude; the beat cuts on medium/large; the same
  unit is callable standalone (the Journey seam).
- **Unit 7:** the world's-turn block appears only under debug; it captures the clock delta, fired events,
  and every pool's roll; non-debug responses carry no `reasoning_log` key.

## Build order & scope

- **Build order:** Unit 1 (world-time + duration_class) → Unit 3 (ledger crossing) → Unit 4 (pressure +
  roll) → Unit 5 (World Actor seat) → Unit 6 (world's-turn composer wiring the beat loop) → Unit 7 (trace).
  Unit 2 (ordering) is realized by where Unit 6 is called. Ledger and pressure are deterministic and land
  before the LLM seat, so the seam is testable before the World Actor is real.
- **Its own plan / body of work**, stacked on `feat/grounded-reasoning` (needs the fact-sheet/trace and the
  F contract functions). Branch off the GR tip.
- **Out of scope:** the Journey (its own plan — this station only exposes the seam); off-scene eruptions +
  distance/sense fan-out ("B" — deferred, grown into later); personality evolution and deep memory (their
  own stations).
