# Concepts as knowledge

**Date:** 2026-09-02 · **Status:** design, settled by a sparring session with the founder. Concept
registration shipped on `engine/concepts-and-factions-are-entities`: a concept is now an
`entity_registry` row carrying its truth. Positions, grades, and the roll (§3–§5 below) are not
built and are deferred to `SPEC-051`.
**Scope: MVP ONLY.** This is the minimum that lets a world's ideas exist and reach the models that
write and run it. The full concept-and-knowledge system is deliberately future work — see `SPEC-051`.

**What this closes.** The fill authors ideas a world argues over and the engine throws them away. This
document says what an idea is, what it means for a character to hold one, and how both reach the
models. It adds three quantities and one law. It adds no subsystem.

---

## 1. The problem, measured

The fill authors `concepts` and `factions` in their own content waves. When this was measured on the
committed Los Andantes world (`464550bd-7815-4ce7-9d2d-0835d1e5d09f`), **neither reached the engine**.
Concepts now do; factions still do not:

| the document holds | count | reaches the engine as |
|---|---|---|
| `cast` | 29 | 29 `actor` |
| `places` | 10 | 11 `location` |
| `objects` + `ways` | 6 + 9 | 15 `artifact` (ways become portals) |
| `history` | 8 | `canon_event` + participants + perceptions |
| **`factions`** | **6** | **nothing** |
| **`concepts`** | **7** | 7 `concept` entities carrying their truth (`entity_registry.descriptor`) |

`factions` still appears nowhere in `worldgenesiscommit.go`; `concepts` now does, at line 363, where
each concept registers as an `entity_kind='concept'` row carrying `what_it_is` as its descriptor. Six
named things — the factions — are still discarded per world, and with them their reference graph:
every cast member carries `belongs_to`, so the engine sees twenty-nine unaffiliated people; factions
carry `seat`, `controls`, `publishes`, `buries`. A concept's `contested` and `taught_by` axes are
still discarded — only its truth registers.

**The history weaver already gets each concept's meaning, and this document originally claimed it
did not.** Corrected 2026-09-04 after the claim was implemented and reverted. `canonSchedule` passes
`Concepts: conceptNames(doc)`, which reads like a bare-names payload and is not one: `scope.Concepts`
is a **selector**, matched by exact string equality in `buildWorldFillPrompt` (`want(scope.Concepts,
strings.TrimSpace(c.CanonicalName))`), and the renderer beside it already emits

```
- concept "Auscultation"
    is: The craft of reading a beast's health through its deep pulse.
    contested: Whether the monthly report tells the full truth.
```

So the weaver receives the truth **and** the axis of dispute already. Changing the selector to carry
joined `"name — meaning"` lines made every concept fail the equality check and dropped all of them from
the prompt — the exact opposite of the intent, silently, with no log and no failing test. It also
recreated a format this repo removed after a live failure: the prompt marker at `worldidentity.go:78`
reads *"Cross-reference these by the EXACT string inside the quotes and nothing else — never the
descriptor, never the quotes, never the two joined."*

**What is actually missing is only the second half:** after genesis nothing survives, so the models
that run *play* never hear of a concept, and no character can hold a position on one.

Nothing is broken in play today, and that is worth saying plainly: an NPC's knowledge reaches a seat
as **prose**, and prose can already say "Del Vas thinks the pulse was regular". What cannot be done is
ask the world a question — who holds this, who differs, what does the Colegio publish — because loose
text is not attached to anything.

---

## 2. Why this is not a new subsystem

Two register rules already govern it:

- **B-6** — *"Contradiction lives in perception, never in canon. Perception-level contradictions are
  preserved. Canon is never contradictory."* One truth, many conflicting accounts of it. That is this
  design, already law.
- **B-5** — *"Append-only time; canon and perception are forward-moving; no mutable `updated_at`
  domain fields."* Accounts are never overwritten and a character's hold on one has a validity window.

The founder's framing was that a concept is **a type of knowledge**, not a thing bent into the entity
model. The engine already works that way: `name_knowledge (holder_id, entity_id, name, learned_tick,
source_event_id)` is a separate table for a separate kind of knowing. The person exists as an entity;
knowing their name is its own row. Both, not either.

One hard fact forces the same split here: **`perception_record.source_event_id` is `NOT NULL`.** A
perception cannot exist without an event. Del Vas knowing Auscultación after twenty years has no single
event to point at, so this knowledge cannot be a perception. It is the mirror of `SPEC-040` (canon with
no perceptions): knowledge with no event.

---

## 3. Three quantities

**The truth.** What the idea actually is. Authored at genesis as **world identity, not canon** — it did
not enter the world through an event, it is part of how the world works, like the condition and the
bargain. One per concept. **Never spoken.** No character ever receives it as text.

**A position.** One written account of the truth, *shared*: written once and pointed at by many
characters. Not one row per believer.

**A grade.** How fluently a given character holds a given position. It is **not** closeness to truth —
that idea was raised and discarded, because truth is permanently obscured. A character can be right and
will never know it. Grade is investment: how reliably they can bring the position to bear, and
therefore how likely they are to fail at it.

```
Concept truth  1 ──┬── Position A ──── Characters {Del Vas, Onn, Lur}
   (identity,      └── Position B ──── Characters {Marbán, Ciso}
    never spoken)
```

**Divergence forks.** When two characters holding one position begin to differ, a **new** position is
created pointing at the same concept. The old one survives for whoever still holds it. A fork is a
**canon event** — someone realised something, and that is an action.

**The holders of a position are a plain list, not a `group` entity.** Everyone who happens to believe
something is not an organisation and must not be able to act like one.

**A latent silent-drop, unreachable today.** `apply_mutation`
(`core/db/migrations/20260610090006_apply_mutation_and_triggers.sql:11-32`) dispatches
`IF actor / ELSIF location / ELSIF artifact / ELSIF relationship` with no `ELSE`.
`apply_attribute_writes` (`20260724100002_apply_ruled_event.sql:255-268`) only `CONTINUE`s when the
kind is NULL, so a concept target would write a `state_mutation` row with `status='applied'` (the
column default), the trigger would fire, the chain would fall through every branch, and nothing
would be projected — while `core/api/orchestrator.go:1667`'s guard compares row *count*, not effect,
so it would pass. Ledger says applied; nothing applied. It is unreachable today only because a
concept has no state row and no location, so it never enters a slice (`gather_slice`), never gets
whitelisted by `core/api/verdict.go:142`, and can never be named as an `attribute_write` target — a
concept is not rejected, it is unnameable. No gate is added for this; it is recorded so the deferred
`SPEC-051` work does not walk into it blind.

**The ten-dispatch-site review, recorded.** Every place a concept's `entity_kind` reaches a branch
that assumes actor/location/artifact was checked before this document was approved. Three classes:

1. **Kind-equality filters** (~30 sites, e.g. `core/api/artcommission.go:96,105,123`,
   `core/api/imagehandler.go:373,781,797`, the name-token wall) — a concept fails them all; safe by
   construction.
2. **CASE dispatch with a fallback** (`gather_slice`, `core/api/journey.go:288-292` via `COALESCE`)
   — safe.
3. **CASE/IF with no fallback** (`apply_mutation`, `fn_target_position`, `fn_distance`, mint
   persistence) — safe only because a concept cannot reach them; see the silent-drop above.

Two unfiltered registry enumerations were also cleared: `fn_unearned_names` / `fn_viewer_text` do
not leak the truth only because `fn_display_name` does not read `entity_registry.descriptor` — an
accident of which column it reads, not a designed boundary (see the caution on that column in
`core/api/worldgenesiscommit.go`, above `registerEntities`) — and `fn_names_in_text` has no kind
filter, so an NPC speaking a concept's name creates a `name_knowledge` row for a concept entity
(benign, but a new undocumented row type).

---

## 4. Two axes that must never be conflated

| axis | measures | who sees it | affects |
|---|---|---|---|
| position ↔ truth | how complete the account is | nobody, ever | what is *possible* |
| character ↔ position | how fluently it is held | rendered as prose | how *reliably* it is managed |

The founder's example settles it. Pyromancy's truth is *elemental power, focused*. One branch says
emotion-driven, another says rune-driven. **Both cast fire. Both work.** So a position is not false, it
is **partial** — a working route described incompletely. A shaky apprentice holding a perfectly good
branch still fizzles. Different axis.

**Consequence: nothing stores "is this wrong."** There is no flag. There is a truth and there are
positions, and whether a position is complete is something the weaver resolves and never says out loud.
A stored wrongness flag would eventually reach a prompt and the irony would collapse.

---

## 5. One law

Reusing `core/api/pressure.go`, which already does exactly this for world pressure:

```
chance = f(grade)
roll   = hash(world, tick, holder, position)      -- pure; never math/rand, never wall-clock
fired  = roll < chance
```

`pressure.go` states the constraint: *"replay must reproduce the identical result byte-for-byte"*
(invariant **I-1**). So the grade must be committed data and the same beat replayed must fumble
identically. The roll is **recomputed, never stored**.

**Two outcomes, by degree.** Recall thins first; application fails at the bottom.

1. **Recall** — the position is not brought to mind this beat, or arrives thinner.
2. **Application** — it is brought to mind, acted on, and does not work. The auscultation reads
   nothing; the fire does not catch.

Reaching for a rival branch's answer was considered and **rejected**: it is a third mechanic for no
gain.

### What follows without any new rule

**Does the character know they fumbled?** Not a setting. It is whether the action left something
observable. Fire that fails to catch is visible, so they know. A misread silence produces nothing, so
they do not. Pure theory knowledge, no. Knowledge that takes shape and can be seen, yes.

**Is a fumble canon?** Everything that is action is canon (**ADR-010**, **C-5**, and **D-1** for how it
gets there):

- a fumbled **application** is an action → **an event**
- a fumbled **recall** is not an action → **nothing is written.** An absence in that beat, and the roll
  is recomputable anyway.

---

## 6. What reaches the models

This is the point of the work, and the half the engine currently fails.

**The history weaver gets the truth** — and it already does, via the renderer described in §1. It
always resolves the truth and does not speak about it. Nothing needs building here. What is missing is
the *positions*: it cannot yet be told that each character holds their own understanding, because no
character holds one. That is what will let it write a past in which people are wrong — the Colegio
publishing the recognised position, *hoping* it is the truth — and it arrives with the knowledge
table, not before.

A faction's published position is **just another position** — carrying authority, carrying no guarantee.

**A play seat gets prose, never numbers.** Not `depth: 4`. Instead:

> "Your Auscultación knowledge taught you how to read and understand the sounds and movements of the
> giant's heart."

and, in a specific moment:

> "Your knowledge of the topic tells you silence is death."

**The seat is never told its character is wrong.** The truth surfaces only when it clashes with the
position *and* the character infers a new understanding — which is a fork, which is an event.

---

## 7. Common knowledge is an overloaded holder, not a bug

**This section originally claimed a leak, and that claim was wrong.** It was corrected on
2026-09-02 when an implementer refused to make the failing tests pass and reported instead.

```sql
fn_visible_perceptions:
  WHERE pr.holder_id = p_viewer_id
     OR pr.holder_id IN (SELECT entity_id FROM entity_registry
                          WHERE entity_kind IN ('faction','group'))
```

That branch is **the common-knowledge implementation**, and common knowledge is a mandated
knowledge path — **B-2**: *"unknown facts enter a perspective's knowledge only through valid
in-world paths — observation, told, record, broadcast, inference, propagation, **or common
knowledge**"*, with the Glossary defining it as *"World facts the current perspective is presumed
to know without an explicit acquisition event."* The dev seed implements it by registering a
`faction` pseudo-entity literally named **"Common Knowledge"**
(`eeeeeeee-…-eeee`, `seed_mara_0A.sql:24`) and attributing public facts to it as holder. Removing
the branch broke **11 assertions across 7 pgTAP files**.

So the earlier claim that it is "dead code because no faction is ever registered" was false: a
faction *is* registered, on purpose, and the branch is load-bearing.

**The real defect is narrower and still real.** The mechanism cannot distinguish two different
things, because both are expressed the same way — `holder_id` pointing at a `faction`/`group`
entity:

- *common knowledge*, which every character is presumed to hold
- *a faction's private position*, which only its members should hold

There is no membership representation anywhere in the schema (`belongs_to` is parsed into
`genesisActor.BelongsTo` and never persisted), so the two cannot be told apart. The consequence
stands even though the diagnosis changed: **registering a real faction would publish its private
position to every character in the world**, because it would be indistinguishable from the
"Common Knowledge" holder.

**Superseded 2026-09-04 by a founder ruling, and the overload stops mattering.** Membership was never
going to grant knowledge: *"if belonging to a faction or group makes you automatically have that
knowledge… it sounds more like a character creator / validator to check what position does the
character have in that faction and assign knowledge accordingly."* `B-2` backs it — *belonging* is not
one of the valid knowledge paths; being **told** is. Knowledge is assigned by `standing` at creation,
joining or promotion, using `told` / `taught` / `granted`.

So a faction holding a *private* perception was never legitimate, and the only proper collective holder
is the ambient "Common Knowledge" path this branch already uses. **`fn_visible_perceptions` needs no
change, and factions become transcribable** — registering plus membership, with no visibility
consequence. A faction's `publishes` is public (an outsider knows the official line, which is what lets
it be a lie everyone repeats); its `buries` is assigned only to standings close enough to it. Full
ruling, reasoning and the remaining open questions: `SPEC-051` item 8.

## 8. Vocabulary

Acquisition modes map onto the existing `epistemic_type` set (`direct`, `shared`, `told`, `overheard`,
`public`, `rumor`, `inference`, `mistaken`, `confirmed`, `disputed`):

| mode | existing | note |
|---|---|---|
| experienced | `direct` | as-is |
| inferred | `inference` | as-is — and this is what a fork is |
| taught | — | `told` means *someone said a thing happened*. Training someone is not that. **Add `taught`.** |
| granted | — | transmitted memory or knowledge. Nothing covers it. **Add `granted`.** |

`mistaken` and `disputed` already existing is further evidence this design is not new ground.

---

## 9. What this MVP is not

Explicitly out of scope, recorded as **SPEC-051**:

- no skill tree, no numeric display, no player-facing mastery UI
- no concept-to-concept relationships (prerequisites, schools containing doctrines)
- no automatic detection of divergence — a fork is authored or inferred, never inferred by a scanner
- no teaching economy, no training time model
- no compendium surface for concepts yet
- no concept participation in events (`event_participant`'s closed set is left intact; about-ness is
  `perception_subject`, per **ADR-035**)

**Deliberately not built:** no validation gate, no belt check, no "knowledge subsystem", no test suite
proving a taxonomy. The founder's constraint, in his words: keep it clean as *speed, velocity and the
physics engine*. Three quantities, one law.

---

## 10. Law relied on

`B-5` append-only time · `B-6` contradiction lives in perception · `D-1` nothing mutates canon directly ·
`C-5` a beat may produce zero, one or several canonical changes · `ADR-006` invalidation never deletion ·
`ADR-010` mechanical actions write accepted events · `ADR-035` about-ness is an explicit junction ·
`I-1` replay determinism · `D-5` additions to the frozen set are ADR-gated, not code workarounds.

Any schema this needs goes through an engine ADR in `docs/law/02_world_state_adrs.md`, the route
`ADR-035` took. **No such ADR is written yet, and none should be until this document is approved.**
