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
| consumes | **Canon & Time** (WE-1) | a committed event | `generate_perceptions(event_id)` runs once per accepted event, inside the commit. It reads **`state_mutation`** for WHAT CHANGED — never the payload for state (which is `{}`); the payload carries only `spoken` words (SPEC-033), read by the Communicated arm of `generate_perceptions` AND by `fn_unearned_names` (the wall's vocabulary source, `schema.sql:3013`). One home for this fact: tech.md §The write path (SPEC-034's receipt lives there). |
| consumes | **Actions** | the closed event vocabulary | Only the closed set reaches this domain. Each event type needs its own arm in `generate_perceptions`; a type with no arm perceives nothing, silently. |
| consumes | **Space & Journey** (WE-5) | place-level binary co-presence, via `fn_actors_at(world, location)` | Place-level and binary. There is no sub-place geometry, so *"could they see it from there"* has no geometric answer today. Do not invent one here — that is a Space decision. |
| consumes | **Physics** | `physical` blocks — *was it impossible to perceive?* | Founder ruling, 2026-08-26: concealment is a Physics seam (`ADR-P025`). Physics answers whether sight is blocked; this domain decides who that means perceives. Perception never computes occlusion itself. **Nothing crosses this seam yet** — it is a dependency, not an implementation. |

## What this domain provides

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| provides | **WE-4 · The naming wall** | the holder and the source event | The wall owns `name_knowledge` and the substitution, and applies on every read path. **Perception never performs name substitution; the wall never decides who perceives.** WE-4's package is not written yet — until it is, its side of this row lives only here. |
| provides | **Platform & Contracts** | every page, index and timeline read | No surface reads canon (`B-1`). Hidden truth is absent from the payload, not hidden by the UI. List the consumers with `grep -n 'fn_visible_perceptions\|fn_entity_visible' core/db/schema.sql` rather than counting them. |
| provides | **Play Loop** (WE-7) | `fn_unearned_names` — consumed as `fn_viewer_text` on perception writes and as the `NamingWall` belt (`core/api/namingwall.go`) at beat emit | Seat output is walled before it reaches a player. A leak that reaches a player is a wall failure, not a model failure. |
| provides | **NPC Cognition** (WE-8) | the trigger | Cognition fires when an actor perceives something, never on a timer or a tick (`B-11`). Perception is the event; cognition is the consumer. |
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
