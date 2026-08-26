# PRD: World Creation Depth — a landing contract that turns authored fiction into operative structure

> **Status:** Draft v3 (2026-08-20) | **Owner:** TBD
> **Supersedes:** v1 (rejected: reached into `buildCognitionPrompt`, the narrator and the interview
> seat for one feature; headline guarantee was a CI test). v2 (rejected by unanimous architecture
> review: its headline property `Operate`-non-empty failed three independent ways, and its norm
> delivery channel was unsound). This draft is built **from** those attacks; §2 records the kills so
> they cannot be re-proposed.
> **Amends:** `prd_world_creation.md` — extends, never re-litigates.
> **Provenance:** four-seat adversarial debate — two content rounds, veto audit, architecture round —
> in `docs/superpowers/debates/2026-08-20-world-creation-depth/`. Every claim below about current
> behaviour is `file:line`-verified by at least two independent seats.

## 1. Problem Statement

**Surface problem:** a brief that states a *system* is committed as *flavor*. "The world is heavily
ruled by a social caste called Alphas" implies collectives, asymmetric power, enforcement and
per-holder epistemics; the pipeline flattens all of it into prose nothing downstream reads.

**Architectural problem, which is the real one:** the commit path **accretes rather than composes**.
Each authored concept re-answers the same questions in bespoke code — what identity it mints, what
event grounds it, who comes to know it and by what path, what engine-read surface it writes, what
refuses it — spread across `registerEntities` (`worldgenesiscommit.go:285`), `writeNamingEvent`
(`:354`), `writeMinds`/`insertMind` (`:479`, `:510`), `writeHistory` (`:414`), `writeOpeningState`
(`:596`), `writeArrival` (`:768`) and one growing `validate()` (`worldgenesis.go:249-495`).

Two consequences, both already realised here:

1. **A concept can be authored and inert, and nothing structural prevents it.** `cast[].standing`
   (`world_genesis.v1.schema.json:130-134`) is schema-required (`:117`), refused when blank
   (`worldgenesis.go:310-311`), and written by no commit path. Same class: `arrival.why` labels a
   button (`worldgenesishandler.go:534,661`) and never persists (`worldgenesiscommit.go:201`);
   `places[].kind` is dropped while `objects[].kind` survives (`:677`).
2. **Provenance is re-implemented per stage, so one stage got it wrong.** A perception grounded on
   the `world_genesis` event renders as an entity's *name* (`fn_perceived_name`,
   `schema.sql:2584-2599`) — the live bug that printed an archivist's forgery scheme where her name
   belonged (`worldgenesiscommit.go:550-555`).

**The feature is the contract; depth is its first customer.** The measure of success is that concept
N+1 is one declaration, and that neither defect class above can recur.

**The general case, not the example.** A caste, a ship's rota, a debt falling due, a hive protocol
are the same object to this engine. A criterion only a hierarchical world can satisfy is itself a
GA-2/GA-3 violation.

## 2. What v2 got wrong (do not re-propose)

Recorded because each was verified, and because the failures constrain the design.

- **`Operate(item) []stateWrite` required non-empty — killed three ways.**
  (a) It excludes concepts that correctly write no state: backstory events get `canon_event` +
  `perception_record` and **zero** `state_mutation` (`20260813142100_world_templates.sql:141-167`
  vs `:274-300`), deliberately — *"AttributeChanged with no mutations"* (`worldgenesiscommit.go:404-407`).
  Only `writeOpeningState` (`:596`) and `writeArrival` (`:768`) insert state at all.
  (b) It is per-**concept** while the defect is per-**leaf**: a `cast` landing that never parses
  `standing` but writes `actor_state` passes it carrying the identical bug. v2's claim to replace the
  leaf test was false, and it retired the only check that caught the defect.
  (c) It is not checkable at registration: a slice return needs an item to call. "A program that does
  not build" was untrue.
- **`norms[]` → `personality_core.traits` entries — unsound channel.** Traits render raw
  (`sb.Write(m.Traits)`, `cognitionprompt.go:143-146`) with no naming-wall pass, so a norm sentence
  naming a person leaks an unearned name into every bound mind's prompt (I-3). Trait entries are
  `{value, manner}` with `value` from a strength class (`worldgenesiscommit.go:516-525`) — a norm has
  no strength, so the runner would invent a number the seat is forbidden to emit. And traits are
  subject to malleability and `trait_pool` threshold accrual (`schema.sql:3950-3956`), so a law would
  **decay per mind during play**. Worst, as design: if the law is who a mind *is*, there can be no
  defiant NPC, no corrupt official, no one who looks away — which is exactly where a rule becomes a
  story. **Obligation is not disposition.**
- **"One runner inside the existing single transaction" — factually wrong.** `commitArrival` is a
  separate, retryable transaction, possibly another process, rebuilding ids via `loadGenesisIDs`
  (`worldgenesiscommit.go:224-256`) and minting `newCast` late, grounded in the arrival event
  (`:134-146`).
- **"All eight concepts migrate with no escape hatches" — fails at design time on five.** `world` is
  the bootstrap that mints `worldID` before any landing can run (`:94`, `:124-130`); `history` and the
  naming event write no state; `cast`'s grounding event is *selected from `history`'s items*
  (`writeMinds:479-491`); `arrival` is cross-transaction with a guarded conditional update.
- **Per-item `Ground` breaks canon shape.** Places, ways, objects and every cast placement share
  **one** scene-genesis event with order carried by `beat_seq` (`:577-604`). Per-item event specs
  would mint N events and fail any byte-identical comparison.

## 3. The contract

Two halves, because the v2 failure was conflating them: a **static declaration** checked at
registration, and an **apply** step that runs.

```go
// Static. Read at registration. No items, no database, no runtime.
type Declaration struct {
    Concept   string          // "collectives"
    Consumes  []LeafPath      // every schema leaf this concept claims
    Mints     []EntityKind    // may be empty
    Event     EventMode       // creates(spec) | references(concept, selector) | shares(key) | none
    Readers   []Reader        // REQUIRED, non-empty — see R1
    Phase     Phase           // content | arrival
    DependsOn []string        // read and write dependencies, both
}

// Dynamic. Runs inside the phase's transaction.
type Landing interface {
    Declare() Declaration
    Parse(*genesisDoc) []item
    Apply(ctx, scope, resolver, []item) error   // resolver exposes other concepts' minted ids
    Refuse(item, resolver) error
}
```

**R1 — a `Reader` is a sum type, so "does this reach the engine" is answerable statically without
excluding honest concepts:**

```
Reader = state(path)              // e.g. actor_state.attrs.location_id
       | perception(holderRule)   // who comes to know it
       | referenced(concept)      // another landing grounds on this one's event
```

`history` declares `perception + referenced(cast)` and passes **honestly** — no dummy state write.
`collectives` declares `state`. A declaration with zero readers fails registration.

**R2 — leaf coverage is the structural property, and it is what kills `standing`.** Registration
computes `⋃ Consumes` across all declarations and diffs it against every non-numeric leaf of
`world_genesis/N`. An unclaimed leaf is a **registration failure naming the leaf**. This is a static
set-difference over schema × declarations — no items, no fixture, no runtime call. It is per-leaf,
which is where the defect lives, and it is the property v2 claimed and did not have.

**R3 — the runner owns the invariants, once.** It mints every uuid, assigns every tick and `beat_seq`,
resolves every class to its number, stamps `source_event_id` on every perception and asserts
`acquired_tick ≥` its event's tick. I-2 and I-9 then hold for every future concept, and the seat still
emits no uuid, tick, coordinate or number (baseline AC-7) because the contract gives it nowhere to put
one.

**R4 — grounding is a sum type, which enforces the archivist rule centrally.** `references(concept,
selector)` expresses `cast`-grounds-on-`history` honestly instead of hiding it in runner internals,
and the runner **refuses `references(world_genesis)` for any perception that is not a name** — the bug
at `worldgenesiscommit.go:550-555` becomes unavailable to every concept, not patched for one.

**R5 — `shares(key)` preserves canon shape.** Concepts landing on the scene-genesis event declare
`shares("scene_genesis")`; the runner mints one event and assigns `beat_seq` in dependency order.

**R6 — phases are explicit, matching the shipped shape.** `content` and `arrival` are separate
transactions; `arrival` is retryable and may run in another process. A landing declares its phase; the
runner does not pretend there is one transaction.

## 4. Acceptance Criteria — the contract

1. **Declaration/Apply split, with registration-time checks.** *AC: registration performs R1 (≥1
   reader) and R2 (leaf coverage) with no database and no items; a deliberately-inert test declaration
   and a deliberately-unclaimed test leaf each produce a named registration failure.*
2. **The runner owns ids, ticks, class resolution and provenance (R3), and enforces R4.** *AC: no
   landing contains a uuid, tick, coordinate or class→number call; a test landing attempting
   `references(world_genesis)` for a non-name perception is refused by the runner, not by a
   concept-specific check.*
3. **Tuning is owned and reviewable, not diffused.** R3 centralises class→number *resolution*; it must
   not centralise *game tuning*. The constants that decide whether a world feels big, tense or slow —
   notably `genesisPlaceCoords` ringing rooms at 0.6 of region radius **specifically so leaving a room
   can exceed a beat budget and become a journey** (`worldgenesiscommit.go:884-892`, the SPEC-030
   lesson), against `tension`→budget (`tension.go:28-45`) and movement speed — live in one named,
   commented place with that rationale attached. *AC: a reviewer can find every world-feel constant in
   one file; changing one is a reviewed design diff, never an incidental runner edit.*
4. **`world` is the bootstrap, declared as such.** It mints `worldID` before any landing runs; it is
   not a landing and does not pretend to be. *AC: the contract has exactly one documented
   non-landing, and it is `world`.*

## 5. Acceptance Criteria — depth as the first customer

Three new concepts, each one declaration. All optional: a brief implying none authors none at zero
added token cost, and the pipeline is unchanged.

5. **`collectives[]`.** `{descriptor, canonical_name, standing, legibility}`; `standing` free prose,
   `legibility` `marked|concealed`, no kind field ever. `cast[].belongs_to[]` join-keys against it
   exactly as `starts_in` does against `places` (`worldgenesis.go:318-319`) and replaces the dead
   `standing`, so position is stated once and referenced by members. `Mints: [group]` →
   `entity_registry` (legal today, no DDL, unused by genesis). *AC: a brief implying a collective
   yields one queryable group row and resolvable membership; one implying none yields zero rows.*
6. **`norms[]` — obligation, in its own channel, beside disposition not inside it.**
   `{canonical_name, stated, binds[]}`; `binds` may be empty, meaning everyone (a taboo, a tide).
   `Reader: state(personality_core.traits.norms)` — a **top-level key in the traits jsonb, a sibling
   of `speech_manner`**, which is already a top-level string rather than a trait object
   (`worldgenesiscommit.go:514-519`; template `20260813142100_world_templates.sql:201`). Consequences,
   each answering a v2 kill: no `value` is invented; the personality module never touches it, so law
   does not drift with malleability or pool to a threshold; a mind may break a law without being out
   of character, which is where the story is; and it renders through the existing prompt path with no
   new read path. The runner applies a naming-wall pass to norm text before render, because trait
   JSON is emitted raw (`cognitionprompt.go:143-146`) and a norm sentence may name a person.
   *AC: the law appears in the prompt of every mind it binds and no other, carries no magnitude, is
   unchanged by any personality-module update across a 20-beat run, and leaks no unearned name.*
7. **`near_future[]` — the world is already in motion.** One authored thing about to happen per world;
   `when` as a **class** the runner resolves to `fire_at_tick` (R3); payload by canonical name resolved
   at apply. `Reader: state(pending_event)`, read every clock crossing (`ledger.go:122-220`) and today
   written by nothing but three test inserts. Not a norm mechanism — but where a norm exists it is the
   cheapest demonstration, because the player watches it applied **to someone else**.
   *AC: within the baseline's 5-beat window (`prd_world_creation.md:22`) the player perceives one
   authored event they did not cause, in every world, law or no law.*
8. **Legibility is knowledge, not formatting.** `concealed` membership is expressed as what minds
   *know* — members hold it, others do not — so existing perception machinery hides it with no
   render-surface special case. Nothing walls a group's name today (`fn_display_name` never reads
   `entity_registry` descriptors, `schema.sql:1445-1453`; `fn_unearned_names` cannot return one,
   `:2937`), so display-rule concealment would leak through any other surface. *AC: two briefs
   differing only in marked/concealed produce different knowledge distributions, not different
   formatting; vacuous when no collective is authored.*

## 6. Acceptance Criteria — did it reach the game

The only victory conditions here that are not structural. A mechanism passing §4–§5 and failing these
is deleted.

9. **The law changes what minds do, or the concept is cut.** *AC: one norm-implying brief built with
   and without the `norms` landing produces differing NPC decisions within five beats — diffed, not
   asserted.*
10. **Someone can break it.** A law nobody can violate is scenery. *AC: across the eval runs, at least
    one authored world produces an NPC act that contravenes a bound norm, and the world responds —
    through a person, a lock, or a stated consequence.*
11. **Extraction is faithful across unlike shapes.** N≥20 planted briefs whose implied norms share no
    shape — an order, a debt, a forbidden place, a duty rota, a non-human protocol — plus a negative
    control implying no norm. 0% invented structure, <10% missed structure, control authors nothing
    extra. Sampled human audit per I-6 (`07_test_and_invariant_spec.md:26-27`), never a CI equality.
    **If the fixtures are all hierarchies, the harness is the ontology.**
12. **The user owns the inference before spend.** Entailed collectives, norms and near-future render
    as strikeable world-language statements before the genesis call — constitution, never plot; AC-7
    secrecy intact. Every statement travels back, accepted as affirmations and amendments verbatim, as
    `InterviewAnswer` rows into the ANSWERS block (`prompts/world_genesis.txt:2`,
    `worldgenesis.go:213-228`); a shown-but-unsent confirmation is consent over a document the user
    cannot read. Zero derived statements ⇒ no screen, no tap, identical single genesis call.

## 7. Migration — staged, and not a gate on shipping depth

v2 made "all eight concepts migrate, byte-identical" the gate. That gate fails at design time (§2) and
the byte-identical diff is itself a second project: the benchmark world is plpgsql with fixed uuids and
hand-placed coordinates (`20260813142100:100-101`) that never passes through a `genesisDoc`, so
round-tripping requires reverse-authoring a document first.

1. **Contract + runner** (AC-1…4). No concepts moved.
2. **New concepts land on it first** (AC-5…8) — none has a cross-concept grounding problem, so they
   prove the contract cheaply.
3. **Migrate the concepts that fit** — `places`, `ways`, `objects` via `shares("scene_genesis")`.
   Equivalence is asserted on *canon shape* (entities, events, seq order, perception provenance), not
   on a byte diff that would freeze known-bad tuning as correctness.
4. **The hard cases get a decision, not indefinite coexistence** — `cast` (grounding selected from
   `history`), `history` (perception-only), `arrival` (cross-transaction, guarded update, late
   `newCast`). Each either migrates under R4/R6 or is documented as a permanent non-landing with its
   reason. **This decision is dated, not deferred**; two conventions may not coexist unexplained.

## 8. Non-Goals

Unanimous cuts, with the deciding argument.

- **No social-structure identifier anywhere** — no `caste|rank|tier|faction|guild|hierarchy|authority`
  in code, prompt, schema key, fixture or test name. Includes ordering keys: a dominance DAG
  (`standing_over[]`) was proposed and **retracted by its own author** — a tidal world cannot fill it.
- **No numeric authority tier, no Tier-1 growth** (`tier1.go:3` grows only when code checks a key).
- **No `relationship_state` writes** — zero readers in `core/api`; the `[RELATIONSHIPS]` block
  (`06_context_assembly_spec.md:76,88`) is unrendered. Under R1 it is not a legal `Reader`, so this is
  now structural rather than a matter of taste. Re-entry: render the block.
- **No group-held `perception_record` at genesis.** Disproven at both ends: cognition reads only
  `fn_public_moment`/`fn_private_records`, both filtered `holder_id = ANY(p_present)`
  (`schema.sql:2679-2684`, `:2732`), and a group is never present, so the law reaches **no mind**;
  meanwhile the player's payload comes from `fn_visible_perceptions` (`beathandler.go:213-228`), whose
  group branch has no viewer filter, so every norm lands in the player's **first** payload. A codex
  screen in SQL.
- **No engine-side norm enforcement** — no sanction pass, no viewer-aware `fn_portal_permits`. A
  permission predicate must know what confers permission, the one thing the service may not learn.
  Access is a `locked` way whose key an authored person carries, plus a mind in the doorway
  (`cognition.txt:8`, `resolve.txt:6`).
- **No new tables, no DDL. No post-commit correction. No new interview question type. No rules/codex
  panel. No second authoring seat** (one genesis call, no repair loop — `worldgenesis.go:172-174`).
- **No threshold-accumulation mechanics** — no `threshold_ledger` exists.

## 9. Open Questions

1. **Does `Refuse(item, resolver)`'s resolver become the new god object?** Every cross-concept rule
   ("somebody is already in the arrival room", `worldgenesis.go:386-395`) wants to live there, and a
   resolver rich enough for eight concepts is the old accretion renamed. Bound it deliberately in
   step 1, or the contract rots from this seam.
2. **Does centralising class→number resolution create a single point of failure?** It trades ~40 named
   `refuse()` messages that point at cause for one path. AC-3 mitigates for tuning; error legibility
   needs its own answer.
3. **Do the hard cases (`cast`, `history`, `arrival`) drive a contract v2, or stay documented
   non-landings?** Answered empirically at migration step 4.
4. **Which SPEC entries does this file?** The landing contract, the `world_genesis/2` bump, the
   `pending_event` genesis writer, the playback turn. Assigned by the implementing chunk (D-5).

## 10. Expelled — real defects, not this feature

- **Refusal states no cause.** `gate_reject` renders *"NOTHING RESOLVED: the attempted action did not
  happen"* with no obstacle (`narrateprompt.go:227-228`); `halt_reason` reaches the client as a bare
  machine string. Own change, narrator surface — **worth shipping before this PRD**, it is a defect
  today.
- **The interview lies about why it stopped.** A seat error collapses to `{done:true}` 200 OK,
  byte-identical to a genuine "nothing worth asking" (`worldinterview.go:71-84`, swallowed at
  `worldgenesishandler.go:155-159`). One boolean, own change.
- **`fn_visible_perceptions`' group branch has no viewer filter** (`schema.sql:3080-3086`). Dormant
  because genesis mints no groups; AC-5 mints the first ones, making it live latent risk. Not exploited
  here (AC-6 uses the traits channel), but file it.

## 11. Final Product Rule

A created world must be indistinguishable, to the engine, from a world a human hand-wrote (baseline
§9). This PRD adds a second rule of the same kind: **an authored concept must be indistinguishable, to
the runner, from every other authored concept — and where it genuinely is not, the exception is
declared, named and dated rather than hidden in a function body.** If the next concept after depth
cannot be added as one declaration, this feature failed regardless of how well the caste system plays.
