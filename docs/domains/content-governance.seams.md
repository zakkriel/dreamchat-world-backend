# content-governance · seams

**Repo:** `dreamchat-world-backend` · **Cluster:** CG-1 · Private, public, and the latitude ·
**Parent bounded context:** Content Governance

Each row declares an expectation — one side owns a fact, the other consumes it and must not re-derive
or re-decide it. `content-governance.product.md` holds what the domain means;
`content-governance.tech.md` holds what is built.

---

## What this domain provides

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| provides | **image-service** (Image Platform, `dreamchat-Image-Platform`) | already-classified, already-authorized generation requests, as the seven-field governance envelope | The platform *"receives generation requests only after the core platform has classified and authorized the request"* (`ADR-P016:24`; `D-3`; `E-1`). Its gate **verifies and stores, never interprets** — `content_class` is opaque to it (`image:ADR-I002`). The platform never re-derives policy; this side never relies on the platform to catch a wrong classification — under its `log_only` default a bad envelope still returns 202 and runs (`tech.md` trap 1). Signature is the stub sentinel until core-signing exists (below). Seam wording agreed with the image-service writer, 2026-08-27. |
| provides | **art-and-image-seam** (backend) | the meaning of the seven envelope fields, the classification stance, the `content_class` vocabulary | Governance owns what the fields **mean**; the art seam owns the **transport** (`newGovEnvelope` and its call sites live in the art seam's files — `imageclient.go`, `imagehandler.go`; `tech.md` §The classification path). The art seam never invents classification values; this domain never re-decides transport discipline (issued_at pinning, idempotency, polling). Agreed with the art-seam writer, 2026-08-27. |
| provides | **every seat-owning domain** (play-loop, world-genesis, NPC cognition) | the latitude block, byte-identical, immediately after the role line | Seat owners must not paraphrase, trim, or drop it — *"two seats with differently-worded thresholds disagree mid-scene"* (`ADR-P022`). Adding a prompt file means carrying the block; `promptlatitude_test.go` fails the suite otherwise. The block's text is founder-ruled (`tech.md` §may-not-decide-alone 3). |

## What this domain consumes

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| consumes | **provider content policy** (via image-service) | a provider's refusal, surfaced verbatim as `provider_content_rejected` | Provider moderation is *"outside our control and the only remaining limit"* (`ADR-P022`). The platform neither hides a rejection nor engineers around it — no fallback walk, no retry on another route (`image:ADR-I002` wave rules). This side owes the mirror: never resubmit the same content elsewhere to dodge a moderation verdict. The job's terminal `error_code` is the signal (`imageclient.go:529-536`). |
| consumes | **world-genesis / play surfaces** | the content whose regime is being decided | Nothing crosses yet beyond the constant: with no sharing built, every world is private by construction and the envelope says so. When a sharing surface exists, it consumes this domain's verdict — it does not classify. |

## The seams that do not exist

- **Core-signing.** The envelope's `signature` is `"stub-unsigned-v1"`; `StubSignatureVerifier`
  passes any non-empty string. The canonicalization is *"a cross-system contract with core that is
  not yet designed (`TODO(core-signing)`)"* and the platform will not invent it
  (`imageclient.go:71-75`; platform `AGENTS.md:71` documents the sentinel). Core owns the design;
  it is undesigned. Both writers state this identically, 2026-08-27. Building either half alone
  breaks the other's startup checks (enforce + stub is refused in the platform's live env).
- **Classification of real content.** No classifier, eligibility service, or `world_content_profile`
  exists (`tech.md` §What exists). The seam from world content to a classification verdict is paper
  only (the two schema JSONs). An agent asked to "make visibility real" is starting the PRD's phase 1,
  which needs a ruling — see `tech.md` open question 3.
- **Publishing / sharing.** No endpoint, column, or UI. The entire public-governed half of
  `ADR-P016` (review states, asset visibility, audit events, takedown) is law with no code behind it.
  Deliberate (`product.md` §not built), and the reason the private-constant envelope is honest today.
