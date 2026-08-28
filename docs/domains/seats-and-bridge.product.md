# seats-and-bridge · product

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-13 · Seats, the LLM bridge, and cost ·
**Parent bounded context:** World Engine

This file holds what the domain *means* — its job, its language, its product rules, what is
deliberately not built. `seats-and-bridge.tech.md` holds how it is built;
`seats-and-bridge.seams.md` holds what crosses its boundary.

---

## What this domain is for

**One job: every model call the engine makes goes through a named seat, and the seat is accountable —
for what it may emit, what it must be capable of, what standard it is held to, and what it cost.**

The product reason it exists as its own domain: the engine's fiction is only as trustworthy as its
least-governed model call. A seat that can emit outside the closed vocabulary, censor a scene on its
own judgement, or spend money invisibly is not a quality problem — it is a different product. This
domain is where none of that is possible by construction.

## Ubiquitous language

| Term | Means, precisely |
|---|---|
| **Seat** | One LLM job with a name and a **capability floor** (`core/api/bridge.go`, `Seat`). Ten exist; `allSeatNames` in `seatconfig.go` is the list — do not count them here. |
| **Driver** | One bound model behind the `Driver` interface. Drivers live only in the bridge layer, never in `core/db` (`D-13`). |
| **Capability** | A **reported fact, never a config label** — `BindSeat` trusts only what the driver's `Capabilities()` returns (`bridge.go:21-22`; the lesson is credited to the Image Platform, `image:ADR-016`'s route-label defect). |
| **The leash** | Structured output constraining generation to a schema, so an illegal answer is unexpressible at token time — not post-hoc. |
| **The belt** | The consumer-side validator (`DecodeAndValidateChain` and kin). Defense in depth; owned by the seat's consumer, never by this domain. |
| **The latitude block** | Five paragraphs, byte-identical in every seat prompt, telling the seat what this app is (`ADR-P022`). |
| **Provider alias** | An operator-chosen name in the environment. No vendor name appears in code; the code knows wire *dialects* (`seatconfig.go:26-30`). |
| **Fake / stand-in** | A deterministic dev/CI driver, required to be **at least as strict** as the real one (`bridge_fakes.go:3`). |
| **Cost sink** | The per-beat spend ledger (`costsink.go`). Cost is an instrumented fact, not a billing estimate. |

## What this domain is not

- **Not the prompts' content.** What decompose or narrate is *told* belongs to the seat's consumer
  domain (play-loop, cognition, world-genesis). This domain owns only the latitude contract those
  files must carry, and the routing that delivers them.
- **Not the belts.** Acceptance of a seat's output is the consumer's decision (`beatsstream.go`,
  `beatseats.go`). The bridge transports; it never judges content.
- **Not canonization.** Every seat is quarantined: propose-only, regardless of bound model
  (`bridge.go:9-10`; `D-1`). The gate is the only canonization point.
- **Not the naming wall.** Seat output is walled downstream by WE-4; the bridge never substitutes a
  name (no wall reference exists in `bridge.go`/`anthropic.go`/`openaicompat.go`).

## Product rules — decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `E-2` | Mature fictional content exists in private worlds within legal bounds; the latitude block is the seats being *told* that standard, not a new grant (`ADR-P022`). | A seat left untold does what models do: fades, softens, disclaims — decisions nobody here made. |
| `ADR-P022` | One latitude block, byte-identical, in every seat, affirmative not prohibitive; **THE ONE FLOOR** is the only content limit. | A paraphrase gives two seats different thresholds and the prose contradicts itself mid-scene. |
| `ADR-P016` (via `ADR-P022`) | The private/public discriminator is **who can access the content** — payment never moves it. | Treating a paid private world as public-governed sanitises content the law already cleared. |
| `D-13` | Model choice affects output quality, never canon integrity or perception isolation — the quarantine holds per seat regardless of bound model. | Trusting a "better" model with authority is re-deciding the engine's shape. |

## What is deliberately not built here

- **No seat-config boot gate.** `ADR-P024` decides a seat's required configuration is part of its
  release, and names its own gap: *"The gate is worth building and is not built"* — the ADR records
  the sequence discipline (config applied → merged → deploy observed → path exercised) that stands in
  until it exists. Building it is welcome; pretending it exists is not.
- **No tool-call forcing in the openai-compat dialect.** Only the anthropic driver forces tool-use;
  `openaicompat.go` leashes via system message + `response_format`. Reason recorded in the
  provider-neutral design (digest S06 §Topic 27): *"an unexercised third branch on the seat path is
  the kind of surface that hides a defect until a live beat finds it."*
- **No `ObjectRelocated` in the dev decompose stand-in.** Its destination is the viewer, whose id is
  not in the candidate block; *"binding dest_id to a guess would make the carry path silently wrong,
  which is worse than absent"* (`bridge_fakes.go:414-417`).
