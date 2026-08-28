# art-and-image-seam · product

**Repo:** `dreamchat-world-backend` · **Cluster:** IP-2 · The seam, from the engine's side ·
**Parent bounded context:** World Engine

This file holds what the domain *means* — its job, its language, its product rules, what is
deliberately not built. `art-and-image-seam.tech.md` holds how it is built;
`art-and-image-seam.seams.md` holds what crosses its boundary. The platform's own side of this seam
is IP-1's package, in `dreamchat-Image-Platform`.

---

## What this domain is for

**One job: every picture the world shows exists because this domain noticed an empty slot and asked
a separate service to fill it.** The engine keeps the entity→image mapping; the platform keeps the
assets, provenance and cost (`D-3`). The founding trap is `docs/00_workspace/failure-log.md` row 40:
a world shipped with **no images**, because creation paths relied on someone remembering a second
call. The answer is `ADR-P021` — art is reconciled, never commissioned.

## Ubiquitous language

| Term | Means, precisely |
|---|---|
| **Slot** | A place a picture belongs: `(world, owner kind, owner, variant)`. Not an image — it holds the last known asset id or nothing, and nothing is the ordinary state. |
| **Reconciler** | The only creation path for art: ask what has no picture, fill exactly that. Per world, idempotent, indifferent to which path created the entity (`ADR-P021`). |
| **Style** | A named, addressable profile, never a per-request prompt — the platform's reuse cache keys on `style_profile_id` (`ADR-P023`). Five presets plus `custom:<prose>`; presets are rendering media, not genres (`GA-2`). |
| **Variant** | One of a closed CHECK set of emotion renders (`default` plus four) — an open column would let a typo'd variant vanish silently from every reader. |
| **Asset reference** | What a payload carries: asset id + a path back to this service (`image_ref/1`). Never a presigned URL — those expire in ~15 minutes and rot in any cache or log. |
| **Pull** | The only channel: store the job id, poll to a terminal status; webhooks are a latency hint the engine ignores entirely (`image:ADR-006` — the platform's own ADR series, prefixed). |

## What this domain is not

- **Not the platform.** Providers, storage, cost, retrieval-before-generation are IP-1's
  (`dreamchat-Image-Platform`); this side must not re-derive routing or asset storage (`D-3`).
- **Not canon.** An image is illustrative and creates no world truth; no event is written when a
  portrait appears (migration `20260808100005`'s own comment).
- **Not perception.** Generation deliberately reads authoritative state, and the image field is not
  perception-scoped — the receipts live in `tech.md` §The read path.
- **Not governance.** The envelope's seven fields cross this domain's wire, but what they *mean* —
  classification, `content_class` vocabulary, the `E-1` ordering — is CG-1's (see `seams.md`).

## Product rules — decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `ADR-P021` | Art is reconciled: genesis kicks, a ticker sweeps. **Do not add a commissioning call to a creation path.** | Failure-log row 40 recurs: a new path ships unillustrated worlds, or double-commissions. |
| `ADR-P023` | A style's look, label and prompt prose live in one module; a style is a named profile. | Surfaces disagree about what a world chose; an unaddressable style is never reused. |
| `D-3` | The platform never owns world truth; the mapping lives here or nobody has it. | The platform cannot answer "current asset for entity X" — losing the mapping loses the art. |
| `D-8` | Images run async; `image` is null until it is not. | Generation inside the genesis transaction turns a provider outage into a destroyed world. |
| `E-1`/`E-2` | Classification precedes any media request; the core classifies, the platform never decides policy. | Policy moves into a service built to never hold it. |
| `image:ADR-006` | Generation is async jobs, pulled. (The citation gate resolves the bare `ADR-006` suffix; the prefix is for the reader.) | Trusting pushes waits forever on transitions that deliberately never emit. |
| `GA-2` | Style keys name how a thing is drawn, never what it is about. | A preset named `cyberpunk` or `high-fantasy` smuggles genre into system vocabulary. |

## What is deliberately not built here

- **No manual commissioning in any creation flow.** The bounded endpoints exist for operators;
  nothing else may call them — a new creation path inherits art for free (`ADR-P021`).
- **No real-time generation during normal play.** Two rejected placements are recorded in
  `core/api/imagehandler.go`: on entity creation (an instant self-inflicted 429) and on first view
  ("a read that mutates is a read you cannot retry"). Generate visual range early, reuse in play.
- **No async channel for image arrival.** The reference is a projection field on payloads already
  read; a channel whose only message a read already carries is scaffolding (`20260808100006`).
- **No webhook consumer.** At-least-once latency hints; "a consumer that ignores webhooks entirely
  is still correct", and this one does (`core/api/imageclient.go` header).
- **No place-pack requests.** "Several images we did not ask for and an identity we have nothing
  to anchor" (`imageclient.go`).
- **No sprite-sheet pipeline.** The ten-expression sheet-and-slice design is the platform's PRD 08
  (`dreamchat-Image-Platform/prds/08_npc_expression_sprite_pipeline.md`); the engine's built
  vocabulary is four emotions and a bust contract (`tech.md`). Building the ten here re-decides
  the seam.
