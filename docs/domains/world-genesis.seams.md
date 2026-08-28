# world-genesis · seams

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-10 · World genesis and world creation ·
**Parent bounded context:** World Engine

A seam belongs to two domains, so it gets its own file. Each row declares an expectation — one side
owns a fact, the other consumes it and must not re-derive or re-decide it. Most neighbours' packages
are unwritten; until they exist, their side of a row lives only here.

---

## What this domain consumes

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| consumes | **Canon & Time** (WE-1 / canon-recording) | the right to write world truth at creation | Genesis writes with `origin='fast_path'` — the ONE documented `D-1` exception, existing because the actors an event would reference do not exist yet. Everything after creation goes through `apply_event` / `apply_ruled_event`. Genesis never grows a second bypass, and nothing else ever uses `fast_path`. |
| consumes | **Seats & the LLM bridge** (WE-13) | the per-seat driver, budget, and config | Every genesis call is driven and budgeted per seat (`E-1`); prompts carry the byte-identical latitude block (`ADR-P022`); a seat's config exists in the environment before the merge that needs it (`ADR-P024`). Genesis never hardcodes a model name or SDK. |
| consumes | **Contracts & Platform** | schema publishing mechanics | SPEC-011 (a published schema has a captured payload), the five-artifact pin rule, CI. Genesis owns what its shapes *say*; the mechanics of publishing are not re-derived here. |
| consumes | **Art & Assets** | `ResolveArtStyle` | A style is a module of named profiles (`ADR-P023`); genesis validates the key early (refusal before spend) and never learns what a style looks like. |

## What this domain provides

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| provides | **Canon & Time** (canon-recording) | the authored world: entities, places, coordinates, the genesis document | Whether any of it becomes *true in play* is canon-recording's; genesis does not decide post-creation truth. |
| provides | **Perception** (WE-3) | the authored truth | Genesis authors what IS; perception decides who may know it (`B-1`). Genesis never writes a perception rule and never decides visibility. |
| provides | **Art & Assets** | the kick, at the end of a build | Detached — the stream has already ended (`ADR-P021`). The reconciler, the ticker, the platform call, and the retry policy are the other side's; genesis fires once and forgets. |
| provides | **Space & Journey** (WE-5) | authored places and coordinates | Geometry, distance and duration are derived there (`fn_distance`, duration classes). Genesis never states a travel time. |
| provides | **Play Loop** (WE-7) | the world identity, for tier-3 minting | Content minted during play answers to the same identity that governed genesis (design §7.2: "available on request" is stage nine of the pipeline). HOW identity reaches the minting moment is design Q5 — open, see "seams that do not exist yet". |
| provides | **`dream-weaver-visuals`** | `world_genesis_frame/3`, `world_kickstart_turn/2`, `world_interview_turn/1`, `world_directory/2`, `world_refreshed/1` | Pinned by exact string equality; vendored byte-identically. A change here is a cross-repo round (`../harness/check.sh contract-drift` is the only gate that sees both sides). The frontend renders frames; it never decides a question or invents a field (`D-7`). |
| provides | **WE-11 · The world-model contract** | the filled document | Genesis fills the document; WE-11 owns what the document IS. Its package is unwritten and its schema has no machine representation — the blocker for genesis step 5, recorded at the design's close. |

## The seams that do not exist yet

Name them, because this is the section an agent will otherwise improvise into.

- **Identity → play-loop minting (design Q5).** Does identity travel inside the emitted document or
  beside it? Inside changes what the schema is; beside must be loaded everywhere the document is and
  can drift. Nothing states the answer. An agent wiring identity into play is deciding something new.
- **Enforcement of a world's own rules (`SPEC-036`).** Exist-kind rules are model-enforced at
  creation moments; happen-kind rules map to acts the play loop can check only when the engine
  recognises the act. No engine surface exists for either yet; a rule naming an unknown act is
  narrator guidance and genesis must mark it as such (design §9).
- **Auth in front of creation (B1).** `world.player_entity_id` landed; the model did not. Until it
  does, `POST /worlds` is unauthenticated and per-user isolation does not exist — now
  security-load-bearing (area dossier §3, SPEC-028's own note).
- **Genre-reference briefs (design Q3).** "Like Dune but underwater" collides with `GA-2`/`GA-3` and
  `prd_world_creation.md:177`. Refusing loses the most informative thing the author said; absorbing
  imports what the rules forbid. Unresolved, and it arrives on day one.
