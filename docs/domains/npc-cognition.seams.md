# npc-cognition · seams

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-8 · NPC cognition and minds ·
**Parent bounded context:** World Engine

A seam belongs to two domains; each row declares an expectation — one side owns a fact, the other
consumes it and must not re-derive or re-decide it.

---

## What this domain consumes

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| consumes | **Perception & Knowledge** (WE-3) | perception records, via the three lookups (`tech.md` §The lookups) | WE-3 owns who knows what and guarantees about-ness links (`ADR-035`); cognition intersects ids and never re-decides who perceived. The perception *is* the trigger: cognition fires when an actor perceives, never on a timer (`B-11`) — WE-3's package states the same row from its side. |
| consumes | **Play Loop** (WE-7) | the invocation and the imminent attempt (the wind-up, not committed history only) | The loop calls `worldFirst` once per action and commits every decision through the player's own pipeline (`D-1`, no bypass). `worldFirst`/`applyNPCDecisions` live in the loop's file (`core/api/orchestrator.go`) — this domain owns their behaviour toward minds, never the beat sequencing around them. |
| consumes | **The naming wall** (WE-4) | viewer-relative display labels, per seat | The wall owns `fn_display_name`/`fn_batch_display_name` and applies its own exclusion: name-knowledge never enters the private lookups — *"a name is an identity substrate, never a secret"* (migration `20260726100001…`, grep the phrase). Batch labels are the every-mind-agrees intersection; an isolated NPC reads the room as *she* knows it. Cognition never re-derives a label. |
| consumes | **Space & grounded reasoning** (WE-5) | the COMPUTED FACTS block — `fn_fact_sheet`, computed per seat's viewer | Engine-measured truth (distance, duration, reachability) the minds reason FROM and may not contradict. Perception-scoped, so the sheet cannot carry a secret into the batch (`orchestrator.go`, grep `truth_side=FALSE`). Cognition never computes geometry. |
| consumes | **Seats & the LLM bridge** (WE-13) | two bound drivers: `cognition_batch`, `cognition_isolated` | Capability-floored, swappable per `D-13`; the Go validator re-checks the schema-leashed output regardless of bound model. |
| consumes | **World genesis** (WE-10) | authored minds: `personality_core` + `trait_provenance` rows | Genesis writes cores with provenance (`worldgenesiscommit.go`, grep `writeMinds`) and may lag the engine — a missing core degrades to a name-only mind, never a failed beat. Cognition never authors a core. |

## What this domain provides

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| provides | **Play Loop** (WE-7) | decisions: `none` \| `commit` \| `telegraph`, typed as the same six-type attempts the player uses | Proposals only (`D-1`): the loop's gate rules, refuses, converts; a refusal must stay an ordinary answer (the livelock trap, `tech.md`). A telegraph is the loop's held-outcome input; depth-1 is the loop's cap to enforce, not this domain's. |
| provides | **Living World** (WE-12) | a boundary, not data | The World Actor is **not** a cognition seat: it is the one omniscient seat, reasons over the world, fires on clock-driven pressure; cognition seats reason over a scene and fire on perception (`B-11`). Neither may borrow the other's scope. Agreed with WE-12's writer 2026-08-27. |

## The seams that do not exist

- **No relationship input.** WE-9 models stance; nothing carries it into a cognition prompt —
  verified against `cognitionprompt.go`'s section list 2026-08-27. `[INFER]` when it lands it
  should arrive as the holder's *perceived* stance, not the logged edge (`B-2` pulls that way), but
  nothing states it. An agent hitting this is deciding something new.
- **The per-mind situation section.** The starved-prompt finding: pursuit/opposition authored by
  WE-11's contract has no reader here (`tech.md` §Open questions 1). Do not improvise fields into
  the prompt piecemeal — three review seats already cut exactly that (*"three leaves competing for
  one unbuilt reader"*, `digest/S07a…` §Topic 15).
- **Personality evolution.** `trait_pool` is a socket with no seam: no event magnitude crosses from
  the loop, no core change flows back. Filling it is a station, not a fix (`product.md` §Not built).
