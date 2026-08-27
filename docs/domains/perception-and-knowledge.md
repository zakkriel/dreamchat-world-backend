# Domain — Perception & Knowledge

**Repo:** `dreamchat-world-backend` · **Kind:** core · **Status:** draft for founder review, 2026-08-26

> **Read this before you touch a perception file, not after.** It exists so you do not need the other
> 95 decisions in this project — only the ones that govern here. If your question is not answered by
> this document, that is a defect in this document; say so rather than deciding locally.
>
> **`[INFERRED]`** marks a claim I derived rather than found stated. Every one is a question for the
> founder, not a fact you may build on.

---

## 1. What this domain is about

**One job: who knows what, and how they came to know it.**

Perception is the layer between what is true and every surface a person sees. Canon records that a
thing happened. This domain decides *whether anyone noticed*, *what they would call it*, and *what
they are allowed to be told about it*. Nothing renders from canon; everything renders from here.

The product reason it exists as its own domain: **a world where everyone knows everything is not a
world, it is a database.** Secrets, mistaken beliefs, rumour, and the slow arrival of information are
the substance of the fiction. This domain is where that substance lives, and it is the reason a
DreamChat world cannot be replaced by a chatbot with a good memory.

### Ubiquitous language

Use these words with these meanings. They are not interchangeable, and the codebase enforces some of
the distinctions mechanically.

| Term | Means, precisely |
|---|---|
| **Perception** | One record of one holder's knowledge, acquired at a tick with an epistemic type. Never edited — invalidated or superseded (`ADR-006`). |
| **Holder** | The entity whose knowledge it is. A perception always belongs to someone. |
| **Subject** | The entity the perception is *about*. **Load-bearing:** `fn_entity_visible` reads subjects, so a perception that names the right holder and the wrong subjects is invisible. This caused SPEC-034. |
| **Epistemic type** | *How* it was acquired. A closed set of ten: `direct · shared · told · overheard · public · rumor · inference · mistaken · confirmed · disputed`. Adding one is not a local decision (see §4). |
| **Earned name** | A canonical name this holder has learned, recorded in `name_knowledge`. A name not earned must never reach them. |
| **The naming wall** | The mechanical substitution that replaces unearned canonical names with labels before text reaches a viewer. Not a convention — a function, applied on every read path. |
| **Projection** | Any derived read of perception: a page, an index, the timeline. Recomputable. |
| **Synthesis** | A summary derived **deterministically** from stored perception versions. Never regenerated on read (`B-9`) — regeneration drift is a bug, not a refresh. |
| **Perception version** | The append-only history of how one holder's understanding evolved. Older versions stay queryable: "how my understanding changed" is a product feature. |

### What this domain is *not*

- **Not whether something happened.** That is *Canon & Time*. If you find yourself deciding what is
  true, you are in the wrong domain.
- **Not what an NPC thinks or feels about what it knows.** That is *NPC Cognition*, which consumes
  perception and must never be confused with it. A perception is an acquisition; a belief is an
  appraisal.
- **Not the rendering.** The frontend owns presentation only (`D-7`).
- **Not relationship state.** That is *Social & Relationships*. `[INFERRED, and the last open seam]` —
  a relationship that changed because of something an actor never perceived would mean the world is
  reacting to information the character does not have, which contradicts `B-2`. So the direction looks
  forced: perception is upstream of relationship. But **nothing states it anywhere**, and `B-3` forbids
  the relationship UI in MVP, so this is a domain with a suppressed surface and an unstated contract.

---

## 2. How it connects — the seams

Domains here are not isolated. They connect through **declared expectations**: one side owns a fact,
the other consumes it and must not re-derive or re-decide it.

### What this domain consumes

| From | What crosses | The expectation |
|---|---|---|
| **Canon & Time** | a committed event | `generate_perceptions(event_id)` runs **once per accepted event, inside the commit**. It reads `state_mutation` for what changed — **never `canon_event.payload`**, which is `{}` on commit. Getting this wrong produces a fix that applies cleanly and does nothing (SPEC-034's receipt). |
| **Actions** | the event vocabulary | Only the closed set reaches this domain. Each event type needs its **own arm** in `generate_perceptions`; a type with no arm perceives **nothing, silently**. Three arms exist today: `Communicated`/`private_disclosure`, `move`/`ActorMoved`, `ObjectRelocated`. A new event type without an arm is the SPEC-034 failure repeating. |
| **Space & Journey** | co-presence, via `fn_actors_at(world, location)` | Place-level and **binary**. There is no sub-place geometry, so *"could they see it from there"* has **no geometric answer today**. Do not invent one here — that is a Space decision. |
| **Physics** | `physical` blocks — *was it impossible to perceive?* | **Founder ruling, 2026-08-26: concealment is a Physics seam.** Physics answers whether sight is blocked; this domain decides who that means perceives. **Perception never computes occlusion itself.** Nothing crosses this seam yet — it is a dependency, not an implementation: concealment is blocked on the Physics domain existing. |

### What this domain provides

| To | What crosses | The expectation |
|---|---|---|
| **Platform & Contracts** | every page, index and timeline read | **22 functions** consume perception (`fn_actor_page`, `fn_location_page`, `fn_artifact_page`, `fn_compendium_*`, `fn_timeline`, `fn_carrying`, …). No surface may read canon directly (`B-1`). Hidden truth is **absent from the payload**, not hidden by the UI. |
| **Play Loop** | `fn_viewer_text` at the emit boundary | Seat output is walled **before** it reaches a player. Narration validation *rejects and re-asks*; NPC telegraphs get deterministic substitution with no retry loop. A leak that reaches a player is a wall failure, not a model failure. |
| **NPC Cognition** | the trigger | Cognition fires **when an actor perceives something**, never on a timer or a tick (`B-11`). Perception is the event; cognition is the consumer. |
| **Social & Relationships** | `[INFERRED]` perceived interactions | Relationship state should derive from what was perceived, not from what happened. Unstated anywhere. |
| **Art & Assets** | the asset *reference*, not the asset | **Two-sided, and the asymmetry is deliberate — verified in `imagehandler.go:669-672`.** Generation reads `*_state` (authoritative), **not** perception, on purpose: *"a picture is of the THING, not of anyone's opinion of it, and the prompt goes to a private service, never to a player."* The perception gate applies where the **reference** enters a payload — `fn_image_ref` inside `fn_actor_page`, which is already perception-bound. So `B-1` is satisfied at the frontend boundary, where what crosses is an asset id and a path. **An agent "fixing" generation to read perception would be breaking a documented decision.** |

### The seam that does not exist and should

**Concealment.** There is no visibility or concealment signal anywhere in the schema — measured
2026-08-25. The flagship example in the PRDs is *"You saw Seren pass a sealed note to a cloaked
figure"* — a **witnessed** handover, which now works (SPEC-035). A **concealed** one does not. It would
need either a Physics seam (occlusion) or an Actions seam (the actor declares concealment), and
**neither is decided.** This is the largest open question in the domain.

---

## 3. The architecture you work on

### Storage

| Table | Shape and the part that matters |
|---|---|
| `perception_record` | `perception_id · world_id · holder_id · source_event_id · content · epistemic_type · sensory_mode · confidence · distortion_level · acquired_tick · valid_tick · invalid_tick · visibility_scope · importance` — note `invalid_tick`: perceptions are **invalidated, never deleted** (`ADR-006`). |
| `perception_subject` | `perception_id · entity_id · world_id` — the join that makes visibility work. **This is the table SPEC-034 was about.** |
| `name_knowledge` | `world_id · holder_id · entity_id · name · learned_tick · source_event_id` — who has earned which name, and from which event. |

### The write path

`apply_event` commits → `generate_perceptions(event_id)` → one arm per event type → inserts
`perception_record` + its `perception_subject` rows.

**The arms are the whole domain's surface area today — and `ADR-P025` shrinks them.** Under that
decision an arm's job becomes only *who could conceivably have perceived this*, and a chain of typed
blockers decides the rest. Until that migration lands, adding an event type means adding an arm; forgetting
means silence. Each arm decides three things: *who perceives*, *what subjects the perception names*,
and *what epistemic type it carries*.

### The read path

- **Visibility:** `fn_entity_visible(world, holder, entity)` — asks whether the holder holds a
  perception whose **subject** is that entity. `fn_visible_perceptions` for the set.
- **The wall:** `fn_unearned_names` → `fn_viewer_text` (substitution), plus `fn_names_in_text`,
  `fn_perceived_name`, `fn_display_name`, `fn_batch_display_name`, `fn_display_names_distinct`.
- **Projections:** the 22 consumers listed in §2.

### Where the code lives

`core/db/migrations/*perception*`, `*naming_wall*`, `*spoken_words*`, `*hearing_teaches*` ·
`core/api/perception*.go` `namingwall.go` `narration.go` · pgTAP `core/db/tests/`

---

## 4. Decisions already made — do not re-decide these

| Id | What it settles | Consequence if you ignore it |
|---|---|---|
| `B-1` | Every user-facing surface renders from the holder's perception; hidden truth is **absent from the payload**. | A UI-level filter is a `B-1` violation even if it looks correct. |
| `B-2` | Knowledge enters only through valid in-world paths — observation, told, record, broadcast. | Assigning knowledge directly is not a shortcut, it is a different product. |
| `B-6` | Contradiction lives in perception, **never** in canon. Preserve source, timing, uncertainty. | "Resolving" a contradiction in canon destroys the fiction's memory. |
| `B-7` | Knowledge transfer never copies memory — a propagated perception is a **new record** with its own epistemic type, teller and confidence. | Copying the row makes hearsay indistinguishable from witness. |
| `B-9` | Syntheses derive deterministically from stored versions. | Regeneration on reload is drift, and it is a bug. |
| `B-11` | Cognition is event-driven, never on a timer. | A polling loop reintroduces the determinism bug `SPEC-002` removed. |
| `I-3` | No hidden-canon leakage. **CI-enforced.** | The build fails. |
| `C-4` | Play mode shows the perceived world; only creator/debug may show authoritative state. | A debug surface leaking into play is a `B-1` breach with extra steps. |
| `ADR-006` | Three time axes; invalidation, never deletion. | A deleted perception erases the history the product sells. |
| `SPEC-034` | **LANDED.** A handover makes the object perceptible to the holders the event names — and the **object must be a subject**. | — |
| `SPEC-035` | **LANDED.** The event **names** its witnesses (`role_qualifier='witness'`); co-presence is necessary, not sufficient. Malformed input is refused, never dropped. | — |
| `ADR-P025` | **The shape of this whole domain.** Perception is a **pipeline of typed blockers**: an actor perceives unless something stopped them. Links may **block, never grant** — which is why order does not change the outcome. Blocks are typed `physical` (Physics, deterministic) or `attentional` (the resolve seat's judgement). Overrides are **named permissions declared at module setup**, never numeric ranks: keen senses may beat `attentional`, never `physical`. Read it before changing who perceives anything. |
| `SPEC-016` | **OPEN.** Per-attribute perceivability does not exist. | This blocks stats, HP, immateriality, mood, condition — every new actor attribute needs *"can another actor see this?"* and there is no answer yet. |

### What you may **not** decide alone

1. **Adding an epistemic type.** The set of ten is closed. A new one changes what the world can mean.
2. **Widening who perceives an event.** SPEC-035 set the precedent: this needed a founder ruling
   (*"holders and co-present as long as they can see it — just because they were there doesn't mean
   they saw it"*). Do not widen a perceiver set by inference.
3. **Anything that lets an unearned name into a payload.** Not a bug to be fixed later — a breach.
4. **Making perception a module.** It is permanently core (`D-2`, `ADR-W005`). Recurring request,
   settled answer.

---

## 5. Testing strategy — what excellence means here

A perception change is correct only when all four hold. This is stricter than the repo default, on
purpose: this domain's failures are **silent** — nothing errors, someone just cannot see something.

1. **Reproduced on the seed before it was written.** Show the defect on a real world first. A fix for
   a defect you never observed is a guess. SPEC-034's first draft read `ev.payload`, applied cleanly,
   passed `make migrate`, and produced **zero perceptions** — only the reproduction caught it.
2. **The subject invariant holds.** Every perception names the entities it is about. `fn_entity_visible`
   reads subjects, so this is what visibility *is*.
3. **The wall still covers every read path.** A new read path is a new place for a name to leak.
   `46_wall_clause_coverage_test.sql` exists because *coverage of the wall clause* was itself the
   defect — not the wall.
4. **Mutation-tested in both classes.** Code-path *and* input-shape.

### The gate

```bash
cd dreamchat-world-backend
make reset && make test                       # pgTAP; the perception suites are 25/40/42/43/44/46/97/121/122/123
make reset && (cd core/api && go test ./... -count=1)
```

`make reset` first, and `-count=1` always: the Go suite is **seed-dependent and not idempotent**, and
the cache will show you a stale pass.

### What a test here must defend, and what is ceremony

- **Defend:** a holder who should see something does; a holder who should not, does not; an unearned
  name does not appear in a payload; an invalidated perception stays queryable.
- **Ceremony:** asserting on seeded rows. The seeded carry edges were authored as **state**, so no
  perception rule can reach them — a suite asserting on them **passes with the arm deleted.** That is
  the definition of a vacuous test, and it is the easiest mistake to make in this domain.
- **Both mutant classes.** Seven code-path mutants were caught across two rounds while a **silent drop
  on malformed input** shipped anyway. For every field you read, answer: **absent · null · wrong type ·
  empty.**

---

## 6. What has already gone wrong here — receipts

| The trap | The receipt |
|---|---|
| **`canon_event.payload` is `{}` on commit.** The relocation lives in `state_mutation`. | SPEC-034: the first fix read the payload, applied cleanly, changed nothing. |
| **Naming the actors is not enough — name the object.** `fn_entity_visible` reads subjects. | SPEC-034: deleting that one line reddens 3 of 8 assertions. |
| **`fn_actors_at` reads `actor_state`.** An actor with no state row is **nowhere**, and every co-presence gate refuses them. | A fixture without `actor_state` reads as "the gate is broken" when it is "your fixture has no world." |
| **A witness is not a holder.** The exclusion is load-bearing: without it a party gets two perceptions of one event. | SPEC-035, mutation-tested. |
| **An invariant maintained by the harness is not an invariant.** The Go suite **backfilled missing subject rows before pgTAP looked**, from 8 call sites. | `failure-log #16`. If you add a repair helper, you may be deleting a guard. |
| **The Go suite poisons pgTAP.** `pressure_test.go` drains `world_eruption`. | Cost two false regression reports in one day. |
| **A 100%-caught mutation table means nothing about malformed input.** | `failure-log #45`. |

---

## Open questions for the founder

1. ~~**Concealment** — Physics or Actions seam?~~ **RULED 2026-08-26: Physics.** Recorded in §2.
   Consequence: concealment cannot be built until the Physics domain exists, and this domain must not
   grow its own occlusion logic in the meantime.
2. ~~**Art & Assets seam**~~ **RESOLVED by reading `imagehandler.go:669-672`** rather than by asking —
   the asymmetry is deliberate and now documented in §2. My inference was wrong in the direction that
   mattered: generation bypasses perception *on purpose*.
3. **Social & Relationships — the one seam still inferred.** Does relationship state derive from what
   was *perceived*, or from what *happened*? `B-2` suggests the former; nothing says it; `B-3` hides
   the surface. **This is the open question.**
4. **`SPEC-016`** — per-attribute perceivability blocks every new actor attribute (stats, HP,
   immateriality, mood, condition). Next, or deliberately parked?
5. Does **"how my understanding changed"** stay a product feature? It constrains everything here: it is
   why perceptions are versioned rather than updated, and it is currently unrendered.
