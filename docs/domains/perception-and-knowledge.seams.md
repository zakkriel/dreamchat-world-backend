# perception-and-knowledge · seams

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-3 · The epistemic layer ·
**Parent bounded context:** World Engine

A seam belongs to two domains, so it gets its own file: one shared file is easier to keep symmetric
than two mirrored sections that drift. Each row declares an expectation — one side owns a fact, the
other consumes it and must not re-derive or re-decide it.

---

## What this domain consumes

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| consumes | **Canon & Time** (WE-1) | accepted event and speech judgment | Ordinary and ruled Communicated share `fn_apply_speech_perception` inside the commit. State changes still come from `state_mutation`; speech words and judgments come from the payload (`ADR-038`). |
| consumes | **Actions** | the closed event vocabulary | Only the closed set reaches this domain. Each event type needs its own arm in `generate_perceptions`; a type with no arm perceives nothing, silently. |
| consumes | **Space & Journey** (WE-5) | place-level binary co-presence, via `fn_actors_at(world, location)` | Place-level and binary. There is no sub-place geometry, so *"could they see it from there"* has no geometric answer today. Do not invent one here — that is a Space decision. |
| consumes | **Physics** | existing computed facts | `fn_speech_perception_facts` includes the existing `fn_fact_sheet` measurements. No hearing radius, occlusion rule, or physical override is invented; concealment remains outside this speech implementation (`ADR-P025`, `ADR-038`). |

## What this domain provides

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| provides | **The naming wall** (WE-4) | accepted associations and holder-specific heard words | Only explicit owner `actor_id` grants sourced `name_knowledge`. Descriptions are retained independently; related actor IDs add subject links but never names. The identity wall renders account prose before adding heard words; the word guard never grants identity (`ADR-038`). |
| provides | **Platform & Contracts** | every page, index and timeline read | No surface reads canon (`B-1`). Hidden truth is absent from the payload, not hidden by the UI. List the consumers with `grep -n 'fn_visible_perceptions\|fn_entity_visible' core/db/schema.sql` rather than counting them. |
| provides | **Play Loop** (WE-7) | `fn_unheard_names` and `fn_perceived_speech` | The output guard and quote evidence use visible holder-specific speech, never canonical source words merely attached to the same event. |
| provides | **NPC Cognition** (WE-8) | committed holder-specific perceptions | Pre-speech interruption sees only a speaking cue. Post-speech cognition reads accepted records; a missed event supplies no words or addressed metadata to that NPC (`ADR-038`). |
| provides | **Social & Relationships** (WE-9) | `[INFER]` perceived interactions | Relationship state should derive from what was perceived, not from what happened. **Unstated anywhere** — see "seams that do not exist" below. |
| provides | **Art & Assets** | the asset *reference*, not the asset | **The asymmetry is deliberate and documented at `core/api/imagehandler.go:669-672`:** generation reads authoritative `*_state`, not perception — *"a picture is of the THING, not of anyone's opinion of it, and the prompt goes to a private service, never to a player."* `B-1` governs what reaches the frontend, and what reaches the frontend is an asset id and a path, through perception-bound pages. **An agent "fixing" generation to read perception would be breaking a documented decision.** |

## The seams that do not exist

Name them, because this is the section an agent will otherwise improvise into.

- **Concealment.** No visibility signal and no within-place geometry exist anywhere in the schema.
  `ADR-P025` routes it to Physics and its own Consequences say it is blocked in practice until Physics
  exists as a domain. `core/db/migrations/20260825130000_object_relocated_witnesses.sql:432-437`
  moved the decision to the caller (the event names who saw it) rather than answering it. Do not
  build an occlusion answer on this side of the seam.
- **Social & Relationships.** The one seam still inferred. Does relationship state derive from what
  was *perceived* or from what *happened*? `B-2` suggests the former; nothing states it; the surface
  is deliberately absent (`product.md`, "What is deliberately not built"). An agent hitting this is
  deciding something new and must say so.
