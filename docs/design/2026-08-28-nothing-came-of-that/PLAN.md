# Plan — a sentence that produces no actions must cost nothing, say so, and be counted

**Status:** LANDED 2026-08-28, plus the piece it deferred. Founder ruled the shape in conversation:
*"yeah stop it. but the user needs to be informed that nothing happened."*
**Recorded as:** `SPEC-037` in `docs/open-spec-items.md` — read that entry for the final state; it is
the durable record and this file is the plan that produced it.

> **The deferral below did not survive the day.** This plan says the journey branch is "left alone
> deliberately" until journeys-run-themselves lands. The founder's answer: *"we were not fixing a bug,
> we were building a feature, why didn't you do it? saying it out loud doesn't exempt you from doing
> it."* It was built the same day — `Orchestrator.runJourneyToCompletion` — and the Continue press,
> its route, its button and its halt copy were all deleted with it. Every paragraph below that reasons
> about "until journeys-run-themselves lands" is history, not standing guidance.

---

## What is true now

A player types a sentence. The parse turns it into a list of actions. Sometimes that list comes back
**empty**, and an empty list is the same thing the Continue button sends — so the engine cannot tell
the two apart.

With no journey running, an empty list means: a moment passes, **the world takes its turn** (NPCs may
act, scheduled events may fire), and the beat reports `completed`. The screen shows no message,
because completing is success. The player's sentence is discarded in silence and their turn is spent.

**What is NOT broken, and must stay working — verified 2026-08-28 by reading the code, not the docs:**

- **Waiting is already a first-class action** with both parameters it needs. A non-move attempt carries
  a `sustain` shape: `{"kind":"for","seconds":N}` for how long, or `until_at` / `until_attr` for what
  it waits on. These become `wait` / `watch` journeys with real thresholds, a per-world watch-horizon
  dial, and an honest ending when the condition never arrives (*"You waited, and it never came"*).
  **Waiting does not produce an empty list and is untouched by this round.**
- **Handing things over is already in the vocabulary** — `ObjectRelocated` (object → actor, location or
  container) and `OwnershipAccessChanged`. An earlier draft of this plan claimed give/take/drop was
  unbuilt; that was read off `bridge_fakes.go`, the **offline stand-in parser used when no API keys are
  present**, and was wrong about the product.
- **Questions-only beats** produce QUERY elements, not an empty list, and correctly pay the stillness
  floor. Untouched.

So the empty list has exactly two sources: the Continue press, and a sentence the parse made nothing of.

## What the spec gets wrong

`SPEC-037` frames this as a journey defect — a typed sentence advancing a journey leg. That is one
symptom. The same silence and the same stolen turn happen standing still in a tavern with no journey
anywhere near it. **This plan fixes the general case.** The journey branch is left alone deliberately:
the founder has ruled that journeys will run themselves and stop only on interruption or arrival, which
deletes the Continue press and the `len(chain) == 0` journey branch entirely. Writing careful new logic
there now would be writing code to be deleted.

**Consequence, stated rather than hidden:** until journeys-run-themselves lands, the journey symptom in
`SPEC-037` remains live. This round does not close `SPEC-037`; it closes the larger half of it.

## Decisions this relies on

Named in plain words. The barcode is for exact lookup only.

- **The AI proposes, the engine decides — nothing is written on the AI's say-so** `[D-1]`. A parse that
  yields no actions is a legitimate outcome of the proposal step. Converting "no proposal" into "a
  moment of stillness" is the engine inventing an action the player never made.
- **Every player-facing surface shows only what that player could know** `[B-1]`. The derivation record
  added below is analytics, never a player surface — and it holds only the player's own typed words and
  entity ids bound from the candidate list, which is already perception-bound to that viewer. It
  reveals nothing new to anyone.
- **The stillness floor exists so that deliberate inaction still costs time** (the Living World "instant
  floor", `orchestrator.go` tail). This round narrows *what counts as deliberate inaction*; it does not
  touch the floor itself.

## What changes

**1. Tell the engine which kind of press this was.**
`RunBeat` currently takes no parameter saying whether the player typed or pressed Continue. The
information exists at the edge — `beatsstream.go` already holds a `continuePress` boolean — and is
thrown away. Thread it through. This is the whole root cause: emptiness is overloaded, so nothing
downstream can distinguish the two meanings.

**2. A typed beat that parsed to nothing does nothing.**
No clock advance. No world's turn. Nothing committed. It reports a distinct stop-reason rather than
`completed`. The journey branch is untouched (see above), so this applies where no journey is active.

**3. The player is told.**
The frontend already shows a message for any stop-reason other than `completed`, and already carries a
**dead** entry, `bounce` — *"That didn't land as possible — say it differently"* — which the backend has
never emitted. Emit `bounce` and rewrite its text to something true:

> **"Nothing came of that. No time passed."**

Two facts, both certain: nothing happened, and it cost you nothing. It does **not** diagnose. We do not
know whether the sentence was unparseable, or unsupported, or simply produced nothing — so the message
must not claim to know. An in-world voice ("nothing here answers to that") is rejected on purpose: if
the real cause was a parse failure, that sentence teaches the player something false about the world,
which is the one thing this engine is built to prevent.

**4. Count it. This is the part that matters most.**
The engine already computes exactly the record needed and throws it away. `BeatTrace.Decompose` holds
one entry per parsed element — its type, the player's words, the ids it bound — but the whole trace is
debug-only and never persisted, so **zero** real beats produce one.

The trace is debug-gated because it carries truth-side material (the referee's reasoning, fact sheets
that can hold a secret). **`Decompose` carries none of that.** Persist that part, on every beat, into a
new analytics table — deliberately NOT into `transcript_entry`, because that is a published contract and
extending it would make this a cross-repo round for no benefit.

Per beat, record: the world, the viewer, the tick, the raw sentence, and what the parse produced —
including **the empty case, which is the entire point.** `NewBeatTrace` builds `Decompose` from the
chain, so an empty chain yields an empty list; the record must be written anyway, with the sentence.

That answers, by query rather than by argument:

- how many sentences produce nothing — **and what they said**
- how many produce ambiguity (`UNRESOLVED`) — where the world's names confuse players
- the spread across action types — what people actually try to do

**Deliberately not built:** a new vocabulary shape for "this world cannot do that." Founder's reasoning,
and it is right: *at best that is a try/catch, at worst a plaster that hides a need.* Decide it from the
data once there is some. If one verb dominates the unassigned pile, the answer is that verb, not an
error message.

## What could break

- **Deliberate stillness.** If "empty list" were treated as not-understood, a player waiting would be
  told they were incomprehensible. It is not: waiting produces a real attempt with a `sustain` shape.
  A test must prove this.
- **QUERY-only beats** must keep paying the floor.
- **The Continue press** must keep advancing a journey leg — it is not a typed beat.
- **Adding a stop-reason** the frontend does not know would show its fallback text. Handled by
  reusing `bounce`, which already exists there.

## How it is verified

Each of these must be watched going **red** before it counts, by reverting the change:

1. A typed sentence that parses to nothing: **clock does not advance, no world's turn runs**, stop-reason
   is `bounce`. Red against today's code.
2. A deliberate wait (`sustain` present) still becomes a wait journey and still costs time. Red if the
   new branch is widened to catch non-empty chains.
3. A QUERY-only beat still pays the stillness floor.
4. A Continue press on an active journey still advances exactly one leg.
5. A derivation row is written for **every** beat, including the empty one, carrying the raw sentence.
6. End to end from the wire: a sentence that parses to nothing returns `ticks_advanced: 0` and the
   player-facing message.

## Open, and not decided here

- **Splitting the reason.** Whether "couldn't parse", "no such verb" and "verb exists but not here" ever
  become distinct is a data question, deferred on purpose.
- **`SPEC-037` stays open** until journeys run themselves.
- **Retention** of the derivation table. Nobody has said how long this data lives, and it grows per beat.
